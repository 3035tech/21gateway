package controllers

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/services"
)

const (
	LegacySendPath     = "/v1/send"
	basicAuthScheme    = "basicAuth"
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
		Summary:     "Enviar SMS",
		Description: "Aceita de 1 a 200 itens em sms, com resultado individual e ordem preservada. Texto encaminhado sem diacríticos e emojis, limitado a 160 caracteres na entrada. No modo assíncrono, cada item válido é persistido e recebe id_sms antes da publicação da referência na SQS. OK confirma aceite para processamento; não confirma envio ao gateway nem entrega. O painel mostra o resultado de submissão; /v1/status lista somente eventos DLR. scheduledDate futuro define quando a referência poderá ser publicada.",
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
	requestID := middleware.GetReqID(ctx)
	results, err := c.service.Send(ctx, credential.ID, input.Body.SMS)
	if err != nil {
		slog.Error("SMS send request failed", "request_id", requestID, "credential_id", credential.ID, "items", len(input.Body.SMS), "error", err)
		return nil, HTTPError(err)
	}
	accepted := 0
	sendIDs := make([]string, 0, len(results))
	for _, result := range results {
		if result.IDSMS != "" {
			accepted++
			sendIDs = append(sendIDs, result.IDSMS)
		}
	}
	slog.Info("SMS send request completed", "request_id", requestID, "credential_id", credential.ID, "items", len(results), "with_send_id", accepted, "send_ids", sendIDs)
	return &dtos.LegacySendOutput{Body: results}, nil
}
