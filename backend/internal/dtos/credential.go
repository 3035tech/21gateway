package dtos

import (
	"github.com/21mobile/sms-gateway/internal/models"
	"time"
)

type CredentialView struct {
	ProviderTokenConfigured bool     `json:"providerTokenConfigured"`
	WhitelistEnabled        bool     `json:"whitelistEnabled"`
	AllowedIPs              []string `json:"allowedIPs"`
	MOCallbackURL           string   `json:"moCallbackUrl"`
	MOCallbacksEnabled      bool     `json:"moCallbacksEnabled"`
	LegacyAccountID         string   `json:"legacyAccountId"`

	ID               uint64     `json:"id"`
	CompanyID        *uint64    `json:"companyId"`
	CompanyName      string     `json:"companyName"`
	Username         string     `json:"username"`
	Active           bool       `json:"active"`
	CallbackURL      string     `json:"callbackUrl"`
	CallbacksEnabled bool       `json:"callbacksEnabled"`
	CreatedAt        time.Time  `json:"createdAt"`
	ExpiresAt        *time.Time `json:"expiresAt"`
}
type CredentialBody struct {
	LegacyPassword     *string    `json:"legacyPassword,omitempty" minLength:"1" maxLength:"128" writeOnly:"true"`
	ManualPassword     *string    `json:"manualPassword,omitempty" minLength:"15" maxLength:"128" writeOnly:"true"`
	GenerateUsername   bool       `json:"generateUsername,omitempty"`
	ExpiresAt          *time.Time `json:"expiresAt,omitempty"`
	ClearExpiry        bool       `json:"clearExpiry,omitempty"`
	ProviderToken      *string    `json:"providerToken,omitempty" maxLength:"4096" writeOnly:"true"`
	ClearProviderToken bool       `json:"clearProviderToken,omitempty"`
	WhitelistEnabled   *bool      `json:"whitelistEnabled,omitempty"`
	AllowedIPs         *[]string  `json:"allowedIPs,omitempty" maxItems:"100"`
	MOCallbackURL      *string    `json:"moCallbackUrl,omitempty" maxLength:"2048"`
	MOCallbacksEnabled *bool      `json:"moCallbacksEnabled,omitempty"`
	LegacyAccountID    *string    `json:"legacyAccountId,omitempty" maxLength:"64" pattern:"^[0-9]+(_[0-9]+)?$"`

	CompanyID        uint64 `json:"companyId" minimum:"1"`
	Username         string `json:"username" minLength:"1" maxLength:"120"`
	Active           bool   `json:"active"`
	CallbackURL      string `json:"callbackUrl" maxLength:"2048"`
	CallbacksEnabled bool   `json:"callbacksEnabled"`
}
type CreateCredentialInput struct{ Body CredentialBody }
type UpdateCredentialInput struct {
	IDInput
	Body CredentialBody
}
type CredentialOutput struct{ Body CredentialView }
type CredentialsOutput struct{ Body Page[CredentialView] }
type CredentialSecret struct {
	Credential CredentialView `json:"credential"`
	Password   string         `json:"password"`
}
type CredentialSecretOutput struct{ Body CredentialSecret }

func CredentialDTO(value models.APICredential) CredentialView {
	allowedIPs := value.AllowedIPs
	if allowedIPs == nil {
		allowedIPs = []string{}
	}
	return CredentialView{ProviderTokenConfigured: value.ProviderTokenEncrypted != "", WhitelistEnabled: value.WhitelistEnabled, AllowedIPs: allowedIPs, MOCallbackURL: value.MOCallbackURL, MOCallbacksEnabled: value.MOCallbacksEnabled, LegacyAccountID: value.LegacyAccountID, ID: value.ID, CompanyID: value.CompanyID, CompanyName: value.CompanyName, Username: value.Username, Active: value.Active, CallbackURL: value.CallbackURL, CallbacksEnabled: value.CallbacksEnabled, CreatedAt: value.CreatedAt, ExpiresAt: value.ExpiresAt}
}
