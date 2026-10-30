package repositories

import (
	"context"
	"time"

	"github.com/21mobile/sms-gateway/internal/dtos"
	"github.com/21mobile/sms-gateway/internal/models"
	"gorm.io/gorm"
)

const notificationSyncLockID = 210026

type Notifications struct{ db *gorm.DB }

func NewNotifications(db *gorm.DB) *Notifications { return &Notifications{db: db} }

// Sync serializes incident transitions across worker replicas. One active event exists per type.
func (r *Notifications) Sync(ctx context.Context, active []models.NotificationEvent) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("SELECT pg_advisory_xact_lock(?)", notificationSyncLockID).Error; err != nil {
			return err
		}
		seen := make(map[models.NotificationType]bool, len(active))
		for _, item := range active {
			seen[item.TypeCode] = true
			var existing models.NotificationEvent
			err := tx.Table("notification_events").Where("type_code = ? AND resolved_at IS NULL", item.TypeCode).Take(&existing).Error
			if err == gorm.ErrRecordNotFound {
				var enabled bool
				if err := tx.Table("notification_types").Select("email_enabled").Where("code = ?", item.TypeCode).Scan(&enabled).Error; err != nil {
					return err
				}
				item.EmailSelected = enabled
				if err := tx.Table("notification_events").Create(&item).Error; err != nil {
					return err
				}
				existing = item
			} else if err != nil {
				return err
			} else if existing.Description != item.Description {
				if err := tx.Table("notification_events").Where("id = ?", existing.ID).Updates(map[string]any{"description": item.Description, "updated_at": time.Now().UTC()}).Error; err != nil {
					return err
				}
			}
			if err := tx.Exec(`INSERT INTO notification_receipts(event_id,user_id,email_next_at)
				SELECT ?, id, CASE WHEN ? THEN NOW() ELSE NULL END FROM system_users
				WHERE role = 'admin' AND active AND archived_at IS NULL
				ON CONFLICT (event_id,user_id) DO NOTHING`, existing.ID, existing.EmailSelected).Error; err != nil {
				return err
			}
		}
		var current []models.NotificationEvent
		if err := tx.Table("notification_events").Where("resolved_at IS NULL").Find(&current).Error; err != nil {
			return err
		}
		for _, item := range current {
			if !seen[item.TypeCode] {
				if err := tx.Exec("UPDATE notification_events SET resolved_at=NOW(),updated_at=NOW() WHERE id=?", item.ID).Error; err != nil {
					return err
				}
			}
		}
		return nil
	})
}

func (r *Notifications) List(ctx context.Context, userID uint64, q dtos.NotificationListInput) (dtos.NotificationList, error) {
	result := dtos.NotificationList{Items: []dtos.NotificationItem{}, Page: q.Page, PageSize: q.PageSize}
	base := func() *gorm.DB {
		return r.db.WithContext(ctx).Table("notification_receipts AS receipt").Joins("JOIN notification_events AS event ON event.id=receipt.event_id").Where("receipt.user_id=?", userID)
	}
	if err := base().Where("receipt.read_at IS NULL").Count(&result.UnreadCount).Error; err != nil {
		return result, err
	}
	query := base()
	if q.Unread {
		query = query.Where("receipt.read_at IS NULL")
	}
	if err := query.Count(&result.Total).Error; err != nil {
		return result, err
	}
	column := "event.opened_at"
	if q.Sort == "title" {
		column = "event.title"
	}
	order := "DESC"
	if q.Order == "asc" {
		order = "ASC"
	}
	items := base()
	if q.Unread {
		items = items.Where("receipt.read_at IS NULL")
	}
	err := items.Select("event.id,event.type_code AS type,event.title,event.description,event.path,event.opened_at,event.resolved_at,receipt.read_at").Order(column + " " + order + ",event.id DESC").Limit(q.PageSize).Offset((q.Page - 1) * q.PageSize).Scan(&result.Items).Error
	return result, err
}

func (r *Notifications) MarkRead(ctx context.Context, userID, eventID uint64) (bool, error) {
	result := r.db.WithContext(ctx).Exec("UPDATE notification_receipts SET read_at=COALESCE(read_at,NOW()) WHERE user_id=? AND event_id=?", userID, eventID)
	return result.RowsAffected == 1, result.Error
}

func (r *Notifications) Preferences(ctx context.Context) (map[models.NotificationType]bool, error) {
	var rows []struct {
		Code         models.NotificationType
		EmailEnabled bool
	}
	err := r.db.WithContext(ctx).Table("notification_types").Select("code,email_enabled").Scan(&rows).Error
	result := make(map[models.NotificationType]bool, len(rows))
	for _, row := range rows {
		result[row.Code] = row.EmailEnabled
	}
	return result, err
}

func (r *Notifications) SetPreference(ctx context.Context, code models.NotificationType, enabled bool) (bool, error) {
	result := r.db.WithContext(ctx).Table("notification_types").Where("code=?", code).Updates(map[string]any{"email_enabled": enabled, "updated_at": time.Now().UTC()})
	return result.RowsAffected == 1, result.Error
}

func (r *Notifications) ClaimEmail(ctx context.Context) (*models.NotificationEmail, error) {
	var email models.NotificationEmail
	err := r.db.WithContext(ctx).Raw(`WITH candidate AS (
		SELECT receipt.event_id,receipt.user_id FROM notification_receipts receipt
		JOIN notification_events event ON event.id=receipt.event_id
		JOIN system_users user_account ON user_account.id=receipt.user_id
		WHERE receipt.email_sent_at IS NULL AND receipt.email_next_at<=NOW()
		AND (receipt.email_lease_until IS NULL OR receipt.email_lease_until<NOW())
		AND receipt.email_attempts<8 AND event.resolved_at IS NULL
		AND user_account.role='admin' AND user_account.active AND user_account.archived_at IS NULL
		ORDER BY receipt.email_next_at,receipt.event_id,receipt.user_id LIMIT 1 FOR UPDATE OF receipt SKIP LOCKED
	), claimed AS (
		UPDATE notification_receipts receipt SET email_lease_until=NOW()+INTERVAL '30 seconds',email_attempts=receipt.email_attempts+1
		FROM candidate WHERE receipt.event_id=candidate.event_id AND receipt.user_id=candidate.user_id
		RETURNING receipt.event_id,receipt.user_id,receipt.email_attempts
	)
	SELECT claimed.event_id,claimed.user_id,user_account.email AS "to",event.title,event.description,event.path,claimed.email_attempts AS attempts
	FROM claimed JOIN notification_events event ON event.id=claimed.event_id
	JOIN system_users user_account ON user_account.id=claimed.user_id`).Scan(&email).Error
	if err != nil || email.EventID == 0 {
		return nil, err
	}
	return &email, nil
}

func (r *Notifications) FinishEmail(ctx context.Context, email models.NotificationEmail, delivered bool) error {
	if delivered {
		return r.db.WithContext(ctx).Exec("UPDATE notification_receipts SET email_sent_at=NOW(),email_next_at=NULL,email_lease_until=NULL WHERE event_id=? AND user_id=? AND email_attempts=?", email.EventID, email.UserID, email.Attempts).Error
	}
	return r.db.WithContext(ctx).Exec("UPDATE notification_receipts SET email_next_at=NOW()+INTERVAL '5 minutes',email_lease_until=NULL WHERE event_id=? AND user_id=? AND email_attempts=?", email.EventID, email.UserID, email.Attempts).Error
}
