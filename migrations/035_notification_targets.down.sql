DROP INDEX IF EXISTS notifications_project_id_ix;
DROP INDEX IF EXISTS notifications_client_id_ix;

ALTER TABLE notifications
  DROP COLUMN IF EXISTS project_id,
  DROP COLUMN IF EXISTS client_id;
