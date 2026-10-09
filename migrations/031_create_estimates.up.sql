CREATE TABLE estimate_catalog_versions (
  id UUID PRIMARY KEY,
  organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
  version TEXT NOT NULL,
  currency TEXT NOT NULL DEFAULT 'EUR',
  floor_cents_per_hour INTEGER NOT NULL CHECK (floor_cents_per_hour > 0),
  target_cents_per_hour INTEGER NOT NULL CHECK (target_cents_per_hour > 0),
  target_multiplier_bps INTEGER NOT NULL DEFAULT 11000 CHECK (target_multiplier_bps > 0),
  created_at TIMESTAMPTZ NOT NULL,
  CONSTRAINT estimate_catalog_versions_currency_check CHECK (currency = 'EUR')
);

CREATE UNIQUE INDEX estimate_catalog_versions_platform_version_ux
  ON estimate_catalog_versions (version)
  WHERE organization_id IS NULL;

CREATE UNIQUE INDEX estimate_catalog_versions_org_version_ux
  ON estimate_catalog_versions (organization_id, version)
  WHERE organization_id IS NOT NULL;

CREATE TABLE estimate_coefficients (
  id UUID PRIMARY KEY,
  catalog_version_id UUID NOT NULL REFERENCES estimate_catalog_versions(id) ON DELETE CASCADE,
  category TEXT NOT NULL,
  mode TEXT NOT NULL DEFAULT 'project' CHECK (mode IN ('project', 'retainer')),
  key TEXT NOT NULL,
  value_numeric NUMERIC NOT NULL,
  meta_json JSONB NOT NULL DEFAULT '{}'::jsonb,
  CONSTRAINT estimate_coefficients_category_key_mode_unique UNIQUE (catalog_version_id, category, mode, key)
);

CREATE INDEX estimate_coefficients_catalog_ix ON estimate_coefficients (catalog_version_id);

CREATE TABLE estimate_runs (
  id UUID PRIMARY KEY,
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  created_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  intake_id UUID,
  client_id UUID REFERENCES clients(id) ON DELETE SET NULL,
  category TEXT NOT NULL,
  mode TEXT NOT NULL CHECK (mode IN ('project', 'retainer')),
  catalog_version_id UUID NOT NULL REFERENCES estimate_catalog_versions(id) ON DELETE RESTRICT,
  currency TEXT NOT NULL DEFAULT 'EUR',
  input_json JSONB NOT NULL,
  result_json JSONB NOT NULL,
  estimated_timeline_days INTEGER,
  hours_per_month NUMERIC,
  minimum_price_cents INTEGER NOT NULL,
  recommended_price_cents INTEGER NOT NULL,
  risk_label TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  CONSTRAINT estimate_runs_currency_check CHECK (currency = 'EUR')
);

CREATE INDEX estimate_runs_org_created_at_id_ix
  ON estimate_runs (organization_id, created_at DESC, id DESC);

CREATE INDEX estimate_runs_org_client_ix
  ON estimate_runs (organization_id, client_id)
  WHERE client_id IS NOT NULL;

ALTER TABLE projects
  ADD COLUMN estimate_run_id UUID REFERENCES estimate_runs(id) ON DELETE SET NULL,
  ADD COLUMN estimated_hours NUMERIC,
  ADD COLUMN target_end_date DATE;

CREATE INDEX projects_estimate_run_id_ix ON projects (estimate_run_id)
  WHERE estimate_run_id IS NOT NULL;

INSERT INTO estimate_catalog_versions (
  id, organization_id, version, currency,
  floor_cents_per_hour, target_cents_per_hour, target_multiplier_bps, created_at
) VALUES (
  '00000000-0000-4000-8000-000000000031',
  NULL, 'v1', 'EUR', 1800, 3500, 11000,
  TIMESTAMPTZ '2026-01-01 00:00:00+00'
);
