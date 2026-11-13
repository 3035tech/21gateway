ALTER TABLE api_credentials ADD COLUMN callback_url_version INTEGER NOT NULL DEFAULT 0;
ALTER TABLE api_credentials ADD COLUMN mo_callback_url_version INTEGER NOT NULL DEFAULT 0;
ALTER TABLE callback_jobs ADD COLUMN url_version INTEGER NOT NULL DEFAULT 0;
ALTER TABLE mo_callback_jobs ADD COLUMN url_version INTEGER NOT NULL DEFAULT 0;
