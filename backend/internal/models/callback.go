package models

import "time"

type DeliveryStatus string

const (
	DeliverySent               DeliveryStatus = "SENT"
	DeliveryDelivered          DeliveryStatus = "DELIVERED"
	DeliveryNotDelivered       DeliveryStatus = "NOT_DELIVERED"
	DeliveryDuplicated         DeliveryStatus = "DUPLICATED"
	DeliveryInvalidDestination DeliveryStatus = "INVALID_DESTINATION_ADDRESS"
	DeliveryBlacklist          DeliveryStatus = "BLACKLIST"
	DeliveryCancelled          DeliveryStatus = "MESSAGE_CANCELLED"
	DeliveryInvalidFormat      DeliveryStatus = "INVALID_MESSAGE_FORMAT"
	DeliveryError              DeliveryStatus = "GENERAL_ERROR"
	DeliveryUnknown            DeliveryStatus = "UNKNOWN"
)

func (s DeliveryStatus) Valid() bool {
	switch s {
	case DeliverySent, DeliveryDelivered, DeliveryNotDelivered, DeliveryDuplicated, DeliveryInvalidDestination, DeliveryBlacklist, DeliveryCancelled, DeliveryInvalidFormat, DeliveryError, DeliveryUnknown:
		return true
	}
	return false
}
func (s DeliveryStatus) Terminal() bool { return s != "" && s != DeliverySent && s != DeliveryUnknown }

type CallbackKind string

const (
	CallbackKindDLR CallbackKind = "dlr"
	CallbackKindMO  CallbackKind = "mo"
)

type CallbackState string

const CallbackContentUnavailable = "CONTENT_UNAVAILABLE"

const (
	CallbackPending        CallbackState = "pending"
	CallbackProcessing     CallbackState = "processing"
	CallbackDelivered      CallbackState = "delivered"
	CallbackFailed         CallbackState = "failed"
	CallbackNetworkError                 = "NETWORK_ERROR"
	CallbackHTTPError                    = "HTTP_ERROR"
	CallbackAccessBlocked                = "ACCESS_BLOCKED"
	CallbackInvalidPayload               = "INVALID_CALLBACK_PAYLOAD"
)

type DeliveryEvent struct {
	ID                uint64
	EventKey          string
	ProviderMessageID string
	Status            DeliveryStatus
	OccurredAt        time.Time
	AttemptID         *uint64
	Processed         bool
	CreatedAt         time.Time
}
type CallbackJob struct {
	ID           uint64
	EventID      uint64
	AttemptID    uint64
	CredentialID uint64
	URL          string
	Status       DeliveryStatus
	State        CallbackState
	Attempts     int
	AvailableAt  time.Time
	LeaseUntil   *time.Time
	LeaseToken   *string
	LastError    string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}
