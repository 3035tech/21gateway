ALTER TABLE mo_events ADD COLUMN content_purged_at TIMESTAMPTZ;
CREATE INDEX mo_events_retention_idx ON mo_events(received_at,id) WHERE content_purged_at IS NULL AND processed;
CREATE INDEX callback_jobs_failed_time_idx ON callback_jobs(updated_at) WHERE state='failed';
CREATE INDEX mo_callback_jobs_failed_time_idx ON mo_callback_jobs(updated_at) WHERE state='failed';
