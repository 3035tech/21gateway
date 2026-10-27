package controllers

import (
	"context"
	"github.com/21mobile/sms-gateway/internal/dtos"
	"github.com/21mobile/sms-gateway/internal/services"
	"github.com/danielgtaylor/huma/v2"
	"net/http"
)

type MOController struct{ service *services.MOService }

func NewMOController(service *services.MOService) *MOController {
	return &MOController{service: service}
}
func (c *MOController) RegisterIngress(api huma.API, token string) {
	huma.Register(api, huma.Operation{OperationID: "receiveNormalizedMO", Method: http.MethodPost, Path: "/internal/mo", Summary: "Receber resposta MO normalizada", Description: "Contrato interno opt-in; não representa o payload do fornecedor. Exige conta, UUID de envio e identificador estável do evento.", DefaultStatus: http.StatusNoContent, Tags: []string{"Callbacks"}, Middlewares: huma.Middlewares{internalBearerAuth(api, token)}}, func(ctx context.Context, input *dtos.MOInput) (*struct{}, error) {
		if err := c.service.Accept(ctx, *input); err != nil {
			return nil, HTTPError(err)
		}
		return nil, nil
	})
}
