package controllers

import (
	"context"
	"github.com/21mobile/sms-gateway/internal/dtos"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/security"
	"github.com/21mobile/sms-gateway/internal/services"
	"github.com/danielgtaylor/huma/v2"
	"net/http"
)

type UserController struct {
	*SessionGuard
	service *services.UserService
}

func NewUserController(service *services.UserService, guard *SessionGuard) *UserController {
	return &UserController{SessionGuard: guard, service: service}
}
func (c *UserController) Register(api huma.API) {
	huma.Register(api, c.operation(api, "getUserByPublicRef", http.MethodGet, "/users/by-ref/{publicRef}", "Consultar usuário pelo endereço público", security.ResourceUsers, security.ActionRead), func(ctx context.Context, input *dtos.PublicRefInput) (*dtos.UserOutput, error) {
		value, err := c.service.UserByPublicRef(ctx, input.PublicRef)
		if err != nil {
			return nil, HTTPError(err)
		}
		return &dtos.UserOutput{Body: value}, nil
	})
	huma.Register(api, c.operation(api, "getUser", http.MethodGet, "/users/{id}", "Consultar usuário", security.ResourceUsers, security.ActionRead), func(ctx context.Context, input *dtos.IDInput) (*dtos.UserOutput, error) {
		value, err := c.service.User(ctx, input.ID)
		if err != nil {
			return nil, HTTPError(err)
		}
		return &dtos.UserOutput{Body: value}, nil
	})
	changePassword := c.operation(api, "changeOwnPassword", http.MethodPost, "/auth/password", "Trocar a própria senha", "", security.ActionRead)
	changePassword.DefaultStatus = http.StatusNoContent
	huma.Register(api, changePassword, func(ctx context.Context, input *dtos.ChangeOwnPasswordInput) (*struct{}, error) {
		user := ctx.Value(userContextKey{}).(models.SystemUser)
		if err := c.service.ChangeOwnPassword(ctx, user, input.Body.CurrentPassword, input.Body.NewPassword); err != nil {
			return nil, HTTPError(err)
		}
		return nil, nil
	})
	huma.Register(api, c.operation(api, "listUsers", http.MethodGet, "/users", "Listar usuários", security.ResourceUsers, security.ActionRead), func(ctx context.Context, input *dtos.ListQuery) (*dtos.UsersOutput, error) {
		values, err := c.service.Users(ctx, *input)
		if err != nil {
			return nil, HTTPError(err)
		}
		return &dtos.UsersOutput{Body: values}, nil
	})
	huma.Register(api, c.operation(api, "createUser", http.MethodPost, "/users", "Criar usuários", security.ResourceUsers, security.ActionWrite), func(ctx context.Context, input *dtos.CreateUserInput) (*dtos.UserOutput, error) {
		value, err := c.service.SaveUser(ctx, actor(ctx), 0, input.Body)
		if err != nil {
			return nil, HTTPError(err)
		}
		return &dtos.UserOutput{Body: value}, nil
	})
	huma.Register(api, c.operation(api, "updateUser", http.MethodPut, "/users/{id}", "Atualizar usuários", security.ResourceUsers, security.ActionWrite), func(ctx context.Context, input *dtos.UpdateUserInput) (*dtos.UserOutput, error) {
		value, err := c.service.SaveUser(ctx, actor(ctx), input.ID, input.Body)
		if err != nil {
			return nil, HTTPError(err)
		}
		return &dtos.UserOutput{Body: value}, nil
	})
	archiveUser := c.operation(api, "archiveUser", http.MethodDelete, "/users/{id}", "Arquivar usuários", security.ResourceUsers, security.ActionWrite)
	archiveUser.DefaultStatus = http.StatusNoContent
	huma.Register(api, archiveUser, func(ctx context.Context, input *dtos.IDInput) (*struct{}, error) {
		if err := c.service.Archive(ctx, actor(ctx), input.ID); err != nil {
			return nil, HTTPError(err)
		}
		return nil, nil
	})

}
