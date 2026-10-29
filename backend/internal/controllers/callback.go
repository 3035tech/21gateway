package controllers

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"github.com/21mobile/sms-gateway/internal/dtos"
	"github.com/21mobile/sms-gateway/internal/security"
	"github.com/21mobile/sms-gateway/internal/services"
	"github.com/danielgtaylor/huma/v2"
	"net/http"
)

type CallbackController struct {
	*SessionGuard
	service *services.CallbackService
}

func NewCallbackController(service *services.CallbackService, guard *SessionGuard) *CallbackController {
	return &CallbackController{SessionGuard: guard, service: service}
}
func (c *CallbackController) Register(api huma.API) {
	huma.Register(api, c.operation(api, "listReceivedCallbacks", http.MethodGet, "/callbacks/received", "Consultar callbacks recebidos", security.ResourceCallbacks, security.ActionRead), func(ctx context.Context, input *dtos.ListQuery) (*dtos.ReceivedCallbacksOutput, error) {
		values, err := c.service.Received(ctx, *input)
		if err != nil {
			return nil, HTTPError(err)
		}
		return &dtos.ReceivedCallbacksOutput{Body: values}, nil
	})
	huma.Register(api, c.operation(api, "listCallbacks", http.MethodGet, "/callbacks", "Consultar callbacks", security.ResourceCallbacks, security.ActionRead), func(ctx context.Context, input *dtos.CallbackListInput) (*dtos.CallbacksOutput, error) {
		values, err := c.service.Callbacks(ctx, *input)
		if err != nil {
			return nil, HTTPError(err)
		}
		return &dtos.CallbacksOutput{Body: values}, nil
	})
	operation := c.operation(api, "retryCallback", http.MethodPost, "/callbacks/{id}/retry", "Reprocessar callback falho", security.ResourceCallbacks, security.ActionWrite)
	operation.DefaultStatus = http.StatusNoContent
	huma.Register(api, operation, func(ctx context.Context, input *dtos.IDInput) (*struct{}, error) {
		if err := c.service.Retry(ctx, actor(ctx), input.ID); err != nil {
			return nil, HTTPError(err)
		}
		return nil, nil
	})
}

// Normalized ingress is an internal contract. It is not a guessed supplier payload.
func (c *CallbackController) RegisterIngress(api huma.API, token string) {
	operation := huma.Operation{OperationID: "receiveNormalizedDLR", Method: http.MethodPost, Path: "/internal/dlr", Summary: "Receber evento de entrega normalizado", Description: "Contrato interno. Habilitar somente após homologar o adaptador do fornecedor.", DefaultStatus: http.StatusNoContent, Tags: []string{"Callbacks"}, Middlewares: huma.Middlewares{internalBearerAuth(api, token)}}
	huma.Register(api, operation, func(ctx context.Context, input *dtos.DeliveryInput) (*struct{}, error) {
		if err := c.service.Accept(ctx, *input); err != nil {
			return nil, HTTPError(err)
		}
		return nil, nil
	})
}

func internalBearerAuth(api huma.API, token string) func(huma.Context, func(huma.Context)) {
	expected := sha256.Sum256([]byte("Bearer " + token))
	return func(ctx huma.Context, next func(huma.Context)) {
		actual := sha256.Sum256([]byte(ctx.Header("Authorization")))
		if len(token) < 32 || subtle.ConstantTimeCompare(expected[:], actual[:]) != 1 {
			_ = huma.WriteErr(api, ctx, http.StatusUnauthorized, "")
			return
		}
		next(ctx)
	}
}
