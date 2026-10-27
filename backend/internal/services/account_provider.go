package services

import (
	"context"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/integrations/whitelabel"
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
func (p *AccountSMSProvider) Prepare(ctx context.Context) (SMSProvider, error) {
	credential, ok := CredentialFromContext(ctx)
	if !ok {
		return nil, apperrors.Wrap(apperrors.CodeUnauthorized, nil)
	}
	token, err := p.vault.Decrypt(credential.Username, credential.ProviderTokenEncrypted)
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
