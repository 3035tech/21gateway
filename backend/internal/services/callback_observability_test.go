package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"testing"

	"github.com/21mobile/sms-gateway/internal/models"
)

func TestCallbackCompletionEvidenceDistinguishesDeliveryAndPersistence(t *testing.T) {
	for _, kind := range []models.CallbackKind{models.CallbackKindDLR, models.CallbackKindMO} {
		for _, tc := range []struct {
			name      string
			attempts  int
			delivered bool
			code      string
			err       error
			level     string
		}{
			{"first success", 1, true, "", nil, ""},
			{"recovered", 2, true, "", nil, "INFO"},
			{"http failure", 1, false, models.CallbackHTTPError, nil, "WARN"},
			{"blocked", 1, false, models.CallbackAccessBlocked, nil, "WARN"},
			{"unavailable content", 1, false, models.CallbackContentUnavailable, nil, "WARN"},
			{"accepted but not committed", 2, true, "", errors.New("private database DSN"), "ERROR"},
			{"failure not committed", 1, false, models.CallbackHTTPError, errors.New("private database DSN"), "ERROR"},
		} {
			t.Run(string(kind)+"/"+tc.name, func(t *testing.T) {
				var output bytes.Buffer
				original := slog.Default()
				slog.SetDefault(slog.New(slog.NewJSONHandler(&output, nil)))
				t.Cleanup(func() { slog.SetDefault(original) })
				recordCallbackCompletion(context.Background(), kind, 3, 9, tc.attempts, tc.delivered, tc.code, tc.err)
				if tc.level == "" {
					if output.Len() != 0 {
						t.Fatal("routine success logged")
					}
					return
				}
				var record map[string]any
				if err := json.Unmarshal(output.Bytes(), &record); err != nil {
					t.Fatal(err)
				}
				if record["level"] != tc.level || record["callback_kind"] != string(kind) || record["job_id"] != float64(3) || record["event_id"] != float64(9) || record["attempts"] != float64(tc.attempts) || record["delivered"] != tc.delivered || record["outcome_persisted"] != (tc.err == nil) || record["failure_code"] != tc.code {
					t.Fatal("incorrect completion evidence", record)
				}
				if strings.Contains(output.String(), "private") {
					t.Fatal("storage cause leaked")
				}
				expectedMessage := "Callback attempt failed"
				if tc.err != nil {
					expectedMessage = "Callback outcome persistence failed"
				} else if tc.delivered {
					expectedMessage = "Callback delivery recovered"
				}
				if record["msg"] != expectedMessage {
					t.Fatal("ambiguous completion message", record["msg"])
				}
			})
		}
	}
}
