package models

import "time"

// SMSDeliveryReceipt records which DLR state a credential has already polled.
type SMSDeliveryReceipt struct {
	AttemptID         uint64 `gorm:"primaryKey"`
	CredentialID      uint64
	DeliveryUpdatedAt time.Time
	DeliveryStatus    string
}

// Keep the existing table and rows when the Go type is renamed.
func (SMSDeliveryReceipt) TableName() string { return "legacy_status_receipts" }
