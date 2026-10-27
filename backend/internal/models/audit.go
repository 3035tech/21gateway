package models

import (
	"time"
)

type AuditEvent struct {
	ID         uint64 `gorm:"primaryKey"`
	ActorID    *uint64
	ActorName  string `gorm:"->;-:migration"`
	Action     string
	Resource   string
	ResourceID *uint64
	CreatedAt  time.Time
}
