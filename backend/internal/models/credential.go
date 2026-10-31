package models

import (
	"time"

	"gorm.io/gorm"
)

type APICredential struct {
	ProviderTokenEncrypted             string
	CompanyProviderTokenEncrypted      string   `gorm:"->;-:migration"`
	CompanyProviderTokenLegacyFallback bool     `gorm:"->;-:migration"`
	CompanySMSProvider                 Provider `gorm:"->;-:migration"`
	WhitelistEnabled                   bool
	AllowedIPs                         []string `gorm:"serializer:json;type:jsonb"`
	MOCallbackURL                      string
	MOCallbacksEnabled                 bool
	MOCallbackAuth                     CallbackAuth `gorm:"serializer:json;type:jsonb"`
	LegacyAccountID                    string

	CompanyID        *uint64
	CompanyName      string `gorm:"->;-:migration"`
	CompanyActive    *bool  `gorm:"->;-:migration"`
	CallbackURL      string
	CallbacksEnabled bool
	CallbackAuth     CallbackAuth `gorm:"serializer:json;type:jsonb"`
	ArchivedAt       *time.Time
	ExpiresAt        *time.Time
	ID               uint64 `gorm:"primaryKey"`
	PublicRef        string `gorm:"type:uuid;default:gen_random_uuid()"`
	Username         string
	PasswordHash     string
	Active           bool
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (APICredential) TableName() string { return "api_credentials" }

func (credential APICredential) ValidAt(now time.Time) bool {
	return credential.Active && (credential.ExpiresAt == nil || now.Before(*credential.ExpiresAt))
}

func (credential *APICredential) BeforeCreate(*gorm.DB) error {
	if credential.AllowedIPs == nil {
		credential.AllowedIPs = []string{}
	}
	return nil
}
