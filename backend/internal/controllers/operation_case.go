package controllers

import (
	"context"
	"net/http"

	"github.com/21mobile/sms-gateway/internal/dtos"
	"github.com/21mobile/sms-gateway/internal/security"
	"github.com/21mobile/sms-gateway/internal/services"
	"github.com/danielgtaylor/huma/v2"
)

type OperationCaseController struct {
	*SessionGuard
	service *services.OperationCaseService
}

func NewOperationCaseController(service *services.OperationCaseService, guard *SessionGuard) *OperationCaseController {
	return &OperationCaseController{SessionGuard: guard, service: service}
}
func (c *OperationCaseController) Register(api huma.API) {
	huma.Register(api, c.operation(api, "listOperationCases", http.MethodGet, "/operation-cases", "Fila operacional", security.ResourceOperationCases, security.ActionRead), func(ctx context.Context, input *dtos.OperationCaseInput) (*dtos.OperationCasesOutput, error) {
		items, err := c.service.List(ctx, *input)
		if err != nil {
			return nil, HTTPError(err)
		}
		return &dtos.OperationCasesOutput{Body: items}, nil
	})
	huma.Register(api, c.operation(api, "updateOperationCase", http.MethodPut, "/operation-cases/{kind}/{id}", "Acompanhar ocorrência", security.ResourceOperationCases, security.ActionWrite), func(ctx context.Context, input *dtos.OperationCaseUpdateInput) (*struct{}, error) {
		if err := c.service.Update(ctx, actor(ctx), input.Kind, input.SourceID, input.Body.Status, input.Body.Note); err != nil {
			return nil, HTTPError(err)
		}
		return nil, nil
	})
	huma.Register(api, c.operation(api, "operationCaseHistory", http.MethodGet, "/operation-cases/{kind}/{id}/history", "Histórico da ocorrência", security.ResourceOperationCases, security.ActionRead), func(ctx context.Context, input *dtos.OperationCaseHistoryInput) (*dtos.OperationCaseHistoryOutput, error) {
		items, err := c.service.History(ctx, input.Kind, input.SourceID)
		if err != nil {
			return nil, HTTPError(err)
		}
		return &dtos.OperationCaseHistoryOutput{Body: items}, nil
	})
}
