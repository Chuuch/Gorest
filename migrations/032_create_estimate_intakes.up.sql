CREATE TABLE estimate_intakes (
  id UUID PRIMARY KEY,
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  created_by_user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  token_hash TEXT NOT NULL UNIQUE,
  category TEXT NOT NULL CHECK (category IN ('website', 'design', 'branding', 'custom_web', 'custom_mobile', 'marketing')),
  mode TEXT NOT NULL CHECK (mode IN ('project', 'retainer')),
  status TEXT NOT NULL CHECK (status IN ('pending', 'submitted', 'consumed', 'expired')),
  expires_at TIMESTAMPTZ NOT NULL,
  respondent_email TEXT,
  respondent_name TEXT,
  company_name TEXT,
  project_name TEXT,
  preferred_deadline DATE,
  client_id UUID REFERENCES clients(id) ON DELETE SET NULL,
  scope_json JSONB,
  submitted_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL,
  CONSTRAINT estimate_intakes_mode_category_check CHECK (
    mode = 'project' OR category = 'marketing'
  )
);

CREATE INDEX estimate_intakes_org_created_at_id_ix
  ON estimate_intakes (organization_id, created_at DESC, id DESC);

CREATE INDEX estimate_intakes_org_status_ix
  ON estimate_intakes (organization_id, status);

ALTER TABLE estimate_runs
  ADD CONSTRAINT estimate_runs_intake_id_fkey
  FOREIGN KEY (intake_id) REFERENCES estimate_intakes(id) ON DELETE SET NULL;
