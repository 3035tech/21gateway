package callback

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/21mobile/sms-gateway/internal/dtos"
	"github.com/21mobile/sms-gateway/internal/models"
	"net/http"
	"strconv"
	"time"
	"unicode/utf8"
)

const maximumMOMessageBytes = 4096
const maximumMOCorrelationCharacters = 50

var legacyMOTimezone = time.FixedZone("BRT", -3*60*60)

// DeliverMO is an outbound adapter only. No supplier contract or ingestion route is implied.
// The caller owns durable IDs, tenant association and authorization before dispatch.
func (c *Client) DeliverMO(ctx context.Context, destination string, message models.MOMessage) (bool, string) {
	target, err := callbackTarget(destination)
	if err != nil || message.ID == 0 || message.AttemptID == 0 || message.ReceivedAt.IsZero() || !utf8.ValidString(message.Message) || len(message.Message) == 0 || len(message.Message) > maximumMOMessageBytes || !utf8.ValidString(message.CorrelationID) || utf8.RuneCountInString(message.CorrelationID) > maximumMOCorrelationCharacters || !validMOPhone(message.PhoneNumber) {
		return false, models.CallbackInvalidPayload
	}
	payload := dtos.LegacyMOCallback{Message: message.Message, Mobile21Identifier: message.ID, CorrelationID: message.CorrelationID, MTID: message.AttemptID, PhoneNumber: message.PhoneNumber, Date: message.ReceivedAt.In(legacyMOTimezone).Format("Mon Jan 02 15:04:05 MST 2006")}
	data, err := json.Marshal(payload)
	if err != nil {
		return false, models.CallbackInvalidPayload
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, target.String(), bytes.NewReader(data))
	if err != nil {
		return false, models.CallbackNetworkError
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-21Mobile-Event-ID", "mo:"+strconv.FormatUint(message.ID, 10))
	return c.dispatch(request)
}
func validMOPhone(phone string) bool {
	if len(phone) == 0 || len(phone) > 15 {
		return false
	}
	for _, digit := range phone {
		if digit < '0' || digit > '9' {
			return false
		}
	}
	return true
}
