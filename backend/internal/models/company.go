package models

import (
	"time"
)

type Company struct {
	ID                          uint64 `gorm:"primaryKey"`
	Name                        string
	Reference                   string
	ProviderTokenEncrypted      string
	ProviderTokenLegacyFallback bool
	Active                      bool
	ArchivedAt                  *time.Time
	CreatedAt                   time.Time
	UpdatedAt                   time.Time
}
