-- Retain invalid destinations from supplier batch results without storing SMS bodies.
-- The supplier owns destination validation for these items.
ALTER TABLE sms_attempts ALTER COLUMN destination TYPE TEXT;
CREATE TABLE send_reservations (
 id BIGSERIAL PRIMARY KEY,
 credential_id BIGINT NOT NULL REFERENCES api_credentials(id),
 messages INTEGER NOT NULL CHECK(messages BETWEEN 1 AND 200),
 created_at TIMESTAMPTZ NOT NULL
);
CREATE INDEX send_reservations_created_idx ON send_reservations(created_at,credential_id);
