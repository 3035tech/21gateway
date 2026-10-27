package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/repositories"
	"github.com/21mobile/sms-gateway/internal/security"
	"gorm.io/gorm"
)

const passwordResetLifetime = 15 * time.Minute
const passwordResetTokenBytes = 32
const passwordResetRequestLimit = 3
const passwordResetIPLimit = 10
const recoveryPurposeReset = "reset"
const recoveryPurposeInvite = "invite"
const recoveryResponseFloor = 150 * time.Millisecond
const recoveryMailTimeout = 12 * time.Second
const recoveryParallelEmails = 4

type PasswordResetMailer interface {
	SendPasswordReset(context.Context, string, string) error
	SendInvitation(context.Context, string, string) error
}
type UserRecoveryService struct {
	repo          *repositories.UserRecovery
	users         *repositories.Users
	mfa           *UserMFAService
	mailer        PasswordResetMailer
	userService   *UserService
	deliverySlots chan struct{}
}

func (s *UserRecoveryService) SetUserService(users *UserService) { s.userService = users }
func (s *UserRecoveryService) Invite(ctx context.Context, actor uint64, body dtos.UserBody) (dtos.UserView, error) {
	if s.mailer == nil || s.userService == nil {
		return dtos.UserView{}, apperrors.Wrap(apperrors.CodeServiceUnavailable, nil)
	}
	raw := make([]byte, passwordResetTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return dtos.UserView{}, apperrors.Wrap(apperrors.CodeInternal, err)
	}
	body.Password = hex.EncodeToString(raw)
	user, err := s.userService.SaveUser(ctx, actor, 0, body)
	if err != nil {
		return dtos.UserView{}, err
	}
	if err := s.issue(ctx, user.ID, user.Email, recoveryPurposeInvite); err != nil {
		return user, err
	}
	return user, nil
}
func (s *UserRecoveryService) ResendInvitation(ctx context.Context, id uint64) error {
	if s.mailer == nil {
		return apperrors.Wrap(apperrors.CodeServiceUnavailable, nil)
	}
	user, err := s.users.User(ctx, id)
	if err != nil {
		return dbError(err)
	}
	if !user.Active {
		return apperrors.Wrap(apperrors.CodeConflict, nil)
	}
	key := security.AuthenticationKey("admin-invite", strings.ToLower(user.Email))
	ok, err := repositories.NewAuthenticationAttempts(s.users.DB()).Reserve(ctx, key, passwordResetRequestLimit)
	if err != nil {
		return dbError(err)
	}
	if !ok {
		return apperrors.Wrap(apperrors.CodeRateLimited, nil)
	}
	return s.issue(ctx, user.ID, user.Email, recoveryPurposeInvite)
}

