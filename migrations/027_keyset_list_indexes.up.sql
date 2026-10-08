DROP INDEX IF EXISTS activity_events_org_created_at_ix;
CREATE INDEX activity_events_org_created_at_id_ix ON activity_events (organization_id, created_at DESC, id DESC);

DROP INDEX IF EXISTS notifications_recipient_created_at_ix;
CREATE INDEX notifications_recipient_created_at_id_ix ON notifications (organization_id, recipient_id, created_at DESC, id DESC);
