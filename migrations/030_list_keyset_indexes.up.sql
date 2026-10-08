CREATE INDEX tickets_org_client_created_at_id_ix ON tickets (organization_id, client_id, created_at DESC, id DESC);
CREATE INDEX tickets_org_created_at_id_ix ON tickets (organization_id, created_at DESC, id DESC);
CREATE INDEX projects_org_client_created_at_id_ix ON projects (organization_id, client_id, created_at DESC, id DESC);
CREATE INDEX tasks_org_project_created_at_id_ix ON tasks (organization_id, project_id, created_at DESC, id DESC);
CREATE INDEX tasks_org_assignee_created_at_id_ix ON tasks (organization_id, assignee_id, created_at DESC, id DESC);
CREATE INDEX invoices_org_client_created_at_id_ix ON invoices (organization_id, client_id, created_at DESC, id DESC);
