package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	apperrors "github.com/21mobile/sms-gateway/internal/errors"
)

func TestAdminErrorLanguageDoesNotChangeLegacyContract(t *testing.T) {
	base := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/problem+json")
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"code":"UNAUTHORIZED","detail":"Autenticação necessária."}`))
	})
	for _, tc := range []struct {
		path, language, expected string
	}{
		{"/admin/users", "en-US", apperrors.CodeUnauthorized.MessageFor("en-US")},
		{"/admin/users", "pt-BR", apperrors.CodeUnauthorized.Message()},
		{"/v1/sms", "en-US", apperrors.CodeUnauthorized.Message()},
	} {
		r := httptest.NewRequest(http.MethodGet, tc.path, nil)
		r.Header.Set("Accept-Language", tc.language)
		w := httptest.NewRecorder()
		localizeAdminErrors(base).ServeHTTP(w, r)
		var body struct {
			Detail string `json:"detail"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || body.Detail != tc.expected {
			t.Fatalf("path=%s lang=%s body=%s err=%v", tc.path, tc.language, w.Body.String(), err)
		}
	}
}
