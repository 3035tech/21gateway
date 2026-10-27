package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/21mobile/sms-gateway/internal/controllers"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/security"
	"github.com/21mobile/sms-gateway/internal/services"
	"github.com/alexedwards/scs/v2"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
)

type attackBudget struct {
	counts   map[string]int
	failures map[string]int
}

func newAttackBudget() *attackBudget {
	return &attackBudget{counts: map[string]int{}, failures: map[string]int{}}
}
func (b *attackBudget) Reserve(_ context.Context, key string, limit int) (bool, error) {
	b.counts[key]++
	return b.counts[key] <= limit, nil
}
func (b *attackBudget) Blocked(_ context.Context, key string, limit int) (bool, error) {
	return b.failures[key] >= limit, nil
}
func (b *attackBudget) Failure(_ context.Context, key string, _ int) error {
	b.failures[key]++
	return nil
}

func TestSecurityAPIPasswordGuessingAcrossPeersAndSpoofedHeaders(t *testing.T) {
	hash, err := security.HashPassword("existing-legacy-password")
	if err != nil {
		t.Fatal(err)
	}
	budget := newAttackBudget()
	auth, err := services.NewAuthServiceWithProtection(testCredentials{value: models.APICredential{ID: 7, Username: "client", PasswordHash: hash, Active: true}}, budget, security.APIAuthenticationIPLimit)
	if err != nil {
		t.Fatal(err)
	}
	attempts := &testAttempts{}
	router := New(nil, false, func(context.Context) error { return nil }, Dependencies{Auth: auth, SMS: services.NewSMSService(attempts, nil)})
	for i := 0; i < security.AuthenticationFailureLimit+3; i++ {
		request := httptest.NewRequest(http.MethodPost, "/v1/send", strings.NewReader(`{"sms":[]}`))
		request.RemoteAddr = fmt.Sprintf("192.0.2.%d:1234", i+1)
		request.Header.Set("Content-Type", "application/json")
		request.SetBasicAuth("client", fmt.Sprintf("guessed-password-%d", i))
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		want := http.StatusUnauthorized
		if i >= security.AuthenticationFailureLimit {
			want = http.StatusTooManyRequests
		}
		if response.Code != want {
			t.Fatalf("guess %d got %d want %d", i, response.Code, want)
		}
		if strings.Contains(response.Body.String(), hash) || strings.Contains(response.Body.String(), "guessed-password") {
			t.Fatal("authentication leaked secret")
		}
	}
	request := httptest.NewRequest(http.MethodPost, "/v1/send", strings.NewReader(`{"sms":[]}`))
	request.RemoteAddr = "192.0.2.99:4321"
	request.Header.Set("Content-Type", "application/json")
	request.SetBasicAuth("client", "existing-legacy-password")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusTooManyRequests {
		t.Fatal("account cooldown bypassed with another peer")
	}
	if len(attempts.created) != 0 {
		t.Fatal("unauthorized attempt reached SMS persistence")
	}
	// All malformed auth attempts still consume the real peer IP budget.
	key := security.AuthenticationKey("api-ip", "198.51.100.4")
	budget.counts[key] = security.APIAuthenticationIPLimit
	for i := 0; i < 3; i++ {
		request := httptest.NewRequest(http.MethodPost, "/v1/send", nil)
		request.RemoteAddr = "198.51.100.4:9999"
		request.Header.Set("X-Forwarded-For", fmt.Sprintf("203.0.113.%d", i+1))
		request.Header.Set("Forwarded", "for=203.0.113.77")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusTooManyRequests {
			t.Fatal("spoofed proxy header bypassed IP budget")
		}
	}
}

type deniedLogins struct {
	fixtureUser
	budget *attackBudget
	calls  int
}

func (f *deniedLogins) AllowLogin(ctx context.Context, key string) error {
	ok, err := f.budget.Reserve(ctx, key, 10)
	if err != nil {
		return err
	}
	if !ok {
		return apperrors.Wrap(apperrors.CodeRateLimited, nil)
	}
	return nil
}

func (f *deniedLogins) Login(context.Context, string, string) (models.SystemUser, error) {
	f.calls++
	return models.SystemUser{}, apperrors.Wrap(apperrors.CodeUnauthorized, nil)
}

