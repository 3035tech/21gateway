package dtos

import (
	"github.com/21mobile/sms-gateway/internal/models"
	"time"
)

type CompanyView struct {
	ID        uint64    `json:"id"`
	Name      string    `json:"name"`
	Reference string    `json:"reference"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"createdAt"`
}
type CompanyBody struct {
	Name      string `json:"name" minLength:"1" maxLength:"160"`
	Reference string `json:"reference" maxLength:"64"`
	Active    bool   `json:"active"`
}
type CreateCompanyInput struct{ Body CompanyBody }
type UpdateCompanyInput struct {
	IDInput
	Body CompanyBody
}
type CompanyOutput struct{ Body CompanyView }
type CompaniesOutput struct{ Body Page[CompanyView] }

func CompanyDTO(value models.Company) CompanyView {
	return CompanyView{ID: value.ID, Name: value.Name, Reference: value.Reference, Active: value.Active, CreatedAt: value.CreatedAt}
}
