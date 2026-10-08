CREATE INDEX clients_org_created_at_id_ix ON clients (organization_id, created_at DESC, id DESC);
