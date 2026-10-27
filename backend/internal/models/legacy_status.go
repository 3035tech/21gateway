package models

import "time"

// Consumption belongs to polling, independently of callback delivery.
type LegacyStatusReceipt struct {
	AttemptID         uint64 `gorm:"primaryKey"`
	CredentialID      uint64
	DeliveryUpdatedAt time.Time
	DeliveryStatus    string
}
