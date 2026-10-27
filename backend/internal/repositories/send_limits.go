package repositories

import (
	"context"
	"gorm.io/gorm"
	"time"
)

const sendLimitLockID = 210016
const sendLimitTimeout = 2 * time.Second
const MaximumProviderMessagesPerSecond = 200

type sendBudgetScope string

const (
	sendBudgetAPI      sendBudgetScope = "api"
	sendBudgetDispatch sendBudgetScope = "dispatch"
	sendBudgetBoth     sendBudgetScope = "both"
)

type DispatchSendLimits struct{ limits *SendLimits }

func NewDispatchSendLimits(db *gorm.DB) *DispatchSendLimits {
	return &DispatchSendLimits{limits: NewSendLimits(db)}
}
func (r *DispatchSendLimits) Allow(ctx context.Context, credentialID uint64, messages int) (bool, error) {
	return r.limits.allow(ctx, credentialID, messages, true)
}

// Shared rolling one-second window. Counts messages, not only HTTP requests.
type SendLimits struct{ db *gorm.DB }

func NewSendLimits(db *gorm.DB) *SendLimits { return &SendLimits{db: db} }
func (r *SendLimits) Allow(ctx context.Context, credentialID uint64, messages int) (bool, error) {
	return r.allow(ctx, credentialID, messages, false)
}
func (r *SendLimits) allow(ctx context.Context, credentialID uint64, messages int, dispatchOnly bool) (bool, error) {
	_, allowed, err := r.reserve(ctx, credentialID, messages, dispatchOnly, false)
	return allowed, err
}
func (r *DispatchSendLimits) ReserveUpTo(ctx context.Context, credentialID uint64, messages int) (int, error) {
	granted, _, err := r.limits.reserve(ctx, credentialID, messages, true, true)
	return granted, err
}
func (r *SendLimits) reserve(ctx context.Context, credentialID uint64, messages int, dispatchOnly, partial bool) (int, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, sendLimitTimeout)
	defer cancel()
	allowed := false
	granted := 0
	if messages < 0 || messages > MaximumProviderMessagesPerSecond || (dispatchOnly && messages == 0) {
		return 0, false, nil
	}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", sendLimitLockID).Error; err != nil {
			return err
		}
		var now time.Time
		if err := tx.Raw("SELECT clock_timestamp()").Scan(&now).Error; err != nil {
			return err
		}
		if err := tx.Exec("DELETE FROM send_reservations WHERE created_at < ?", now.Add(-time.Minute)).Error; err != nil {
			return err
		}
		var used struct {
			Messages   int
			Requests   int
			Dispatches int
		}
		if err := tx.Raw("SELECT COALESCE(SUM(messages),0) AS messages, COUNT(*) FILTER (WHERE credential_id=? AND scope IN (?,?)) AS requests, COUNT(*) FILTER (WHERE credential_id=? AND scope IN (?,?)) AS dispatches FROM send_reservations WHERE created_at > ?", credentialID, sendBudgetAPI, sendBudgetBoth, credentialID, sendBudgetDispatch, sendBudgetBoth, now.Add(-time.Second)).Scan(&used).Error; err != nil {
			return err
		}
		if (!dispatchOnly && used.Requests > 0) || (messages > 0 && used.Dispatches > 0) {
			return nil
		}
		if used.Messages+messages > MaximumProviderMessagesPerSecond {
			if !partial {
				return nil
			}
			messages = MaximumProviderMessagesPerSecond - used.Messages
			if messages <= 0 {
				return nil
			}
		}
		scope := sendBudgetBoth
		if dispatchOnly {
			scope = sendBudgetDispatch
		} else if messages == 0 {
			scope = sendBudgetAPI
		}
		if err := tx.Exec("INSERT INTO send_reservations(credential_id,messages,created_at,scope) VALUES (?,?,?,?)", credentialID, messages, now, scope).Error; err != nil {
			return err
		}
		granted = messages
		allowed = true
		return nil
	})
	return granted, allowed, err
}
