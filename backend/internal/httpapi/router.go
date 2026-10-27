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
	"github.com/21mobile/sms-gateway/internal/services"
)

type Dependencies struct {
	Operations             *services.OperationsService
	OperationsEnabled      bool
	OperationsToken        string
	LegacyMO               *services.LegacyMOService
	MO                     *services.MOService
	MOIngressEnabled       bool
	LegacyStatuses         *services.LegacyStatusService
	Users                  *services.UserService
	UserMFA                *services.UserMFAService
	UserRecovery           *services.UserRecoveryService
	Companies              *services.CompanyService
	Credentials            *services.CredentialService
	Messages               *services.MessageService
	Schedules              *services.ScheduledListService
	Audit                  *services.AuditService
	Summary                *services.SummaryService
	Sessions               *scs.SessionManager
	Callbacks              *services.CallbackService
	CallbackIngressEnabled bool
	CallbackIngressToken   string
	SMS                    *services.SMSService
	Auth                   *services.AuthService
}

func New(origins []string, docsEnabled bool, ping func(context.Context) error, dependencies Dependencies) http.Handler {
	controllers.ConfigureErrors()
	router := chi.NewRouter()
	router.Use(middleware.RequestID, middleware.Recoverer)
	router.Use(localizeAdminErrors)
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	cfg.DocsRenderer = huma.DocsRendererSwaggerUI
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
	controllers.NewHealthController(services.NewHealthService(ping)).Register(api)
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
	router.NotFound(func(w http.ResponseWriter, r *http.Request) { controllers.WriteHTTPError(w, http.StatusNotFound) })
	router.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		controllers.WriteHTTPError(w, http.StatusMethodNotAllowed)
	})
	return router
}
