package callback

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	"github.com/21mobile/sms-gateway/internal/models"
)

func TestCallbackFailureEvidenceDoesNotExposeClientData(t *testing.T) {
	for _, kind := range []string{"dlr", "mo"} {
		for _, tc := range []struct {
			name    string
			status  int
			network bool
			code    string
		}{
			{"unavailable", http.StatusServiceUnavailable, false, models.CallbackHTTPError},
			{"redirect", http.StatusFound, false, models.CallbackHTTPError},
			{"network", 0, true, models.CallbackNetworkError},
			{"success", http.StatusNoContent, false, ""},
		} {
			t.Run(kind+"/"+tc.name, func(t *testing.T) {
				var output bytes.Buffer
				original := slog.Default()
				slog.SetDefault(slog.New(slog.NewJSONHandler(&output, nil)))
				t.Cleanup(func() { slog.SetDefault(original) })
				calls := 0
				client := NewWithTransport(fakeTransport(func(r *http.Request) (*http.Response, error) {
					calls++
					if tc.network {
						return nil, errors.New("transport-private-url-and-token")
					}
					return &http.Response{StatusCode: tc.status, Header: http.Header{"Private": []string{"private-header"}}, Body: io.NopCloser(strings.NewReader("private-response-body"))}, nil
				}))
				target := "https://private-host.example/callback?token=private-query-token"
				var ok bool
				var code string
				wantEvent := "19"
				wantMethod := http.MethodGet
				if kind == "dlr" {
					ok, code = client.Deliver(context.Background(), models.CallbackJob{URL: target, EventID: 19, Status: models.DeliveryDelivered}, models.SMSAttempt{ID: 42, CorrelationID: "private-correlation"})
				} else {
					m := moFixture()
					m.Message = "private-message"
					m.CorrelationID = "private-correlation"
					ok, code = client.DeliverMO(context.Background(), target, m)
					wantEvent = "mo:1370"
					wantMethod = http.MethodPost
				}
				if code != tc.code || ok != (tc.code == "") || calls != 1 {
					t.Fatal("logging changed delivery/retry behavior", ok, code, calls)
				}
				if tc.code == "" {
					if output.Len() != 0 {
						t.Fatal("routine success produced log volume")
					}
					return
				}
				var record map[string]any
				if err := json.Unmarshal(output.Bytes(), &record); err != nil {
					t.Fatal(err)
				}
				if record["msg"] != "Callback HTTP attempt failed" || record["event_id"] != wantEvent || record["method"] != wantMethod || record["http_status"] != float64(tc.status) || record["failure_code"] != tc.code || record["level"] != "WARN" {
					t.Fatal("missing safe delivery evidence", record)
				}
				if duration, ok := record["duration_ms"].(float64); !ok || duration < 0 {
					t.Fatal("missing duration")
				}
				for _, secret := range []string{"private-host", "private-query-token", "private-correlation", "private-message", "5511998877665", "private-response-body", "private-header", "transport-private"} {
					if strings.Contains(output.String(), secret) {
						t.Fatal("client data leaked to logs")
					}
				}
			})
		}
	}
}
