package httpapi

import (
	"database/sql"
	"github.com/21mobile/sms-gateway/internal/controllers"
	"github.com/alexedwards/scs/postgresstore"
	"github.com/alexedwards/scs/v2"
	"net/http"
	"strings"
	"time"
)

const SessionCookieName = "21mobile_session"
const sessionLifetime = 8 * time.Hour

func NewSessions(pool *sql.DB, secure, crossSite bool) *scs.SessionManager {
	sessions := scs.New()
	// Cleanup is owned by the operational worker; avoid library logging raw SQL errors.
	sessions.Store = postgresstore.NewWithCleanupInterval(pool, 0)
	sessions.Lifetime = sessionLifetime
	// Per-session idle limits are enforced by SessionGuard so remembered sessions can differ.
	sessions.IdleTimeout = 0
	sessions.Cookie.Name = SessionCookieName
	sessions.Cookie.Path = "/admin"
	sessions.Cookie.HttpOnly = true
	sessions.Cookie.Secure = secure
	sessions.Cookie.Persist = false
	sessions.Cookie.SameSite = http.SameSiteLaxMode
	if crossSite {
		sessions.Cookie.SameSite = http.SameSiteNoneMode
	}
	sessions.ErrorFunc = func(w http.ResponseWriter, r *http.Request, _ error) {
		controllers.WriteHTTPError(w, http.StatusServiceUnavailable)
	}
	return sessions
}

// Protect only the backoffice. Legacy Basic Auth and supplier ingress have independent authentication.
func AdminMiddleware(sessions *scs.SessionManager, origins []string) func(http.Handler) http.Handler {
	protection := http.NewCrossOriginProtection()
	for _, origin := range origins {
		if err := protection.AddTrustedOrigin(origin); err != nil {
			panic("invalid configured admin origin")
		}
	}
	protection.SetDenyHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { controllers.WriteHTTPError(w, http.StatusForbidden) }))
	return func(next http.Handler) http.Handler {
		protected := protection.Handler(sessions.LoadAndSave(next))
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/admin" || strings.HasPrefix(r.URL.Path, "/admin/") {
				protected.ServeHTTP(w, r)
			} else {
				next.ServeHTTP(w, r)
			}
		})
	}
}
