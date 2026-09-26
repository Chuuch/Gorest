ALTER TABLE tasks
  ADD COLUMN created_by UUID REFERENCES users(id) ON DELETE RESTRICT,
  ADD COLUMN assignee_id UUID REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX idx_tasks_created_by ON tasks (created_by);
CREATE INDEX idx_tasks_assignee_id ON tasks (assignee_id);
