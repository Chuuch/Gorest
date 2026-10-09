# Estimator questionnaires — v1 draft

Status: **v1 fields frozen** (C/S split locked; coefficient *numbers* still placeholders in catalog).  
Parent: [`estimator-v1.md`](./estimator-v1.md).

Purpose: define the **questions** per category and how answers map to engine coefficient keys.  
Numbers (hours/multipliers) stay in the **catalog** — this doc only defines *what we ask* and *which keys they touch*.

### Decisions locked (workshop)

| Topic | Decision |
| --- | --- |
| Ecommerce | Always under **Website** (WordPress / WooCommerce-style), never Custom Web |
| Design & Branding | **Both**: Website add-on toggles *and* standalone Design / Branding categories |
| Marketing modes | Retainer vs project field split **approved** |
| Who fills scope | **Customer** via magic-link intake |
| Pricing visibility | Customer **never** sees price/risk on the form; staff only after Estimator |

---

## Mix model (old Estimator data → questionnaire + staff)

Same **engine inputs** as Zyntera; different **who answers**.

```text
┌─────────────────────────────┐
│ 1. Staff creates intake link│  (picks category / marketing mode)
└──────────────┬──────────────┘
               ▼
┌─────────────────────────────┐
│ 2. Customer questionnaire   │  factual scope they know
│    → scope_json             │  (no prices, no complexity jargon)
└──────────────┬──────────────┘
               ▼
┌─────────────────────────────┐
│ 3. Staff Estimator review   │  overlays + optional scope edits
│    → merge → Estimate()     │  → hours / € / risk
└─────────────────────────────┘
```

**Rule of thumb**

| Ask the **customer** if… | Keep **staff-only** if… |
| --- | --- |
| They know the fact (pages, languages, channels, content ready) | It’s agency judgment (complexity, urgency banding, rates) |
| Plain language (“Do you need a shop?”) | Internal jargon (“Custom complexity ×1.25”) |
| Choosing what they *want included* | Gaming risk (don’t let them set urgency to cheapen/stress timeline alone without staff) |
| Contact / deadline preference | Client record, catalog, double-count guards |

Staff can always **edit** customer answers after submit (wrong page count, etc.) before calculating.

---

## Split matrix — from old Zyntera fields

Legend: **C** = customer questionnaire · **S** = staff overlay / Estimator · **L** = set on intake **link** by staff · **—** = drop for v1

### Shared / meta

| Old field | Who | Notes |
| --- | --- | --- |
| Client / company name | **C** | As respondent + company; no `clients` row yet |
| Project name | **C** | |
| Category | **L** | Staff picks when sending link (Website enquiry, etc.) |
| Complexity | **S** | Agency judgment after reading scope |
| Urgency | **S** | Staff sets; customer only gives `preferred_deadline` |
| Selected services (catalog) | **—** | Dropped on intake (was unused in hour math). Staff may tag later for Offers |
| Notes | **C** + **S** | `customer_notes` vs `internal_notes` |
| *(new)* email | **C** | Needed for confirmation / follow-up |
| *(new)* preferred deadline | **C** | Soft signal → staff urgency |

### Website (old Website/Design tab)

| Old field | Who | Customer wording (draft) |
| --- | --- | --- |
| Website type | **C** | “What kind of site?” Landing / company site / online shop |
| Page count | **C** | “Roughly how many pages?” |
| Multilingual | **C** | “More than one language?” |
| CMS required | **S** or default | Agency default **yes** for Website (WP/Woo); don’t bother customer unless useful |
| Content ready | **C** | “Do you already have final text & images?” |
| SEO setup | **C** | “Include basic SEO setup?” (want / don’t want) |
| UI/UX included | **C** | “Include UI/UX design in this project?” |
| Branding included | **C** | “Include branding (logo/identity)?” |
| Expected revisions | **C** soft or **S** | Prefer **S**: customers under-estimate rounds. Optional soft ask: “How many review rounds do you expect?” |
| Custom features checklist | **C** | Plain checklist (login, payments, booking…) — maps to adders; Custom Web link if “app-like” |

