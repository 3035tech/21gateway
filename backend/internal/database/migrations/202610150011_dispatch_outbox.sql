CREATE TABLE sms_dispatch_batches (
 id UUID PRIMARY KEY,
 credential_id BIGINT NOT NULL REFERENCES api_credentials(id),
 available_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 published_at TIMESTAMPTZ,
 completed_at TIMESTAMPTZ,
 publish_attempts INTEGER NOT NULL DEFAULT 0,
 lease_token UUID,
 lease_until TIMESTAMPTZ,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX sms_dispatch_one_open_account_idx ON sms_dispatch_batches(credential_id) WHERE completed_at IS NULL;
CREATE INDEX sms_dispatch_publish_idx ON sms_dispatch_batches(available_at,id) WHERE completed_at IS NULL;
ALTER TABLE scheduled_sms ADD COLUMN dispatch_batch_id UUID REFERENCES sms_dispatch_batches(id);
CREATE INDEX scheduled_sms_batch_idx ON scheduled_sms(dispatch_batch_id,id) WHERE dispatch_batch_id IS NOT NULL;
CREATE INDEX scheduled_sms_unbatched_due_idx ON scheduled_sms(available_at,due_at,id) WHERE state='queued' AND dispatch_batch_id IS NULL;
CREATE INDEX sms_dispatch_account_history_idx ON sms_dispatch_batches(credential_id,created_at DESC);
