package whitelabel

import (
	"bytes"
	"context"
	"encoding/json"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/google/uuid"
	"io"
	"net/http"
)

const MaximumBatchMessages = 200
const batchResponseBytes = 256 << 10
const batchSendPath = "api/send/batch"

type BatchMessage struct {
	Message     string `json:"message"`
	Destination string `json:"destination_addr"`
	ReferenceID string `json:"reference_id,omitempty"`
	Index       string `json:"index"`
}
type BatchRequest struct {
	Type     MessageType    `json:"type"`
	Encoding Encoding       `json:"encoding"`
	Messages []BatchMessage `json:"messages"`
}
type BatchResult struct {
	MessageID string
	ErrorCode apperrors.Code
}
type batchAcknowledgement struct {
	Success []struct {
		MessageID string `json:"message_id"`
		Position  *int   `json:"position"`
		Index     string `json:"index"`
	} `json:"success"`
	Errors []struct {
		Position *int   `json:"position"`
		Error    string `json:"error"`
	} `json:"errors"`
	TotalItems *int   `json:"total_items"`
	Error      string `json:"error"`
}

func (c *Client) SendBatch(ctx context.Context, input BatchRequest) ([]BatchResult, error) {
	if !c.Enabled() {
		return nil, apperrors.Wrap(apperrors.CodeServiceUnavailable, nil)
	}
	if len(input.Messages) < 1 || len(input.Messages) > MaximumBatchMessages {
		return nil, apperrors.Wrap(apperrors.CodeInvalidRequest, nil)
	}
	body, err := json.Marshal(input)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.CodeInternal, nil)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/batch", bytes.NewReader(body))
	if err != nil {
		return nil, apperrors.Wrap(apperrors.CodeInternal, nil)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(tokenHeader, c.token)
	response, err := c.http.Do(request)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.CodeUnknownOutcome, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusMultiStatus && response.StatusCode != http.StatusBadRequest {
		return nil, apperrors.Wrap(codeForResponse(response.StatusCode), nil)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, batchResponseBytes+1))
	if err != nil || len(data) > batchResponseBytes {
		return nil, apperrors.Wrap(apperrors.CodeUnknownOutcome, nil)
	}
	var ack batchAcknowledgement
	if err := json.Unmarshal(data, &ack); err != nil {
		return nil, apperrors.Wrap(apperrors.CodeUnknownOutcome, nil)
	}
	if response.StatusCode == http.StatusBadRequest && ack.Error != "" && len(ack.Success) == 0 && len(ack.Errors) == 0 {
		return nil, apperrors.Wrap(apperrors.CodeProviderRejected, nil)
	}
	if ack.TotalItems != nil && *ack.TotalItems != len(input.Messages) {
		return nil, apperrors.Wrap(apperrors.CodeUnknownOutcome, nil)
	}
	results := make([]BatchResult, len(input.Messages))
	seen := make([]bool, len(results))
	for i := range results {
		results[i].ErrorCode = apperrors.CodeUnknownOutcome
	}
	ids := map[string]int{}
	assign := func(position *int, value BatchResult) {
		if position == nil || *position < 0 || *position >= len(results) {
			return
		}
		i := *position
		if seen[i] {
			results[i] = BatchResult{ErrorCode: apperrors.CodeUnknownOutcome}
			return
		}
		seen[i] = true
		results[i] = value
	}
	for _, item := range ack.Success {
		parsed, err := uuid.Parse(item.MessageID)
		value := BatchResult{ErrorCode: apperrors.CodeUnknownOutcome}
		if err == nil && parsed != uuid.Nil && item.Position != nil && *item.Position >= 0 && *item.Position < len(results) && (item.Index == "" || item.Index == input.Messages[*item.Position].Index) {
			id := parsed.String()
			if old, duplicate := ids[id]; duplicate {
				results[old] = BatchResult{ErrorCode: apperrors.CodeUnknownOutcome}
			} else {
				ids[id] = *item.Position
				value = BatchResult{MessageID: id}
			}
		}
		assign(item.Position, value)
	}
	for _, item := range ack.Errors {
		value := BatchResult{ErrorCode: apperrors.CodeUnknownOutcome}
		if item.Error != "" {
			value.ErrorCode = apperrors.CodeProviderRejected
		}
		assign(item.Position, value)
	}
	return results, nil
}
