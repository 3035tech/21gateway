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
	huma.Register(api, huma.Operation{OperationID: "dependencies", Method: http.MethodGet, Path: "/health/dependencies", Summary: "Database, SMTP and SQS diagnostics", Tags: []string{"Operations"}}, c.Dependencies)
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

func (c *HealthController) Dependencies(ctx context.Context, input *struct{}) (*dtos.DependenciesOutput, error) {
	result := c.service.CheckDependencies(ctx)
	status := http.StatusOK
	if result.Status == dtos.HealthStatusDegraded {
		status = http.StatusServiceUnavailable
	}
	return &dtos.DependenciesOutput{Status: status, Body: result}, nil
}
