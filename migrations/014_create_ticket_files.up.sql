CREATE TABLE ticket_files (
  id UUID PRIMARY KEY,
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  ticket_id UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
  uploaded_by UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
  object_key TEXT NOT NULL UNIQUE,
  filename TEXT NOT NULL,
  content_type TEXT NOT NULL,
  size_bytes INTEGER NOT NULL,
  created_at TIMESTAMPTZ NOT NULL,
  updated_at TIMESTAMPTZ NOT NULL,
  CONSTRAINT ticket_files_size_bytes_check CHECK (size_bytes >= 1 AND size_bytes <= 10485760)
);

CREATE INDEX idx_ticket_files_organization_id ON ticket_files (organization_id);
CREATE INDEX idx_ticket_files_ticket_id ON ticket_files (ticket_id);
