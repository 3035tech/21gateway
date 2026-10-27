CREATE INDEX mo_events_credential_day_idx ON mo_events(credential_id,received_at,id) WHERE attempt_id IS NOT NULL;
