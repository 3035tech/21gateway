CREATE TABLE user_mfa (
 user_id BIGINT PRIMARY KEY REFERENCES system_users(id) ON DELETE CASCADE,
 secret_encrypted TEXT NOT NULL DEFAULT '',
 pending_encrypted TEXT NOT NULL DEFAULT '',
 pending_expires_at TIMESTAMPTZ,
 last_used_step BIGINT NOT NULL DEFAULT 0,
 updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE TABLE user_mfa_recovery_codes (
 user_id BIGINT NOT NULL REFERENCES system_users(id) ON DELETE CASCADE,
 code_hash BYTEA NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 PRIMARY KEY (user_id, code_hash)
);
