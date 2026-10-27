package controllers

import (
	"context"
	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/services"
	"github.com/danielgtaylor/huma/v2"
	"net/http"
)

const LegacyStatusPath = "/v1/status"

type LegacyStatusController struct {
	service *services.LegacyStatusService
	guard   *LegacyGuard
}

func NewLegacyStatusController(service *services.LegacyStatusService, auth *services.AuthService) *LegacyStatusController {
	return &LegacyStatusController{service: service, guard: NewLegacyGuard(auth)}
}
func (c *LegacyStatusController) Register(api huma.API) {
	c.guard.RegisterSecurity(api)
	huma.Register(api, huma.Operation{OperationID: "consumeLegacySMSStatuses", Method: http.MethodGet, Path: LegacyStatusPath, Summary: "Consulta até 100 estados ainda não consultados", Description: "Somente mensagens com correlationId e estado de entrega recebido. Consulta consome o estado antes da resposta; não confirma compatibilidade integral do legado.", Tags: []string{"SMS"}, Security: []map[string][]string{{basicAuthScheme: {}}}, Middlewares: huma.Middlewares{c.guard.Authenticate(api)}, Errors: []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusTooManyRequests, http.StatusServiceUnavailable}}, c.Consume)
}
func (c *LegacyStatusController) Consume(ctx context.Context, _ *struct{}) (*dtos.LegacyStatusOutput, error) {
	credential, ok := services.CredentialFromContext(ctx)
	if !ok {
		return nil, HTTPError(apperrors.Wrap(apperrors.CodeUnauthorized, nil))
	}
	body, err := c.service.Consume(ctx, credential.ID)
	if err != nil {
		return nil, HTTPError(err)
	}
	return &dtos.LegacyStatusOutput{Body: body}, nil
}
