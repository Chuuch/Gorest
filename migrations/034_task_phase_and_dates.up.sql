ALTER TABLE tasks
  ADD COLUMN phase_id UUID REFERENCES project_phases(id) ON DELETE SET NULL,
  ADD COLUMN starts_on DATE,
  ADD COLUMN due_on DATE;

ALTER TABLE tasks
  ADD CONSTRAINT tasks_dates_check
  CHECK (starts_on IS NULL OR due_on IS NULL OR starts_on <= due_on);

CREATE INDEX idx_tasks_phase_id ON tasks (phase_id);
CREATE INDEX idx_tasks_project_starts_on ON tasks (project_id, starts_on);
