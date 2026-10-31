-- Provider IDs are opaque. Existing UUIDs are retained as text, and callbacks
-- are reconciled only within their authenticated provider namespace.
ALTER TABLE sms_attempts ALTER COLUMN provider_message_id TYPE TEXT USING provider_message_id::text;
ALTER TABLE sms_attempts DROP CONSTRAINT sms_attempts_provider_message_id_key;
ALTER TABLE sms_attempts ADD CONSTRAINT sms_attempts_provider_message_namespace_key UNIQUE (provider_code, provider_message_id);

ALTER TABLE delivery_events ALTER COLUMN provider_message_id TYPE TEXT USING provider_message_id::text;
ALTER TABLE delivery_events ADD COLUMN provider_code VARCHAR(64) NOT NULL DEFAULT 'zoug' CHECK (provider_code <> '');

ALTER TABLE mo_events ALTER COLUMN provider_message_id TYPE TEXT USING provider_message_id::text;
ALTER TABLE mo_events ADD COLUMN provider_code VARCHAR(64) NOT NULL DEFAULT 'zoug' CHECK (provider_code <> '');

ALTER TABLE company_test_sends ALTER COLUMN provider_message_id TYPE TEXT USING provider_message_id::text;
ALTER TABLE company_test_sends ADD COLUMN provider_code VARCHAR(64) NOT NULL DEFAULT 'zoug' CHECK (provider_code <> '');
ALTER TABLE company_test_sends DROP CONSTRAINT company_test_sends_provider_message_id_key;
ALTER TABLE company_test_sends ADD CONSTRAINT company_test_sends_provider_message_namespace_key UNIQUE (provider_code, provider_message_id);
