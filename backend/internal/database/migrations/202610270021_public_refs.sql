ALTER TABLE companies ADD COLUMN public_ref uuid NOT NULL DEFAULT gen_random_uuid();
ALTER TABLE system_users ADD COLUMN public_ref uuid NOT NULL DEFAULT gen_random_uuid();
ALTER TABLE api_credentials ADD COLUMN public_ref uuid NOT NULL DEFAULT gen_random_uuid();

CREATE UNIQUE INDEX companies_public_ref_key ON companies(public_ref);
CREATE UNIQUE INDEX system_users_public_ref_key ON system_users(public_ref);
CREATE UNIQUE INDEX api_credentials_public_ref_key ON api_credentials(public_ref);
