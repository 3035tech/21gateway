package controllers

import (
	"context"
	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/security"
	"github.com/alexedwards/scs/v2"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"net/http"
)

type SessionLogin interface {
	Login(context.Context, string, string) (models.SystemUser, error)
	AllowLogin(context.Context, string) error
}
type SessionController struct {
	*SessionGuard
	service  SessionLogin
	sessions *scs.SessionManager
}

func NewSessionController(users SessionLogin, guard *SessionGuard) *SessionController {
	return &SessionController{SessionGuard: guard, service: users, sessions: guard.sessions}
}
func (c *SessionController) Register(api huma.API) {
	if api.OpenAPI().Components.SecuritySchemes == nil {
		api.OpenAPI().Components.SecuritySchemes = map[string]*huma.SecurityScheme{}
	}
	api.OpenAPI().Components.SecuritySchemes[adminSessionScheme] = &huma.SecurityScheme{Type: "apiKey", In: "cookie", Name: "21mobile_session"}
	huma.Register(api, huma.Operation{OperationID: "adminLogin", Method: http.MethodPost, Path: "/admin/auth/login", Summary: "Entrar no backoffice", Tags: []string{"Backoffice"}, Errors: []int{401, 403, 422, 429, 503}, Middlewares: huma.Middlewares{func(ctx huma.Context, next func(huma.Context)) {
		request, _ := humachi.Unwrap(ctx)
		key := security.AuthenticationKey("admin-ip", security.AuthenticationPeer(request.RemoteAddr))
		if err := c.service.AllowLogin(ctx.Context(), key); err != nil {
			_ = huma.WriteErr(api, ctx, HTTPError(err).GetStatus(), "")
			return
		}
		next(ctx)
	}}}, func(ctx context.Context, input *dtos.LoginInput) (*dtos.UserOutput, error) {
		user, err := c.service.Login(ctx, input.Body.Email, input.Body.Password)
		if err != nil {
			return nil, HTTPError(err)
		}
		if err := c.sessions.RenewToken(ctx); err != nil {
			return nil, HTTPError(apperrors.Wrap(apperrors.CodeServiceUnavailable, err))
		}
		c.sessions.Put(ctx, sessionUserID, int64(user.ID))
		c.sessions.Put(ctx, sessionVersion, user.SessionVersion)
		return &dtos.UserOutput{Body: dtos.UserDTO(user)}, nil
	})
	huma.Register(api, c.operation(api, "adminMe", http.MethodGet, "/auth/me", "Sessão atual", "", security.ActionRead), func(ctx context.Context, _ *struct{}) (*dtos.UserOutput, error) {
		return &dtos.UserOutput{Body: dtos.UserDTO(ctx.Value(userContextKey{}).(models.SystemUser))}, nil
	})
	// Logout remains available when the account was blocked or its session version changed.
	huma.Register(api, huma.Operation{OperationID: "adminLogout", Method: http.MethodPost, Path: "/admin/auth/logout", DefaultStatus: http.StatusNoContent, Tags: []string{"Backoffice"}}, func(ctx context.Context, _ *struct{}) (*struct{}, error) {
		if err := c.sessions.Destroy(ctx); err != nil {
			return nil, HTTPError(apperrors.Wrap(apperrors.CodeServiceUnavailable, err))
		}
		return nil, nil
	})

}
