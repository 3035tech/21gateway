package controllers

import (
	"context"
	"github.com/21mobile/sms-gateway/internal/dtos"
	"github.com/21mobile/sms-gateway/internal/security"
	"github.com/21mobile/sms-gateway/internal/services"
	"github.com/danielgtaylor/huma/v2"
	"net/http"
)

type MessageController struct {
	*SessionGuard
	service *services.MessageService
}

func NewMessageController(service *services.MessageService, guard *SessionGuard) *MessageController {
	return &MessageController{SessionGuard: guard, service: service}
}
func (c *MessageController) Register(api huma.API) {
	huma.Register(api, c.operation(api, "listMessages", http.MethodGet, "/messages", "Consultar tentativas de envio", security.ResourceMessages, security.ActionRead), func(ctx context.Context, input *dtos.ListQuery) (*dtos.MessagesOutput, error) {
		values, err := c.service.Messages(ctx, *input)
		if err != nil {
			return nil, HTTPError(err)
		}
		return &dtos.MessagesOutput{Body: values}, nil
	})

}
