package httpapi

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAPIResponsesSendHSTS(t *testing.T) {
	handler := New(nil, false, func(context.Context) error { return nil }, Dependencies{})
	for _, path := range []string{"/healthz", "/unknown"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if got := response.Header().Get("Strict-Transport-Security"); got != "max-age=31536000" {
			t.Fatalf("%s: HSTS=%q", path, got)
		}
	}
}
