package httpapi

import (
	"github.com/21mobile/sms-gateway/internal/controllers"
	"github.com/alexedwards/scs/v2"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAdminCSRFAndLegacyIsolation(t *testing.T) {
	controllers.ConfigureErrors()
	sessions := scs.New()
	handler := AdminMiddleware(sessions, []string{"https://panel.example"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNoContent) }))
	cases := []struct {
		path, method, origin, site string
		status                     int
	}{{"/admin/companies", "POST", "https://evil.example", "cross-site", 403}, {"/admin/auth/login", "POST", "http://panel.example", "cross-site", 403}, {"/admin/auth/login", "POST", "https://panel.example", "cross-site", 204}, {"/admin/companies", "GET", "https://evil.example", "cross-site", 204}, {"/v1/send", "POST", "https://evil.example", "cross-site", 204}}
	for _, c := range cases {
		r := httptest.NewRequest(c.method, "https://api.example"+c.path, nil)
		r.Header.Set("Origin", c.origin)
		r.Header.Set("Sec-Fetch-Site", c.site)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		if w.Code != c.status {
			t.Errorf("%s %s %s: %d want %d", c.method, c.path, c.origin, w.Code, c.status)
		}
	}
}
func TestProductionSessionCookieFlags(t *testing.T) {
	sessions := NewSessions(nil, true, true)
	if !sessions.Cookie.Secure || !sessions.Cookie.HttpOnly || sessions.Cookie.SameSite != http.SameSiteNoneMode || sessions.Cookie.Path != "/admin" {
		t.Fatal("unsafe production cookie configuration")
	}
}
