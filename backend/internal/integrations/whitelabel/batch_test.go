package whitelabel

import (
	"context"
	"encoding/json"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/jarcoal/httpmock"
	"net/http"
	"testing"
)

func TestBatchResultsUsePositionAndPreserveUnknowns(t *testing.T) {
	transport := httpmock.NewMockTransport()
	transport.RegisterResponder("POST", "https://provider.example/api/send/batch", func(r *http.Request) (*http.Response, error) {
		if r.Header.Get(tokenHeader) != "account-token" {
			t.Fatal("wrong account token")
		}
		var body BatchRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || len(body.Messages) != 3 {
			t.Fatal("wrong batch body")
		}
		return httpmock.NewJsonResponse(207, map[string]any{"total_items": 3, "success": []map[string]any{{"position": 2, "message_id": "550e8400-e29b-41d4-a716-446655440002", "index": "3"}}, "errors": []map[string]any{{"position": 0, "error": "destination_addr is required"}}})
	})
	client, err := NewWithTransport("https://provider.example", "unused-shared-token", transport)
	if err != nil {
		t.Fatal(err)
	}
	outcomes, err := client.ForToken("account-token").SendBatch(context.Background(), BatchRequest{Type: MessageTypeSMS, Encoding: EncodingGSM7, Messages: []BatchMessage{{Index: "1"}, {Index: "2"}, {Index: "3"}}})
	if err != nil || len(outcomes) != 3 || outcomes[0].ErrorCode != apperrors.CodeProviderRejected || outcomes[1].ErrorCode != apperrors.CodeUnknownOutcome || outcomes[2].MessageID == "" {
		t.Fatal("position mapping or unknown outcome lost")
	}
	if transport.GetTotalCallCount() != 1 {
		t.Fatal("batch retried")
	}
}
func TestDuplicateBatchUUIDDoesNotCreateTwoAcceptances(t *testing.T) {
	transport := httpmock.NewMockTransport()
	transport.RegisterResponder("POST", "https://provider.example/api/send/batch", httpmock.NewStringResponder(200, `{"success":[{"position":0,"message_id":"550e8400-e29b-41d4-a716-446655440001"},{"position":1,"message_id":"550e8400-e29b-41d4-a716-446655440001"}],"errors":[],"total_items":2}`))
	client, _ := NewWithTransport("https://provider.example", "token", transport)
	outcomes, err := client.SendBatch(context.Background(), BatchRequest{Messages: make([]BatchMessage, 2)})
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range outcomes {
		if v.MessageID != "" || v.ErrorCode != apperrors.CodeUnknownOutcome {
			t.Fatal("duplicate provider UUID accepted")
		}
	}
}
