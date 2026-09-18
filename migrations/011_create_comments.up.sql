CREATE TABLE comments (
  id UUID PRIMARY KEY,
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  body TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  CONSTRAINT comments_body_check CHECK (char_length(body) >= 1 AND char_length(body) <= 2000)
);

CREATE INDEX idx_comments_organization_id ON comments (organization_id);
CREATE INDEX idx_comments_task_id ON comments (task_id);
CREATE INDEX idx_comments_user_id ON comments (user_id);
