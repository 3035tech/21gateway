-- Each scheduled_sms row is the durable outbox entry for one SMS. Existing
-- batch-bound rows remain owned by the v1 batch dispatcher until drained.
ALTER TABLE scheduled_sms ADD COLUMN publish_after TIMESTAMPTZ;
ALTER TABLE scheduled_sms ADD COLUMN publish_token UUID;
ALTER TABLE scheduled_sms ADD COLUMN published_at TIMESTAMPTZ;
ALTER TABLE scheduled_sms ADD COLUMN publish_attempts INTEGER NOT NULL DEFAULT 0;
ALTER TABLE scheduled_sms ADD COLUMN send_reference BOOLEAN NOT NULL DEFAULT FALSE;
CREATE INDEX scheduled_sms_send_publish_idx ON scheduled_sms(publish_after,due_at,id)
 WHERE state='queued' AND send_reference=TRUE;
