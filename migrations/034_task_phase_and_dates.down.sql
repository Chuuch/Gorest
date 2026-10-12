DROP INDEX IF EXISTS idx_tasks_project_starts_on;
DROP INDEX IF EXISTS idx_tasks_phase_id;

ALTER TABLE tasks
  DROP CONSTRAINT IF EXISTS tasks_dates_check;

ALTER TABLE tasks
  DROP COLUMN IF EXISTS due_on,
  DROP COLUMN IF EXISTS starts_on,
  DROP COLUMN IF EXISTS phase_id;
