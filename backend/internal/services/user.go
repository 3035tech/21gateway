package services

import (
	"context"
	"errors"
	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/repositories"
	"github.com/21mobile/sms-gateway/internal/security"
	"gorm.io/gorm"
	"net/mail"
	"strings"
	"time"
)

type UserService struct {
	repository  *repositories.Users
	permissions *security.Permissions
	dummyHash   string
	checks      chan struct{}
	mfa         *UserMFAService
}

func NewUserService(repository *repositories.Users) (*UserService, error) {
	permissions, err := security.NewPermissions()
	if err != nil {
		return nil, err
	}
	dummy, err := security.HashPassword("dummy-backoffice-not-an-account")
	if err != nil {
		return nil, err
	}
	return &UserService{repository: repository, permissions: permissions, dummyHash: dummy, checks: make(chan struct{}, security.MaxConcurrentPasswordChecks)}, nil
}
func (s *UserService) SetMFA(service *UserMFAService) { s.mfa = service }
func (s *UserService) Allowed(role models.Role, resource security.Resource, action security.Action) bool {
	return s.permissions.Allow(role, resource, action)
}
func (s *UserService) Login(ctx context.Context, email, password string) (models.SystemUser, error) {
	return s.LoginWithFactor(ctx, email, password, "")
}
func (s *UserService) LoginWithFactor(ctx context.Context, email, password, factor string) (models.SystemUser, error) {
	if len(password) > security.MaximumPasswordLength {
		return models.SystemUser{}, apperrors.Wrap(apperrors.CodeUnauthorized, nil)
	}
	protection := repositories.NewAuthenticationAttempts(s.repository.DB())
	accountKey := security.AuthenticationKey("admin-account", strings.ToLower(strings.TrimSpace(email)))
	blocked, err := protection.Blocked(ctx, accountKey, security.AuthenticationFailureLimit)
	if err != nil {
		return models.SystemUser{}, dbError(err)
	}
	if blocked {
		return models.SystemUser{}, apperrors.Wrap(apperrors.CodeRateLimited, nil)
	}
	select {
	case s.checks <- struct{}{}:
		defer func() { <-s.checks }()
	default:
		return models.SystemUser{}, apperrors.Wrap(apperrors.CodeServiceUnavailable, nil)
	}
	user, err := s.repository.UserByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return user, dbError(err)
	}
	found := err == nil
	hash := s.dummyHash
	if found {
		hash = user.PasswordHash
	}
	match, err := security.VerifyPassword(password, hash)
	if err != nil {
		return models.SystemUser{}, apperrors.Wrap(apperrors.CodeServiceUnavailable, err)
	}
	if !found || !match || !user.Active {
		if err := protection.Failure(ctx, accountKey, security.AuthenticationFailureLimit); err != nil {
			return models.SystemUser{}, dbError(err)
		}
		return models.SystemUser{}, apperrors.Wrap(apperrors.CodeUnauthorized, nil)
	}
	if s.mfa != nil {
		if err := s.mfa.VerifyLogin(ctx, user.ID, factor); err != nil {
			var appErr *apperrors.Error
			if errors.As(err, &appErr) && appErr.Code == apperrors.CodeMFAInvalid {
				if failure := protection.Failure(ctx, accountKey, security.AuthenticationFailureLimit); failure != nil {
					return models.SystemUser{}, dbError(failure)
				}
			}
			return models.SystemUser{}, err
		}
	}
	now := time.Now().UTC()
	user.LastLoginAt = &now
	if err := s.repository.Transaction(ctx, func(repo *repositories.Users) error {
		if err := repo.TouchLogin(ctx, user, now); err != nil {
			return err
		}
		return repo.Audit(ctx, &user.ID, auditLogin, string(security.ResourceUsers), &user.ID)
	}); err != nil {
		return models.SystemUser{}, dbError(err)
	}
	user.PasswordHash = ""
	return user, nil
}
func (s *UserService) SessionUser(ctx context.Context, id uint64, version int64) (models.SystemUser, error) {
	if id == 0 || version < 1 {
		return models.SystemUser{}, apperrors.Wrap(apperrors.CodeUnauthorized, nil)
	}
	user, err := s.repository.User(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return user, apperrors.Wrap(apperrors.CodeUnauthorized, nil)
		}
		return user, dbError(err)
	}
	if !user.Active || user.SessionVersion != version {
		return models.SystemUser{}, apperrors.Wrap(apperrors.CodeUnauthorized, nil)
	}
	user.PasswordHash = ""
	return user, nil
}
func (s *UserService) ChangeOwnPassword(ctx context.Context, actor models.SystemUser, currentPassword, newPassword string) error {
	if len(currentPassword) == 0 || len(currentPassword) > security.MaximumPasswordLength || !security.ValidNewSystemPassword(newPassword) || currentPassword == newPassword {
		return apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	accountKey := security.AuthenticationKey("admin-account", strings.ToLower(actor.Email))
	protection := repositories.NewAuthenticationAttempts(s.repository.DB())
	blocked, err := protection.Blocked(ctx, accountKey, security.AuthenticationFailureLimit)
	if err != nil {
		return dbError(err)
	}
	if blocked {
		return apperrors.Wrap(apperrors.CodeRateLimited, nil)
	}
	select {
	case s.checks <- struct{}{}:
		defer func() { <-s.checks }()
	default:
		return apperrors.Wrap(apperrors.CodeServiceUnavailable, nil)
	}
	user, err := s.repository.User(ctx, actor.ID)
	if err != nil {
		return dbError(err)
	}
	if user.SessionVersion != actor.SessionVersion || !user.Active {
		return apperrors.Wrap(apperrors.CodeUnauthorized, nil)
	}
	match, err := security.VerifyPassword(currentPassword, user.PasswordHash)
	if err != nil {
		return apperrors.Wrap(apperrors.CodeServiceUnavailable, err)
	}
	if !match {
		if err := protection.Failure(ctx, accountKey, security.AuthenticationFailureLimit); err != nil {
			return dbError(err)
		}
		return apperrors.Wrap(apperrors.CodeInvalidCurrentPassword, nil)
	}
	hash, err := security.HashPassword(newPassword)
	if err != nil {
		return apperrors.Wrap(apperrors.CodeInternal, err)
	}
	return dbError(s.repository.Transaction(ctx, func(repo *repositories.Users) error {
		if err := repo.ReplaceOwnPassword(ctx, actor.ID, actor.SessionVersion, user.PasswordHash, hash); err != nil {
			return err
		}
		return repo.Audit(ctx, &actor.ID, auditRotated, string(security.ResourceUsers), &actor.ID)
	}))
}
func (s *UserService) Users(ctx context.Context, q dtos.ListQuery) (dtos.Page[dtos.UserView], error) {
	values, err := s.repository.Users(ctx, q)
	return mapPage(values, dtos.UserDTO), dbError(err)
}
func validRole(role models.Role) bool {
	return role == models.RoleAdmin || role == models.RoleOperator || role == models.RoleViewer
}
func (s *UserService) SaveUser(ctx context.Context, actor, id uint64, body dtos.UserBody) (dtos.UserView, error) {
	var result models.SystemUser
	address, err := mail.ParseAddress(strings.TrimSpace(body.Email))
	if err != nil || address.Address != strings.TrimSpace(body.Email) || strings.TrimSpace(body.Name) == "" || !validRole(body.Role) {
		return dtos.UserView{}, apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	hash := ""
	if body.Password != "" {
		if !security.ValidNewSystemPassword(body.Password) {
			return dtos.UserView{}, apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
		}
		hash, err = security.HashPassword(body.Password)
		if err != nil {
			return dtos.UserView{}, apperrors.Wrap(apperrors.CodeInternal, err)
		}
	} else if id == 0 {
		return dtos.UserView{}, apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	err = s.repository.Transaction(ctx, func(repo *repositories.Users) error {
		if err := repo.LockUsers(ctx); err != nil {
			return err
		}
		action := auditCreated
		if id != 0 {
			var err error
			result, err = repo.User(ctx, id)
			if err != nil {
				return err
			}
			if result.Role == models.RoleAdmin && result.Active && (!body.Active || body.Role != models.RoleAdmin) {
				count, err := repo.AdminCount(ctx, id)
				if err != nil {
					return err
				}
				if count == 0 {
					return apperrors.Wrap(apperrors.CodeLastAdmin, nil)
				}
			}
			action = auditUpdated
		} else {
			result.SessionVersion = initialSessionVersion
		}
		if id != 0 && (hash != "" || result.Role != body.Role || result.Active != body.Active) {
			result.SessionVersion++
		}
		if hash != "" {
			result.PasswordHash = hash
		}
		result.Name = strings.TrimSpace(body.Name)
		result.Email = strings.ToLower(address.Address)
		result.Role = body.Role
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
		return repo.Audit(ctx, &actor, action, string(security.ResourceUsers), &result.ID)
	})
	return dtos.UserDTO(result), dbError(err)
}
func (s *UserService) BootstrapAdmin(ctx context.Context, name, email, password string) error {
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email || strings.TrimSpace(name) == "" || !security.ValidNewSystemPassword(password) {
		return apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	hash, err := security.HashPassword(password)
	if err != nil {
		return err
	}
	return dbError(s.repository.Transaction(ctx, func(repo *repositories.Users) error {
		if err := repo.LockUsers(ctx); err != nil {
			return err
		}
		count, err := repo.UserCount(ctx)
		if err != nil {
			return err
		}
		if count != 0 {
			return apperrors.Wrap(apperrors.CodeConflict, nil)
		}
		user := models.SystemUser{Name: strings.TrimSpace(name), Email: strings.ToLower(email), PasswordHash: hash, Role: models.RoleAdmin, Active: true, SessionVersion: initialSessionVersion}
		if err := repo.Create(ctx, &user); err != nil {
			return err
		}
		return repo.Audit(ctx, nil, auditCreated, string(security.ResourceUsers), &user.ID)
	}))
}
func (s *UserService) AllowLogin(ctx context.Context, key string) error {
	ok, err := s.repository.AllowLogin(ctx, key)
	if err != nil {
		return dbError(err)
	}
	if !ok {
		return apperrors.Wrap(apperrors.CodeRateLimited, nil)
	}
	return nil
}
func (s *UserService) Archive(ctx context.Context, actor, id uint64) error {
	return dbError(s.repository.Transaction(ctx, func(repo *repositories.Users) error {
		now := time.Now().UTC()
		if err := repo.LockUsers(ctx); err != nil {
			return err
		}
		value, found := repo.User(ctx, id)
		if found != nil {
			return found
		}
		if value.Role == models.RoleAdmin && value.Active {
			count, err := repo.AdminCount(ctx, id)
			if err != nil {
				return err
			}
			if count == 0 {
				return apperrors.Wrap(apperrors.CodeLastAdmin, nil)
			}
		}
		value.Active = false
		value.ArchivedAt = &now
		value.SessionVersion++
		if err := repo.Save(ctx, &value); err != nil {
			return err
		}
		return repo.Audit(ctx, &actor, auditArchived, string(security.ResourceUsers), &id)
	}))
}
