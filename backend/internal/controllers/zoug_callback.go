package controllers

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/netip"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/integrations/whitelabel"
	"github.com/21mobile/sms-gateway/internal/security"
)

const zougCallbackBodyLimit = 64 << 10

// ZougCallbackController receives the supplier's global webhook. The secret is
// carried only in the configured URL; the application stores its SHA-256 hash.
type ZougCallbackController struct {
	delivery interface {
		AcceptWhiteLabelDelivery(context.Context, []byte) error
	}
	mo interface {
		AcceptWhiteLabelMO(context.Context, whitelabel.CallbackPayload) error
	}
	tokenHash string
	allowed   []netip.Addr
	logFull   bool
}

func NewZougCallbackController(delivery interface {
	AcceptWhiteLabelDelivery(context.Context, []byte) error
}, mo interface {
	AcceptWhiteLabelMO(context.Context, whitelabel.CallbackPayload) error
}, tokenHash string, allowedIPs []string, logFull ...bool) *ZougCallbackController {
	allowed := make([]netip.Addr, 0, len(allowedIPs))
	for _, raw := range allowedIPs {
		if ip, err := netip.ParseAddr(strings.TrimSpace(raw)); err == nil {
			allowed = append(allowed, ip.Unmap())
		}
	}
	return &ZougCallbackController{delivery: delivery, mo: mo, tokenHash: strings.ToLower(tokenHash), allowed: allowed, logFull: len(logFull) > 0 && logFull[0]}
}

func (c *ZougCallbackController) Register(router chi.Router) {
	router.Post("/webhooks/zoug/{token}", c.receive)
}

func (c *ZougCallbackController) receive(w http.ResponseWriter, r *http.Request) {
	status := http.StatusNoContent
	reason := "persisted"
	var body []byte
	defer func() {
		if !c.logFull {
			return
		}
		attrs := []any{"request_id", middleware.GetReqID(r.Context()), "method", r.Method,
			"route", "/webhooks/zoug/{token}", "peer", r.RemoteAddr,
			"client_ip", security.ClientIP(r), "http_status", status, "reason", reason}
		if body != nil {
			attrs = append(attrs, "request_headers", callbackLogHeaders(r.Header), "request_body", string(body))
		}
		slog.Info("Zoug callback exchange", attrs...)
	}()
	digest := sha256.Sum256([]byte(chi.URLParam(r, "token")))
	expected, err := hex.DecodeString(c.tokenHash)
	if err != nil || len(expected) != sha256.Size || subtle.ConstantTimeCompare(digest[:], expected) != 1 {
		status, reason = http.StatusNotFound, "invalid_token"
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	peer, err := netip.ParseAddr(security.ClientIP(r))
	if err != nil || !c.allowedIP(peer.Unmap()) {
		status, reason = http.StatusForbidden, "ip_not_allowed"
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" {
		status, reason = http.StatusUnsupportedMediaType, "invalid_content_type"
		http.Error(w, "Unsupported Media Type", http.StatusUnsupportedMediaType)
		return
	}
	body, err = io.ReadAll(io.LimitReader(r.Body, zougCallbackBodyLimit+1))
	if err != nil || len(body) > zougCallbackBodyLimit {
		status, reason = http.StatusRequestEntityTooLarge, "body_too_large"
		http.Error(w, "Payload Too Large", http.StatusRequestEntityTooLarge)
		return
	}
	payload, err := whitelabel.DecodeCallback(body)
	if err != nil {
		status, reason = http.StatusUnprocessableEntity, "invalid_callback"
		http.Error(w, "Invalid callback", http.StatusUnprocessableEntity)
		return
	}
	switch payload.CallbackType {
	case whitelabel.CallbackSubmission, whitelabel.CallbackDelivery:
		err = c.delivery.AcceptWhiteLabelDelivery(r.Context(), body)
	case whitelabel.CallbackMO:
		err = c.mo.AcceptWhiteLabelMO(r.Context(), payload)
	default:
		status, reason = http.StatusUnprocessableEntity, "unsupported_callback"
		http.Error(w, "Unsupported callback", http.StatusUnprocessableEntity)
		return
	}
	if err != nil {
		var appErr *apperrors.Error
		if errors.As(err, &appErr) && appErr.Code == apperrors.CodeInvalidRequest {
			status, reason = http.StatusUnprocessableEntity, "invalid_callback"
			http.Error(w, "Invalid callback", http.StatusUnprocessableEntity)
		} else {
			status, reason = http.StatusServiceUnavailable, "persistence_unavailable"
			http.Error(w, "Callback unavailable", http.StatusServiceUnavailable)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func callbackLogHeaders(headers http.Header) http.Header {
	copy := headers.Clone()
	for _, name := range []string{"Authorization", "Proxy-Authorization", "Cookie", "Set-Cookie", "X-API-TOKEN"} {
		if copy.Get(name) != "" {
			copy.Set(name, "[redacted]")
		}
	}
	return copy
}

func (c *ZougCallbackController) allowedIP(ip netip.Addr) bool {
	for _, allowed := range c.allowed {
		if ip == allowed {
			return true
		}
	}
	return false
}
