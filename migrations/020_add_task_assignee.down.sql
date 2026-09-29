DROP INDEX IF EXISTS idx_tasks_assignee_id;
DROP INDEX IF EXISTS idx_tasks_created_by;

ALTER TABLE tasks
  DROP COLUMN IF EXISTS assignee_id,
  DROP COLUMN IF EXISTS created_by;
