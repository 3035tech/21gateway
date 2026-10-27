package controllers

import (
	"context"
	"net/http"

	"github.com/21mobile/sms-gateway/internal/services"
	"github.com/danielgtaylor/huma/v2"
)

// CredentialCheckController verifies the legacy Basic Auth and peer IP from
// the caller's network without consuming statuses or sending an SMS.
type CredentialCheckController struct{ auth *services.AuthService }

func NewCredentialCheckController(auth *services.AuthService) *CredentialCheckController {
	return &CredentialCheckController{auth: auth}
}

func (c *CredentialCheckController) Register(api huma.API) {
	guard := NewLegacyGuard(c.auth)
	guard.RegisterSecurity(api)
	operation := huma.Operation{
		OperationID:   "checkCredentialAccess",
		Method:        http.MethodGet,
		Path:          "/v1/auth/check",
		Summary:       "Verificar acesso à API pela rede do cliente",
		Tags:          []string{"SMS"},
		DefaultStatus: http.StatusNoContent,
		Errors:        []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusTooManyRequests, http.StatusServiceUnavailable},
		Security:      []map[string][]string{{basicAuthScheme: {}}},
		Middlewares:   huma.Middlewares{guard.Authenticate(api)},
	}
	huma.Register(api, operation, func(context.Context, *struct{}) (*struct{}, error) {
		return nil, nil
	})
}
