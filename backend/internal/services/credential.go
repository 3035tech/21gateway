package services

import (
	"context"
	"crypto/rand"
	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/repositories"
	"github.com/21mobile/sms-gateway/internal/security"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"
)

type CredentialService struct {
	repository *repositories.Credentials
	vault      *security.TokenVault
}

func NewCredentialService(repo *repositories.Credentials) *CredentialService {
	return &CredentialService{repository: repo}
}
func NewCredentialServiceWithVault(repo *repositories.Credentials, vault *security.TokenVault) *CredentialService {
	return &CredentialService{repository: repo, vault: vault}
}
func (s *CredentialService) Credentials(ctx context.Context, q dtos.ListQuery) (dtos.Page[dtos.CredentialView], error) {
	values, err := s.repository.Credentials(ctx, q)
	return mapPage(values, dtos.CredentialDTO), dbError(err)
}
func (s *CredentialService) Credential(ctx context.Context, id uint64) (dtos.CredentialView, error) {
	value, err := s.repository.CredentialView(ctx, id)
	return dtos.CredentialDTO(value), dbError(err)
}
func (s *CredentialService) CredentialByPublicRef(ctx context.Context, publicRef string) (dtos.CredentialView, error) {
	value, err := s.repository.CredentialViewByPublicRef(ctx, publicRef)
	return dtos.CredentialDTO(value), dbError(err)
}
func ValidateCallbackURL(raw string) bool {
	if raw == "" {
		return true
	}
	parsed, err := url.Parse(raw)
	return err == nil && parsed.Scheme == "https" && parsed.Hostname() != "" && parsed.User == nil && parsed.Fragment == "" && (parsed.Port() == "" || parsed.Port() == "443")
}
func (s *CredentialService) SaveCredential(ctx context.Context, actor, id uint64, body dtos.CredentialBody) (dtos.CredentialSecret, error) {
	if body.ProviderToken != nil || body.ClearProviderToken {
		return dtos.CredentialSecret{}, apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	var result models.APICredential
	secret := ""
	username := body.Username
	if body.GenerateUsername {
		if id != 0 || username != "" {
			return dtos.CredentialSecret{}, apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
		}
		username = "gw_" + rand.Text()
	}
	if strings.TrimSpace(username) == "" || strings.ContainsAny(username, ":\r\n") || !ValidateCallbackURL(body.CallbackURL) || (body.CallbacksEnabled && body.CallbackURL == "") || (body.ManualPassword != nil && body.LegacyPassword != nil) || (body.ExpiresAt != nil && body.ClearExpiry) {
		return dtos.CredentialSecret{}, apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	if body.ExpiresAt != nil && !body.ExpiresAt.After(time.Now()) {
		return dtos.CredentialSecret{}, apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	hash := ""
	var err error
	if id != 0 && (body.LegacyPassword != nil || body.ManualPassword != nil) {
		return dtos.CredentialSecret{}, apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	if id == 0 {
		password := ""
		if body.ManualPassword != nil {
			password = *body.ManualPassword
			if utf8.RuneCountInString(strings.TrimSpace(password)) < 15 || len(password) > security.MaximumPasswordLength {
				return dtos.CredentialSecret{}, apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
			}
		} else if body.LegacyPassword != nil {
			password = *body.LegacyPassword
			if len(password) < 1 || len(password) > security.MaximumPasswordLength {
				return dtos.CredentialSecret{}, apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
			}
		} else {
			secret = rand.Text()
			password = secret
		}
		hash, err = security.HashPassword(password)
		if err != nil {
			return dtos.CredentialSecret{}, err
		}
	}
	err = s.repository.Transaction(ctx, func(repo *repositories.Credentials) error {
		company, err := repositories.NewCompanies(repo.DB()).Company(ctx, body.CompanyID)
		if err != nil {
			return err
		}
		action := auditCreated
		if id != 0 {
			result, err = repo.Credential(ctx, id)
			if err != nil {
				return err
			}
			if result.Username != username || (result.CompanyID != nil && *result.CompanyID != body.CompanyID) {
				return apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
			}
			action = auditUpdated
		} else {
			result.Username = username
			result.PasswordHash = hash
		}
		result.CompanyID = &body.CompanyID
		result.CompanyName = company.Name
		result.CompanyProviderTokenEncrypted = company.ProviderTokenEncrypted
		result.CompanyProviderTokenLegacyFallback = company.ProviderTokenLegacyFallback
		result.Active = body.Active
		result.CallbackURL = body.CallbackURL
		result.CallbacksEnabled = body.CallbacksEnabled
		if body.ClearExpiry {
			result.ExpiresAt = nil
		} else if body.ExpiresAt != nil {
			expiresAt := body.ExpiresAt.UTC()
			result.ExpiresAt = &expiresAt
		}
		if err := s.applySettings(&result, body); err != nil {
			return err
		}
		if id == 0 {
			err = repo.Create(ctx, &result)
		} else {
			err = repo.Save(ctx, &result)
		}
		if err != nil {
			return err
		}
		return repo.Audit(ctx, &actor, action, string(security.ResourceCredentials), &result.ID)
	})
	return dtos.CredentialSecret{Credential: dtos.CredentialDTO(result), Password: secret}, dbError(err)
}
func (s *CredentialService) RotateCredential(ctx context.Context, actor, id uint64) (dtos.CredentialSecret, error) {
	secret := rand.Text()
	hash, err := security.HashPassword(secret)
	if err != nil {
		return dtos.CredentialSecret{}, err
	}
	var result models.APICredential
	err = s.repository.Transaction(ctx, func(repo *repositories.Credentials) error {
		var err error
		result, err = repo.Credential(ctx, id)
		if err != nil {
			return err
		}
		result.PasswordHash = hash
		if err := repo.Save(ctx, &result); err != nil {
			return err
		}
		return repo.Audit(ctx, &actor, auditRotated, string(security.ResourceCredentials), &result.ID)
	})
	return dtos.CredentialSecret{Credential: dtos.CredentialDTO(result), Password: secret}, dbError(err)
}
func (s *CredentialService) Archive(ctx context.Context, actor, id uint64) error {
	return dbError(s.repository.Transaction(ctx, func(repo *repositories.Credentials) error {
		now := time.Now().UTC()
		value, found := repo.Credential(ctx, id)
		if found != nil {
			return found
		}
		value.Active = false
		value.ArchivedAt = &now
		if err := repo.Save(ctx, &value); err != nil {
			return err
		}
		return repo.Audit(ctx, &actor, auditArchived, string(security.ResourceCredentials), &id)
	}))
}

func (s *CredentialService) applySettings(result *models.APICredential, body dtos.CredentialBody) error {
	if body.WhitelistEnabled != nil {
		result.WhitelistEnabled = *body.WhitelistEnabled
	}
	if body.AllowedIPs != nil {
		values, err := security.NormalizeAllowedIPs(*body.AllowedIPs)
		if err != nil {
			return err
		}
		result.AllowedIPs = values
	}
	if result.WhitelistEnabled && len(result.AllowedIPs) == 0 {
		return apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	if body.MOCallbackURL != nil {
		result.MOCallbackURL = *body.MOCallbackURL
	}
	if body.MOCallbacksEnabled != nil {
		result.MOCallbacksEnabled = *body.MOCallbacksEnabled
	}
	if !ValidateCallbackURL(result.MOCallbackURL) || (result.MOCallbacksEnabled && result.MOCallbackURL == "") {
		return apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	if body.LegacyAccountID != nil {
		if result.LegacyAccountID != "" && result.LegacyAccountID != *body.LegacyAccountID {
			return apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
		}
		result.LegacyAccountID = *body.LegacyAccountID
	}
	return nil
}
