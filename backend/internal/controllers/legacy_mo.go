package controllers

import (
	"context"
	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/services"
	"github.com/danielgtaylor/huma/v2"
	"net/http"
)

const LegacyMOPath = "/v1/consultmo"

type LegacyMOController struct {
	service *services.LegacyMOService
	guard   *LegacyGuard
}

func NewLegacyMOController(service *services.LegacyMOService, auth *services.AuthService) *LegacyMOController {
	return &LegacyMOController{service: service, guard: NewLegacyGuard(auth)}
}
func (c *LegacyMOController) Register(api huma.API) {
	c.guard.RegisterSecurity(api)
	huma.Register(api, huma.Operation{OperationID: "queryLegacyMO", Method: http.MethodGet, Path: LegacyMOPath, Summary: "Consultar respostas recebidas em um dia", Description: "date é Unix em segundos e identifica o dia em America/Sao_Paulo. Consulta não consome eventos ou callbacks. Datas/erros ainda sujeitos à homologação do legado.", Tags: []string{"SMS"}, Security: []map[string][]string{{basicAuthScheme: {}}}, Middlewares: huma.Middlewares{c.guard.Authenticate(api)}, Errors: []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusUnprocessableEntity, http.StatusTooManyRequests, http.StatusServiceUnavailable}}, c.Query)
}
func (c *LegacyMOController) Query(ctx context.Context, input *dtos.LegacyMOQueryInput) (*dtos.LegacyMOOutput, error) {
	credential, ok := services.CredentialFromContext(ctx)
	if !ok {
		return nil, HTTPError(apperrors.Wrap(apperrors.CodeUnauthorized, nil))
	}
	body, err := c.service.Query(ctx, credential.ID, input.Date)
	if err != nil {
		return nil, HTTPError(err)
	}
	return &dtos.LegacyMOOutput{Body: body}, nil
}
