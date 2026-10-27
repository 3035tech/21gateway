package httpapi

import (
	"context"
	"encoding/json"
	"github.com/21mobile/sms-gateway/internal/services"
	"github.com/alexedwards/scs/v2"
	"net/http/httptest"
	"testing"
)

func TestAllAdministrativeRoutesRegister(t *testing.T) {
	users, err := services.NewUserService(nil)
	if err != nil {
		t.Fatal(err)
	}
	handler := New([]string{"http://localhost:5173"}, true, func(context.Context) error { return nil }, Dependencies{Users: users, Sessions: scs.New(), Companies: services.NewCompanyService(nil), Credentials: services.NewCredentialService(nil), Messages: services.NewMessageService(nil), Audit: services.NewAuditService(nil), Summary: services.NewSummaryService(nil), Callbacks: services.NewCallbackService(nil), CallbackIngressEnabled: true, CallbackIngressToken: "test-token-only-for-schema-registration"})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "http://localhost:8080/openapi.json", nil))
	if w.Code != 200 {
		t.Fatal("OpenAPI unavailable", w.Code)
	}
	var document struct {
		Paths map[string]map[string]struct {
			Parameters []struct {
				Name string `json:"name"`
			} `json:"parameters"`
		} `json:"paths"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	parameters := map[string]bool{}
	for _, parameter := range document.Paths["/admin/audit"]["get"].Parameters {
		parameters[parameter.Name] = true
	}
	for _, name := range []string{"actorId", "action", "resource", "page", "pageSize", "search", "sort", "direction"} {
		if !parameters[name] {
			t.Fatalf("audit parameter %s missing from OpenAPI", name)
		}
	}
	for _, name := range []string{"active", "companyId", "state"} {
		if parameters[name] {
			t.Fatalf("unused audit parameter %s exposed in OpenAPI", name)
		}
	}
}
