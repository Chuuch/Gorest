DROP INDEX IF EXISTS idx_tasks_project_ticket_unique;
DROP INDEX IF EXISTS idx_tasks_ticket_id;
ALTER TABLE tasks DROP COLUMN IF EXISTS ticket_id;
