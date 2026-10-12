ALTER TABLE notifications
  ADD COLUMN client_id UUID REFERENCES clients(id) ON DELETE SET NULL,
  ADD COLUMN project_id UUID REFERENCES projects(id) ON DELETE SET NULL;

CREATE INDEX notifications_client_id_ix
  ON notifications (client_id)
  WHERE client_id IS NOT NULL;

CREATE INDEX notifications_project_id_ix
  ON notifications (project_id)
  WHERE project_id IS NOT NULL;
