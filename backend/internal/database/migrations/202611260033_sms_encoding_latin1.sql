ALTER TABLE sms_attempts
    DROP CONSTRAINT sms_attempts_encoding_check;

ALTER TABLE sms_attempts
    ADD CONSTRAINT sms_attempts_encoding_check
    CHECK (encoding IN ('0', '3', '8'));
