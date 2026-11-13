package controllers

import (
	"context"
	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/services"
	"github.com/danielgtaylor/huma/v2"
	"net/http"
)

const V1StatusPath = "/v1/status"
const V1StatusByDatePath = "/v1/status/date"
const V1StatusByDateSlashPath = V1StatusByDatePath + "/"

type V1StatusController struct {
	service *services.V1StatusService
	guard   *V1Guard
}

func NewV1StatusController(service *services.V1StatusService, auth *services.AuthService) *V1StatusController {
	return &V1StatusController{service: service, guard: NewV1Guard(auth)}
}
func (c *V1StatusController) Register(api huma.API) {
	c.guard.RegisterSecurity(api)
	huma.Register(api, huma.Operation{OperationID: "getSMSDeliveryStatuses", Method: http.MethodGet, Path: V1StatusPath, Summary: "Consultar estados de entrega", Description: "Retorna até 100 estados de entrega ainda não consultados para mensagens com correlationId. A leitura consome os registros retornados; repetições da consulta não os devolvem novamente.", Tags: []string{"SMS"}, Security: []map[string][]string{{basicAuthScheme: {}}}, Middlewares: huma.Middlewares{c.guard.Authenticate(api)}, Errors: []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusTooManyRequests, http.StatusServiceUnavailable}}, c.Consume)
	operation := huma.Operation{OperationID: "getSMSDeliveryStatusesByDate", Method: http.MethodGet, Path: V1StatusByDatePath, Summary: "Consultar estados de entrega por data", Description: "Retorna até 100 registros do dia informado para a credencial autenticada. A consulta não consome registros; repetições devolvem os mesmos dados. Paginação ainda depende de confirmação do contrato legado; se houver mais de 100 resultados, retorna indisponibilidade em vez de uma lista parcial. O parâmetro destination deve incluir o prefixo 55.", Tags: []string{"SMS"}, Security: []map[string][]string{{basicAuthScheme: {}}}, Middlewares: huma.Middlewares{c.guard.Authenticate(api)}, Errors: []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusUnprocessableEntity, http.StatusTooManyRequests, http.StatusServiceUnavailable}}
	huma.Register(api, operation, c.ByDate)
	operation.OperationID = "getSMSDeliveryStatusesByDateTrailingSlash"
	operation.Path = V1StatusByDateSlashPath
	huma.Register(api, operation, c.ByDate)
}
func (c *V1StatusController) ByDate(ctx context.Context, input *dtos.LegacyStatusByDateInput) (*dtos.LegacyStatusOutput, error) {
	credential, ok := services.CredentialFromContext(ctx)
	if !ok {
		return nil, HTTPError(apperrors.Wrap(apperrors.CodeUnauthorized, nil))
	}
	body, err := c.service.ByDate(ctx, credential.ID, input.Date, input.CorrelationID, input.Destination)
	if err != nil {
		return nil, HTTPError(err)
	}
	return &dtos.LegacyStatusOutput{Body: body}, nil
}

func (c *V1StatusController) Consume(ctx context.Context, _ *struct{}) (*dtos.LegacyStatusOutput, error) {
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
