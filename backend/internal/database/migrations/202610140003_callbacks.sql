CREATE TABLE login_limits (
 key TEXT PRIMARY KEY, window_start TIMESTAMPTZ NOT NULL, attempts INTEGER NOT NULL
);
CREATE TABLE delivery_events (
 id BIGSERIAL PRIMARY KEY, event_key VARCHAR(200) NOT NULL UNIQUE,
 provider_message_id UUID NOT NULL, status VARCHAR(64) NOT NULL,
 occurred_at TIMESTAMPTZ NOT NULL, attempt_id BIGINT REFERENCES sms_attempts(id),
 processed BOOLEAN NOT NULL DEFAULT FALSE, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX delivery_events_unmatched_idx ON delivery_events(created_at,id) WHERE NOT processed;
CREATE TABLE callback_jobs (
 id BIGSERIAL PRIMARY KEY, event_id BIGINT NOT NULL UNIQUE REFERENCES delivery_events(id),
 attempt_id BIGINT NOT NULL REFERENCES sms_attempts(id), credential_id BIGINT NOT NULL REFERENCES api_credentials(id),
 url TEXT NOT NULL, status VARCHAR(64) NOT NULL,
 state VARCHAR(16) NOT NULL CHECK(state IN ('pending','processing','delivered','failed')),
 attempts INTEGER NOT NULL DEFAULT 0, available_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 lease_until TIMESTAMPTZ, lease_token UUID, last_error VARCHAR(64) NOT NULL DEFAULT '',
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX callback_jobs_available_idx ON callback_jobs(available_at,id) WHERE state IN ('pending','processing');
CREATE INDEX callback_jobs_attempt_idx ON callback_jobs(attempt_id,id,state);
ALTER TABLE sms_attempts ADD COLUMN company_id BIGINT REFERENCES companies(id);
UPDATE sms_attempts AS sms SET company_id=creds.company_id FROM api_credentials AS creds WHERE creds.id=sms.credential_id;
CREATE INDEX sms_attempts_company_created_idx ON sms_attempts(company_id,created_at,id);
