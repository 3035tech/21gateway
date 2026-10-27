CREATE TABLE api_credentials (
 id BIGSERIAL PRIMARY KEY,
 username VARCHAR(120) NOT NULL UNIQUE,
 password_hash TEXT NOT NULL,
 active BOOLEAN NOT NULL DEFAULT TRUE,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE sms_attempts (
 id BIGSERIAL PRIMARY KEY,
 credential_id BIGINT NOT NULL REFERENCES api_credentials(id),
 destination VARCHAR(13) NOT NULL,
 correlation_id VARCHAR(50) NOT NULL DEFAULT '',
 encoding VARCHAR(1) NOT NULL CHECK (encoding IN ('0', '8')),
 provider_message_id UUID UNIQUE,
 state VARCHAR(16) NOT NULL CHECK (state IN ('pending', 'accepted', 'rejected', 'unknown')),
 failure_code VARCHAR(64) NOT NULL DEFAULT '',
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX sms_attempts_credential_created_idx ON sms_attempts(credential_id, created_at, id);
