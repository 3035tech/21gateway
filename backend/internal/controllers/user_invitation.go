package controllers

import (
	"context"
	"net/http"

	"github.com/21mobile/sms-gateway/internal/dtos"
	"github.com/21mobile/sms-gateway/internal/security"
	"github.com/21mobile/sms-gateway/internal/services"
	"github.com/danielgtaylor/huma/v2"
)

type UserInvitationController struct {
	*SessionGuard
	service *services.UserRecoveryService
}

func NewUserInvitationController(service *services.UserRecoveryService, guard *SessionGuard) *UserInvitationController {
	return &UserInvitationController{SessionGuard: guard, service: service}
}
func (c *UserInvitationController) Register(api huma.API) {
	huma.Register(api, c.operation(api, "inviteUser", http.MethodPost, "/users/invite", "Convidar usuário interno", security.ResourceUsers, security.ActionWrite), func(ctx context.Context, input *dtos.InviteUserInput) (*dtos.UserOutput, error) {
		user, err := c.service.Invite(ctx, actor(ctx), dtos.UserBody{Name: input.Body.Name, Email: input.Body.Email, Role: input.Body.Role, Active: input.Body.Active})
		if err != nil {
			return nil, HTTPError(err)
		}
		return &dtos.UserOutput{Body: user}, nil
	})
	resend := c.operation(api, "resendUserInvitation", http.MethodPost, "/users/{id}/invite", "Reenviar convite", security.ResourceUsers, security.ActionWrite)
	resend.DefaultStatus = http.StatusNoContent
	huma.Register(api, resend, func(ctx context.Context, input *dtos.IDInput) (*struct{}, error) {
		if err := c.service.ResendInvitation(ctx, input.ID); err != nil {
			return nil, HTTPError(err)
		}
		return nil, nil
	})
}