func TestSecurityBackofficeBruteForceDoesNotTrustForwardedIP(t *testing.T) {
	controllers.ConfigureErrors()
	sessions := scs.New()
	permissions, err := security.NewPermissions()
	if err != nil {
		t.Fatal(err)
	}
	users := &deniedLogins{fixtureUser: fixtureUser{permissions: permissions}, budget: newAttackBudget()}
	router := chi.NewRouter()
	router.Use(AdminMiddleware(sessions, []string{"https://panel.example"}))
	api := humachi.New(router, huma.DefaultConfig("assessment", "1"))
	controllers.NewSessionController(users, controllers.NewSessionGuard(users, sessions)).Register(api)
	for i := 0; i < 13; i++ {
		body, _ := json.Marshal(map[string]string{"email": "admin@example.test", "password": fmt.Sprintf("guess-%d", i)})
		request := httptest.NewRequest(http.MethodPost, "https://api.example/admin/auth/login", strings.NewReader(string(body)))
		request.RemoteAddr = fmt.Sprintf("192.0.2.5:%d", 1000+i)
		request.Header.Set("Origin", "https://panel.example")
		request.Header.Set("Content-Type", "application/json")
		request.Header.Set("X-Forwarded-For", fmt.Sprintf("203.0.113.%d", i+1))
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		want := http.StatusUnauthorized
		if i >= 10 {
			want = http.StatusTooManyRequests
		}
		if response.Code != want {
			t.Fatalf("login %d got %d want %d", i, response.Code, want)
		}
		if len(response.Result().Cookies()) > 0 {
			t.Fatal("failed login created a session")
		}
	}
	if users.calls != 10 {
		t.Fatal("throttled login still reached password verification")
	}
}

func TestSecurityAnonymousBackofficeMutationAndHostileOrigin(t *testing.T) {
	users, err := services.NewUserService(nil)
	if err != nil {
		t.Fatal(err)
	}
	router := New([]string{"https://panel.example"}, false, func(context.Context) error { return nil }, Dependencies{Users: users, Sessions: scs.New(), Companies: services.NewCompanyService(nil), Credentials: services.NewCredentialService(nil)})
	for _, tc := range []struct {
		method, path, origin string
		want                 int
	}{
		{http.MethodGet, "/admin/companies", "", http.StatusUnauthorized},
		{http.MethodPut, "/admin/companies/1", "https://panel.example", http.StatusUnauthorized},
		{http.MethodDelete, "/admin/credentials/2", "https://panel.example", http.StatusUnauthorized},
		{http.MethodPost, "/admin/auth/login", "https://evil.example", http.StatusForbidden},
		{http.MethodPost, "/admin/auth/logout", "https://evil.example", http.StatusForbidden},
	} {
		request := httptest.NewRequest(tc.method, "https://api.example"+tc.path, strings.NewReader(`{"name":"' OR 1=1 --"}`))
		request.Header.Set("Content-Type", "application/json")
		if tc.origin != "" {
			request.Header.Set("Origin", tc.origin)
			request.Header.Set("Sec-Fetch-Site", "cross-site")
		}
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != tc.want {
			t.Fatalf("%s %s got %d want %d", tc.method, tc.path, response.Code, tc.want)
		}
		if response.Header().Get("X-Frame-Options") != "DENY" || response.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("security headers absent on rejection")
		}

		if response.Header().Get("Access-Control-Allow-Origin") == "https://evil.example" {
			t.Fatal("hostile origin allowed")
		}
	}
	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	for key, want := range map[string]string{"X-Content-Type-Options": "nosniff", "X-Frame-Options": "DENY", "Referrer-Policy": "no-referrer", "Cache-Control": "no-store"} {
		if response.Header().Get(key) != want {
			t.Fatal("missing security header", key)
		}
	}
}

func TestSecurityAPIEnumerationAndOversizedBody(t *testing.T) {
	hash, err := security.HashPassword("existing-legacy-password")
	if err != nil {
		t.Fatal(err)
	}
	auth, err := services.NewAuthServiceWithProtection(testCredentials{value: models.APICredential{ID: 7, Username: "client", PasswordHash: hash, Active: true}}, newAttackBudget(), security.APIAuthenticationIPLimit)
	if err != nil {
		t.Fatal(err)
	}
	attempts := &testAttempts{}
	router := New(nil, false, func(context.Context) error { return nil }, Dependencies{Auth: auth, SMS: services.NewSMSService(attempts, nil)})
	previous := ""
	for _, username := range []string{"client", "unknown", "' OR '1'='1' --"} {
		request := httptest.NewRequest(http.MethodPost, "/v1/send", strings.NewReader(`{"sms":[]}`))
		request.Header.Set("Content-Type", "application/json")
		request.SetBasicAuth(username, "incorrect-password")
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusUnauthorized || response.Header().Get("WWW-Authenticate") == "" {
			t.Fatal("authentication not rejected uniformly")
		}
		if previous != "" && previous != response.Body.String() {
			t.Fatal("account existence exposed through response")
		}
		previous = response.Body.String()
	}
	body := `{"sms":[{"messageText":"` + strings.Repeat("x", (1<<20)+1) + `","destination":"11987654321"}]}`
	request := httptest.NewRequest(http.MethodPost, "/v1/send", strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.SetBasicAuth("client", "existing-legacy-password")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatal("oversized body not rejected", response.Code)
	}
	if len(attempts.created) != 0 {
		t.Fatal("invalid requests reached message persistence")
	}
}
