CREATE TABLE client_users (
  id UUID PRIMARY KEY,
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  client_id UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  created_at TIMESTAMPTZ NOT NULL,
  CONSTRAINT client_users_user_id_unique UNIQUE (user_id)
);

CREATE INDEX idx_client_users_organizations_id ON client_users (organization_id);
CREATE INDEX idx_client_users_client_id ON client_users (client_id);
