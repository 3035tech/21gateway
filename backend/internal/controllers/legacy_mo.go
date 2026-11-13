package controllers

import (
	"context"
	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/services"
	"github.com/danielgtaylor/huma/v2"
	"net/http"
)

const V1MOPath = "/v1/consultmo"

type V1MOController struct {
	service *services.V1MOService
	guard   *V1Guard
}

func NewV1MOController(service *services.V1MOService, auth *services.AuthService) *V1MOController {
	return &V1MOController{service: service, guard: NewV1Guard(auth)}
}
func (c *V1MOController) Register(api huma.API) {
	c.guard.RegisterSecurity(api)
	huma.Register(api, huma.Operation{OperationID: "getMOReplies", Method: http.MethodGet, Path: V1MOPath, Summary: "Consultar respostas recebidas em um dia", Description: "date é Unix em segundos e identifica o dia em America/Sao_Paulo. A consulta não consome respostas ou callbacks. Validar datas e erros com a equipe 21 Mobile durante a homologação.", Tags: []string{"SMS"}, Security: []map[string][]string{{basicAuthScheme: {}}}, Middlewares: huma.Middlewares{c.guard.Authenticate(api)}, Errors: []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusUnprocessableEntity, http.StatusTooManyRequests, http.StatusServiceUnavailable}}, c.Query)
}
func (c *V1MOController) Query(ctx context.Context, input *dtos.LegacyMOQueryInput) (*dtos.LegacyMOOutput, error) {
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
