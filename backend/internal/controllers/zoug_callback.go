package controllers

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/netip"
	"strings"

	"github.com/go-chi/chi/v5"

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
}

func NewZougCallbackController(delivery interface {
	AcceptWhiteLabelDelivery(context.Context, []byte) error
}, mo interface {
	AcceptWhiteLabelMO(context.Context, whitelabel.CallbackPayload) error
}, tokenHash string, allowedIPs []string) *ZougCallbackController {
	allowed := make([]netip.Addr, 0, len(allowedIPs))
	for _, raw := range allowedIPs {
		if ip, err := netip.ParseAddr(strings.TrimSpace(raw)); err == nil {
			allowed = append(allowed, ip.Unmap())
		}
	}
	return &ZougCallbackController{delivery: delivery, mo: mo, tokenHash: strings.ToLower(tokenHash), allowed: allowed}
}

func (c *ZougCallbackController) Register(router chi.Router) {
	router.Post("/webhooks/zoug/{token}", c.receive)
}

func (c *ZougCallbackController) receive(w http.ResponseWriter, r *http.Request) {
	digest := sha256.Sum256([]byte(chi.URLParam(r, "token")))
	expected, err := hex.DecodeString(c.tokenHash)
	if err != nil || len(expected) != sha256.Size || subtle.ConstantTimeCompare(digest[:], expected) != 1 {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}
	peer, err := netip.ParseAddr(security.ClientIP(r))
	if err != nil || !c.allowedIP(peer.Unmap()) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" {
		http.Error(w, "Unsupported Media Type", http.StatusUnsupportedMediaType)
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, zougCallbackBodyLimit+1))
	if err != nil || len(body) > zougCallbackBodyLimit {
		http.Error(w, "Payload Too Large", http.StatusRequestEntityTooLarge)
		return
	}
	payload, err := whitelabel.DecodeCallback(body)
	if err != nil {
		http.Error(w, "Invalid callback", http.StatusUnprocessableEntity)
		return
	}
	switch payload.CallbackType {
	case whitelabel.CallbackSubmission, whitelabel.CallbackDelivery:
		err = c.delivery.AcceptWhiteLabelDelivery(r.Context(), body)
	case whitelabel.CallbackMO:
		err = c.mo.AcceptWhiteLabelMO(r.Context(), payload)
	default:
		http.Error(w, "Unsupported callback", http.StatusUnprocessableEntity)
		return
	}
	if err != nil {
		var appErr *apperrors.Error
		if errors.As(err, &appErr) && appErr.Code == apperrors.CodeInvalidRequest {
			http.Error(w, "Invalid callback", http.StatusUnprocessableEntity)
		} else {
			http.Error(w, "Callback unavailable", http.StatusServiceUnavailable)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (c *ZougCallbackController) allowedIP(ip netip.Addr) bool {
	for _, allowed := range c.allowed {
		if ip == allowed {
			return true
		}
	}
	return false
}
