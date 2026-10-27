package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	apperrors "github.com/21mobile/sms-gateway/internal/errors"
)

// localizeAdminErrors translates only administrative problem responses.
// Legacy /v1 response contracts keep their established Portuguese text.
func localizeAdminErrors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/admin/") && r.URL.Path != "/admin" {
			next.ServeHTTP(w, r)
			return
		}
		language := "pt-BR"
		if strings.HasPrefix(strings.ToLower(r.Header.Get("Accept-Language")), "en") {
			language = "en-US"
		}
		w.Header().Set("Content-Language", language)
		w.Header().Add("Vary", "Accept-Language")
		response := &adminErrorWriter{ResponseWriter: w}
		next.ServeHTTP(response, r)
		if response.status < http.StatusBadRequest {
			return
		}
		contents := response.buffer.Bytes()
		var problem map[string]json.RawMessage
		if json.Unmarshal(contents, &problem) == nil {
			var code apperrors.Code
			if json.Unmarshal(problem["code"], &code) == nil && code != apperrors.CodeNone {
				detail, _ := json.Marshal(code.MessageFor(language))
				problem["detail"] = detail
				if translated, err := json.Marshal(problem); err == nil {
					contents = translated
				}
			}
		}
		w.Header().Del("Content-Length")
		w.WriteHeader(response.status)
		_, _ = w.Write(contents)
	})
}

type adminErrorWriter struct {
	http.ResponseWriter
	status int
	buffer bytes.Buffer
}

func (w *adminErrorWriter) WriteHeader(status int) {
	if w.status != 0 {
		return
	}
	w.status = status
	if status < http.StatusBadRequest {
		w.ResponseWriter.WriteHeader(status)
	}
}

func (w *adminErrorWriter) Write(data []byte) (int, error) {
	if w.status == 0 {
		w.WriteHeader(http.StatusOK)
	}
	if w.status >= http.StatusBadRequest {
		return w.buffer.Write(data)
	}
	return w.ResponseWriter.Write(data)
}
