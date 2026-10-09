# Estimator v1 — engineering breakdown

Status: **planning** (no product code until this doc is approved).  
Related legacy: `zyntera-operations` Estimator (reference only — do not port as-is).  
Frontend: Flourish (`features/estimates`).

---

## 1. Product intent

Internal (and later possibly client-facing) decision tool that turns a **category questionnaire** into:

- estimated effort (hours, or hours/month for retainers)
- timeline
- floor vs recommended price (**EUR** in v1)
- risk + human-readable drivers

Primary v1 actions after a run:

1. **Save run**
2. **Create project** (linked to the run)
3. **Copy / print-friendly summary** (for sharing; not a full Offers module)

**Not** invoices: existing invoices remain time-and-materials billing from `time_entries`. Do not overload them as proposals.

---

## 2. Locked decisions

| Topic | Decision |
| --- | --- |
| Rates | Configurable **catalog** from day 1; seed placeholders; editable later |
| Categories | Website, Design, Marketing, Custom Web, Custom Mobile, Branding |
| Marketing | Both **project** (campaign) and **retainer** (`mode`) |
| Offers / quotes | **Out of v1** |
| Capacity / next slot | **Defer** |
| Estimator currency | **EUR only** (org multi-currency stays for invoices/settings) |
| Engine location | **Server-side** (Gorest); Flourish is UI only |
| Dual engines | **Forbidden** — one formula path |
| Intake UX | Questionnaire/wizard **per category** — draft fields in [`estimator-questionnaires-v1.md`](./estimator-questionnaires-v1.md) |
| Who fills scope | **Customer** via agency-sent link; agency then runs Estimator on the submission |
| What customer sees | Scope questions only — **no** prices / internal rates / risk scores |

---

## 3. Roadmap (phases)

| Phase | Name | Notes |
| --- | --- | --- |
| **1a** | Engine + catalog + estimate runs + staff review UI | Core math + “open submission → estimate” |
| **1b** | Customer intake links | Magic-link questionnaire; submit → intake record |
| **2** | Offers / quotes | Commercial artifact; accept → project |
| **3** | Capacity | Dept caps, schedule impact |
| **4** | Calibration | Estimate vs `time_entries` |
| **5** | Delivery plan calendar | Spread estimate into tasks/milestones; track deadline |

**Canonical flow (locked):**

```text
Enquiry
  → Agency creates intake link (category fixed or chosen)
  → Customer opens link, fills questionnaire, submits
  → Agency opens submission in Estimator
  → Staff adds overlays (complexity, urgency, notes)
  → Engine → hours / price / risk
  → Save run · Create project · Copy summary
```

---

## 4. Domain model (v1)

### 4.1 Tables (suggested)

```text
estimate_catalog_versions
  id, organization_id NULLABLE  -- NULL = platform default seed
  version INT / semver TEXT
  currency CHAR(3)              -- 'EUR' for v1
  rates_json JSONB              -- floor_cents_per_hour, target_cents_per_hour, target_multiplier_bps
  created_at

estimate_coefficients
  id, catalog_version_id
  category TEXT                 -- website | design | marketing | custom_web | custom_mobile | branding
  mode TEXT                     -- project | retainer (marketing uses both; others usually project)
  key TEXT                      -- e.g. base.landing_page, adder.multilingual, mult.complexity.high
  value_numeric NUMERIC         -- hours or multiplier
  meta_json JSONB               -- optional labels, clamp min/max

estimate_intakes                 -- customer submission (scope only)
  id, organization_id
  token_hash                     -- magic link; never store raw token
  category, mode
  status: pending | submitted | consumed | expired
  expires_at
  respondent_email NULLABLE
  respondent_name NULLABLE
  client_id NULLABLE             -- linked after staff claims / creates client
  scope_json JSONB               -- customer answers only
  submitted_at NULLABLE
  created_by_user_id             -- staff who sent the link
  created_at

estimate_runs
  id, organization_id, created_by_user_id
  intake_id NULLABLE            -- source questionnaire if any
  client_id NULLABLE
  category, mode
  catalog_version_id
  currency                      -- 'EUR'
  input_json JSONB              -- scope + staff overlays (complexity, urgency, …)
  result_json JSONB
  estimated_hours NUMERIC
  estimated_timeline_days INT NULLABLE
  hours_per_month NUMERIC NULLABLE
  minimum_price_cents INT
  recommended_price_cents INT
  risk_level TEXT
  created_at

ALTER TABLE projects
  ADD COLUMN estimate_run_id UUID NULL REFERENCES estimate_runs(id),
  ADD COLUMN estimated_hours NUMERIC NULL,
  ADD COLUMN target_end_date DATE NULL;
```

Money: store **integer cents** (align with invoices). Display EUR in UI.

### 4.2 Engine output (result_json shape)

```text
estimated_hours | hours_per_month
estimated_timeline_days (project mode)
minimum_price_cents, recommended_price_cents, currency
eur_per_hour (derived)
risk_level, risk_factors[]
drivers[], expensive_factors[]
capacity_impact: omitted in v1
catalog_version
```

### 4.3 Modes

| Mode | Categories | Pricing semantics |
| --- | --- | --- |
| `project` | All | Total €, timeline days |
| `retainer` | Marketing (primary) | €/month, hours/month; timeline optional/N/A |

---

## 5. Engine design

Package: `internal/estimator` (pure calculation + catalog load).

```text
Estimate(input, catalog) → Result
```

