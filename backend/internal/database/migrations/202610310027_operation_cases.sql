CREATE TABLE operation_cases (
  id BIGSERIAL PRIMARY KEY,
  kind VARCHAR(24) NOT NULL CHECK (kind IN ('unknown_send','overdue_schedule','dlr_callback','mo_callback')),
  source_id BIGINT NOT NULL,
  attempt_id BIGINT NOT NULL REFERENCES sms_attempts(id),
  company_id BIGINT NOT NULL REFERENCES companies(id),
  status VARCHAR(16) NOT NULL DEFAULT 'open' CHECK (status IN ('open','in_progress','completed')),
  owner_id BIGINT REFERENCES system_users(id),
  note VARCHAR(1000) NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  UNIQUE (kind, source_id)
);
CREATE INDEX operation_cases_company_status_idx ON operation_cases(company_id, status, updated_at DESC);
CREATE TABLE operation_case_events (
  id BIGSERIAL PRIMARY KEY,
  case_id BIGINT NOT NULL REFERENCES operation_cases(id),
  actor_id BIGINT NOT NULL REFERENCES system_users(id),
  status VARCHAR(16) NOT NULL CHECK (status IN ('open','in_progress','completed')),
  note VARCHAR(1000) NOT NULL DEFAULT '',
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX operation_case_events_case_idx ON operation_case_events(case_id, id);

CREATE VIEW operation_case_sources AS
  SELECT 'unknown_send'::varchar(24) AS kind, a.id AS source_id, a.id AS attempt_id, a.company_id, a.created_at, a.updated_at AS source_changed_at
  FROM sms_attempts a
  WHERE a.state = 'unknown' AND COALESCE(a.delivery_status,'') IN ('','SENT','UNKNOWN')
    AND a.company_id IS NOT NULL
  UNION ALL
  SELECT 'overdue_schedule', s.attempt_id, s.attempt_id, a.company_id, s.due_at, s.updated_at
  FROM scheduled_sms s JOIN sms_attempts a ON a.id = s.attempt_id
  WHERE s.state = 'queued' AND a.company_id IS NOT NULL
  UNION ALL
  SELECT 'dlr_callback', j.id, j.attempt_id, a.company_id, j.created_at, j.updated_at
  FROM callback_jobs j JOIN sms_attempts a ON a.id = j.attempt_id
  WHERE j.state = 'failed' AND a.company_id IS NOT NULL
  UNION ALL
  SELECT 'mo_callback', j.id, j.attempt_id, a.company_id, j.created_at, j.updated_at
  FROM mo_callback_jobs j JOIN sms_attempts a ON a.id = j.attempt_id
  WHERE j.state = 'failed' AND a.company_id IS NOT NULL;
