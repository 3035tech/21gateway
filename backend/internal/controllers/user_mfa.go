package controllers

import (
	"context"
	"net/http"

	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/security"
	"github.com/21mobile/sms-gateway/internal/services"
	"github.com/danielgtaylor/huma/v2"
)

type UserMFAController struct {
	*SessionGuard
	service *services.UserMFAService
}

func NewUserMFAController(service *services.UserMFAService, guard *SessionGuard) *UserMFAController {
	return &UserMFAController{SessionGuard: guard, service: service}
}
func (c *UserMFAController) Register(api huma.API) {
	huma.Register(api, c.operation(api, "mfaStatus", http.MethodGet, "/auth/mfa", "Estado do segundo fator", "", security.ActionRead), func(ctx context.Context, _ *struct{}) (*dtos.MFAStatusOutput, error) {
		active, err := c.service.Status(ctx, actor(ctx))
		if err != nil {
			return nil, HTTPError(err)
		}
		out := &dtos.MFAStatusOutput{}
		out.Body.Enabled = active
		return out, nil
	})
	huma.Register(api, c.operation(api, "mfaBegin", http.MethodPost, "/auth/mfa/setup", "Iniciar segundo fator", "", security.ActionRead), func(ctx context.Context, input *dtos.MFABeginInput) (*dtos.MFABeginOutput, error) {
		user := ctx.Value(userContextKey{}).(models.SystemUser)
		secret, uri, err := c.service.Begin(ctx, user, input.Body.CurrentPassword)
		if err != nil {
			return nil, HTTPError(err)
		}
		out := &dtos.MFABeginOutput{}
		out.Body.Secret = secret
		out.Body.URI = uri
		return out, nil
	})
	huma.Register(api, c.operation(api, "mfaConfirm", http.MethodPost, "/auth/mfa/confirm", "Confirmar segundo fator", "", security.ActionRead), func(ctx context.Context, input *dtos.MFAConfirmInput) (*dtos.MFAConfirmOutput, error) {
		if err := c.sessions.RenewToken(ctx); err != nil {
			return nil, HTTPError(apperrors.Wrap(apperrors.CodeServiceUnavailable, err))
		}
		codes, err := c.service.Confirm(ctx, actor(ctx), input.Body.Code)
		if err != nil {
			return nil, HTTPError(err)
		}
		out := &dtos.MFAConfirmOutput{}
		out.Body.RecoveryCodes = codes
		user := ctx.Value(userContextKey{}).(models.SystemUser)
		c.sessions.Put(ctx, sessionVersion, user.SessionVersion+1)
		return out, nil
	})
	disable := c.operation(api, "mfaDisable", http.MethodPost, "/auth/mfa/disable", "Desativar segundo fator", "", security.ActionRead)
	disable.DefaultStatus = http.StatusNoContent
	huma.Register(api, disable, func(ctx context.Context, input *dtos.MFADisableInput) (*struct{}, error) {
		user := ctx.Value(userContextKey{}).(models.SystemUser)
		if err := c.sessions.RenewToken(ctx); err != nil {
			return nil, HTTPError(apperrors.Wrap(apperrors.CodeServiceUnavailable, err))
		}
		if err := c.service.Disable(ctx, user, input.Body.CurrentPassword, input.Body.Code); err != nil {
			return nil, HTTPError(err)
		}
		c.sessions.Put(ctx, sessionVersion, user.SessionVersion+1)
		return nil, nil
	})
}
