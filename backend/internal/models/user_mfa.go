package models

import "time"

type UserMFA struct {
	UserID           uint64 `gorm:"primaryKey"`
	SecretEncrypted  string
	PendingEncrypted string
	PendingExpiresAt *time.Time
	LastUsedStep     int64
	UpdatedAt        time.Time
}

func (UserMFA) TableName() string { return "user_mfa" }

type UserMFARecoveryCode struct {
	UserID    uint64 `gorm:"primaryKey"`
	CodeHash  []byte `gorm:"primaryKey"`
	CreatedAt time.Time
}

func (UserMFARecoveryCode) TableName() string { return "user_mfa_recovery_codes" }
