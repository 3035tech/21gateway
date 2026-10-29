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
	huma.Register(api, c.operation(api, "getMessageContent", http.MethodGet, "/messages/{id}/content", "Consultar texto do SMS", security.ResourceMessages, security.ActionRead), func(ctx context.Context, input *dtos.IDInput) (*dtos.MessageContentOutput, error) {
		value, err := c.service.Content(ctx, input.ID)
		if err != nil {
			return nil, HTTPError(err)
		}
		return &dtos.MessageContentOutput{Body: value}, nil
	})
	huma.Register(api, c.operation(api, "listMessages", http.MethodGet, "/messages", "Consultar tentativas de envio", security.ResourceMessages, security.ActionRead), func(ctx context.Context, input *dtos.ListQuery) (*dtos.MessagesOutput, error) {
		values, err := c.service.Messages(ctx, *input)
		if err != nil {
			return nil, HTTPError(err)
		}
		return &dtos.MessagesOutput{Body: values}, nil
	})
	huma.Register(api, c.operation(api, "sendCredentialTestSMS", http.MethodPost, "/credentials/{id}/test-send", "Enviar um SMS de homologação", security.ResourceMessages, security.ActionWrite), func(ctx context.Context, input *dtos.TestSendInput) (*dtos.TestSendOutput, error) {
		value, err := c.service.SendTest(ctx, actor(ctx), input.ID, input.Body.Destination, input.Body.MessageText)
		if err != nil {
			return nil, HTTPError(err)
		}
		return &dtos.TestSendOutput{Body: value}, nil
	})
	huma.Register(api, c.operation(api, "sendCompanyTokenTestSMS", http.MethodPost, "/companies/by-ref/{publicRef}/test-send", "Testar token Zoug da empresa com um SMS", security.ResourceMessages, security.ActionWrite), func(ctx context.Context, input *dtos.CompanyTestSendInput) (*dtos.TestSendOutput, error) {
		value, err := c.service.SendCompanyTest(ctx, actor(ctx), input.PublicRef, input.Body.Destination, input.Body.MessageText)
		if err != nil {
			return nil, HTTPError(err)
		}
		return &dtos.TestSendOutput{Body: value}, nil
	})
	huma.Register(api, c.operation(api, "listCompanyTokenTests", http.MethodGet, "/companies/by-ref/{publicRef}/test-sends", "Consultar últimos SMS de teste da empresa", security.ResourceMessages, security.ActionRead), func(ctx context.Context, input *dtos.CompanyTestHistoryInput) (*dtos.CompanyTestHistoryOutput, error) {
		items, err := c.service.CompanyTests(ctx, input.PublicRef)
		if err != nil {
			return nil, HTTPError(err)
		}
		return &dtos.CompanyTestHistoryOutput{Body: items}, nil
	})
	huma.Register(api, c.operation(api, "diagnoseZougResponses", http.MethodPost, "/companies/by-ref/{publicRef}/zoug-diagnostic", "Diagnosticar respostas Zoug sem SMS real", security.ResourceMessages, security.ActionWrite), func(ctx context.Context, input *dtos.ZougDiagnosticInput) (*dtos.ZougDiagnosticOutput, error) {
		value, err := c.service.DiagnoseCompany(ctx, actor(ctx), input.PublicRef)
		if err != nil {
			return nil, HTTPError(err)
		}
		return &dtos.ZougDiagnosticOutput{Body: value}, nil
	})

}
