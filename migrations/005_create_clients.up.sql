CREATE TABLE clients (
  id UUID PRIMARY KEY,
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  notes TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  CONSTRAINT clients_org_name_unique UNIQUE (organization_id, name)
);

CREATE INDEX idx_clients_organization_id ON clients (organization_id);
