package webhook

import (
	"net/http"
	"net/netip"
	"time"

	"github.com/21mobile/sms-gateway/internal/dtos"
	"github.com/21mobile/sms-gateway/internal/models"
)

// Event contains one validated provider callback, translated into gateway terms.
type Event struct {
	Delivery *dtos.DeliveryInput
	MO       *MOMessage
}

type MOMessage struct {
	ProviderMessageID string
	EventKey          string
	Message           string
	ReceivedAt        time.Time
}

// Adapter owns the provider's webhook authentication and JSON contract.
// Authentication runs before Decode; the JSON body never selects its adapter.
type Adapter interface {
	Code() models.Provider
	AuthenticateToken(string) bool
	AllowPeer(netip.Addr) bool
	AuthenticateBody(*http.Request, []byte) bool
	Decode([]byte) (Event, error)
}
