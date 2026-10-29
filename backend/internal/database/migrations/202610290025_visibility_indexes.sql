CREATE INDEX delivery_events_attempt_trace_idx ON delivery_events(attempt_id, occurred_at, id) WHERE attempt_id IS NOT NULL;
CREATE INDEX mo_events_attempt_trace_idx ON mo_events(attempt_id, received_at, id) WHERE attempt_id IS NOT NULL;
CREATE INDEX mo_callback_jobs_attempt_idx ON mo_callback_jobs(attempt_id, id);
CREATE INDEX sms_attempts_unresolved_company_idx ON sms_attempts(company_id, created_at) WHERE state='unknown' AND COALESCE(delivery_status,'') IN ('','SENT','UNKNOWN');
