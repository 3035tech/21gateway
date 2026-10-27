package models

import (
	"time"
)

type Role string

const (
	RoleAdmin    Role = "admin"
	RoleOperator Role = "operator"
	RoleViewer   Role = "viewer"
)

type SystemUser struct {
	ID             uint64 `gorm:"primaryKey"`
	Name           string
	Email          string
	PasswordHash   string
	Role           Role
	Active         bool
	SessionVersion int64
	ArchivedAt     *time.Time
	LastLoginAt    *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
