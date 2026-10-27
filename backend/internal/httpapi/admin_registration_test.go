package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
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
		Info struct {
			Description string `json:"description"`
		} `json:"info"`
		Paths      map[string]json.RawMessage `json:"paths"`
		Components struct {
			SecuritySchemes map[string]json.RawMessage `json:"securitySchemes"`
			Schemas         map[string]json.RawMessage `json:"schemas"`
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
	if len(document.Components.SecuritySchemes) != 1 || document.Components.SecuritySchemes["basicAuth"] == nil {
		t.Fatalf("unexpected public security schemes: %v", document.Components.SecuritySchemes)
	}
	if !strings.Contains(string(document.Components.SecuritySchemes["basicAuth"]), "gerados e fornecidos pela 21 Mobile") {
		t.Fatal("Basic Auth guidance missing from security scheme")
	}
	for name := range document.Components.Schemas {
		if strings.Contains(name, "Legacy") {
			t.Fatalf("internal schema name leaked into client documentation: %s", name)
		}
	}
	for _, name := range []string{"SMSSendRequest", "SMSRequestMessage", "SMSSendResult", "SMSDeliveryStatusResponse", "MORepliesResponse"} {
		if document.Components.Schemas[name] == nil {
			t.Fatalf("client-facing schema name missing: %s", name)
		}
	}
	if document.Components.Schemas["CompanyView"] != nil {
		t.Fatal("administrative company schema leaked into client documentation")
	}
	if strings.Contains(w.Body.String(), "Legacy") {
		t.Fatal("internal compatibility name leaked into public OpenAPI")
	}
	if !strings.Contains(document.Info.Description, "21 Mobile gera e entrega") {
		t.Fatal("credential issuance guidance missing from public introduction")
	}
	var send struct {
		Post struct {
			Description string `json:"description"`
			RequestBody struct {
				Content map[string]struct {
					Examples map[string]json.RawMessage `json:"examples"`
				} `json:"content"`
			} `json:"requestBody"`
			Responses map[string]struct {
				Content map[string]struct {
					Examples map[string]json.RawMessage `json:"examples"`
				} `json:"content"`
			} `json:"responses"`
		} `json:"post"`
	}
	if err := json.Unmarshal(document.Paths["/v1/send"], &send); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(send.Post.Description, "não evita duplicações") || len(send.Post.RequestBody.Content["application/json"].Examples) == 0 || len(send.Post.Responses["200"].Content["application/json"].Examples) < 2 {
		t.Fatal("send operation must explain behavior and show request/result examples")
	}
	for _, constraint := range []string{`"minLength":11`, `"maxLength":11`, `"maximum":253402300799`} {
		if !strings.Contains(w.Body.String(), constraint) {
			t.Fatalf("public OpenAPI missing DTO constraint %s", constraint)
		}
	}
	for _, path := range []string{"/v1/status", "/v1/consultmo"} {
		var operation struct {
			Get struct {
				Description string `json:"description"`
				Responses   map[string]struct {
					Content map[string]struct {
						Examples map[string]json.RawMessage `json:"examples"`
					} `json:"content"`
				} `json:"responses"`
			} `json:"get"`
		}
		if err := json.Unmarshal(document.Paths[path], &operation); err != nil {
			t.Fatal(err)
		}
		if len(operation.Get.Description) < 200 || len(operation.Get.Responses["200"].Content["application/json"].Examples) < 2 {
			t.Fatalf("query operation lacks integration guidance or examples: %s", path)
		}
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
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/docs", nil))
	if w.Code != http.StatusOK || !strings.Contains(strings.ToLower(w.Body.String()), "scalar") {
		t.Fatalf("Scalar documentation unavailable: %d", w.Code)
	}
}
