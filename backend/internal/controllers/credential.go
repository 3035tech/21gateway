package controllers

import (
	"context"
	"github.com/21mobile/sms-gateway/internal/dtos"
	"github.com/21mobile/sms-gateway/internal/security"
	"github.com/21mobile/sms-gateway/internal/services"
	"github.com/danielgtaylor/huma/v2"
	"net/http"
)

type CredentialController struct {
	*SessionGuard
	service *services.CredentialService
}

func NewCredentialController(service *services.CredentialService, guard *SessionGuard) *CredentialController {
	return &CredentialController{SessionGuard: guard, service: service}
}
func (c *CredentialController) Register(api huma.API) {
	huma.Register(api, c.operation(api, "listCredentials", http.MethodGet, "/credentials", "Listar credenciais", security.ResourceCredentials, security.ActionRead), func(ctx context.Context, input *dtos.ListQuery) (*dtos.CredentialsOutput, error) {
		values, err := c.service.Credentials(ctx, *input)
		if err != nil {
			return nil, HTTPError(err)
		}
		return &dtos.CredentialsOutput{Body: values}, nil
	})
	huma.Register(api, c.operation(api, "createCredential", http.MethodPost, "/credentials", "Criar credenciais", security.ResourceCredentials, security.ActionWrite), func(ctx context.Context, input *dtos.CreateCredentialInput) (*dtos.CredentialSecretOutput, error) {
		value, err := c.service.SaveCredential(ctx, actor(ctx), 0, input.Body)
		if err != nil {
			return nil, HTTPError(err)
		}
		return &dtos.CredentialSecretOutput{Body: value}, nil
	})
	huma.Register(api, c.operation(api, "updateCredential", http.MethodPut, "/credentials/{id}", "Atualizar credenciais", security.ResourceCredentials, security.ActionWrite), func(ctx context.Context, input *dtos.UpdateCredentialInput) (*dtos.CredentialSecretOutput, error) {
		value, err := c.service.SaveCredential(ctx, actor(ctx), input.ID, input.Body)
		if err != nil {
			return nil, HTTPError(err)
		}
		return &dtos.CredentialSecretOutput{Body: value}, nil
	})
	archiveCredential := c.operation(api, "archiveCredential", http.MethodDelete, "/credentials/{id}", "Arquivar credenciais", security.ResourceCredentials, security.ActionWrite)
	archiveCredential.DefaultStatus = http.StatusNoContent
	huma.Register(api, archiveCredential, func(ctx context.Context, input *dtos.IDInput) (*struct{}, error) {
		if err := c.service.Archive(ctx, actor(ctx), input.ID); err != nil {
			return nil, HTTPError(err)
		}
		return nil, nil
	})
	huma.Register(api, c.operation(api, "rotateCredential", http.MethodPost, "/credentials/{id}/rotate", "Trocar senha da API", security.ResourceCredentials, security.ActionWrite), func(ctx context.Context, input *dtos.IDInput) (*dtos.CredentialSecretOutput, error) {
		value, err := c.service.RotateCredential(ctx, actor(ctx), input.ID)
		if err != nil {
			return nil, HTTPError(err)
		}
		return &dtos.CredentialSecretOutput{Body: value}, nil
	})

}
