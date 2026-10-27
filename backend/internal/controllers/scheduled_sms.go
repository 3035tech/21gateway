package controllers

import (
	"context"
	"net/http"

	"github.com/21mobile/sms-gateway/internal/dtos"
	"github.com/21mobile/sms-gateway/internal/security"
	"github.com/21mobile/sms-gateway/internal/services"
	"github.com/danielgtaylor/huma/v2"
)

type ScheduledSMSController struct {
	*SessionGuard
	service *services.ScheduledListService
}

func NewScheduledSMSController(service *services.ScheduledListService, guard *SessionGuard) *ScheduledSMSController {
	return &ScheduledSMSController{SessionGuard: guard, service: service}
}

func (c *ScheduledSMSController) Register(api huma.API) {
	huma.Register(api, c.operation(api, "listSchedules", http.MethodGet, "/schedules", "Consultar SMS agendados", security.ResourceMessages, security.ActionRead), func(ctx context.Context, input *dtos.ScheduledListQuery) (*dtos.SchedulesOutput, error) {
		values, err := c.service.List(ctx, *input)
		if err != nil {
			return nil, HTTPError(err)
		}
		return &dtos.SchedulesOutput{Body: values}, nil
	})
}
