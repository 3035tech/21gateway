package whitelabel

import (
	"context"
	"fmt"

	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/models"
	smsprovider "github.com/21mobile/sms-gateway/internal/providers/sms"
	"github.com/21mobile/sms-gateway/internal/security"
)

// Adapter owns Zoug authentication and translates the neutral SMS contract.
type Adapter struct {
	client *Client
	vault  *security.TokenVault
}

func NewAdapter(client *Client, vault *security.TokenVault) *Adapter {
	return &Adapter{client: client, vault: vault}
}
func (*Adapter) Code() models.Provider { return models.ProviderZoug }

func (a *Adapter) PrepareCompany(_ context.Context, company models.Company) (smsprovider.Sender, error) {
	if a.client == nil || a.vault == nil || !a.vault.Enabled() || company.ProviderTokenEncrypted == "" {
		return nil, apperrors.Wrap(apperrors.CodeTestNotConfigured, nil)
	}
	token, err := a.vault.Decrypt(fmt.Sprintf("company:%d", company.ID), company.ProviderTokenEncrypted)
	if err != nil {
		return nil, err
	}
	return &sender{client: a.client.ForToken(token)}, nil
}

func (a *Adapter) PrepareCredential(_ context.Context, credential models.APICredential) (smsprovider.Sender, error) {
	if a.client == nil || a.vault == nil || !a.vault.Enabled() {
		return nil, apperrors.Wrap(apperrors.CodeServiceUnavailable, nil)
	}
	associatedData := credential.Username
	encrypted := credential.ProviderTokenEncrypted
	if credential.CompanyProviderTokenEncrypted != "" && credential.CompanyID != nil {
		associatedData = fmt.Sprintf("company:%d", *credential.CompanyID)
		encrypted = credential.CompanyProviderTokenEncrypted
	} else if credential.CompanyID != nil && !credential.CompanyProviderTokenLegacyFallback {
		encrypted = ""
	}
	token, err := a.vault.Decrypt(associatedData, encrypted)
	if err != nil {
		return nil, err
	}
	return &sender{client: a.client.ForToken(token)}, nil
}

type sender struct{ client *Client }

func (s *sender) Enabled() bool { return s != nil && s.client != nil && s.client.Enabled() }
func (s *sender) Send(ctx context.Context, input smsprovider.SendRequest) (smsprovider.SendResponse, error) {
	if input.Type != models.ChannelSMS {
		return smsprovider.SendResponse{}, apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	result, err := s.client.Send(ctx, SendRequest{
		Type: MessageTypeSMS, Message: input.Message, Encoding: input.Encoding,
		Destination: input.Destination, ReferenceID: input.ReferenceID,
		Index: input.Index, CallbackLevel: CallbackLevel(input.CallbackLevel),
	})
	return smsprovider.SendResponse{MessageID: result.MessageID, Status: string(result.Status)}, err
}
func (s *sender) SendBatch(ctx context.Context, input smsprovider.BatchRequest) ([]smsprovider.BatchResult, error) {
	if input.Type != models.ChannelSMS {
		return nil, apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	messages := make([]BatchMessage, len(input.Messages))
	for i, message := range input.Messages {
		messages[i] = BatchMessage{Message: message.Message, Destination: message.Destination,
			ReferenceID: message.ReferenceID, Index: message.Index}
	}
	result, err := s.client.SendBatch(ctx, BatchRequest{Type: MessageTypeSMS, Encoding: input.Encoding, Messages: messages})
	if err != nil {
		return nil, err
	}
	out := make([]smsprovider.BatchResult, len(result))
	for i, item := range result {
		out[i] = smsprovider.BatchResult{MessageID: item.MessageID, ErrorCode: item.ErrorCode}
	}
	return out, nil
}
func (s *sender) Diagnose(ctx context.Context, runID string, companyID, actorID uint64, destination string) []DiagnosticResult {
	return s.client.Diagnose(ctx, runID, companyID, actorID, destination)
}
