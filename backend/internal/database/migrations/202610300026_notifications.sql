CREATE TABLE notification_types (
 code VARCHAR(48) PRIMARY KEY,
 email_enabled BOOLEAN NOT NULL DEFAULT FALSE,
 updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
INSERT INTO notification_types(code) VALUES
 ('unknown_sends'), ('expired_leases'), ('dlr_pending'), ('mo_pending'),
 ('dlr_unmatched'), ('mo_unmatched'), ('schedule_overdue'), ('callback_failed');

CREATE TABLE notification_events (
 id BIGSERIAL PRIMARY KEY,
 type_code VARCHAR(48) NOT NULL REFERENCES notification_types(code),
 title VARCHAR(160) NOT NULL,
 description VARCHAR(500) NOT NULL,
 path VARCHAR(80) NOT NULL,
 email_selected BOOLEAN NOT NULL DEFAULT FALSE,
 opened_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
 resolved_at TIMESTAMPTZ
);
CREATE UNIQUE INDEX notification_events_active_type_idx ON notification_events(type_code) WHERE resolved_at IS NULL;
CREATE INDEX notification_events_recent_idx ON notification_events(opened_at DESC, id DESC);

CREATE TABLE notification_receipts (
 event_id BIGINT NOT NULL REFERENCES notification_events(id),
 user_id BIGINT NOT NULL REFERENCES system_users(id),
 read_at TIMESTAMPTZ,
 email_sent_at TIMESTAMPTZ,
 email_attempts INTEGER NOT NULL DEFAULT 0,
 email_next_at TIMESTAMPTZ,
 email_lease_until TIMESTAMPTZ,
 PRIMARY KEY (event_id, user_id)
);
CREATE INDEX notification_receipts_user_idx ON notification_receipts(user_id, event_id DESC);
CREATE INDEX notification_receipts_email_due_idx ON notification_receipts(email_next_at) WHERE email_sent_at IS NULL AND email_next_at IS NOT NULL;
