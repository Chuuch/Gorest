CREATE TABLE time_entries (
  id UUID PRIMARY KEY,
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
  user_id UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  minutes INTEGER NOT NULL,
  notes TEXT NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  CONSTRAINT time_entries_minutes_check CHECK (minutes >= 1 AND minutes <= 1440)
);

CREATE INDEX idx_time_entries_organization_id ON time_entries (organization_id);
CREATE INDEX idx_time_entries_task_id ON time_entries (task_id);
CREATE INDEX idx_time_entries_user_id ON time_entries (user_id);
