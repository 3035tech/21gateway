CREATE TABLE legacy_status_receipts (
 attempt_id BIGINT PRIMARY KEY REFERENCES sms_attempts(id),
 credential_id BIGINT NOT NULL REFERENCES api_credentials(id),
 delivery_updated_at TIMESTAMPTZ NOT NULL,
 delivery_status TEXT NOT NULL
);
CREATE INDEX sms_attempts_legacy_status_idx ON sms_attempts (credential_id, id)
 WHERE correlation_id <> '' AND delivery_status <> '' AND delivery_updated_at IS NOT NULL;
