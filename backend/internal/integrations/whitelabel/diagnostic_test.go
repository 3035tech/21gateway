package whitelabel

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/jarcoal/httpmock"
)

func TestDiagnosticsNeverUseDeliverableDestinationAndCaptureResponses(t *testing.T) {
	transport := httpmock.NewMockTransport()
	transport.RegisterResponder(http.MethodPost, testBaseURL+"/api/send", func(r *http.Request) (*http.Response, error) {
		body, _ := io.ReadAll(r.Body)
		var payload map[string]any
		if json.Unmarshal(body, &payload) == nil {
			if destination, ok := payload["destination_addr"].(string); ok && destination != "invalid" {
				t.Fatalf("deliverable destination in diagnostic: %q", destination)
			}
		}
		return httpmock.NewStringResponse(http.StatusUnprocessableEntity, `{"status":"INVALID_REQUEST","token":"private-token"}`), nil
	})
	client, _ := NewWithTransport(testBaseURL, "private-token", transport)
	results := client.Diagnose(context.Background(), "run", 1, 2)
	if len(results) != 10 || transport.GetTotalCallCount() != 10 {
		t.Fatalf("results=%d calls=%d", len(results), transport.GetTotalCallCount())
	}
	for _, result := range results {
		if result.HTTPStatus != http.StatusUnprocessableEntity || strings.Contains(result.ResponseBody, "private-token") || !strings.Contains(result.ResponseBody, "INVALID_REQUEST") {
			t.Fatalf("bad diagnostic result: %+v", result)
		}
	}
}
