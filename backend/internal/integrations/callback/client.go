package callback

import (
	"context"
	"errors"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/doyensec/safeurl"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const DeliveryTimeout = 10 * time.Second
const responseLimit = 64 << 10

type HTTPDoer interface {
	Do(*http.Request) (*http.Response, error)
}
type Client struct {
	http        HTTPDoer
	close       func()
	resolveAuth func(context.Context, uint64, models.CallbackKind) (RequestAuth, error)
}

type RequestAuth struct {
	Mode       models.CallbackAuthMode
	HeaderName string
	Username   string
	Secret     string
}

func (c *Client) SetAuthResolver(resolve func(context.Context, uint64, models.CallbackKind) (RequestAuth, error)) {
	c.resolveAuth = resolve
}

func New() *Client {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = nil
	transport.MaxConnsPerHost = 4
	transport.MaxIdleConns = 32
	transport.ResponseHeaderTimeout = DeliveryTimeout
	config := safeurl.GetConfigBuilder().SetAllowedSchemes("https").SetAllowedPorts(443).EnableIPv6(false).SetTimeout(DeliveryTimeout).SetTransport(transport).SetCheckRedirect(func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }).Build()
	client := safeurl.Client(config)
	return &Client{http: client, close: client.CloseIdleConnections}
}
func NewWithTransport(client HTTPDoer) *Client { return &Client{http: client, close: func() {}} }
func (c *Client) Close()                       { c.close() }
func (c *Client) Deliver(ctx context.Context, job models.CallbackJob, attempt models.SMSAttempt) (bool, string) {
	target, err := callbackTarget(job.URL)
	if err != nil {
		return false, models.CallbackNetworkError
	}
	query := target.Query()
	query.Set("idSms", strconv.FormatUint(attempt.ID, 10))
	query.Set("correlationId", attempt.CorrelationID)
	query.Set("status", string(job.Status))
	target.RawQuery = query.Encode()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return false, models.CallbackNetworkError
	}
	request.Header.Set("X-21Mobile-Event-ID", strconv.FormatUint(job.EventID, 10))
	if !c.applyAuth(request, job.CredentialID, models.CallbackKindDLR) {
		return false, models.CallbackNetworkError
	}
	return c.dispatch(request)
}

func (c *Client) applyAuth(request *http.Request, credentialID uint64, kind models.CallbackKind) bool {
	if c.resolveAuth == nil {
		return true
	}
	auth, err := c.resolveAuth(request.Context(), credentialID, kind)
	if err != nil {
		return false
	}
	switch auth.Mode {
	case "", models.CallbackAuthNone:
		return true
	case models.CallbackAuthHeader:
		if auth.HeaderName == "" || auth.Secret == "" {
			return false
		}
		request.Header.Set(auth.HeaderName, auth.Secret)
	case models.CallbackAuthBasic:
		if auth.Username == "" || auth.Secret == "" {
			return false
		}
		request.SetBasicAuth(auth.Username, auth.Secret)
	default:
		return false
	}
	return true
}

func callbackTarget(raw string) (*url.URL, error) {
	target, err := url.Parse(raw)
	if err != nil {
		return nil, err
	}
	if target.Scheme != "https" || target.Hostname() == "" || target.User != nil || target.Fragment != "" || (target.Port() != "" && target.Port() != "443") {
		return nil, errors.New("invalid callback target")
	}
	return target, nil
}

func (c *Client) dispatch(request *http.Request) (bool, string) {
	started := time.Now()
	response, err := c.http.Do(request)
	if err != nil {
		logHTTPFailure(request, started, 0, models.CallbackNetworkError)
		return false, models.CallbackNetworkError
	}
	defer response.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(response.Body, responseLimit))
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		logHTTPFailure(request, started, response.StatusCode, models.CallbackHTTPError)
		return false, models.CallbackHTTPError
	}
	return true, ""
}

// Do not log the request URL/body, response body/headers or transport error:
// all can contain client secrets or message content. Zero means no HTTP response.
func logHTTPFailure(request *http.Request, started time.Time, status int, code string) {
	slog.LogAttrs(request.Context(), slog.LevelWarn, "Callback HTTP attempt failed",
		slog.String("event_id", request.Header.Get("X-21Mobile-Event-ID")),
		slog.String("method", request.Method),
		slog.Int("http_status", status),
		slog.String("failure_code", code),
		slog.Int64("duration_ms", time.Since(started).Milliseconds()))
}
