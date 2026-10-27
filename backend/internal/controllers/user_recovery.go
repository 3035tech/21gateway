package controllers

import (
	"context"
	"net/http"

	"github.com/21mobile/sms-gateway/internal/dtos"
	"github.com/21mobile/sms-gateway/internal/security"
	"github.com/21mobile/sms-gateway/internal/services"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
)

type UserRecoveryController struct{ service *services.UserRecoveryService }

func NewUserRecoveryController(service *services.UserRecoveryService) *UserRecoveryController {
	return &UserRecoveryController{service: service}
}
func (c *UserRecoveryController) Register(api huma.API) {
	rateLimit := func(ctx huma.Context, next func(huma.Context)) {
		request, _ := humachi.Unwrap(ctx)
		if err := c.service.AllowIP(ctx.Context(), security.ClientIP(request)); err != nil {
			_ = huma.WriteErr(api, ctx, HTTPError(err).GetStatus(), "")
			return
		}
		next(ctx)
	}
	requestOp := huma.Operation{OperationID: "requestPasswordRecovery", Method: http.MethodPost, Path: "/admin/auth/recovery", DefaultStatus: http.StatusNoContent, Tags: []string{"Backoffice"}, Middlewares: huma.Middlewares{rateLimit}, Errors: []int{422, 429, 503}}
	huma.Register(api, requestOp, func(ctx context.Context, input *dtos.RecoveryRequestInput) (*struct{}, error) {
		if err := c.service.Request(ctx, input.Body.Email); err != nil {
			return nil, HTTPError(err)
		}
		return nil, nil
	})
	resetOp := huma.Operation{OperationID: "resetPassword", Method: http.MethodPost, Path: "/admin/auth/recovery/reset", DefaultStatus: http.StatusNoContent, Tags: []string{"Backoffice"}, Middlewares: huma.Middlewares{rateLimit}, Errors: []int{401, 422, 429, 503}}
	huma.Register(api, resetOp, func(ctx context.Context, input *dtos.RecoveryResetInput) (*struct{}, error) {
		if err := c.service.Reset(ctx, input.Body.Token, input.Body.Factor, input.Body.NewPassword); err != nil {
			return nil, HTTPError(err)
		}
		return nil, nil
	})
}
