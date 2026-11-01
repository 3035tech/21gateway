package google

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/providers/rcs"
	"github.com/google/uuid"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

const (
	rcsEndpoint           = "https://rcsbusinessmessaging.googleapis.com/v1/phones/"
	rcsScope              = "https://www.googleapis.com/auth/rcsbusinessmessaging"
	requestTimeout        = 10 * time.Second
	maximumResponseBytes  = 64 << 10
	maximumTextCharacters = 3072
)

var (
	phonePattern      = regexp.MustCompile(`^\+[1-9][0-9]{1,14}$`)
	ErrInvalidRequest = errors.New("invalid RCS request")
	ErrUnauthorized   = errors.New("RCS provider authorization failed")
	ErrUnreachable    = errors.New("RCS recipient unreachable")
	ErrRejected       = errors.New("RCS provider rejected the request")
	ErrUnknownOutcome = errors.New("RCS provider outcome unknown")
)

type Client struct {
	tokens   oauth2.TokenSource
	http     *http.Client
	endpoint string
}

type textMessage struct {
	ContentMessage struct {
		Text string `json:"text"`
	} `json:"contentMessage"`
}

// NewFromServiceAccountJSON prepares Google OAuth without registering a route or sending traffic.
func NewFromServiceAccountJSON(ctx context.Context, credentials []byte) (*Client, error) {
	config, err := google.JWTConfigFromJSON(credentials, rcsScope)
	if err != nil {
		return nil, ErrInvalidRequest
	}
	return NewWithTokenSource(config.TokenSource(ctx), http.DefaultTransport), nil
}

// NewWithTokenSource allows deterministic protocol tests and alternative secret sources.
func NewWithTokenSource(tokens oauth2.TokenSource, transport http.RoundTripper) *Client {
	if transport == nil {
		transport = http.DefaultTransport
	}
	if tokens != nil {
		tokens = oauth2.ReuseTokenSource(nil, tokens)
	}
	return &Client{
		tokens:   tokens,
		http:     &http.Client{Transport: transport, Timeout: requestTimeout, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
		endpoint: rcsEndpoint,
	}
}

func (*Client) Code() models.Provider { return models.ProviderGoogle }

func (c *Client) SendText(ctx context.Context, input rcs.TextRequest) (rcs.TextResponse, error) {
	if c == nil || c.tokens == nil || !validRequest(input) {
		return rcs.TextResponse{}, ErrInvalidRequest
	}
	token, err := c.tokens.Token()
	if err != nil || token == nil || token.AccessToken == "" {
		return rcs.TextResponse{}, ErrUnauthorized
	}
	params := url.Values{"agentId": {input.AgentID}, "messageId": {input.MessageID}}
	address := c.endpoint + url.PathEscape(input.Phone) + "/agentMessages?" + params.Encode()
	message := textMessage{}
	message.ContentMessage.Text = input.Text
	body, err := json.Marshal(message)
	if err != nil {
		return rcs.TextResponse{}, ErrInvalidRequest
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, address, bytes.NewReader(body))
	if err != nil {
		return rcs.TextResponse{}, ErrInvalidRequest
	}
	request.Header.Set("Authorization", "Bearer "+token.AccessToken)
	request.Header.Set("Content-Type", "application/json")
	response, err := c.http.Do(request)
	if err != nil {
		return rcs.TextResponse{}, ErrUnknownOutcome
	}
	defer response.Body.Close()
	switch response.StatusCode {
	case http.StatusOK:
		var acknowledged struct {
			Name string `json:"name"`
		}
		data, err := io.ReadAll(io.LimitReader(response.Body, maximumResponseBytes+1))
		if err != nil || len(data) > maximumResponseBytes || json.Unmarshal(data, &acknowledged) != nil || acknowledged.Name != "phones/"+input.Phone+"/agentMessages/"+input.MessageID {
			return rcs.TextResponse{}, ErrUnknownOutcome
		}
		return rcs.TextResponse{MessageID: input.MessageID}, nil
	case http.StatusUnauthorized, http.StatusForbidden:
		return rcs.TextResponse{}, ErrUnauthorized
	case http.StatusNotFound:
		return rcs.TextResponse{}, ErrUnreachable
	case http.StatusBadRequest, http.StatusUnprocessableEntity:
		return rcs.TextResponse{}, ErrRejected
	default:
		return rcs.TextResponse{}, ErrUnknownOutcome
	}
}

func validRequest(input rcs.TextRequest) bool {
	if !phonePattern.MatchString(input.Phone) || input.AgentID == "" || strings.TrimSpace(input.AgentID) != input.AgentID || !utf8.ValidString(input.AgentID) {
		return false
	}
	for _, character := range input.AgentID {
		if unicode.IsControl(character) {
			return false
		}
	}
	id, err := uuid.Parse(input.MessageID)
	return err == nil && id != uuid.Nil && id.String() == input.MessageID && input.Text != "" && utf8.ValidString(input.Text) && utf8.RuneCountInString(input.Text) <= maximumTextCharacters
}

var _ rcs.Sender = (*Client)(nil)
