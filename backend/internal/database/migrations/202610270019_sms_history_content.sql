ALTER TABLE sms_attempts ADD COLUMN content_ref UUID;
ALTER TABLE sms_attempts ADD COLUMN encrypted_message TEXT NOT NULL DEFAULT '';
ALTER TABLE sms_attempts ADD COLUMN content_purged_at TIMESTAMPTZ;
CREATE INDEX idx_sms_attempts_content_retention ON sms_attempts ((COALESCE(dispatched_at, scheduled_at, created_at)), id) WHERE encrypted_message <> '';
