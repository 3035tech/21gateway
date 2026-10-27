package controllers

import (
	"context"
	"github.com/21mobile/sms-gateway/internal/dtos"
	"github.com/21mobile/sms-gateway/internal/security"
	"github.com/21mobile/sms-gateway/internal/services"
	"github.com/danielgtaylor/huma/v2"
	"net/http"
)

type SummaryController struct {
	*SessionGuard
	service *services.SummaryService
}

func NewSummaryController(service *services.SummaryService, guard *SessionGuard) *SummaryController {
	return &SummaryController{SessionGuard: guard, service: service}
}
func (c *SummaryController) Register(api huma.API) {
	huma.Register(api, c.operation(api, "adminSummary", http.MethodGet, "/summary", "Resumo operacional", security.ResourceMessages, security.ActionRead), func(ctx context.Context, _ *struct{}) (*dtos.SummaryOutput, error) {
		value, err := c.service.Summary(ctx)
		if err != nil {
			return nil, HTTPError(err)
		}
		return &dtos.SummaryOutput{Body: value}, nil
	})

}
