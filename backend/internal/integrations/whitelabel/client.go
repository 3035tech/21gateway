package whitelabel

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/warthog618/sms/encoding/gsm7"

	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/models"
)

type Encoding = models.SMSEncoding
type MessageType string
type CallbackLevel string
type SendStatus string

const (
	EncodingGSM7          Encoding      = models.SMSEncodingGSM7
	EncodingUCS2          Encoding      = models.SMSEncodingUCS2
	MessageTypeSMS        MessageType   = "SMS"
	CallbackNone          CallbackLevel = "NONE"
	StatusSent            SendStatus    = "sent"
	tokenHeader                         = "X-API-TOKEN"
	sendPath                            = "api/send"
	requestTimeout                      = 10 * time.Second
	maxResponseBytes                    = 64 << 10
	maxConnectionsPerHost               = 100
	idleConnectionTimeout               = 90 * time.Second
)

type SendRequest struct {
	Type          MessageType   `json:"type"`
	Message       string        `json:"message"`
	Encoding      Encoding      `json:"encoding"`
	Destination   string        `json:"destination_addr"`
	ReferenceID   string        `json:"reference_id,omitempty"`
	Index         string        `json:"index"`
	CallbackLevel CallbackLevel `json:"callback_level"`
}

type SendResponse struct {
	MessageID string     `json:"message_id"`
	Status    SendStatus `json:"status"`
}

type Client struct {
	endpoint                  string
	token                     string
	http                      *http.Client
	logUnexpectedResponseBody bool
}

func New(baseURL, token string) (*Client, error) {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConns = maxConnectionsPerHost
	transport.MaxIdleConnsPerHost = maxConnectionsPerHost
	transport.MaxConnsPerHost = maxConnectionsPerHost
	transport.IdleConnTimeout = idleConnectionTimeout
	return NewWithTransport(baseURL, token, transport)
}

// NewWithTransport permits deterministic HTTP tests without opening sockets.
// Production callers use New, which supplies a pooled TLS-capable transport.
func NewWithTransport(baseURL, token string, transport http.RoundTripper) (*Client, error) {
	endpoint, err := url.JoinPath(baseURL, sendPath)
	if err != nil {
		return nil, apperrors.Wrap(apperrors.CodeConfigurationInvalid, err)
	}
	if transport == nil {
		return nil, apperrors.Wrap(apperrors.CodeConfigurationInvalid, nil)
	}
	return &Client{endpoint: endpoint, token: token, http: &http.Client{Transport: transport, Timeout: requestTimeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}, nil
}

func (c *Client) Enabled() bool { return strings.TrimSpace(c.token) != "" }
func (c *Client) Close()        { c.http.CloseIdleConnections() }
func (c *Client) SetLogUnexpectedResponseBody(enabled bool) {
	c.logUnexpectedResponseBody = enabled
}

func SelectEncoding(message string) Encoding {
	if _, err := gsm7.Encode([]byte(message)); err == nil {
		return EncodingGSM7
	}
	return EncodingUCS2
}

func (c *Client) Send(ctx context.Context, input SendRequest) (SendResponse, error) {
	var result SendResponse
	if !c.Enabled() {
		return result, apperrors.Wrap(apperrors.CodeServiceUnavailable, nil)
	}
	body, err := json.Marshal(input)
	if err != nil {
		return result, apperrors.Wrap(apperrors.CodeInternal, err)
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(body))
	if err != nil {
		return result, apperrors.Wrap(apperrors.CodeInternal, err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set(tokenHeader, c.token)
	response, err := c.http.Do(request)
	if err != nil {
		slog.Warn("Zoug SMS response unavailable", "reason", "transport", "provider_index", input.Index, "error", err)
		return result, apperrors.Wrap(apperrors.CodeUnknownOutcome, err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		code := codeForResponse(response.StatusCode)
		slog.Warn("Zoug SMS request not accepted", "provider_index", input.Index, "http_status", response.StatusCode, "code", code)
		return result, apperrors.Wrap(code, nil)
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err != nil || len(data) > maxResponseBytes {
		slog.Warn("Zoug SMS response unavailable", "reason", "response_body_unreadable_or_too_large", "provider_index", input.Index, "error", err)
		return result, apperrors.Wrap(apperrors.CodeUnknownOutcome, err)
	}
	if err := json.Unmarshal(data, &result); err != nil {
		slog.Warn("Zoug SMS response unavailable", "reason", "invalid_json", "provider_index", input.Index, "error", err)
		return result, apperrors.Wrap(apperrors.CodeUnknownOutcome, err)
	}
	parsedID, err := uuid.Parse(result.MessageID)
	if err != nil || parsedID == uuid.Nil || result.Status != StatusSent {
		attributes := []any{"reason", "invalid_acknowledgement", "provider_index", input.Index, "message_id_valid", err == nil && parsedID != uuid.Nil, "status_sent", result.Status == StatusSent}
		if c.logUnexpectedResponseBody {
			attributes = append(attributes, "http_status", response.StatusCode, "response_body", string(data))
		}
		slog.Warn("Zoug SMS response unavailable", attributes...)
		if err == nil && parsedID != uuid.Nil {
			result.MessageID = parsedID.String()
		} else {
			result.MessageID = ""
		}
		return result, apperrors.Wrap(apperrors.CodeUnknownOutcome, errors.New("invalid provider acknowledgement"))
	}
	result.MessageID = parsedID.String()
	return result, nil
}

func codeForResponse(status int) apperrors.Code {
	switch status {
	case http.StatusUnauthorized:
		return apperrors.CodeProviderUnauthorized
	case http.StatusPaymentRequired:
		return apperrors.CodeInsufficientBalance
	case http.StatusTooManyRequests:
		return apperrors.CodeRateLimited
	case http.StatusBadRequest, http.StatusForbidden, http.StatusNotFound, http.StatusUnprocessableEntity:
		return apperrors.CodeProviderRejected
	default:
		return apperrors.CodeUnknownOutcome
	}
}

// ForToken shares transport and connections without mutating another account.
func (c *Client) ForToken(token string) *Client {
	return &Client{endpoint: c.endpoint, token: token, http: c.http, logUnexpectedResponseBody: c.logUnexpectedResponseBody}
}
