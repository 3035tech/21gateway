package controllers

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/services"
)

const SubmissionStatusPath = "/v1/send/result"

type SubmissionController struct {
	service *services.SubmissionService
	guard   *LegacyGuard
}

func NewSubmissionController(service *services.SubmissionService, auth *services.AuthService) *SubmissionController {
	return &SubmissionController{service: service, guard: NewLegacyGuard(auth)}
}

func (c *SubmissionController) Register(api huma.API) {
	c.guard.RegisterSecurity(api)
	huma.Register(api, huma.Operation{
		OperationID: "getSMSSubmission",
		Method:      http.MethodGet,
		Path:        SubmissionStatusPath,
		Summary:     "Consultar o resultado de submissão de um SMS",
		Description: "Consulta por id_sms o estado de uma tentativa da credencial autenticada. A leitura não consome registros. 404 também é devolvido para ID pertencente a outra credencial.",
		Tags:        []string{"SMS"},
		Security:    []map[string][]string{{basicAuthScheme: {}}},
		Middlewares: huma.Middlewares{c.guard.Authenticate(api)},
		Errors:      []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusTooManyRequests, http.StatusServiceUnavailable},
	}, c.Status)
}

func (c *SubmissionController) Status(ctx context.Context, input *dtos.SMSSubmissionQuery) (*dtos.SMSSubmissionOutput, error) {
	credential, ok := services.CredentialFromContext(ctx)
	if !ok {
		return nil, HTTPError(apperrors.Wrap(apperrors.CodeUnauthorized, nil))
	}
	status, err := c.service.Status(ctx, credential.ID, input.IDSMS)
	if err != nil {
		return nil, HTTPError(err)
	}
	return &dtos.SMSSubmissionOutput{Body: status}, nil
}
