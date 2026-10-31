CREATE TABLE company_provider_routes (
 company_id BIGINT NOT NULL REFERENCES companies(id),
 channel VARCHAR(32) NOT NULL CHECK (channel <> ''),
 provider_code VARCHAR(64) NOT NULL CHECK (provider_code <> ''),
 PRIMARY KEY (company_id, channel)
);
INSERT INTO company_provider_routes(company_id, channel, provider_code)
SELECT id, 'sms', 'zoug' FROM companies;
ALTER TABLE sms_attempts ADD COLUMN provider_code VARCHAR(64) NOT NULL DEFAULT 'zoug' CHECK (provider_code <> '');

