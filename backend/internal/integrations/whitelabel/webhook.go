package whitelabel

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/providers/webhook"
)

type WebhookAdapter struct {
	tokenHash []byte
	allowed   []netip.Addr
}

func NewWebhookAdapter(tokenHash string, allowedIPs []string) *WebhookAdapter {
	hash, _ := hex.DecodeString(strings.ToLower(tokenHash))
	allowed := make([]netip.Addr, 0, len(allowedIPs))
	for _, raw := range allowedIPs {
		if ip, err := netip.ParseAddr(strings.TrimSpace(raw)); err == nil {
			allowed = append(allowed, ip.Unmap())
		}
	}
	return &WebhookAdapter{tokenHash: hash, allowed: allowed}
}

func (*WebhookAdapter) Code() models.Provider { return models.ProviderZoug }

func (a *WebhookAdapter) AuthenticateToken(token string) bool {
	digest := sha256.Sum256([]byte(token))
	return len(a.tokenHash) == sha256.Size && subtle.ConstantTimeCompare(digest[:], a.tokenHash) == 1
}

func (a *WebhookAdapter) AllowPeer(peer netip.Addr) bool {
	for _, allowed := range a.allowed {
		if peer.Unmap() == allowed {
			return true
		}
	}
	return false
}

func (*WebhookAdapter) AuthenticateBody(_ *http.Request, _ []byte) bool { return true }

func (*WebhookAdapter) Decode(raw []byte) (webhook.Event, error) {
	payload, err := DecodeCallback(raw)
	if err != nil {
		return webhook.Event{}, err
	}
	if payload.CallbackType != CallbackMO {
		delivery, err := NormalizeDeliveryCallback(raw)
		return webhook.Event{Delivery: &delivery}, err
	}
	receivedAt, err := time.Parse(time.RFC3339Nano, payload.ReportDate)
	if err != nil || payload.ShortMessage == nil {
		return webhook.Event{}, ErrInvalidCallback
	}
	identity, _ := json.Marshal(struct {
		MessageID string
		Date      string
		Message   string
	}{payload.MessageID, receivedAt.UTC().Format(time.RFC3339Nano), *payload.ShortMessage})
	digest := sha256.Sum256(identity)
	return webhook.Event{MO: &webhook.MOMessage{
		ProviderMessageID: payload.MessageID,
		EventKey:          "zoug:" + hex.EncodeToString(digest[:]),
		Message:           *payload.ShortMessage,
		ReceivedAt:        receivedAt,
	}}, nil
}
