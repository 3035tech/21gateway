package controllers

import (
	"context"
	"net/http"

	"github.com/21mobile/sms-gateway/internal/dtos"
	"github.com/21mobile/sms-gateway/internal/security"
	"github.com/21mobile/sms-gateway/internal/services"
	"github.com/danielgtaylor/huma/v2"
)

type NotificationController struct {
	*SessionGuard
	service *services.NotificationService
}

func NewNotificationController(service *services.NotificationService, guard *SessionGuard) *NotificationController {
	return &NotificationController{SessionGuard: guard, service: service}
}

func (c *NotificationController) Register(api huma.API) {
	huma.Register(api, c.operation(api, "adminNotifications", http.MethodGet, "/notifications", "Listar notificações", security.ResourceNotifications, security.ActionRead), func(ctx context.Context, input *dtos.NotificationListInput) (*dtos.NotificationListOutput, error) {
		result, err := c.service.List(ctx, actor(ctx), *input)
		if err != nil {
			return nil, HTTPError(err)
		}
		return &dtos.NotificationListOutput{Body: result}, nil
	})
	huma.Register(api, c.operation(api, "adminNotificationRead", http.MethodPost, "/notifications/{id}/read", "Marcar notificação como lida", security.ResourceNotifications, security.ActionWrite), func(ctx context.Context, input *dtos.NotificationReadInput) (*struct{}, error) {
		if err := c.service.MarkRead(ctx, actor(ctx), input.ID); err != nil {
			return nil, HTTPError(err)
		}
		return nil, nil
	})
	huma.Register(api, c.operation(api, "adminNotificationPreferences", http.MethodGet, "/notification-preferences", "Consultar tipos de alerta", security.ResourceNotifications, security.ActionRead), func(ctx context.Context, _ *struct{}) (*dtos.NotificationPreferencesOutput, error) {
		items, err := c.service.Preferences(ctx)
		if err != nil {
			return nil, HTTPError(err)
		}
		return &dtos.NotificationPreferencesOutput{Body: struct {
			Items          []dtos.NotificationPreference `json:"items"`
			EmailAvailable bool                          `json:"emailAvailable"`
		}{Items: items, EmailAvailable: c.service.EmailAvailable()}}, nil
	})
	huma.Register(api, c.operation(api, "adminNotificationPreferenceUpdate", http.MethodPut, "/notification-preferences/{code}", "Configurar e-mail do alerta", security.ResourceNotifications, security.ActionWrite), func(ctx context.Context, input *dtos.NotificationPreferenceInput) (*dtos.NotificationPreferenceOutput, error) {
		item, err := c.service.SetPreference(ctx, input.Code, input.Body.EmailEnabled)
		if err != nil {
			return nil, HTTPError(err)
		}
		return &dtos.NotificationPreferenceOutput{Body: item}, nil
	})
}
