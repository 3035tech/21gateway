package services

import (
	"context"

	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/integrations/whitelabel"
	"github.com/21mobile/sms-gateway/internal/models"
)

// DiagnoseCompany is an explicitly Zoug-only operational diagnostic. It does
// not form part of the SMS provider contract.
func (s *SMSService) DiagnoseCompany(ctx context.Context, company models.Company, runID string, actorID uint64, destination string) ([]whitelabel.DiagnosticResult, error) {
	preparer, ok := s.provider.(interface {
		PrepareCompany(context.Context, models.Company) (SMSProvider, error)
	})
	if !ok {
		return nil, apperrors.Wrap(apperrors.CodeTestNotConfigured, nil)
	}
	provider, err := preparer.PrepareCompany(ctx, company)
	if err != nil {
		return nil, err
	}
	diagnostic, ok := provider.(interface {
		Diagnose(context.Context, string, uint64, uint64, string) []whitelabel.DiagnosticResult
	})
	if !ok {
		return nil, apperrors.Wrap(apperrors.CodeTestNotConfigured, nil)
	}
	return diagnostic.Diagnose(ctx, runID, company.ID, actorID, destination), nil
}
