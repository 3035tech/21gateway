package controllers

import (
	"context"
	"github.com/21mobile/sms-gateway/internal/dtos"
	"github.com/21mobile/sms-gateway/internal/security"
	"github.com/21mobile/sms-gateway/internal/services"
	"github.com/danielgtaylor/huma/v2"
	"net/http"
)

type CompanyController struct {
	*SessionGuard
	service *services.CompanyService
}

func NewCompanyController(service *services.CompanyService, guard *SessionGuard) *CompanyController {
	return &CompanyController{SessionGuard: guard, service: service}
}
func (c *CompanyController) Register(api huma.API) {
	huma.Register(api, c.operation(api, "getCompany", http.MethodGet, "/companies/{id}", "Consultar empresa", security.ResourceCompanies, security.ActionRead), func(ctx context.Context, input *dtos.IDInput) (*dtos.CompanyOutput, error) {
		value, err := c.service.Company(ctx, input.ID)
		if err != nil {
			return nil, HTTPError(err)
		}
		return &dtos.CompanyOutput{Body: value}, nil
	})
	huma.Register(api, c.operation(api, "listCompanies", http.MethodGet, "/companies", "Listar empresas", security.ResourceCompanies, security.ActionRead), func(ctx context.Context, input *dtos.ListQuery) (*dtos.CompaniesOutput, error) {
		values, err := c.service.Companies(ctx, *input)
		if err != nil {
			return nil, HTTPError(err)
		}
		return &dtos.CompaniesOutput{Body: values}, nil
	})
	huma.Register(api, c.operation(api, "createCompany", http.MethodPost, "/companies", "Criar empresas", security.ResourceCompanies, security.ActionWrite), func(ctx context.Context, input *dtos.CreateCompanyInput) (*dtos.CompanyOutput, error) {
		value, err := c.service.SaveCompany(ctx, actor(ctx), 0, input.Body)
		if err != nil {
			return nil, HTTPError(err)
		}
		return &dtos.CompanyOutput{Body: value}, nil
	})
	huma.Register(api, c.operation(api, "updateCompany", http.MethodPut, "/companies/{id}", "Atualizar empresas", security.ResourceCompanies, security.ActionWrite), func(ctx context.Context, input *dtos.UpdateCompanyInput) (*dtos.CompanyOutput, error) {
		value, err := c.service.SaveCompany(ctx, actor(ctx), input.ID, input.Body)
		if err != nil {
			return nil, HTTPError(err)
		}
		return &dtos.CompanyOutput{Body: value}, nil
	})
	archiveCompany := c.operation(api, "archiveCompany", http.MethodDelete, "/companies/{id}", "Arquivar empresas", security.ResourceCompanies, security.ActionWrite)
	archiveCompany.DefaultStatus = http.StatusNoContent
	huma.Register(api, archiveCompany, func(ctx context.Context, input *dtos.IDInput) (*struct{}, error) {
		if err := c.service.Archive(ctx, actor(ctx), input.ID); err != nil {
			return nil, HTTPError(err)
		}
		return nil, nil
	})

}
