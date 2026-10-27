package models

import "time"

// A batch is a durable wake-up reference; message content stays in scheduled_sms.
type DispatchBatch struct {
	ID              string `gorm:"primaryKey"`
	CredentialID    uint64
	AvailableAt     time.Time
	PublishedAt     *time.Time
	CompletedAt     *time.Time
	PublishAttempts int
	LeaseToken      *string
	LeaseUntil      *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (DispatchBatch) TableName() string { return "sms_dispatch_batches" }
