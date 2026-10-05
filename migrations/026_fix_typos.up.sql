-- Fix typo creatd_at → created_at on ticket_comments
DO $$
BEGIN
  IF EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_name = 'ticket_comments' AND column_name = 'creatd_at'
  ) AND NOT EXISTS (
    SELECT 1
    FROM information_schema.columns
    WHERE table_name = 'ticket_comments' AND column_name = 'created_at'
  ) THEN
    ALTER TABLE ticket_comments RENAME COLUMN creatd_at TO created_at;
  END IF;
END $$;

-- Fix tasks status typo in_progres → in_progress
ALTER TABLE tasks DROP CONSTRAINT IF EXISTS tasks_status_check;

UPDATE tasks SET status = 'in_progress' WHERE status = 'in_progres';

ALTER TABLE tasks
  ADD CONSTRAINT tasks_status_check
  CHECK (status IN ('todo', 'in_progress', 'done'));
