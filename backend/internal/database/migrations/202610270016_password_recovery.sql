CREATE TABLE user_password_resets (
 user_id BIGINT PRIMARY KEY REFERENCES system_users(id) ON DELETE CASCADE,
 token_hash BYTEA NOT NULL,
 purpose VARCHAR(16) NOT NULL CHECK (purpose IN ('reset','invite')),
 expires_at TIMESTAMPTZ NOT NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX user_password_resets_expiry_idx ON user_password_resets(expires_at);
