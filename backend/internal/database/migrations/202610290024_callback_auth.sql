ALTER TABLE api_credentials
 ADD COLUMN callback_auth JSONB NOT NULL DEFAULT '{"mode":"none"}',
 ADD COLUMN mo_callback_auth JSONB NOT NULL DEFAULT '{"mode":"none"}';
