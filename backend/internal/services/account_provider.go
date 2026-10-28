package services

import (
	"context"
	"fmt"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/integrations/whitelabel"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/security"
)

// Each request receives its own token view of a shared, pooled HTTP client.
// Tokens never enter request context, response DTOs or a global mutable header.
type AccountSMSProvider struct {
	client *whitelabel.Client
	vault  *security.TokenVault
}

func NewAccountSMSProvider(client *whitelabel.Client, vault *security.TokenVault) *AccountSMSProvider {
	return &AccountSMSProvider{client: client, vault: vault}
}
func (p *AccountSMSProvider) Enabled() bool { return p.vault.Enabled() }
func (p *AccountSMSProvider) PrepareCompany(company models.Company) (SMSProvider, error) {
	if !p.Enabled() || company.ProviderTokenEncrypted == "" {
		return nil, apperrors.Wrap(apperrors.CodeTestNotConfigured, nil)
	}
	token, err := p.vault.Decrypt(fmt.Sprintf("company:%d", company.ID), company.ProviderTokenEncrypted)
	if err != nil {
		return nil, err
	}
	return p.client.ForToken(token), nil
}
func (p *AccountSMSProvider) Prepare(ctx context.Context) (SMSProvider, error) {
	credential, ok := CredentialFromContext(ctx)
	if !ok {
		return nil, apperrors.Wrap(apperrors.CodeUnauthorized, nil)
	}
	associatedData := credential.Username
	encrypted := credential.ProviderTokenEncrypted
	if credential.CompanyProviderTokenEncrypted != "" && credential.CompanyID != nil {
		associatedData = fmt.Sprintf("company:%d", *credential.CompanyID)
		encrypted = credential.CompanyProviderTokenEncrypted
	} else if credential.CompanyID != nil && !credential.CompanyProviderTokenLegacyFallback {
		encrypted = ""
	}
	token, err := p.vault.Decrypt(associatedData, encrypted)
	if err != nil {
		return nil, err
	}
	return p.client.ForToken(token), nil
}
func (p *AccountSMSProvider) Send(ctx context.Context, input whitelabel.SendRequest) (whitelabel.SendResponse, error) {
	provider, err := p.Prepare(ctx)
	if err != nil {
		return whitelabel.SendResponse{}, err
	}
	return provider.Send(ctx, input)
}
