package whitelabel

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// These fixed requests have no valid destination. Never accept a body or
// destination from the administrative HTTP request for this diagnostic.
type diagnosticCase struct {
	id, body, contentType, auth, method, path string
}

var diagnosticCases = []diagnosticCase{
	{"missing_token", `{"type":"SMS","message":"Diagnostico","destination_addr":"invalid"}`, "application/json", "missing", "", ""},
	{"invalid_token", `{"type":"SMS","message":"Diagnostico","destination_addr":"invalid"}`, "application/json", "invalid", "", ""},
	{"malformed_json", `{`, "application/json", "configured", "", ""},
	{"empty_object", `{}`, "application/json", "configured", "", ""},
	{"empty_array", `[]`, "application/json", "configured", "", ""},
	{"wrong_content_type", `{"type":"SMS","message":"Diagnostico","destination_addr":"invalid"}`, "text/plain", "configured", "", ""},
	{"missing_destination", `{"type":"SMS","message":"Diagnostico","encoding":"0"}`, "application/json", "configured", "", ""},
	{"invalid_destination", `{"type":"SMS","message":"Diagnostico","encoding":"0","destination_addr":"invalid"}`, "application/json", "configured", "", ""},
	{"empty_message", `{"type":"SMS","message":"","encoding":"0","destination_addr":"invalid"}`, "application/json", "configured", "", ""},
	{"invalid_type_encoding", `{"type":"INVALID","message":"Diagnostico","encoding":"INVALID","destination_addr":"invalid"}`, "application/json", "configured", "", ""},
	{"null_body", `null`, "application/json", "configured", "", ""},
	{"batch_empty_object", `{}`, "application/json", "configured", "POST", "batch"},
	{"batch_empty_messages", `{"type":"SMS","messages":[]}`, "application/json", "configured", "POST", "batch"},
	{"batch_invalid_message", `{"type":"SMS","messages":[{"message":"","destination_addr":"invalid"}]}`, "application/json", "configured", "POST", "batch"},
	{"unknown_route", `{}`, "application/json", "configured", "POST", "unknown"},
	{"wrong_method", ``, "application/json", "configured", "GET", ""},
}

type DiagnosticResult struct {
	CaseID          string              `json:"caseId"`
	RequestMethod   string              `json:"requestMethod"`
	RequestPath     string              `json:"requestPath"`
	RequestBody     string              `json:"requestBody"`
	ContentType     string              `json:"contentType"`
	Auth            string              `json:"auth"`
	HTTPStatus      int                 `json:"httpStatus"`
	ResponseBody    string              `json:"responseBody"`
	ResponseHeaders map[string][]string `json:"responseHeaders,omitempty"`
	DurationMS      int64               `json:"durationMs"`
	ErrorCode       string              `json:"errorCode,omitempty"`
}