Rules:

- Deterministic; no DB I/O inside pure calc (catalog passed in).
- Category modules: one file/strategy per category (shared multipliers for complexity/urgency).
- Coefficients **only** from catalog (no magic numbers in Go except fallbacks that fail closed in prod if catalog missing).
- Seed v1 coefficients from cleaned zyntera `logic.ts` where applicable; invent sensible seeds for Custom Mobile / Branding / Custom Web until questionnaires are finalized.
- Website-style **clamps**: either make clamps catalog-driven and documented, or drop silent clamps in v1 (prefer: catalog `clamp_min` / `clamp_max` per subtype, visible in drivers when applied).

**Do not** reimplement a second path in Flourish.

---

## 6. HTTP API (v1)

All staff-authenticated (`RequireAuth` + staff), org-scoped.

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/api/v1/estimate-catalog` | Active catalog for org (fallback platform seed) |
| `POST` | `/api/v1/estimates/preview` | Stateless calculate (no persist) |
| `POST` | `/api/v1/estimates` | Persist `estimate_runs` |
| `GET` | `/api/v1/estimates` | List runs (filter client_id, category) |
| `GET` | `/api/v1/estimates/{id}` | Get run |
| `POST` | `/api/v1/estimates/{id}/create-project` | Create project + link `estimate_run_id` |

Idempotency: use existing idempotency middleware on POSTs that mutate.

Catalog **edit** API: optional in v1 (seed + SQL/admin later). Minimum: read path. Prefer a thin `PATCH` for rates only if cheap.

---

## 7. Flourish UI (v1)

Feature: `src/features/estimates/` (+ public intake route outside staff shell)

### Staff

1. **Send intake** — pick category/mode → generate link → copy/email to customer  
2. **Inbox** — list submitted intakes  
3. **Open in Estimator** — prefill scope from `scope_json`  
4. Set **staff overlays** (complexity, urgency, internal notes)  
5. Preview → Save run · Create project · Copy summary  

Routes (suggested):

- `/estimates` — runs  
- `/estimates/intakes` — sent / submitted questionnaires  
- `/estimates/intakes/:id` — review + “Run estimator”  
- `/estimates/:id` — saved run detail  

### Customer (public)

- `/intake/:token` — questionnaire only; thank-you on submit  
- No login, no prices, no risk, no catalog  

i18n: staff + public strings. Rate-limit + expire tokens (e.g. 14 days).

---

## 8. Work tickets (implementation order)

Approve this list before coding.

**1a — engine & staff review**

1. Migrations — catalog, coefficients, estimate_runs, project columns  
2. Seed catalog  
3. Go engine + unit tests  
4. API — preview, persist runs, create-project  
5. Flourish — open/edit estimate from manual or intake-shaped input; results + actions  

**1b — customer intake**

6. Migrations — `estimate_intakes` + token security  
7. API — staff create intake / list; public get-by-token + submit (scoped, rate-limited)  
8. Flourish public `/intake/:token` wizard (scope fields only)  
9. Staff intakes inbox → “Run estimator”  
10. Freeze questionnaire JSON schema in docs  

Order: **1a can ship before 1b**, but product story is incomplete until 1b exists. Prefer building schema so intake `scope_json` == engine input minus staff overlays.

---

## 9. Test plan

- Unit: each category golden input → expected hours/price band (tolerance 0 if deterministic)  
- Unit: retainer vs project marketing divergence  
- Unit: catalog version stamped on result  
- API: preview does not write; persist writes one row  
- API: create-project links run and refuses cross-org  
- UI: smoke wizard → preview → save (Vitest / Playwright later)  

No capacity tests in v1.

---

## 10. Open product decisions (next conversations)

### 10.1 Questionnaires (fields)

Draft field lists + coefficient key mapping:  
→ **[`estimator-questionnaires-v1.md`](./estimator-questionnaires-v1.md)** (in review).

### 10.2 Intake channel — **locked**

**Customer fills the questionnaire; agency estimates from that submission.**

Mechanism: staff-created **magic link** (token URL), no customer account required (works for first enquiry). Closest to former option **D/C hybrid**.

**Intake ops — locked / advised**

| Topic | Decision |
| --- | --- |
| Notify staff on submit | **Yes** — in-app notification (+ optional email later) |
| Create `clients` row on submit | **No** — intake stays a lead until commercial agreement |
| When to create client | After offer accepted / kickoff (Phase 2+); staff may create earlier only if they choose |
| Portal | **After** client exists — invite `client_users`; portal is not for first-contact intake |
| Token TTL | **14 days**, single-submit |
| Customer confirmation email | **Yes** (Resend) on successful submit |
| Existing clients later | Same magic-link pattern in v1; optional portal-hosted form later |

---

## 11. Explicit non-goals (v1)

- Offers/quotes module  
- Capacity forecasting  
- Generating invoices from estimates  
- Multi-currency estimates  
- Plan calendar / task spread (Phase 5)  
- Customer-visible pricing on intake  
- Portal-only intake (v1 uses magic links)  
- Learning/ML pricing  

---

## 12. Approval checklist

- [ ] Domain tables + API list accepted  
- [ ] Seed rate placeholders accepted (editable later)  
- [x] Intake = customer link → staff estimator (no price on form)  
- [x] Questionnaire fields frozen (`estimator-questionnaires-v1.md`)  
- [x] Intake ops locked — §10.2  

When checkboxes are done → implement tickets §8 in order (1a then 1b).
