ALTER TABLE tasks DROP CONSTRAINT IF EXISTS tasks_status_check;
ALTER TABLE tasks
  ADD CONSTRAINT tasks_status_check
  CHECK (status IN ('todo', 'in_progres', 'done'));

UPDATE tasks SET status = 'in_progres' WHERE status = 'in_progress';

DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_name = 'ticket_comments' AND column_name = 'created_at'
  ) AND NOT EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_name = 'ticket_comments' AND column_name = 'creatd_at'
  ) THEN
    ALTER TABLE ticket_comments RENAME COLUMN created_at TO creatd_at;
  END IF;
END $$;
