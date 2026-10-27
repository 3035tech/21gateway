package controllers

import (
	"context"
	"github.com/21mobile/sms-gateway/internal/dtos"
	"github.com/21mobile/sms-gateway/internal/security"
	"github.com/21mobile/sms-gateway/internal/services"
	"github.com/danielgtaylor/huma/v2"
	"net/http"
)

type AuditController struct {
	*SessionGuard
	service *services.AuditService
}

func NewAuditController(service *services.AuditService, guard *SessionGuard) *AuditController {
	return &AuditController{SessionGuard: guard, service: service}
}
func (c *AuditController) Register(api huma.API) {
	huma.Register(api, c.operation(api, "listAudit", http.MethodGet, "/audit", "Consultar auditoria", security.ResourceAudit, security.ActionRead), func(ctx context.Context, input *dtos.AuditListQuery) (*dtos.AuditOutput, error) {
		values, err := c.service.Audits(ctx, *input)
		if err != nil {
			return nil, HTTPError(err)
		}
		return &dtos.AuditOutput{Body: values}, nil
	})
}
