ALTER TABLE api_credentials ADD COLUMN provider_token_encrypted TEXT NOT NULL DEFAULT '';
ALTER TABLE api_credentials ADD COLUMN whitelist_enabled BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE api_credentials ADD COLUMN allowed_ips JSONB NOT NULL DEFAULT '[]'::jsonb;
ALTER TABLE api_credentials ADD COLUMN mo_callback_url TEXT NOT NULL DEFAULT '';
ALTER TABLE api_credentials ADD COLUMN mo_callbacks_enabled BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE api_credentials ADD COLUMN legacy_account_id VARCHAR(64) NOT NULL DEFAULT '';
CREATE UNIQUE INDEX api_credentials_legacy_account_unique ON api_credentials(legacy_account_id) WHERE legacy_account_id<>'' AND archived_at IS NULL;
