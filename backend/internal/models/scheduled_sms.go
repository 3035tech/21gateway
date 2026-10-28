package models

import (
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"time"
)

type ScheduleState string

const (
	ScheduleQueued      ScheduleState = "queued"
	ScheduleClaimed     ScheduleState = "claimed"
	ScheduleDispatching ScheduleState = "dispatching"
	ScheduleDone        ScheduleState = "done"
	ScheduleFailed      ScheduleState = "failed"
	ScheduleUnknown     ScheduleState = "unknown"
)
const ScheduledBatchSize = 200
const ScheduleLease = 30 * time.Second

type ScheduledSMS struct {
	DispatchBatchID  *string
	PublishAfter     *time.Time
	PublishToken     *string
	PublishedAt      *time.Time
	SendReference    bool
	ID               uint64
	AttemptID        uint64
	CredentialID     uint64
	EventKey         string
	EncryptedMessage string
	DueAt            time.Time
	AvailableAt      time.Time
	State            ScheduleState
	LeaseToken       *string
	LeaseUntil       *time.Time
	LastError        apperrors.Code
	CreatedAt        time.Time
	UpdatedAt        time.Time
	Attempt          SMSAttempt `gorm:"-"`
}

func (ScheduledSMS) TableName() string { return "scheduled_sms" }