### Design (standalone — new emphasis)

| Field | Who | Notes |
| --- | --- | --- |
| Size / screens / deliverables | **C** | What they need designed |
| Revisions / brand exists | **C** or **S** | Facts **C**; scoring **S** |

### Branding (standalone)

| Field | Who | Notes |
| --- | --- | --- |
| Package / deliverables | **C** | Logo only vs full identity |
| Stakeholder count | **C** | Affects risk; plain ask |

### Marketing

| Old field | Who | Notes |
| --- | --- | --- |
| Channels | **C** | “Which channels?” |
| Creatives / month | **C** | Retainer |
| Reporting level | **C** soft → **S** refine | Customer: basic/detailed; staff maps to Basic/Standard/Advanced |
| Campaign complexity | **S** | Agency judgment |
| Landing page support | **C** | Want included? |
| Mode project vs retainer | **L** (or **C** if both offered on same link) | Staff usually chooses link type |

### Custom Web / Mobile (new vs old “Custom Web App”)

| Field | Who | Notes |
| --- | --- | --- |
| Platforms, auth, payments, etc. | **C** | Capability checklist in plain language |
| App scale mvp/growth/complex | **S** | Derived from answers + staff confirm |
| Tech approach (Flutter, etc.) | **S** | Agency decision |

### Never on customer form

Rates · hours · prices · risk scores · capacity · catalog coefficients · `client_id`

---

## Visibility rules

| Audience | Scope | Price / risk / hours |
| --- | --- | --- |
| Customer (intake) | **C** fields only | **no** |
| Staff (Estimator) | **C** (editable) + **S** overlays | **yes** |

---

## 1. Website (`category: website`, `mode: project`) — **locked**

Classic / CMS sites (WordPress, WooCommerce, etc.). Full custom apps → **Custom Web**.  
**Ecommerce** = Website subtype (shop on CMS), not Custom Web.

### Questions

| Field | Who | Type | Options / wording | Coefficient key(s) |
| --- | --- | --- | --- | --- |
| `website_type` | **C** | select | Landing · company site · online shop | `base.website.landing` / `.corporate` / `.ecommerce` |
| `page_count` | **C** | int ≥ 1 | Rough page count | `adder.website.extra_page` × max(0, pages−5) |
| `multilingual` | **C** | bool | More than one language? | `adder.website.multilingual` |
| `content_ready` | **C** | bool | Texts & images ready? (yes/no) | `adder.website.content_not_ready` if false |
| `seo_setup` | **C** | bool | Include basic SEO setup? | `adder.website.seo` |
| `uiux_included` | **C** | bool | Include UI/UX in this project? | `adder.website.uiux` |
| `branding_included` | **C** | bool | Include branding? | `adder.website.branding` |
| `integrations` | **C** | multi | CRM/forms, newsletter, analytics, booking, payments, maps, live chat, other | `adder.website.integration` × n |
| `cms_required` | **S** | bool | **default true** (WP/Woo) — not asked | `adder.website.cms` |
| `expected_revisions` | **S** | select | default medium | `adder.website.revisions.*` |

### Packaging note (Design / Branding)

- Website toggles on → bundled estimate.  
- Standalone Design/Branding → separate intake/category.  
- Avoid double-count if both a Website add-on and a standalone run exist.

---

## 2. Design (`category: design`, `mode: project`) — **locked**

UI/UX only (no build). Coexists with Website’s `uiux_included` toggle (see packaging note above).

### Questions

| Field | Who | Type | Options / wording | Coefficient key(s) |
| --- | --- | --- | --- | --- |
| `deliverables` | **C** | multi | wireframes · hi-fi UI · design system · prototype · handoff | `adder.design.*` |
| `screen_count` | **C** | int | Rough number of screens | `adder.design.extra_screen` (threshold) |
| `brand_guidelines_exist` | **C** | bool | Already have brand guidelines? | `adder.design.no_brand` if false |
| `includes_mobile` | **C** | bool | Design for mobile | `adder.design.mobile` |
| `includes_desktop` | **C** | bool | Design for desktop (default on) | — |
| `design_size` | **S** | select | small · medium · large | `base.design.*` (staff; may auto-derive from screens later) |
| `expected_revisions` | **S** | select | default medium | `adder.design.revisions.*` |