// Diagnose runs fixed, bounded requests. The optional allowlisted destination
// appears only in cases with multiple invalid required fields. Bodies and
// responses are logged for this opt-in action; token values are redacted.
func (c *Client) Diagnose(ctx context.Context, runID string, companyID, actorID uint64, destination string) []DiagnosticResult {
	probes := append([]diagnosticCase(nil), diagnosticCases[:11]...)
	if destination != "" {
		address := "55" + destination
		probes = append(probes,
			diagnosticCase{id: "valid_destination_invalid_type", body: `{"type":"INVALID","message":"","encoding":"INVALID","destination_addr":"` + address + `"}`, contentType: "application/json", auth: "configured"},
			diagnosticCase{id: "valid_destination_null_message", body: `{"type":"INVALID","message":null,"encoding":"INVALID","destination_addr":"` + address + `"}`, contentType: "application/json", auth: "configured"},
		)
	}
	probes = append(probes, diagnosticCases[11:]...)
	results := make([]DiagnosticResult, 0, len(probes))
	slog.Info("Zoug diagnostic started", "run_id", runID, "company_id", companyID, "actor_id", actorID, "cases", len(probes))
	for _, probe := range probes {
		started := time.Now()
		result := DiagnosticResult{CaseID: probe.id, RequestBody: probe.body, ContentType: probe.contentType, Auth: probe.auth}
		caseCtx, cancel := context.WithTimeout(ctx, time.Second)
		method, endpoint := probe.method, c.endpoint
		if method == "" {
			method = http.MethodPost
		}
		if probe.path != "" {
			endpoint += "/" + probe.path
		}
		result.RequestMethod = method
		result.RequestPath = "/api/send"
		if probe.path != "" {
			result.RequestPath += "/" + probe.path
		}
		request, err := http.NewRequestWithContext(caseCtx, method, endpoint, strings.NewReader(probe.body))
		if err == nil {
			request.Header.Set("Content-Type", probe.contentType)
			switch probe.auth {
			case "configured":
				request.Header.Set(tokenHeader, c.token)
			case "invalid":
				request.Header.Set(tokenHeader, "invalid-diagnostic-token")
			}
			response, requestErr := c.http.Do(request)
			if requestErr != nil {
				err = requestErr
			} else {
				result.HTTPStatus = response.StatusCode
				result.ResponseHeaders = safeDiagnosticHeaders(response.Header, c.token, destination)
				data, readErr := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
				_ = response.Body.Close()
				if readErr != nil {
					err = readErr
				} else if len(data) > maxResponseBytes {
					result.ErrorCode = "RESPONSE_TOO_LARGE"
				} else {
					result.ResponseBody = strings.ReplaceAll(strings.ToValidUTF8(string(data), "�"), "invalid-diagnostic-token", "[REDACTED]")
					if c.token != "" {
						result.ResponseBody = strings.ReplaceAll(result.ResponseBody, c.token, "[REDACTED]")
					}
					if destination != "" {
						result.ResponseBody = strings.ReplaceAll(result.ResponseBody, "55"+destination, "[TEST_DESTINATION]")
						result.ResponseBody = strings.ReplaceAll(result.ResponseBody, destination, "[TEST_DESTINATION]")
					}
				}
			}
		}
		if err != nil {
			result.ErrorCode = "TRANSPORT_ERROR"
			if caseCtx.Err() != nil {
				result.ErrorCode = "TIMEOUT"
			}
		}
		cancel()
		result.DurationMS = time.Since(started).Milliseconds()
		loggedBody := result.RequestBody
		if destination != "" {
			loggedBody = strings.ReplaceAll(loggedBody, "55"+destination, "[TEST_DESTINATION]")
			loggedBody = strings.ReplaceAll(loggedBody, destination, "[TEST_DESTINATION]")
		}
		slog.Info("Zoug diagnostic case", "run_id", runID, "company_id", companyID, "case_id", result.CaseID, "method", method, "path", result.RequestPath, "request_body", loggedBody, "content_type", result.ContentType, "auth", result.Auth, "duration_ms", result.DurationMS, "http_status", result.HTTPStatus, "response_headers", result.ResponseHeaders, "response_body", result.ResponseBody, "error_code", result.ErrorCode)
		results = append(results, result)
		if result.HTTPStatus >= 200 && result.HTTPStatus < 300 {
			slog.Warn("Zoug diagnostic stopped after unexpected acceptance", "run_id", runID, "company_id", companyID, "case_id", result.CaseID, "http_status", result.HTTPStatus)
			break
		}
	}
	slog.Info("Zoug diagnostic completed", "run_id", runID, "company_id", companyID, "cases", len(results))
	return results
}

func safeDiagnosticHeaders(headers http.Header, token, destination string) map[string][]string {
	safe := make(map[string][]string, len(headers))
	for name, values := range headers {
		lower := strings.ToLower(name)
		if strings.Contains(lower, "token") || strings.Contains(lower, "auth") || strings.Contains(lower, "cookie") || strings.Contains(lower, "key") || strings.Contains(lower, "secret") || strings.Contains(lower, "session") {
			safe[name] = []string{"[REDACTED]"}
			continue
		}
		for _, value := range values {
			if token != "" {
				value = strings.ReplaceAll(value, token, "[REDACTED]")
			}
			if destination != "" {
				value = strings.ReplaceAll(value, "55"+destination, "[TEST_DESTINATION]")
				value = strings.ReplaceAll(value, destination, "[TEST_DESTINATION]")
			}
			safe[name] = append(safe[name], value)
		}
	}
	return safe
}

func NewDiagnosticRunID() string { return uuid.NewString() }
