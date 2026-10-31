package models

import (
	"time"
)

type Company struct {
	ID                          uint64 `gorm:"primaryKey"`
	PublicRef                   string `gorm:"type:uuid;default:gen_random_uuid()"`
	Name                        string
	Reference                   string
	ProviderTokenEncrypted      string
	ProviderTokenLegacyFallback bool
	SMSProvider                 Provider `gorm:"->;-:migration"`
	Active                      bool
	ArchivedAt                  *time.Time
	CreatedAt                   time.Time
	UpdatedAt                   time.Time
}