func NewUserRecoveryService(repo *repositories.UserRecovery, users *repositories.Users, mfa *UserMFAService, mailer PasswordResetMailer) *UserRecoveryService {
	return &UserRecoveryService{repo: repo, users: users, mfa: mfa, mailer: mailer, deliverySlots: make(chan struct{}, recoveryParallelEmails)}
}
func (s *UserRecoveryService) AllowIP(ctx context.Context, peer string) error {
	key := security.AuthenticationKey("password-reset-ip", peer)
	ok, err := repositories.NewAuthenticationAttempts(s.users.DB()).Reserve(ctx, key, passwordResetIPLimit)
	if err != nil {
		return dbError(err)
	}
	if !ok {
		return apperrors.Wrap(apperrors.CodeRateLimited, nil)
	}
	return nil
}
func (s *UserRecoveryService) Request(ctx context.Context, email string) error {
	started := time.Now()
	defer func() {
		if remaining := recoveryResponseFloor - time.Since(started); remaining > 0 {
			select {
			case <-time.After(remaining):
			case <-ctx.Done():
			}
		}
	}()
	if s.mailer == nil {
		return apperrors.Wrap(apperrors.CodeServiceUnavailable, nil)
	}
	email = strings.ToLower(strings.TrimSpace(email))
	if len(email) > 254 || !strings.Contains(email, "@") {
		return apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	key := security.AuthenticationKey("password-reset", email)
	ok, err := repositories.NewAuthenticationAttempts(s.users.DB()).Reserve(ctx, key, passwordResetRequestLimit)
	if err != nil {
		return dbError(err)
	}
	if !ok {
		return apperrors.Wrap(apperrors.CodeRateLimited, nil)
	}
	user, err := s.users.UserByEmail(ctx, email)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil
	}
	if err != nil {
		return dbError(err)
	}
	if !user.Active {
		return nil
	}
	// Deliver after responding so SMTP latency does not reveal account existence.
	token, err := s.issueToken(ctx, user.ID, recoveryPurposeReset)
	if err != nil {
		slog.Error("password recovery request failed")
		return nil
	}
	select {
	case s.deliverySlots <- struct{}{}:
		go func() {
			defer func() { <-s.deliverySlots }()
			mailCtx, cancel := context.WithTimeout(context.Background(), recoveryMailTimeout)
			defer cancel()
			if err := s.mailer.SendPasswordReset(mailCtx, user.Email, token); err != nil {
				slog.Error("password recovery mail delivery failed")
			}
		}()
	default:
		slog.Error("password recovery mail capacity exhausted")
	}
	return nil
}
func (s *UserRecoveryService) issue(ctx context.Context, id uint64, email, purpose string) error {
	token, err := s.issueToken(ctx, id, purpose)
	if err != nil {
		return err
	}
	if purpose == recoveryPurposeInvite {
		if err := s.mailer.SendInvitation(ctx, email, token); err != nil {
			return apperrors.Wrap(apperrors.CodeInvitationDeliveryFailed, err)
		}
		return nil
	}
	if err := s.mailer.SendPasswordReset(ctx, email, token); err != nil {
		return apperrors.Wrap(apperrors.CodeServiceUnavailable, err)
	}
	return nil
}
func (s *UserRecoveryService) issueToken(ctx context.Context, id uint64, purpose string) (string, error) {
	raw := make([]byte, passwordResetTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", apperrors.Wrap(apperrors.CodeInternal, err)
	}
	token := hex.EncodeToString(raw)
	hash := sha256.Sum256([]byte(token))
	if err := s.repo.Store(ctx, id, hash[:], purpose, time.Now().UTC().Add(passwordResetLifetime)); err != nil {
		return "", dbError(err)
	}
	return token, nil
}
func (s *UserRecoveryService) Reset(ctx context.Context, token, factor, newPassword string) error {
	if len(token) != passwordResetTokenBytes*2 || !security.ValidNewSystemPassword(newPassword) {
		return apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	if _, err := hex.DecodeString(token); err != nil {
		return apperrors.Wrap(apperrors.CodeInvalidResetToken, nil)
	}
	hash := sha256.Sum256([]byte(token))
	entry, err := s.repo.ByHash(ctx, hash[:])
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return apperrors.Wrap(apperrors.CodeInvalidResetToken, nil)
		}
		return dbError(err)
	}
	if s.mfa != nil {
		if err := s.mfa.VerifyLogin(ctx, entry.UserID, factor); err != nil {
			return err
		}
	}
	passwordHash, err := security.HashPassword(newPassword)
	if err != nil {
		return apperrors.Wrap(apperrors.CodeInternal, err)
	}
	return dbError(s.repo.Transaction(ctx, func(repo *repositories.UserRecovery) error {
		used, err := repo.Consume(ctx, entry.UserID, hash[:])
		if err != nil {
			return err
		}
		if !used {
			return apperrors.Wrap(apperrors.CodeInvalidResetToken, nil)
		}
		if err := repo.SetPassword(ctx, entry.UserID, passwordHash); err != nil {
			return err
		}
		action := auditPasswordRecovered
		if entry.Purpose == recoveryPurposeInvite {
			action = auditInviteAccepted
		}
		return repo.Audit(ctx, entry.UserID, action)
	}))
}
