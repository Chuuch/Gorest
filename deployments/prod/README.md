# Production ops notes

Companion files:

- `../../docker-compose.prod.yml` — Compose overlay (no MinIO/Mailpit; R2 + Resend)
- `./.env.example` — copy to `Gorest/.env.prod` on the VPS
- `../loki/loki.prod.yml` — 14-day log retention
- Flourish: `Flourish/docker-compose.prod.yml` + `Flourish/deployments/prod/.env.example`

## Bring-up (single VPS)

```bash
# once
docker network create flourish-edge

# API + DB + observability
cd /opt/gorest
docker compose -f docker-compose.yml -f docker-compose.prod.yml --env-file .env.prod up -d

# web + Caddy
cd /opt/flourish
docker compose -f docker-compose.yml -f docker-compose.prod.yml --env-file .env.prod up -d
```

## Env checklist (R2 / Resend / Sentry / Caddy)

| Item | Where | Notes |
| --- | --- | --- |
| Domain DNS | Cloudflare | `A`/`AAAA` → VPS IP; grey-cloud at first |
| `FLOURISH_SITE` | Flourish `.env.prod` | hostname only, e.g. `app.example.com` |
| `CADDY_EMAIL` | Flourish `.env.prod` | Let's Encrypt account email |
| `APP_API_URL` | Flourish `.env.prod` | `https://app.example.com/api/v1` |
| `GOREST_APP_PUBLIC_URL` | Gorest `.env.prod` | `https://app.example.com` (invite/reset links) |
| `GOREST_CORS_ALLOWED_ORIGINS` | Gorest `.env.prod` | `https://app.example.com` |
| `GOREST_AUTH_COOKIE_SECURE` | overlay forces `true` | HTTPS only |
| R2 bucket + API token | Cloudflare R2 | private bucket; custom domain → `GOREST_STORAGE_PUBLIC_ENDPOINT` |
| Resend domain | Resend + CF DNS | SPF/DKIM(/DMARC); `GOREST_MAILER_FROM=…@yourdomain` |
| Sentry DSN | Flourish build arg / `VITE_SENTRY_DSN` | optional; free Developer plan |
| Grafana | `127.0.0.1:3000` | SSH tunnel or Tailscale only |

### Known blocker before R2

`GOREST_STORAGE_BUCKET` is defined in `.env.example`, but `StorageConfig.Bucket` currently binds to env tag `ACCESS_KEY` (same as the access key). Fix that one-line tag to `BUCKET` before relying on R2 bucket names.

## Health + uptime

Already in the apps:

- API: `GET /api/v1/health` → `{"status":"ok"}`
- Web: `GET /healthz` (nginx)

**Uptime check** = an external service that hits those URLs every few minutes and emails/Slack you if they fail.

| Option | Free tier | Pick when |
| --- | --- | --- |
| **UptimeRobot** | ~50 monitors, 5‑min interval | Simplest; fine for agency launch |
| **Better Stack (Uptime)** | Limited free monitors + nicer incidents | If you want status page / on-call later |
| Cloudflare Health Checks | Paid add-on on many plans | Skip for now |

**Recommendation:** UptimeRobot, one HTTP(S) monitor on `https://app.example.com/api/v1/health` (and optionally `/healthz`). Keyword or status 200 is enough.

## Firewall (22 key-only, 80, 443)

Set this **on the Hetzner VPS** (host OS), not in Docker.

```bash
# Ubuntu/Debian example
sudo apt install ufw
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow OpenSSH          # 22 — use SSH keys only (PasswordAuthentication no)
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable
sudo ufw status
```

Hetzner Cloud Firewall (console → Firewalls) can mirror the same rules as a second layer. Do **not** publish Postgres/Grafana/MinIO to the world (prod overlay already binds Grafana to localhost and drops public API port).

## Staging

**Where:** usually the **same VPS** (second Compose project + subdomain) or a cheaper second CX box later.

**Extra cost:** subdomain is free (DNS). Same VPS ≈ €0 extra if RAM fits. Second VPS ≈ another €5–10/mo.

Example on one box:

```bash
# DNS: staging.example.com → same IP
# Separate dirs + env files + project name so containers/volumes do not clash

cd /opt/gorest-staging
docker compose -p gorest-staging \
  -f docker-compose.yml -f docker-compose.prod.yml \
  --env-file .env.staging up -d

cd /opt/flourish-staging
# FLOURISH_SITE=staging.example.com, separate edge network or shared with care
docker compose -p flourish-staging \
  -f docker-compose.yml -f docker-compose.prod.yml \
  --env-file .env.staging up -d
```

Use a **separate Postgres volume, R2 bucket prefix/bucket, Resend from-address, and JWT secrets**. Staging is optional until you have paying/external users; for internal agency use, a careful prod + backups is acceptable short-term.

## Loki retention

Local Loki had no delete/compactor retention (disk grows). Prod overlay mounts `deployments/loki/loki.prod.yml` with **14-day** retention (`retention_period: 336h` + compactor). Prometheus already keeps **15d** (`--storage.tsdb.retention.time=15d`).

## Deploy pipeline

**Not automated yet.** CI runs tests (`.github/workflows/ci.yml`); there is no “build image → push → SSH → migrate → restart” workflow.

Minimal target:

1. On push to `main` / tag: build & push `ghcr.io/.../gorest` and `flourish`
2. SSH to VPS: `docker compose pull && migrate && up -d`
3. Health-check curl; rollback = previous image tag

Until that exists: manual `docker build` / `compose build` on the box is fine for first deploy.

## Rate limits

**Already wired.** Auth login/refresh (and related) use in-memory limiters (`internal/platform/middleware/rate_limit.go`), e.g. login 10 / 15m. No extra work for launch. Note: limits are per-process (fine on one API replica).

## Legal (agency-first)

Internal agency use → privacy/ToS can wait. Add when you invite external customers or take payment. Keep a short internal note of what data you store (accounts, files, tickets).

## Restore drill

**Meaning:** prove backups work by restoring once on purpose (staging or throwaway DB), not only writing backups.

**Setup later (after first prod backup exists):**

1. Nightly `pg_dump` (cron) → encrypted copy off-box (R2/Backblaze)
2. Once: restore dump into a scratch database, boot API against it, spot-check login + one client
3. Write the exact commands in this folder when you add the backup script

**When:** within the first week of having real data on prod—not before the first successful backup.

## Suggested order

1. VPS + UFW + DNS + Caddy TLS  
2. Fix storage `BUCKET` env tag → R2  
3. Resend domain verify  
4. Prod compose up + migrate + seed/admin user  
5. UptimeRobot on `/api/v1/health`  
6. Postgres backup cron  
7. One restore drill  
8. Optional: staging subdomain, deploy workflow, Sentry DSN  
