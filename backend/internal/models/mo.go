package models

import "time"

// MOMessage is normalized transient data, not a persisted or supplier payload.
// IDs refer to the gateway MO event and original SMS, never supplier UUIDs.
type MOMessage struct {
	ID            uint64
	CredentialID  uint64
	AttemptID     uint64
	CorrelationID string
	PhoneNumber   string
	Message       string
	ReceivedAt    time.Time
}

type MOEvent struct {
	ContentPurgedAt   *time.Time
	ID                uint64
	EventKey          string
	CredentialID      uint64
	ProviderMessageID string
	AttemptID         *uint64
	EncryptedMessage  string
	Fingerprint       string
	PhoneNumber       string
	ReceivedAt        time.Time
	Processed         bool
	CreatedAt         time.Time
}

func (MOEvent) TableName() string { return "mo_events" }

type MOCallbackJob struct {
	ID           uint64
	EventID      uint64
	AttemptID    uint64
	CredentialID uint64
	URL          string
	State        CallbackState
	Attempts     int
	AvailableAt  time.Time
	LeaseUntil   *time.Time
	LeaseToken   *string
	LastError    string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (MOCallbackJob) TableName() string { return "mo_callback_jobs" }
