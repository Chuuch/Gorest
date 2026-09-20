CREATE TABLE tickets (
  id UUID PRIMARY KEY,
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  client_id UUID NOT NULL REFERENCES clients(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  kind TEXT NOT NULL,
  status TEXT NOT NULL,
  title TEXT NOT NULL,
  body TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  CONSTRAINT tickets_kind_check CHECK (kind IN ('bug', 'feature', 'question', 'other')),
  CONSTRAINT tickets_status_check CHECK (status IN ('open', 'in_progress', 'resolved', 'closed')),
  CONSTRAINT tickets_title_check CHECK (char_length(title) >= 4 AND char_length(title) <= 100),
  CONSTRAINT tickets_body_check CHECK (char_length(body) >= 1 AND char_length(body) <= 2000)
);

CREATE INDEX idx_tickets_organization_id ON tickets (organization_id);
CREATE INDEX idx_tickets_client_id ON tickets (client_id);
