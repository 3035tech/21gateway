package dtos

import (
	"github.com/21mobile/sms-gateway/internal/models"
	"time"
)

type CompanyView struct {
	ProviderTokenConfigured     bool      `json:"providerTokenConfigured"`
	LegacyProviderTokenFallback bool      `json:"legacyProviderTokenFallback"`
	ID                          uint64    `json:"id"`
	PublicRef                   string    `json:"publicRef" format:"uuid"`
	Name                        string    `json:"name"`
	Reference                   string    `json:"reference"`
	Active                      bool      `json:"active"`
	CreatedAt                   time.Time `json:"createdAt"`
}
type CompanyBody struct {
	ProviderToken      *string `json:"providerToken,omitempty" maxLength:"4096" writeOnly:"true"`
	ClearProviderToken bool    `json:"clearProviderToken,omitempty"`
	Name               string  `json:"name" minLength:"1" maxLength:"160"`
	Reference          string  `json:"reference" maxLength:"64"`
	Active             bool    `json:"active"`
}
type CreateCompanyInput struct{ Body CompanyBody }
type UpdateCompanyInput struct {
	IDInput
	Body CompanyBody
}
type CompanyOutput struct{ Body CompanyView }
type CompaniesOutput struct{ Body Page[CompanyView] }

func CompanyDTO(value models.Company) CompanyView {
	return CompanyView{ProviderTokenConfigured: value.ProviderTokenEncrypted != "", LegacyProviderTokenFallback: value.ProviderTokenLegacyFallback, ID: value.ID, PublicRef: value.PublicRef, Name: value.Name, Reference: value.Reference, Active: value.Active, CreatedAt: value.CreatedAt}
}
