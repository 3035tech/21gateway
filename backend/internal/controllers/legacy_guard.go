package controllers

import (
	"github.com/21mobile/sms-gateway/internal/security"
	"github.com/21mobile/sms-gateway/internal/services"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"net/http"
)

type LegacyGuard struct{ auth *services.AuthService }

func NewLegacyGuard(auth *services.AuthService) *LegacyGuard { return &LegacyGuard{auth: auth} }
func (g *LegacyGuard) RegisterSecurity(api huma.API) {
	if api.OpenAPI().Components.SecuritySchemes == nil {
		api.OpenAPI().Components.SecuritySchemes = map[string]*huma.SecurityScheme{}
	}
	api.OpenAPI().Components.SecuritySchemes[basicAuthScheme] = &huma.SecurityScheme{Type: "http", Scheme: "basic"}
}
func (g *LegacyGuard) Authenticate(api huma.API) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		request, _ := humachi.Unwrap(ctx)
		clientIP := security.ClientIP(request)
		credential, err := g.auth.AuthenticateRequest(ctx.Context(), ctx.Header("Authorization"), clientIP)
		if err == nil {
			err = g.auth.AuthorizeIP(credential, clientIP)
		}
		if err != nil {
			public := HTTPError(err)
			if public.GetStatus() == http.StatusUnauthorized {
				ctx.SetHeader("WWW-Authenticate", basicAuthChallenge)
			}
			_ = huma.WriteErr(api, ctx, public.GetStatus(), "")
			return
		}
		next(huma.WithContext(ctx, services.WithCredential(ctx.Context(), credential)))
	}
}