Customer is **not** asked `design_size` (avoids double-asking with screen count).

---

## 3. Branding (`category: branding`, `mode: project`) — **locked**

Standalone brand work. Coexists with Website’s `branding_included` toggle (see packaging note under Website).

### Questions

| Field | Who | Type | Options / wording | Coefficient key(s) |
| --- | --- | --- | --- | --- |
| `package` | **C** | select | Logo only · Identity · Full brand | `base.branding.logo` / `.identity` / `.full` |
| `deliverables` | **C** | multi | logo, palette, typography, guidelines PDF, social kit, business cards, … | `adder.branding.*` |
| `has_existing_brand` | **C** | bool | Refreshing an existing brand? | refresh vs greenfield |
| `competitor_research` | **C** | bool | Include competitor/brand research? | `adder.branding.research` |
| `stakeholder_count` | **C** | select | 1–2 · 3–5 · 6+ sign-off | risk + optional adder |
| `expected_revisions` | **S** | select | default medium | `adder.branding.revisions.*` |

---

## 4. Custom Web (`category: custom_web`, `mode: project`) — **locked**

Web apps / complex products (auth, roles, dashboards, etc.).

### Questions

| Field | Who | Type | Options / wording | Coefficient key(s) |
| --- | --- | --- | --- | --- |
| `platforms` | **C** | multi | User-facing app · **Admin panel** · Marketing site | `adder.custom_web.platform.*` |
| `auth` | **C** | select | none · email · SSO/OAuth · both | `adder.custom_web.auth.*` |
| `roles_permissions` | **C** | bool | “Different user roles?” | `adder.custom_web.rbac` |
| `payments` | **C** | select | none · one-time · subscriptions · marketplace | `adder.custom_web.payments.*` |
| `integrations` | **C** | multi | email, CRM, analytics, accounting, maps, other API | `adder.custom_web.integration` × n |
| `realtime` | **C** | bool | Live updates / instant notifications | `adder.custom_web.realtime` |
| `file_uploads` | **C** | bool | Users upload files/images | `adder.custom_web.files` |
| `multilingual` | **C** | bool | More than one language | `adder.custom_web.i18n` |
| `design_included` | **C** | bool | Include UI/UX design in this project | `adder.custom_web.design` |
| `content_ready` | **C** | bool | Yes/no: texts & images ready? | risk + small adder if false |
| `app_scale` | **S** | select | mvp · growth · complex | `base.custom_web.*` |
| `expected_revisions` | **S** | select | default medium | `adder.custom_web.revisions.*` |
| `hosting_devops` | **S** | select | none · basic deploy · CI/CD + staging | `adder.custom_web.devops.*` |

**Dropped:** separate `admin_dashboard` — covered by platforms → Admin panel.  
**`content_ready`:** customer yes/no only (locked).  
Design-in-run via `design_included` is enough for v1.

---

## 5. Custom Mobile (`category: custom_mobile`, `mode: project`) — **locked**

Always built with **React Native** (not asked — fixed catalog assumption).

### Questions

| Field | Who | Type | Options / wording | Coefficient key(s) |
| --- | --- | --- | --- | --- |
| `platforms` | **C** | multi | iOS · Android · both | `adder.custom_mobile.platform.*` |
| `backend` | **C** | select | None · We have an API · Build API for us | `adder.custom_mobile.backend.*` |
| `auth` | **C** | select | none · email · SSO · both | `adder.custom_mobile.auth.*` |
| `offline` | **C** | bool | Works without internet? | `adder.custom_mobile.offline` |
| `push_notifications` | **C** | bool | | `adder.custom_mobile.push` |
| `payments_in_app` | **C** | bool | In-app purchases / payments | `adder.custom_mobile.iap` |
| `store_release` | **C** | bool | Submit to App Store / Play for you? | `adder.custom_mobile.store` |
| `design_included` | **C** | bool | Include UI/UX design? | `adder.custom_mobile.design` |
| `device_features` | **C** | multi | camera, maps, bluetooth, biometrics | `adder.custom_mobile.device.*` |
| `content_ready` | **C** | bool | Texts & images ready? (yes/no) | same as web |
| `app_scale` | **S** | select | mvp · growth · complex | `base.custom_mobile.*` |
| `expected_revisions` | **S** | select | default medium | `adder.custom_mobile.revisions.*` |

