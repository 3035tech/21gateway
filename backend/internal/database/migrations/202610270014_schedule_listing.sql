-- Paginated administrative history includes terminal and queued schedules.
CREATE INDEX scheduled_sms_due_all_idx ON scheduled_sms(due_at,id);
CREATE INDEX scheduled_sms_credential_due_all_idx ON scheduled_sms(credential_id,due_at,id);
