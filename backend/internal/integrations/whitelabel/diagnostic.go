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
	id, body, contentType, auth string
}

var diagnosticCases = []diagnosticCase{
	{"missing_token", `{"type":"SMS","message":"Diagnostico","destination_addr":"invalid"}`, "application/json", "missing"},
	{"invalid_token", `{"type":"SMS","message":"Diagnostico","destination_addr":"invalid"}`, "application/json", "invalid"},
	{"malformed_json", `{`, "application/json", "configured"},
	{"empty_object", `{}`, "application/json", "configured"},
	{"empty_array", `[]`, "application/json", "configured"},
	{"wrong_content_type", `{"type":"SMS","message":"Diagnostico","destination_addr":"invalid"}`, "text/plain", "configured"},
	{"missing_destination", `{"type":"SMS","message":"Diagnostico","encoding":"0"}`, "application/json", "configured"},
	{"invalid_destination", `{"type":"SMS","message":"Diagnostico","encoding":"0","destination_addr":"invalid"}`, "application/json", "configured"},
	{"empty_message", `{"type":"SMS","message":"","encoding":"0","destination_addr":"invalid"}`, "application/json", "configured"},
	{"invalid_type_encoding", `{"type":"INVALID","message":"Diagnostico","encoding":"INVALID","destination_addr":"invalid"}`, "application/json", "configured"},
}

type DiagnosticResult struct {
	CaseID       string `json:"caseId"`
	RequestBody  string `json:"requestBody"`
	ContentType  string `json:"contentType"`
	Auth         string `json:"auth"`
	HTTPStatus   int    `json:"httpStatus"`
	ResponseBody string `json:"responseBody"`
	ErrorCode    string `json:"errorCode,omitempty"`
}

// Diagnose runs exactly ten bounded, non-deliverable requests. Bodies and
// responses are logged for this opt-in action; token values are redacted.
func (c *Client) Diagnose(ctx context.Context, runID string, companyID, actorID uint64) []DiagnosticResult {
	results := make([]DiagnosticResult, 0, len(diagnosticCases))
	slog.Info("Zoug diagnostic started", "run_id", runID, "company_id", companyID, "actor_id", actorID, "cases", len(diagnosticCases))
	for _, probe := range diagnosticCases {
		result := DiagnosticResult{CaseID: probe.id, RequestBody: probe.body, ContentType: probe.contentType, Auth: probe.auth}
		caseCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		request, err := http.NewRequestWithContext(caseCtx, http.MethodPost, c.endpoint, strings.NewReader(probe.body))
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
		slog.Info("Zoug diagnostic case", "run_id", runID, "company_id", companyID, "case_id", result.CaseID, "request_body", result.RequestBody, "content_type", result.ContentType, "auth", result.Auth, "http_status", result.HTTPStatus, "response_body", result.ResponseBody, "error_code", result.ErrorCode)
		results = append(results, result)
	}
	slog.Info("Zoug diagnostic completed", "run_id", runID, "company_id", companyID, "cases", len(results))
	return results
}

func NewDiagnosticRunID() string { return uuid.NewString() }