**Dropped as a question:** `tech_approach` — constant `react_native` in catalog/seed.

---

## 6. Marketing (`category: marketing`) — **locked**

Mode (`retainer` | `project`) set on intake **link** by staff (**L**).

### Retainer (monthly)

| Field | Who | Type | Options / wording | Coefficient key(s) |
| --- | --- | --- | --- | --- |
| `channels` | **C** | multi | Facebook · Google · SEO · Social · Email | `adder.marketing.channel` × n (cap) |
| `creatives_per_month` | **C** | int | | `adder.marketing.creative` × n (cap) |
| `reporting_detail` | **C** | select | Basic overview · Detailed | soft; staff maps to reporting level |
| `landing_page_support` | **C** | bool | Include landing page support? | `adder.marketing.landing_support` |
| `ad_spend_band` | **C** | select | optional &lt;1k · 1–5k · 5k+/mo | risk / optional hours |
| `reporting_level` | **S** | select | basic · standard · advanced | drives tier |
| `campaign_complexity` | **S** | select | low · medium · high | drives tier |
| `tier` | **S** | derived | from reporting + complexity (legacy-style) | `base.marketing.retainer.*` |

Output: **hours/month** + **€/month**.

### Project (finite campaign)

| Field | Who | Type | Options / wording | Coefficient key(s) |
| --- | --- | --- | --- | --- |
| `campaign_goal` | **C** | select | launch · lead gen · awareness · migration | base / informational |
| `channels` | **C** | multi | same as retainer | channel adders |
| `duration_weeks` | **C** | int | How many weeks? | timeline / scale |
| `creatives_one_shot` | **C** | int | Creatives needed | `adder.marketing.creative` |
| `setup_includes` | **C** | multi | pixel/analytics · landing page · email sequences · creative pack | `adder.marketing.setup.*` |
| `reporting_detail` | **C** | select | Basic · Detailed | soft |
| `reporting_level` | **S** | select | | |
| `campaign_complexity` | **S** | select | | multipliers |

Output: **total hours** + **total €** + timeline days.

---

## Coefficient key naming convention

```text
base.<category>[.<subtype>]
adder.<category>.<feature>
mult.complexity.<low|medium|high|custom>
mult.urgency.<normal|fast|urgent>
clamp.<category>.<subtype>.min|max
rate.floor_cents_per_hour
rate.target_cents_per_hour
rate.target_multiplier_bps
```

All numeric values live in `estimate_coefficients` / catalog JSON — not hardcoded in the UI.

---

## Wizard UX notes (Flourish)

- One category per run (v1). Bundles = multiple runs or Phase 2 Offers with line items.  
- Show **live preview** after enough required fields (or explicit Calculate).  
- Progressive disclosure: hide retainer fields in project mode and vice versa.  
- Help text under each control explaining “why we ask” (builds trust if later client-facing).

---

## Review checklist (this workshop)

- [x] Website vs Custom Web — ecommerce under Website (WP/Woo)  
- [x] UI/UX & Branding — Website add-on toggles **and** standalone categories  
- [x] Marketing retainer vs project field split  
- [x] Custom Web / Mobile / Branding / Design field walks  
- [x] Mobile tech = React Native (fixed)  
- [x] Intake = customer magic-link; staff Estimator; no auto-client; 14d TTL  
- [x] C vs S split frozen for v1  

---

## Next

1. Optional: machine-readable `input_json` schema snippet in this repo.  
2. Implement Phase **1a** (engine + staff review) then **1b** (intake) per [`estimator-v1.md`](./estimator-v1.md).
