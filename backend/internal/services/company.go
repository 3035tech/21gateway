package services

import (
	"context"
	"fmt"
	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/repositories"
	"github.com/21mobile/sms-gateway/internal/security"
	"strings"
	"time"
)

type CompanyService struct {
	repository *repositories.Companies
	vault      *security.TokenVault
}

func NewCompanyService(repo *repositories.Companies) *CompanyService {
	return &CompanyService{repository: repo}
}
func NewCompanyServiceWithVault(repo *repositories.Companies, vault *security.TokenVault) *CompanyService {
	return &CompanyService{repository: repo, vault: vault}
}
func (s *CompanyService) Companies(ctx context.Context, q dtos.ListQuery) (dtos.Page[dtos.CompanyView], error) {
	values, err := s.repository.Companies(ctx, q)
	if err == nil {
		err = s.attachProviderRoutes(ctx, values.Items)
	}
	return mapPage(values, dtos.CompanyDTO), dbError(err)
}
func (s *CompanyService) Company(ctx context.Context, id uint64) (dtos.CompanyView, error) {
	value, err := s.repository.CompanyView(ctx, id)
	if err == nil {
		companies := []models.Company{value}
		err = s.attachProviderRoutes(ctx, companies)
		value = companies[0]
	}
	return dtos.CompanyDTO(value), dbError(err)
}
func (s *CompanyService) CompanyByPublicRef(ctx context.Context, publicRef string) (dtos.CompanyView, error) {
	value, err := s.repository.CompanyViewByPublicRef(ctx, publicRef)
	if err == nil {
		companies := []models.Company{value}
		err = s.attachProviderRoutes(ctx, companies)
		value = companies[0]
	}
	return dtos.CompanyDTO(value), dbError(err)
}

func (s *CompanyService) attachProviderRoutes(ctx context.Context, companies []models.Company) error {
	ids := make([]uint64, len(companies))
	byID := make(map[uint64]*models.Company, len(companies))
	for i := range companies {
		ids[i] = companies[i].ID
		byID[ids[i]] = &companies[i]
	}
	routes, err := s.repository.ProviderRoutes(ctx, ids)
	if err != nil {
		return err
	}
	for _, route := range routes {
		if company := byID[route.CompanyID]; company != nil {
			company.ProviderRoutes = append(company.ProviderRoutes, route)
		}
	}
	return nil
}
func (s *CompanyService) Health(ctx context.Context, publicRef string) (dtos.CompanyHealth, error) {
	company, err := s.repository.CompanyViewByPublicRef(ctx, publicRef)
	if err != nil {
		return dtos.CompanyHealth{}, dbError(err)
	}
	value, err := s.repository.Health(ctx, company.ID)
	return value, dbError(err)
}
func (s *CompanyService) SaveCompany(ctx context.Context, actor, id uint64, body dtos.CompanyBody) (dtos.CompanyView, error) {
	var result models.Company
	if id == 0 && security.Scope(ctx).Restricted {
		return dtos.CompanyView{}, apperrors.Wrap(apperrors.CodeForbidden, nil)
	}
	if strings.TrimSpace(body.Name) == "" {
		return dtos.CompanyView{}, apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	if body.ProviderToken != nil && body.ClearProviderToken {
		return dtos.CompanyView{}, apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	if body.SMSProvider != "" && !body.SMSProvider.AvailableForSMS() {
		return dtos.CompanyView{}, apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	plannedRoutes, err := validatePlannedRoutes(body.PlannedRoutes)
	if err != nil {
		return dtos.CompanyView{}, err
	}
	err = s.repository.Transaction(ctx, func(repo *repositories.Companies) error {
		action := auditCreated
		if id != 0 {
			var err error
			result, err = repo.Company(ctx, id)
			if err != nil {
				return err
			}
			action = auditUpdated
		}
		provider := result.SMSProvider
		if provider == "" {
			provider = models.ProviderZoug
		}
		if body.SMSProvider != "" {
			provider = body.SMSProvider
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
		if err = repo.SetSMSProvider(ctx, result.ID, provider); err != nil {
			return err
		}
		if plannedRoutes != nil {
			if err = repo.ReplacePlannedRoutes(ctx, result.ID, plannedRoutes); err != nil {
				return err
			}
		}
		result.SMSProvider = provider
		if body.ProviderToken != nil {
			if s.vault == nil {
				return apperrors.Wrap(apperrors.CodeServiceUnavailable, nil)
			}
			result.ProviderTokenEncrypted, err = s.vault.Encrypt(fmt.Sprintf("company:%d", result.ID), *body.ProviderToken)
			if err != nil {
				return err
			}
			result.ProviderTokenLegacyFallback = false
			if err = repo.Save(ctx, &result); err != nil {
				return err
			}
		} else if body.ClearProviderToken {
			result.ProviderTokenEncrypted = ""
			result.ProviderTokenLegacyFallback = false
			if err = repo.Save(ctx, &result); err != nil {
				return err
			}
		}
		return repo.Audit(ctx, &actor, action, string(security.ResourceCompanies), &result.ID)
	})
	if err == nil {
		companies := []models.Company{result}
		err = s.attachProviderRoutes(ctx, companies)
		result = companies[0]
	}
	return dtos.CompanyDTO(result), dbError(err)
}

func validatePlannedRoutes(input *[]dtos.PlannedProviderRoute) ([]models.CompanyProviderRoute, error) {
	if input == nil {
		return nil, nil
	}
	seen := make(map[models.Channel]bool, len(*input))
	routes := make([]models.CompanyProviderRoute, 0, len(*input))
	for _, route := range *input {
		if seen[route.Channel] || !route.Provider.PlannedFor(route.Channel) {
			return nil, apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
		}
		seen[route.Channel] = true
		routes = append(routes, models.CompanyProviderRoute{Channel: route.Channel, ProviderCode: route.Provider, Status: models.ProviderRoutePlanned})
	}
	return routes, nil
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
