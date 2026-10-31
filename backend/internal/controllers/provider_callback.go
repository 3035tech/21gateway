package controllers

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"net/netip"
	"strings"

	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/providers/webhook"
	"github.com/21mobile/sms-gateway/internal/security"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

const providerCallbackBodyLimit = 64 << 10
const providerCallbackRoute = "/webhooks/{provider}/{token}"

type ProviderCallbackController struct {
	delivery interface {
		AcceptProviderDelivery(context.Context, models.Provider, dtos.DeliveryInput) error
	}
	mo interface {
		AcceptProviderMO(context.Context, models.Provider, webhook.MOMessage) error
	}
	adapters map[models.Provider]webhook.Adapter
	logFull  bool
}

func NewProviderCallbackController(delivery interface {
	AcceptProviderDelivery(context.Context, models.Provider, dtos.DeliveryInput) error
}, mo interface {
	AcceptProviderMO(context.Context, models.Provider, webhook.MOMessage) error
}, logFull bool, adapters ...webhook.Adapter) *ProviderCallbackController {
	registry := make(map[models.Provider]webhook.Adapter, len(adapters))
	for _, adapter := range adapters {
		if adapter == nil || adapter.Code() == "" || registry[adapter.Code()] != nil {
			panic("invalid webhook adapter registry")
		}
		registry[adapter.Code()] = adapter
	}
	return &ProviderCallbackController{delivery: delivery, mo: mo, adapters: registry, logFull: logFull}
}
func (c *ProviderCallbackController) Register(router chi.Router) {
	router.Post("/webhooks/{provider}", c.receive)
	router.Post(providerCallbackRoute, c.receive)
}

func (c *ProviderCallbackController) receive(w http.ResponseWriter, r *http.Request) {
	status, reason := http.StatusNoContent, "persisted"
	providerCode := models.Provider(chi.URLParam(r, "provider"))
	var body []byte
	defer func() {
		if !c.logFull || providerCode != models.ProviderZoug {
			return
		}
		attrs := []any{"request_id", middleware.GetReqID(r.Context()), "method", r.Method, "route", providerCallbackRoute, "peer", r.RemoteAddr, "client_ip", security.ClientIP(r), "http_status", status, "reason", reason}
		if body != nil {
			attrs = append(attrs, "request_headers", callbackLogHeaders(r.Header), "request_body", string(body))
		}
		slog.Info("provider callback exchange", attrs...)
	}()
	adapter := c.adapters[providerCode]
	if adapter == nil || !adapter.AuthenticateToken(chi.URLParam(r, "token")) {
		status, reason = http.StatusNotFound, "invalid_provider_or_token"
		http.Error(w, "Not Found", status)
		return
	}
	peer, err := netip.ParseAddr(security.ClientIP(r))
	if err != nil || !adapter.AllowPeer(peer) {
		status, reason = http.StatusForbidden, "ip_not_allowed"
		http.Error(w, "Forbidden", status)
		return
	}
	contentType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || contentType != "application/json" {
		status, reason = http.StatusUnsupportedMediaType, "invalid_content_type"
		http.Error(w, "Unsupported Media Type", status)
		return
	}
	body, err = io.ReadAll(io.LimitReader(r.Body, providerCallbackBodyLimit+1))
	if err != nil || len(body) > providerCallbackBodyLimit {
		status, reason = http.StatusRequestEntityTooLarge, "body_too_large"
		http.Error(w, "Payload Too Large", status)
		return
	}
	if !adapter.AuthenticateBody(r, body) {
		status, reason = http.StatusNotFound, "invalid_body_signature"
		http.Error(w, "Not Found", status)
		return
	}
	event, err := adapter.Decode(body)
	if err != nil || (event.Delivery == nil) == (event.MO == nil) {
		status, reason = http.StatusUnprocessableEntity, "invalid_callback"
		http.Error(w, "Invalid callback", status)
		return
	}
	key := ""
	if event.Delivery != nil {
		key = event.Delivery.Body.EventKey
	} else {
		key = event.MO.EventKey
	}
	if !strings.HasPrefix(key, string(adapter.Code())+":") {
		status, reason = http.StatusUnprocessableEntity, "invalid_callback_namespace"
		http.Error(w, "Invalid callback", status)
		return
	}
	if event.Delivery != nil {
		err = c.delivery.AcceptProviderDelivery(r.Context(), adapter.Code(), *event.Delivery)
	} else {
		err = c.mo.AcceptProviderMO(r.Context(), adapter.Code(), *event.MO)
	}
	if err != nil {
		var appErr *apperrors.Error
		if errors.As(err, &appErr) && appErr.Code == apperrors.CodeInvalidRequest {
			status, reason = http.StatusUnprocessableEntity, "invalid_callback"
			http.Error(w, "Invalid callback", status)
		} else {
			status, reason = http.StatusServiceUnavailable, "persistence_unavailable"
			http.Error(w, "Callback unavailable", status)
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
