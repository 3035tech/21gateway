package controllers

import (
	"context"
	"github.com/21mobile/sms-gateway/internal/dtos"
	"github.com/21mobile/sms-gateway/internal/security"
	"github.com/21mobile/sms-gateway/internal/services"
	"github.com/danielgtaylor/huma/v2"
	"net/http"
)

type OperationsController struct{ service *services.OperationsService }

func NewOperationsController(service *services.OperationsService) *OperationsController {
	return &OperationsController{service: service}
}
func (c *OperationsController) Register(api huma.API, token string) {
	huma.Register(api, huma.Operation{OperationID: "operationsSnapshot", Method: http.MethodGet, Path: "/internal/operations", Summary: "Diagnóstico protegido das filas", Description: "Contagens sem URLs, telefones, conteúdo ou segredos. Desativado por padrão; Bearer operacional próprio.", Tags: []string{"Operations"}, Middlewares: huma.Middlewares{internalBearerAuth(api, token)}, Errors: []int{http.StatusUnauthorized, http.StatusServiceUnavailable}}, func(ctx context.Context, _ *struct{}) (*dtos.OperationsOutput, error) {
		body, err := c.service.Snapshot(ctx)
		if err != nil {
			return nil, HTTPError(err)
		}
		return &dtos.OperationsOutput{Body: body}, nil
	})
}
func (c *OperationsController) RegisterAdmin(api huma.API, guard *SessionGuard) {
	operation := guard.operation(api, "adminOperationsSnapshot", http.MethodGet, "/operations", "Consultar alertas operacionais", security.ResourceAudit, security.ActionRead)
	huma.Register(api, operation, func(ctx context.Context, _ *struct{}) (*dtos.OperationsOutput, error) {
		body, err := c.service.Snapshot(ctx)
		if err != nil {
			return nil, HTTPError(err)
		}
		return &dtos.OperationsOutput{Body: body}, nil
	})
}
