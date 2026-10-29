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
	"time"
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
	router.Get("/admin/test/backdate", func(w http.ResponseWriter, r *http.Request) {
		sessions.Put(r.Context(), "last_activity", time.Now().Add(-2*time.Hour).Unix())
		w.WriteHeader(204)
	})
	router.Get("/admin/test/backdate-week", func(w http.ResponseWriter, r *http.Request) {
		sessions.Put(r.Context(), "last_activity", time.Now().Add(-8*24*time.Hour).Unix())
		w.WriteHeader(204)
	})
	api := humachi.New(router, huma.DefaultConfig("test", "1"))
	guard := controllers.NewSessionGuard(users, sessions)
	controllers.NewSessionController(users, guard).Register(api)
	controllers.NewCompanyController(services.NewCompanyService(nil), guard).Register(api)
	operations := &operationsMemory{}
	controllers.NewOperationsController(services.NewOperationsService(operations)).RegisterAdmin(api, guard)
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
	anonymous := call("GET", "/admin/auth/me", "", nil)
	if anonymous.Code != 401 {
		t.Fatal("anonymous access", anonymous.Code)
	}
	if cookies := anonymous.Result().Cookies(); len(cookies) != 0 {
		t.Fatal("anonymous access created a session cookie")
	}
	if code := call("GET", "/admin/companies/by-ref/11111111-1111-4111-8111-111111111111", "", nil).Code; code != 401 {
		t.Fatal("anonymous public reference lookup", code)
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
	backdated := call("GET", "/admin/test/backdate", "", cookie)
	cookie = backdated.Result().Cookies()[0]
	if code := call("GET", "/admin/auth/me", "", cookie).Code; code != 200 {
		t.Fatal("regular session expired after two idle hours", code)
	}
	remembered := call("POST", "/admin/auth/login", `{"email":"viewer@example.test","password":"anything","remember":true}`, cookie)
	if remembered.Code != 200 {
		t.Fatal("remembered login", remembered.Code, remembered.Body.String())
	}
	if remembered.Result().Cookies()[0].MaxAge < int((13 * 24 * time.Hour).Seconds()) {
		t.Fatal("remembered session cookie is shorter than expected")
	}
	cookie = remembered.Result().Cookies()[0]
	backdated = call("GET", "/admin/test/backdate-week", "", cookie)
	cookie = backdated.Result().Cookies()[0]
	if code := call("GET", "/admin/auth/me", "", cookie).Code; code != 200 {
		t.Fatal("remembered session expired after eight idle days", code)
	}
	if code := call("POST", "/admin/companies", `{"name":"Denied","reference":"","active":true}`, cookie).Code; code != 403 {
		t.Fatal("viewer may write", code)
	}
	if code := call("GET", "/admin/operations", "", cookie).Code; code != 403 || operations.calls != 0 {
		t.Fatal("viewer may read global operations", code, operations.calls)
	}
	users.user.Role = models.RoleAdmin
	if code := call("GET", "/admin/operations", "", cookie).Code; code != 200 || operations.calls != 1 {
		t.Fatal("admin cannot read operations", code, operations.calls)
	}
	users.user.Role = models.RoleViewer
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
