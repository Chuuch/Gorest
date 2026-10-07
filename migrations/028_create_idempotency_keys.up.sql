CREATE TABLE idempotency_keys (
  organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
  key TEXT NOT NULL,
  method TEXT NOT NULL,
  path TEXT NOT NULL,
  request_hash TEXT NOT NULL,
  status_code INT,
  response_body BYTEA,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  expires_at TIMESTAMPTZ NOT NULL,
  PRIMARY KEY (organization_id, key)
);

CREATE INDEX idempotency_keys_expires_at_ix ON idempotency_keys (expires_at);
