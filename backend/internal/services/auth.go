package services

import (
	"context"
	"errors"
	"net/http"
	"time"

	"gorm.io/gorm"

	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/security"
)

type CredentialReader interface {
	FindByUsername(context.Context, string) (models.APICredential, error)
}
type AuthenticationProtection interface {
	Reserve(context.Context, string, int) (bool, error)
	Blocked(context.Context, string, int) (bool, error)
	Failure(context.Context, string, int) error
}
type credentialContextKey struct{}

type AuthService struct {
	repository CredentialReader
	protection AuthenticationProtection
	ipLimit    int
	dummyHash  string
	checks     chan struct{}
}

func NewAuthService(repository CredentialReader) (*AuthService, error) {
	dummy, err := security.HashPassword("dummy-credential-not-an-account")
	if err != nil {
		return nil, apperrors.Wrap(apperrors.CodeInternal, err)
	}
	return &AuthService{repository: repository, dummyHash: dummy, checks: make(chan struct{}, security.MaxConcurrentPasswordChecks)}, nil
}

// Runtime uses shared PostgreSQL protection; the base constructor permits isolated tests.
func NewAuthServiceWithProtection(repository CredentialReader, protection AuthenticationProtection, ipLimit int) (*AuthService, error) {
	if protection == nil || ipLimit < 1 || ipLimit > security.MaximumAuthenticationIPLimit {
		return nil, apperrors.Wrap(apperrors.CodeConfigurationInvalid, nil)
	}
	service, err := NewAuthService(repository)
	if err != nil {
		return nil, err
	}
	service.protection = protection
	service.ipLimit = ipLimit
	return service, nil
}
func (s *AuthService) AuthenticateRequest(ctx context.Context, authorization, remote string) (models.APICredential, error) {
	if s.protection != nil {
		allowed, err := s.protection.Reserve(ctx, security.AuthenticationKey("api-ip", security.AuthenticationPeer(remote)), s.ipLimit)
		if err != nil {
			return models.APICredential{}, apperrors.Wrap(apperrors.CodeServiceUnavailable, err)
		}
		if !allowed {
			return models.APICredential{}, apperrors.Wrap(apperrors.CodeRateLimited, nil)
		}
	}
	return s.Authenticate(ctx, authorization)
}
func (s *AuthService) Authenticate(ctx context.Context, authorization string) (models.APICredential, error) {
	var empty models.APICredential
	request := &http.Request{Header: http.Header{"Authorization": []string{authorization}}}
	username, password, ok := request.BasicAuth()
	if !ok || username == "" || len(username) > security.MaximumUsernameLength || len(password) > security.MaximumPasswordLength {
		return empty, apperrors.Wrap(apperrors.CodeUnauthorized, nil)
	}
	accountKey := security.AuthenticationKey("api-account", username)
	if s.protection != nil {
		blocked, err := s.protection.Blocked(ctx, accountKey, security.AuthenticationFailureLimit)
		if err != nil {
			return empty, apperrors.Wrap(apperrors.CodeServiceUnavailable, err)
		}
		if blocked {
			return empty, apperrors.Wrap(apperrors.CodeRateLimited, nil)
		}
	}
	select {
	case s.checks <- struct{}{}:
		defer func() { <-s.checks }()
	default:
		return empty, apperrors.Wrap(apperrors.CodeServiceUnavailable, nil)
	}
	credential, err := s.repository.FindByUsername(ctx, username)
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return empty, apperrors.Wrap(apperrors.CodeServiceUnavailable, err)
	}
	found := err == nil
	hash := s.dummyHash
	if found {
		hash = credential.PasswordHash
	}
	match, err := security.VerifyPassword(password, hash)
	if err != nil {
		return empty, apperrors.Wrap(apperrors.CodeServiceUnavailable, err)
	}
	if !found || !match || !credential.ValidAt(time.Now()) || (credential.CompanyID != nil && (credential.CompanyActive == nil || !*credential.CompanyActive)) {
		if s.protection != nil {
			if err := s.protection.Failure(ctx, accountKey, security.AuthenticationFailureLimit); err != nil {
				return empty, apperrors.Wrap(apperrors.CodeServiceUnavailable, err)
			}
		}
		return empty, apperrors.Wrap(apperrors.CodeUnauthorized, nil)
	}
	// The password hash is not needed downstream and must not enter request logs.
	credential.PasswordHash = ""
	return credential, nil
}

func WithCredential(ctx context.Context, credential models.APICredential) context.Context {
	return context.WithValue(ctx, credentialContextKey{}, credential)
}
func CredentialFromContext(ctx context.Context) (models.APICredential, bool) {
	credential, ok := ctx.Value(credentialContextKey{}).(models.APICredential)
	return credential, ok
}

func (s *AuthService) AuthorizeIP(credential models.APICredential, remoteAddr string) error {
	if credential.WhitelistEnabled && !security.IPAllowed(remoteAddr, credential.AllowedIPs) {
		return apperrors.Wrap(apperrors.CodeForbidden, nil)
	}
	return nil
}
