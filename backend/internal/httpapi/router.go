package httpapi

import (
	"context"
	"github.com/alexedwards/scs/v2"
	"net/http"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	"github.com/21mobile/sms-gateway/internal/config"
	"github.com/21mobile/sms-gateway/internal/controllers"
	"github.com/21mobile/sms-gateway/internal/security"
	"github.com/21mobile/sms-gateway/internal/services"
)

type Dependencies struct {
	ClientIPResolver              security.ClientIPResolver
	SMTPProbe                     func(context.Context) error
	SQSProbe                      func(context.Context) error
	Operations                    *services.OperationsService
	OperationsEnabled             bool
	OperationsToken               string
	HealthDependenciesTokenSHA256 string
	LegacyMO                      *services.LegacyMOService
	MO                            *services.MOService
	MOIngressEnabled              bool
	LegacyStatuses                *services.LegacyStatusService
	Users                         *services.UserService
	UserMFA                       *services.UserMFAService
	UserRecovery                  *services.UserRecoveryService
	Companies                     *services.CompanyService
	Credentials                   *services.CredentialService
	Messages                      *services.MessageService
	Schedules                     *services.ScheduledListService
	Audit                         *services.AuditService
	Summary                       *services.SummaryService
	Sessions                      *scs.SessionManager
	Callbacks                     *services.CallbackService
	CallbackIngressEnabled        bool
	CallbackIngressToken          string
	SMS                           *services.SMSService
	Auth                          *services.AuthService
}

func New(origins []string, docsEnabled bool, ping func(context.Context) error, dependencies Dependencies) http.Handler {
	controllers.ConfigureErrors()
	router := chi.NewRouter()
	router.Use(dependencies.ClientIPResolver.Middleware)
	router.Use(middleware.RequestID, middleware.Recoverer)
	router.Use(localizeAdminErrors)
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Referrer-Policy", "no-referrer")
			w.Header().Set("Cache-Control", "no-store")
			r.Body = http.MaxBytesReader(w, r.Body, config.MaxBodyBytes)
			next.ServeHTTP(w, r)
		})
	})
	router.Use(cors.Handler(cors.Options{AllowedOrigins: origins, AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"}, AllowedHeaders: []string{"Accept", "Accept-Language", "Authorization", "Content-Type", "X-CSRF-Token"}, AllowCredentials: true, MaxAge: config.CORSMaxAgeSeconds}))
	if dependencies.Sessions != nil {
		router.Use(AdminMiddleware(dependencies.Sessions, origins))
	}
	cfg := huma.DefaultConfig("21 Mobile Gateway", "0.1.0")
	cfg.DocsRenderer = huma.DocsRendererScalar
	cfg.DocsRendererConfig = map[string]any{"theme": "purple", "hideModels": false}
	cfg.SchemasPath = ""
	if !docsEnabled {
		cfg.DocsPath = ""
		cfg.OpenAPIPath = ""
		cfg.SchemasPath = ""
	}
	api := humachi.New(router, cfg)
	if dependencies.UserRecovery != nil {
		controllers.NewUserRecoveryController(dependencies.UserRecovery).Register(api)
	}
	if dependencies.Users != nil && dependencies.Sessions != nil {
		guard := controllers.NewSessionGuard(dependencies.Users, dependencies.Sessions)
		controllers.NewSessionController(dependencies.Users, guard).Register(api)
		if dependencies.Callbacks != nil {
			controllers.NewCallbackController(dependencies.Callbacks, guard).Register(api)
		}
		if dependencies.Companies != nil {
			controllers.NewCompanyController(dependencies.Companies, guard).Register(api)
		}
		if dependencies.Credentials != nil {
			controllers.NewCredentialController(dependencies.Credentials, guard).Register(api)
		}
		controllers.NewUserController(dependencies.Users, guard).Register(api)
		if dependencies.UserMFA != nil {
			controllers.NewUserMFAController(dependencies.UserMFA, guard).Register(api)
		}
		if dependencies.UserRecovery != nil {
			controllers.NewUserInvitationController(dependencies.UserRecovery, guard).Register(api)
		}
		if dependencies.Messages != nil {
			controllers.NewMessageController(dependencies.Messages, guard).Register(api)
		}
		if dependencies.Schedules != nil {
			controllers.NewScheduledSMSController(dependencies.Schedules, guard).Register(api)
		}
		if dependencies.Audit != nil {
			controllers.NewAuditController(dependencies.Audit, guard).Register(api)
		}
		if dependencies.Summary != nil {
			controllers.NewSummaryController(dependencies.Summary, guard).Register(api)
		}
	}
	if dependencies.Callbacks != nil && dependencies.CallbackIngressEnabled {
		controllers.NewCallbackController(dependencies.Callbacks, nil).RegisterIngress(api, dependencies.CallbackIngressToken)
	}
	if dependencies.MO != nil && dependencies.MOIngressEnabled {
		controllers.NewMOController(dependencies.MO).RegisterIngress(api, dependencies.CallbackIngressToken)
	}
	if dependencies.Operations != nil && dependencies.OperationsEnabled {
		controllers.NewOperationsController(dependencies.Operations).Register(api, dependencies.OperationsToken)
	}
	controllers.NewHealthController(services.NewHealthService(ping).WithDependencies(dependencies.SMTPProbe, dependencies.SQSProbe)).Register(api, dependencies.HealthDependenciesTokenSHA256)
	if dependencies.SMS != nil && dependencies.Auth != nil {
		controllers.NewSMSController(dependencies.SMS, dependencies.Auth).Register(api)
	}
	if dependencies.Auth != nil {
		controllers.NewCredentialCheckController(dependencies.Auth).Register(api)
	}
	if dependencies.LegacyStatuses != nil && dependencies.Auth != nil {
		controllers.NewLegacyStatusController(dependencies.LegacyStatuses, dependencies.Auth).Register(api)
	}
	if dependencies.LegacyMO != nil && dependencies.Auth != nil {
		controllers.NewLegacyMOController(dependencies.LegacyMO, dependencies.Auth).Register(api)
	}
	if docsEnabled {
		limitOpenAPIToClientIntegration(api.OpenAPI())
		describeClientIntegration(api.OpenAPI())
	}
	router.NotFound(func(w http.ResponseWriter, r *http.Request) { controllers.WriteHTTPError(w, http.StatusNotFound) })
	router.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		controllers.WriteHTTPError(w, http.StatusMethodNotAllowed)
	})
	return router
}

func limitOpenAPIToClientIntegration(spec *huma.OpenAPI) {
	clientPaths := map[string]bool{
		controllers.LegacySendPath:   true,
		controllers.LegacyStatusPath: true,
		controllers.LegacyMOPath:     true,
		"/v1/auth/check":             true,
	}
	for path := range spec.Paths {
		if !clientPaths[path] {
			delete(spec.Paths, path)
		}
	}
	usedSchemes := map[string]bool{}
	for _, path := range spec.Paths {
		for _, operation := range []*huma.Operation{path.Get, path.Post, path.Put, path.Patch, path.Delete, path.Head, path.Options, path.Trace} {
			if operation == nil {
				continue
			}
			for _, requirement := range operation.Security {
				for scheme := range requirement {
					usedSchemes[scheme] = true
				}
			}
		}
	}
	for scheme := range spec.Components.SecuritySchemes {
		if !usedSchemes[scheme] {
			delete(spec.Components.SecuritySchemes, scheme)
		}
	}
}
