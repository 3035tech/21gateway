package controllers

import (
	"context"
	"net/http"

	"github.com/danielgtaylor/huma/v2"

	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/services"
)

const (
	LegacySendPath     = "/v1/send"
	basicAuthScheme    = "legacyBasicAuth"
	basicAuthChallenge = `Basic realm="21 Mobile API", charset="UTF-8"`
)

type SMSController struct {
	service *services.SMSService
	auth    *services.AuthService
}

func NewSMSController(service *services.SMSService, auth *services.AuthService) *SMSController {
	return &SMSController{service: service, auth: auth}
}

func (c *SMSController) Register(api huma.API) {
	guard := NewLegacyGuard(c.auth)
	guard.RegisterSecurity(api)
	operation := huma.Operation{
		OperationID: "sendSMS",
		Method:      http.MethodPost,
		Path:        LegacySendPath,
		Summary:     "Envio imediato de SMS pelo contrato legado",
		Description: "Aceita de 1 a 200 itens em sms, com resultado individual e ordem preservada. Texto encaminhado sem diacríticos e emojis, limitado a 160 caracteres na entrada. Aceite não confirma entrega. scheduledDate futuro pode ser aceito na fila local quando o scheduler estiver habilitado; OK confirma aceite, não envio ou entrega.",
		Tags:        []string{"SMS"},
		// Preserve NOK per-item responses while Huma still handles JSON decoding and schema documentation.
		SkipValidateBody: true,
		Errors:           []int{http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden, http.StatusTooManyRequests, http.StatusRequestEntityTooLarge, http.StatusUnsupportedMediaType, http.StatusUnprocessableEntity, http.StatusServiceUnavailable},
		Security:         []map[string][]string{{basicAuthScheme: {}}},
		Middlewares:      huma.Middlewares{guard.Authenticate(api)},
	}
	huma.Register(api, operation, c.Send)
}

func (c *SMSController) Send(ctx context.Context, input *dtos.LegacySendInput) (*dtos.LegacySendOutput, error) {
	credential, ok := services.CredentialFromContext(ctx)
	if !ok {
		return nil, HTTPError(apperrors.Wrap(apperrors.CodeUnauthorized, nil))
	}
	results, err := c.service.Send(ctx, credential.ID, input.Body.SMS)
	if err != nil {
		return nil, HTTPError(err)
	}
	return &dtos.LegacySendOutput{Body: results}, nil
}
