ALTER TABLE send_reservations ADD COLUMN scope VARCHAR(16) NOT NULL DEFAULT 'both' CHECK(scope IN ('api','dispatch','both'));
ALTER TABLE sms_attempts ADD COLUMN scheduled_at TIMESTAMPTZ;
ALTER TABLE sms_attempts ADD COLUMN dispatched_at TIMESTAMPTZ;
CREATE TABLE scheduled_sms (
 id BIGSERIAL PRIMARY KEY,
 attempt_id BIGINT NOT NULL UNIQUE REFERENCES sms_attempts(id),
 credential_id BIGINT NOT NULL REFERENCES api_credentials(id),
 event_key UUID NOT NULL UNIQUE,
 encrypted_message TEXT NOT NULL,
 due_at TIMESTAMPTZ NOT NULL,
 available_at TIMESTAMPTZ NOT NULL,
 state VARCHAR(16) NOT NULL CHECK(state IN ('queued','claimed','dispatching','done','failed','unknown')),
 lease_token UUID,
 lease_until TIMESTAMPTZ,
 last_error VARCHAR(64) NOT NULL DEFAULT '',
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX scheduled_sms_due_idx ON scheduled_sms(available_at,due_at,id) WHERE state='queued';
CREATE INDEX scheduled_sms_account_idx ON scheduled_sms(credential_id,due_at,id) WHERE state IN ('queued','claimed','dispatching');
CREATE INDEX scheduled_sms_lease_idx ON scheduled_sms(lease_until,id) WHERE state IN ('claimed','dispatching');
CREATE INDEX scheduled_sms_terminal_time_idx ON scheduled_sms(updated_at) WHERE state IN ('failed','unknown');
CREATE INDEX scheduled_sms_inflight_account_idx ON scheduled_sms(credential_id,lease_until) WHERE state IN ('claimed','dispatching');
