CREATE TABLE mo_events (
 id BIGSERIAL PRIMARY KEY,
 event_key VARCHAR(200) NOT NULL,
 credential_id BIGINT NOT NULL REFERENCES api_credentials(id),
 provider_message_id UUID NOT NULL,
 attempt_id BIGINT REFERENCES sms_attempts(id),
 encrypted_message TEXT NOT NULL,
 fingerprint VARCHAR(64) NOT NULL,
 phone_number VARCHAR(15) NOT NULL,
 received_at TIMESTAMPTZ NOT NULL,
 processed BOOLEAN NOT NULL DEFAULT FALSE,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 UNIQUE (credential_id,event_key)
);
CREATE INDEX mo_events_unmatched_idx ON mo_events(created_at,id) WHERE NOT processed;
CREATE TABLE mo_callback_jobs (
 id BIGSERIAL PRIMARY KEY,
 event_id BIGINT NOT NULL UNIQUE REFERENCES mo_events(id),
 attempt_id BIGINT NOT NULL REFERENCES sms_attempts(id),
 credential_id BIGINT NOT NULL REFERENCES api_credentials(id),
 url TEXT NOT NULL,
 state VARCHAR(16) NOT NULL CHECK(state IN ('pending','processing','delivered','failed')),
 attempts INTEGER NOT NULL DEFAULT 0,
 available_at TIMESTAMPTZ NOT NULL,
 lease_until TIMESTAMPTZ,
 lease_token UUID,
 last_error VARCHAR(64) NOT NULL DEFAULT '',
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX mo_callback_jobs_queue_idx ON mo_callback_jobs(available_at,id) WHERE state IN ('pending','processing');
