package controllers

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/21mobile/sms-gateway/internal/dtos"
	"github.com/21mobile/sms-gateway/internal/services"
)

type HealthController struct{ service *services.HealthService }

func NewHealthController(service *services.HealthService) *HealthController {
	return &HealthController{service: service}
}

func (c *HealthController) Register(api huma.API) {
	huma.Register(api, huma.Operation{OperationID: "health", Method: http.MethodGet, Path: "/healthz", Summary: "Process health", Tags: []string{"Operations"}}, c.Health)
	huma.Register(api, huma.Operation{OperationID: "ready", Method: http.MethodGet, Path: "/readyz", Summary: "Database readiness", Tags: []string{"Operations"}}, c.Ready)
}

func (c *HealthController) Health(ctx context.Context, input *struct{}) (*dtos.HealthOutput, error) {
	return &dtos.HealthOutput{Body: dtos.HealthResponse{Status: dtos.HealthStatusOK}}, nil
}

func (c *HealthController) Ready(ctx context.Context, input *struct{}) (*dtos.HealthOutput, error) {
	if err := c.service.CheckReadiness(ctx); err != nil {
		return nil, HTTPError(err)
	}
	return &dtos.HealthOutput{Body: dtos.HealthResponse{Status: dtos.HealthStatusOK}}, nil
}
