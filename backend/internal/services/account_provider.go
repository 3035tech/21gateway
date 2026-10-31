package services

import (
	"context"

	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/models"
	smsprovider "github.com/21mobile/sms-gateway/internal/providers/sms"
)

// AccountSMSProvider resolves the company's SMS route. Each adapter owns its
// authentication and wire protocol; no provider credential enters the context.
type AccountSMSProvider struct {
	adapters map[models.Provider]smsprovider.Adapter
}

func NewAccountSMSProvider(adapters ...smsprovider.Adapter) (*AccountSMSProvider, error) {
	registry := &AccountSMSProvider{adapters: make(map[models.Provider]smsprovider.Adapter, len(adapters))}
	for _, adapter := range adapters {
		if adapter == nil || adapter.Code() == "" || registry.adapters[adapter.Code()] != nil {
			return nil, apperrors.Wrap(apperrors.CodeConfigurationInvalid, nil)
		}
		registry.adapters[adapter.Code()] = adapter
	}
	if len(registry.adapters) == 0 {
		return nil, apperrors.Wrap(apperrors.CodeConfigurationInvalid, nil)
	}
	return registry, nil
}

func (p *AccountSMSProvider) Enabled() bool { return p != nil && len(p.adapters) > 0 }

func selectedSMSProvider(code models.Provider) models.Provider {
	if code == "" {
		return models.ProviderZoug // Compatibility for pre-route credentials.
	}
	return code
}

func (p *AccountSMSProvider) adapter(code models.Provider) (smsprovider.Adapter, error) {
	if p == nil {
		return nil, apperrors.Wrap(apperrors.CodeServiceUnavailable, nil)
	}
	adapter := p.adapters[selectedSMSProvider(code)]
	if adapter == nil {
		return nil, apperrors.Wrap(apperrors.CodeServiceUnavailable, nil)
	}
	return adapter, nil
}

func (p *AccountSMSProvider) PrepareCompany(ctx context.Context, company models.Company) (SMSProvider, error) {
	adapter, err := p.adapter(company.SMSProvider)
	if err != nil {
		return nil, err
	}
	return adapter.PrepareCompany(ctx, company)
}

func (p *AccountSMSProvider) Prepare(ctx context.Context) (SMSProvider, error) {
	credential, ok := CredentialFromContext(ctx)
	if !ok {
		return nil, apperrors.Wrap(apperrors.CodeUnauthorized, nil)
	}
	adapter, err := p.adapter(credential.CompanySMSProvider)
	if err != nil {
		return nil, err
	}
	return adapter.PrepareCredential(ctx, credential)
}

func (p *AccountSMSProvider) Send(ctx context.Context, input smsprovider.SendRequest) (smsprovider.SendResponse, error) {
	provider, err := p.Prepare(ctx)
	if err != nil {
		return smsprovider.SendResponse{}, err
	}
	return provider.Send(ctx, input)
}
