CREATE TABLE company_test_sends (
 id BIGSERIAL PRIMARY KEY,
 company_id BIGINT NOT NULL REFERENCES companies(id),
 actor_id BIGINT NOT NULL REFERENCES system_users(id),
 destination VARCHAR(13) NOT NULL,
 correlation_id VARCHAR(50) NOT NULL UNIQUE,
 provider_message_id UUID UNIQUE,
 state VARCHAR(16) NOT NULL CHECK (state IN ('accepted', 'rejected', 'unknown')),
 failure_code VARCHAR(64) NOT NULL DEFAULT '',
 created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX company_test_sends_company_created_idx ON company_test_sends(company_id, created_at DESC, id DESC);
