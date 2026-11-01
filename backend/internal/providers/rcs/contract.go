package rcs

import "context"

// TextRequest is deliberately independent of the legacy SMS send contract.
// MessageID must be persisted before dispatch and reused when reconciling an unknown outcome.
type TextRequest struct {
	AgentID   string
	Phone     string // E.164, including the leading +.
	MessageID string // Caller-assigned UUID.
	Text      string
}

type TextResponse struct {
	MessageID string
}

type Sender interface {
	SendText(context.Context, TextRequest) (TextResponse, error)
}
