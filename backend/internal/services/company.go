package services

import (
	"context"
	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/repositories"
	"github.com/21mobile/sms-gateway/internal/security"
	"strings"
	"time"
)

type CompanyService struct{ repository *repositories.Companies }

func NewCompanyService(repo *repositories.Companies) *CompanyService {
	return &CompanyService{repository: repo}
}
func (s *CompanyService) Companies(ctx context.Context, q dtos.ListQuery) (dtos.Page[dtos.CompanyView], error) {
	values, err := s.repository.Companies(ctx, q)
	return mapPage(values, dtos.CompanyDTO), dbError(err)
}
func (s *CompanyService) Company(ctx context.Context, id uint64) (dtos.CompanyView, error) {
	value, err := s.repository.CompanyView(ctx, id)
	return dtos.CompanyDTO(value), dbError(err)
}
func (s *CompanyService) SaveCompany(ctx context.Context, actor, id uint64, body dtos.CompanyBody) (dtos.CompanyView, error) {
	var result models.Company
	if strings.TrimSpace(body.Name) == "" {
		return dtos.CompanyView{}, apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	err := s.repository.Transaction(ctx, func(repo *repositories.Companies) error {
		action := auditCreated
		if id != 0 {
			var err error
			result, err = repo.Company(ctx, id)
			if err != nil {
				return err
			}
			action = auditUpdated
		}
		result.Name = strings.TrimSpace(body.Name)
		result.Reference = strings.TrimSpace(body.Reference)
		result.Active = body.Active
		var err error
		if id == 0 {
			err = repo.Create(ctx, &result)
		} else {
			err = repo.Save(ctx, &result)
		}
		if err != nil {
			return err
		}
		return repo.Audit(ctx, &actor, action, string(security.ResourceCompanies), &result.ID)
	})
	return dtos.CompanyDTO(result), dbError(err)
}
func (s *CompanyService) Archive(ctx context.Context, actor, id uint64) error {
	return dbError(s.repository.Transaction(ctx, func(repo *repositories.Companies) error {
		now := time.Now().UTC()
		value, found := repo.Company(ctx, id)
		if found != nil {
			return found
		}
		value.Active = false
		value.ArchivedAt = &now
		if err := repo.Save(ctx, &value); err != nil {
			return err
		}
		return repo.Audit(ctx, &actor, auditArchived, string(security.ResourceCompanies), &id)
	}))
}
