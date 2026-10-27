package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/21mobile/sms-gateway/internal/services"
	"github.com/alexedwards/scs/v2"
)

func TestPublicOpenAPIOnlyShowsClientIntegration(t *testing.T) {
	users, err := services.NewUserService(nil)
	if err != nil {
		t.Fatal(err)
	}
	auth, err := services.NewAuthService(nil)
	if err != nil {
		t.Fatal(err)
	}
	handler := New([]string{"http://localhost:5173"}, true, func(context.Context) error { return nil }, Dependencies{
		Users: users, Sessions: scs.New(), Companies: services.NewCompanyService(nil), Credentials: services.NewCredentialService(nil),
		Messages: services.NewMessageService(nil), Audit: services.NewAuditService(nil), Summary: services.NewSummaryService(nil),
		Callbacks: services.NewCallbackService(nil), CallbackIngressEnabled: true, CallbackIngressToken: "test-token-only-for-schema-registration",
		SMS: services.NewSMSService(nil, nil), Auth: auth, LegacyStatuses: services.NewLegacyStatusService(nil),
		LegacyMO: services.NewLegacyMOService(nil, nil),
	})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/openapi.json", nil))
	if w.Code != http.StatusOK {
		t.Fatal("OpenAPI unavailable", w.Code)
	}
	var document struct {
		Paths      map[string]json.RawMessage `json:"paths"`
		Components struct {
			SecuritySchemes map[string]json.RawMessage `json:"securitySchemes"`
		} `json:"components"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	expected := map[string]bool{"/v1/send": true, "/v1/status": true, "/v1/consultmo": true, "/v1/auth/check": true}
	if len(document.Paths) != len(expected) {
		t.Fatalf("public paths: %v", document.Paths)
	}
	for path := range document.Paths {
		if !expected[path] {
			t.Fatalf("non-integration path in public OpenAPI: %s", path)
		}
	}
	if len(document.Components.SecuritySchemes) != 1 || document.Components.SecuritySchemes["legacyBasicAuth"] == nil {
		t.Fatalf("unexpected public security schemes: %v", document.Components.SecuritySchemes)
	}
	for _, path := range []string{"/admin/audit", "/internal/dlr", "/healthz"} {
		if _, ok := document.Paths[path]; ok {
			t.Fatalf("private path in public OpenAPI: %s", path)
		}
	}
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/admin/audit", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("admin route changed by documentation filtering: %d", w.Code)
	}
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/schemas/SessionOutput", nil))
	if w.Code != http.StatusNotFound {
		t.Fatalf("schema route exposed: %d", w.Code)
	}
}
