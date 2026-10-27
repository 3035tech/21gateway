-- Administrative message and callback lists default to creation order.
CREATE INDEX sms_attempts_created_idx ON sms_attempts(created_at, id);
CREATE INDEX callback_jobs_created_idx ON callback_jobs(created_at, id);
