package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"

	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/repositories"
	"github.com/21mobile/sms-gateway/internal/security"
	"github.com/pquerna/otp/totp"
	"gorm.io/gorm"
)

const mfaPeriod = 30
const mfaSetupLifetime = 10 * time.Minute
const recoveryCodeCount = 10
const recoveryCodeBytes = 16

type UserMFAService struct {
	repo   *repositories.UserMFA
	users  *repositories.Users
	vault  *security.TokenVault
	checks chan struct{}
}

func NewUserMFAService(repo *repositories.UserMFA, users *repositories.Users, vault *security.TokenVault) *UserMFAService {
	return &UserMFAService{repo: repo, users: users, vault: vault, checks: make(chan struct{}, security.MaxConcurrentPasswordChecks)}
}
func (s *UserMFAService) encryptedID(id uint64) string { return strconv.FormatUint(id, 10) }
func (s *UserMFAService) enabled() bool                { return s.vault != nil && s.vault.Enabled() }
func (s *UserMFAService) Status(ctx context.Context, id uint64) (bool, error) {
	row, err := s.repo.Get(ctx, id, false)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}
	return row.SecretEncrypted != "", dbError(err)
}
func (s *UserMFAService) Begin(ctx context.Context, user models.SystemUser, password string) (string, string, error) {
	if !s.enabled() {
		return "", "", apperrors.Wrap(apperrors.CodeServiceUnavailable, nil)
	}
	if err := s.verifyPassword(ctx, user, password); err != nil {
		return "", "", err
	}
	key, err := totp.Generate(totp.GenerateOpts{Issuer: "21 Mobile Gateway", AccountName: user.Email})
	if err != nil {
		return "", "", apperrors.Wrap(apperrors.CodeInternal, err)
	}
	cipher, err := s.vault.Encrypt(s.encryptedID(user.ID), key.Secret())
	if err != nil {
		return "", "", err
	}
	err = s.repo.Transaction(ctx, func(repo *repositories.UserMFA) error {
		row, err := repo.Get(ctx, user.ID, true)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if row.SecretEncrypted != "" {
			return apperrors.Wrap(apperrors.CodeConflict, nil)
		}
		row.UserID = user.ID
		row.PendingEncrypted = cipher
		expires := time.Now().UTC().Add(mfaSetupLifetime)
		row.PendingExpiresAt = &expires
		return repo.Save(ctx, row)
	})
	if err != nil {
		return "", "", dbError(err)
	}
	return key.Secret(), key.URL(), nil
}
func (s *UserMFAService) Confirm(ctx context.Context, id uint64, code string) ([]string, error) {
	if !s.enabled() {
		return nil, apperrors.Wrap(apperrors.CodeServiceUnavailable, nil)
	}
	if err := s.allowFactor(ctx, id); err != nil {
		return nil, err
	}
	codes := make([]string, recoveryCodeCount)
	hashes := make([][]byte, recoveryCodeCount)
	for i := range codes {
		raw := make([]byte, recoveryCodeBytes)
		if _, err := rand.Read(raw); err != nil {
			return nil, apperrors.Wrap(apperrors.CodeInternal, err)
		}
		codes[i] = hex.EncodeToString(raw)
		digest := sha256.Sum256([]byte(codes[i]))
		hashes[i] = digest[:]
	}
	err := s.repo.Transaction(ctx, func(repo *repositories.UserMFA) error {
		row, err := repo.Get(ctx, id, true)
		if err != nil {
			return err
		}
		if row.SecretEncrypted != "" || row.PendingEncrypted == "" || row.PendingExpiresAt == nil || time.Now().After(*row.PendingExpiresAt) {
			return apperrors.Wrap(apperrors.CodeConflict, nil)
		}
		secret, err := s.vault.Decrypt(s.encryptedID(id), row.PendingEncrypted)
		if err != nil {
			return err
		}
		step, valid := validTOTPStep(secret, code, time.Now().UTC(), 0)
		if !valid {
			return apperrors.Wrap(apperrors.CodeMFAInvalid, nil)
		}
		row.SecretEncrypted = row.PendingEncrypted
		row.PendingEncrypted = ""
		row.PendingExpiresAt = nil
		row.LastUsedStep = step
		if err := repo.Save(ctx, row); err != nil {
			return err
		}
		if err := repo.AddCodes(ctx, id, hashes); err != nil {
			return err
		}
		if err := repo.RevokeSessions(ctx, id); err != nil {
			return err
		}
		return repo.Audit(ctx, id, auditMFAEnabled)
	})
	if err != nil {
		return nil, dbError(err)
	}
	return codes, nil
}
func (s *UserMFAService) VerifyLogin(ctx context.Context, id uint64, code string) error {
	active, err := s.Status(ctx, id)
	if err != nil {
		return err
	}
	if !active {
		return nil
	}
	if !s.enabled() {
		return apperrors.Wrap(apperrors.CodeServiceUnavailable, nil)
	}
	if code == "" {
		return apperrors.Wrap(apperrors.CodeMFARequired, nil)
	}
	if err := s.allowFactor(ctx, id); err != nil {
		return err
	}
	return s.repo.Transaction(ctx, func(repo *repositories.UserMFA) error {
		row, err := repo.Get(ctx, id, true)
		if err != nil {
			return dbError(err)
		}
		if row.SecretEncrypted == "" {
			return nil
		}
		if len(code) == recoveryCodeBytes*2 {
			if _, err := hex.DecodeString(code); err == nil {
				digest := sha256.Sum256([]byte(code))
				used, err := repo.ConsumeCode(ctx, id, digest[:])
				if err != nil {
					return dbError(err)
				}
				if used {
					return nil
				}
			}
		}
		secret, err := s.vault.Decrypt(s.encryptedID(id), row.SecretEncrypted)
		if err != nil {
			return err
		}
		step, valid := validTOTPStep(secret, code, time.Now().UTC(), row.LastUsedStep)
		if !valid {
			return apperrors.Wrap(apperrors.CodeMFAInvalid, nil)
		}
		advanced, err := repo.AdvanceStep(ctx, id, step)
		if err != nil {
			return dbError(err)
		}
		if !advanced {
			return apperrors.Wrap(apperrors.CodeMFAInvalid, nil)
		}
		return nil
	})
}
func validTOTPStep(secret, code string, now time.Time, last int64) (int64, bool) {
	if len(code) != 6 {
		return 0, false
	}
	current := now.Unix() / mfaPeriod
	for _, step := range []int64{current, current - 1, current + 1} {
		if step <= last {
			continue
		}
		expected, err := totp.GenerateCode(secret, time.Unix(step*mfaPeriod, 0))
		if err == nil && subtle.ConstantTimeCompare([]byte(expected), []byte(code)) == 1 {
			return step, true
		}
	}
	return 0, false
}
func (s *UserMFAService) Disable(ctx context.Context, user models.SystemUser, password, factor string) error {
	if err := s.verifyPassword(ctx, user, password); err != nil {
		return err
	}
	if err := s.VerifyLogin(ctx, user.ID, factor); err != nil {
		return err
	}
	return dbError(s.repo.Transaction(ctx, func(repo *repositories.UserMFA) error {
		row, err := repo.Get(ctx, user.ID, true)
		if err != nil {
			return err
		}
		if row.SecretEncrypted == "" {
			return apperrors.Wrap(apperrors.CodeConflict, nil)
		}
		row.SecretEncrypted = ""
		row.PendingEncrypted = ""
		row.PendingExpiresAt = nil
		row.LastUsedStep = 0
		if err := repo.Save(ctx, row); err != nil {
			return err
		}
		if err := repo.ClearCodes(ctx, user.ID); err != nil {
			return err
		}
		if err := repo.RevokeSessions(ctx, user.ID); err != nil {
			return err
		}
		return repo.Audit(ctx, user.ID, auditMFADisabled)
	}))
}
func (s *UserMFAService) verifyPassword(ctx context.Context, user models.SystemUser, password string) error {
	if password == "" || len(password) > security.MaximumPasswordLength {
		return apperrors.Wrap(apperrors.CodeInvalidCurrentPassword, nil)
	}
	key := security.AuthenticationKey("admin-account", strings.ToLower(user.Email))
	protection := repositories.NewAuthenticationAttempts(s.users.DB())
	blocked, err := protection.Blocked(ctx, key, security.AuthenticationFailureLimit)
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
	stored, err := s.users.User(ctx, user.ID)
	if err != nil {
		return dbError(err)
	}
	if stored.SessionVersion != user.SessionVersion {
		return apperrors.Wrap(apperrors.CodeUnauthorized, nil)
	}
	match, err := security.VerifyPassword(password, stored.PasswordHash)
	if err != nil {
		return apperrors.Wrap(apperrors.CodeServiceUnavailable, err)
	}
	if !match {
		if err := protection.Failure(ctx, key, security.AuthenticationFailureLimit); err != nil {
			return dbError(err)
		}
		return apperrors.Wrap(apperrors.CodeInvalidCurrentPassword, nil)
	}
	return nil
}
func (s *UserMFAService) allowFactor(ctx context.Context, id uint64) error {
	key := security.AuthenticationKey("admin-mfa", s.encryptedID(id))
	ok, err := repositories.NewAuthenticationAttempts(s.users.DB()).Reserve(ctx, key, security.AuthenticationFailureLimit)
	if err != nil {
		return dbError(err)
	}
	if !ok {
		return apperrors.Wrap(apperrors.CodeRateLimited, nil)
	}
	return nil
}
