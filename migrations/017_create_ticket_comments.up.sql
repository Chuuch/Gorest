CREATE TABLE ticket_comments (
  id UUID PRIMARY KEY,
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  ticket_id UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  body TEXT NOT NULL,
  creatd_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  CONSTRAINT ticket_comments_body_check CHECK (char_length(body) >= 1 AND char_length(body) <= 2000)
);

CREATE INDEX idx_ticket_comments_organization_id ON ticket_comments (organization_id);
CREATE INDEX idx_ticket_comments_ticket_id ON ticket_comments (ticket_id);
CREATE INDEX idx_ticket_comments_user_id ON ticket_comments (user_id);
