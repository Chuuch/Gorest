ALTER TABLE tasks ADD COLUMN ticket_id UUID REFERENCES tickets(id) ON DELETE SET NULL;

CREATE INDEX idx_tasks_ticket_id ON tasks (ticket_id);

CREATE UNIQUE INDEX idx_tasks_project_ticket_unique
  ON tasks (project_id, ticket_id)
  WHERE ticket_id IS NOT NULL;
