CREATE TABLE companies (
 id BIGSERIAL PRIMARY KEY, name VARCHAR(160) NOT NULL, reference VARCHAR(64) NOT NULL DEFAULT '',
 active BOOLEAN NOT NULL DEFAULT TRUE, archived_at TIMESTAMPTZ,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX companies_reference_unique ON companies(reference) WHERE reference <> '' AND archived_at IS NULL;
CREATE INDEX companies_active_idx ON companies(active, id) WHERE archived_at IS NULL;
CREATE TABLE system_users (
 id BIGSERIAL PRIMARY KEY, name VARCHAR(120) NOT NULL, email VARCHAR(254) NOT NULL,
 password_hash TEXT NOT NULL, role VARCHAR(16) NOT NULL CHECK(role IN ('admin','operator','viewer')),
 active BOOLEAN NOT NULL DEFAULT TRUE, session_version BIGINT NOT NULL DEFAULT 1,
 archived_at TIMESTAMPTZ, last_login_at TIMESTAMPTZ,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX system_users_email_unique ON system_users(lower(email)) WHERE archived_at IS NULL;
ALTER TABLE api_credentials ADD COLUMN company_id BIGINT REFERENCES companies(id);
ALTER TABLE api_credentials ADD COLUMN callback_url TEXT NOT NULL DEFAULT '';
ALTER TABLE api_credentials ADD COLUMN callbacks_enabled BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE api_credentials ADD COLUMN archived_at TIMESTAMPTZ;
CREATE INDEX api_credentials_company_idx ON api_credentials(company_id, active, id);
CREATE TABLE sessions (token TEXT PRIMARY KEY, data BYTEA NOT NULL, expiry TIMESTAMPTZ NOT NULL);
CREATE INDEX sessions_expiry_idx ON sessions(expiry);
CREATE TABLE audit_events (
 id BIGSERIAL PRIMARY KEY, actor_id BIGINT REFERENCES system_users(id), action VARCHAR(64) NOT NULL,
 resource VARCHAR(32) NOT NULL, resource_id BIGINT, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX audit_events_created_idx ON audit_events(created_at, id);
ALTER TABLE sms_attempts ADD COLUMN delivery_status VARCHAR(64) NOT NULL DEFAULT '';
ALTER TABLE sms_attempts ADD COLUMN delivery_updated_at TIMESTAMPTZ;
