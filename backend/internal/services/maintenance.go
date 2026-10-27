package services

import (
	"context"
	"github.com/21mobile/sms-gateway/internal/config"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/security"
	"time"
)

const MaintenanceBatchSize = 100
const MaximumRetentionDays = config.MaximumMORetentionDays

type MaintenanceRepository interface {
	CredentialCiphertexts(context.Context, uint64, int) ([]models.CredentialCiphertext, error)
	MOCiphertexts(context.Context, uint64, int) ([]models.MOCiphertext, error)
	ReplaceCredentialCiphertext(context.Context, uint64, string, string) (bool, error)
	ReplaceMOCiphertext(context.Context, uint64, string, string) (bool, error)
	PurgeMO(context.Context, time.Time, int, bool) (int64, error)
}
type MaintenanceService struct {
	repo   MaintenanceRepository
	tokens *security.TokenVault
	mo     *security.MOContentVault
}

func NewMaintenanceService(repo MaintenanceRepository, tokens *security.TokenVault, mo *security.MOContentVault) *MaintenanceService {
	return &MaintenanceService{repo: repo, tokens: tokens, mo: mo}
}
func (s *MaintenanceService) Rotate(ctx context.Context, apply bool) (models.RotationResult, error) {
	result := models.RotationResult{}
	if !s.tokens.Enabled() && !s.mo.Enabled() {
		return result, apperrors.Wrap(apperrors.CodeConfigurationInvalid, nil)
	}
	if s.tokens.Enabled() {
		var after uint64
		for {
			rows, err := s.repo.CredentialCiphertexts(ctx, after, MaintenanceBatchSize)
			if err != nil {
				return result, dbError(err)
			}
			if len(rows) == 0 {
				break
			}
			for _, row := range rows {
				after = row.ID
				result.Scanned++
				plain, err := s.tokens.Decrypt(row.Username, row.ProviderTokenEncrypted)
				if err != nil {
					return result, err
				}
				if s.tokens.Current(row.ProviderTokenEncrypted) {
					continue
				}
				result.Eligible++
				if !apply {
					continue
				}
				next, err := s.tokens.Encrypt(row.Username, plain)
				if err != nil {
					return result, err
				}
				changed, err := s.repo.ReplaceCredentialCiphertext(ctx, row.ID, row.ProviderTokenEncrypted, next)
				if err != nil {
					return result, dbError(err)
				}
				if changed {
					result.Changed++
				} else {
					result.Conflicts++
				}
			}
		}
	}
	if s.mo.Enabled() {
		var after uint64
		for {
			rows, err := s.repo.MOCiphertexts(ctx, after, MaintenanceBatchSize)
			if err != nil {
				return result, dbError(err)
			}
			if len(rows) == 0 {
				break
			}
			for _, row := range rows {
				after = row.ID
				result.Scanned++
				plain, err := s.mo.Decrypt(row.CredentialID, row.EventKey, row.EncryptedMessage)
				if err != nil {
					return result, err
				}
				if s.mo.Current(row.EncryptedMessage) {
					continue
				}
				result.Eligible++
				if !apply {
					continue
				}
				next, err := s.mo.Encrypt(row.CredentialID, row.EventKey, plain)
				if err != nil {
					return result, err
				}
				changed, err := s.repo.ReplaceMOCiphertext(ctx, row.ID, row.EncryptedMessage, next)
				if err != nil {
					return result, dbError(err)
				}
				if changed {
					result.Changed++
				} else {
					result.Conflicts++
				}
			}
		}
	}
	return result, nil
}
func (s *MaintenanceService) PurgeMO(ctx context.Context, days int, apply bool) (int64, error) {
	if days == 0 {
		return 0, nil
	}
	if days < 0 || days > MaximumRetentionDays {
		return 0, apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	count, err := s.repo.PurgeMO(ctx, time.Now().UTC().Add(-time.Duration(days)*24*time.Hour), MaintenanceBatchSize, apply)
	return count, dbError(err)
}
