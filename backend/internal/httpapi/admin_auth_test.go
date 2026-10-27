package httpapi

import (
	"context"
	"github.com/21mobile/sms-gateway/internal/controllers"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/security"
	"github.com/21mobile/sms-gateway/internal/services"
	"github.com/alexedwards/scs/v2"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fixtureUser struct {
	user        models.SystemUser
	permissions *security.Permissions
}

func (f *fixtureUser) Login(context.Context, string, string) (models.SystemUser, error) {
	return f.user, nil
}
func (f *fixtureUser) LoginWithFactor(ctx context.Context, email, password, _ string) (models.SystemUser, error) {
	return f.Login(ctx, email, password)
}
func (f *fixtureUser) AllowLogin(context.Context, string) error { return nil }
func (f *fixtureUser) SessionUser(_ context.Context, id uint64, version int64) (models.SystemUser, error) {
	if id != f.user.ID || version != f.user.SessionVersion || !f.user.Active {
		return models.SystemUser{}, apperrors.Wrap(apperrors.CodeUnauthorized, nil)
	}
	return f.user, nil
}
func (f *fixtureUser) Allowed(role models.Role, resource security.Resource, action security.Action) bool {
	return f.permissions.Allow(role, resource, action)
}
func TestAdministrativeSessionAndPermissions(t *testing.T) {
	controllers.ConfigureErrors()
	permissions, err := security.NewPermissions()
	if err != nil {
		t.Fatal(err)
	}
	users := &fixtureUser{user: models.SystemUser{ID: 1, Name: "Viewer", Email: "viewer@example.test", Active: true, Role: models.RoleViewer, SessionVersion: 1}, permissions: permissions}
	sessions := scs.New()
	sessions.Cookie.Name = SessionCookieName
	sessions.Cookie.Path = "/admin"
	sessions.Cookie.Persist = false
	sessions.IdleTimeout = 0
	router := chi.NewRouter()
	router.Use(AdminMiddleware(sessions, []string{"http://localhost:5173"}))
	router.Get("/admin/test/seed", func(w http.ResponseWriter, r *http.Request) {
		sessions.Put(r.Context(), "seed", true)
		w.WriteHeader(204)
	})
	api := humachi.New(router, huma.DefaultConfig("test", "1"))
	guard := controllers.NewSessionGuard(users, sessions)
	controllers.NewSessionController(users, guard).Register(api)
	controllers.NewCompanyController(services.NewCompanyService(nil), guard).Register(api)
	call := func(method, path, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, "http://localhost:8080"+path, strings.NewReader(body))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Origin", "http://localhost:5173")
		if cookie != nil {
			r.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		router.ServeHTTP(w, r)
		return w
	}
	if code := call("GET", "/admin/auth/me", "", nil).Code; code != 401 {
		t.Fatal("anonymous access", code)
	}
	seed := call("GET", "/admin/test/seed", "", nil).Result().Cookies()[0]
	login := call("POST", "/admin/auth/login", `{"email":"viewer@example.test","password":"anything"}`, seed)
	if login.Code != 200 {
		t.Fatal("login", login.Code, login.Body.String())
	}
	cookie := login.Result().Cookies()[0]
	if cookie.MaxAge > 0 {
		t.Fatal("ordinary session persisted")
	}
	if cookie.Value == seed.Value {
		t.Fatal("session token was not renewed")
	}
	if code := call("GET", "/admin/auth/me", "", cookie).Code; code != 200 {
		t.Fatal("authenticated access", code)
	}
	remembered := call("POST", "/admin/auth/login", `{"email":"viewer@example.test","password":"anything","remember":true}`, cookie)
	if remembered.Code != 200 {
		t.Fatal("remembered login", remembered.Code, remembered.Body.String())
	}
	if remembered.Result().Cookies()[0].MaxAge <= 0 {
		t.Fatal("remembered session has no persistent cookie")
	}
	cookie = remembered.Result().Cookies()[0]
	if code := call("POST", "/admin/companies", `{"name":"Denied","reference":"","active":true}`, cookie).Code; code != 403 {
		t.Fatal("viewer may write", code)
	}
	users.user.SessionVersion++
	if code := call("GET", "/admin/auth/me", "", cookie).Code; code != 401 {
		t.Fatal("revoked session accepted", code)
	}
	if code := call("POST", "/admin/auth/logout", "", cookie).Code; code != 204 {
		t.Fatal("logout revoked session", code)
	}
	if code := call("GET", "/admin/auth/me", "", cookie).Code; code != 401 {
		t.Fatal("logout did not destroy session", code)
	}
}
