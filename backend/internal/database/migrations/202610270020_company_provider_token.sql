ALTER TABLE companies ADD COLUMN provider_token_encrypted TEXT NOT NULL DEFAULT '';
ALTER TABLE companies ADD COLUMN provider_token_legacy_fallback BOOLEAN NOT NULL DEFAULT FALSE;
UPDATE companies SET provider_token_legacy_fallback = TRUE;
