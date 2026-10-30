package services

import (
	"context"
	"testing"
	"time"

	"github.com/21mobile/sms-gateway/internal/dtos"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/repositories"
)

type notificationTestMailer struct{}

func (notificationTestMailer) SendCriticalAlert(context.Context, string, string, string, string) error {
	return nil
}

func TestPostgresNotificationsPreferencesAndRecipients(t *testing.T) {
	ctx, db := postgresFixture(t)
	adminA := models.SystemUser{Name: "Admin A", Email: "admin-a@example.test", PasswordHash: "unused", Role: models.RoleAdmin, Active: true}
	adminB := models.SystemUser{Name: "Admin B", Email: "admin-b@example.test", PasswordHash: "unused", Role: models.RoleAdmin, Active: true}
	operator := models.SystemUser{Name: "Operator", Email: "operator@example.test", PasswordHash: "unused", Role: models.RoleOperator, Active: true}
	for _, user := range []*models.SystemUser{&adminA, &adminB, &operator} {
		if err := db.Create(user).Error; err != nil {
			t.Fatal(err)
		}
	}
	repo := repositories.NewNotifications(db)
	service := NewNotificationService(repo, nil, notificationTestMailer{})
	item := models.NotificationEvent{TypeCode: models.NotificationExpiredLeases, Title: "Processing stalled", Description: "1 lease expired", Path: "/callbacks", OpenedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()}
	if err := repo.Sync(ctx, []models.NotificationEvent{item}); err != nil {
		t.Fatal(err)
	}
	if err := repo.Sync(ctx, []models.NotificationEvent{item}); err != nil {
		t.Fatal(err)
	}
	query := dtos.NotificationListInput{Page: 1, PageSize: 20, Sort: "openedAt", Order: "desc"}
	first, err := service.List(ctx, adminA.ID, query)
	if err != nil || first.Total != 1 || first.UnreadCount != 1 || len(first.Items) != 1 {
		t.Fatalf("first=%+v err=%v", first, err)
	}
	second, err := service.List(ctx, adminB.ID, query)
	if err != nil || second.Total != 1 {
		t.Fatalf("second=%+v err=%v", second, err)
	}
	outside, err := service.List(ctx, operator.ID, query)
	if err != nil || outside.Total != 0 {
		t.Fatalf("operator=%+v err=%v", outside, err)
	}
	if email, err := repo.ClaimEmail(ctx); err != nil || email != nil {
		t.Fatalf("default email=%+v err=%v", email, err)
	}
	if err := service.MarkRead(ctx, adminA.ID, first.Items[0].ID); err != nil {
		t.Fatal(err)
	}
	first, err = service.List(ctx, adminA.ID, query)
	if err != nil || first.UnreadCount != 0 || first.Items[0].ReadAt == nil {
		t.Fatalf("read=%+v err=%v", first, err)
	}
	if second, err = service.List(ctx, adminB.ID, query); err != nil || second.UnreadCount != 1 {
		t.Fatalf("other admin read=%+v err=%v", second, err)
	}
	if _, err := service.SetPreference(ctx, string(models.NotificationExpiredLeases), true); err != nil {
		t.Fatal(err)
	}
	if err := repo.Sync(ctx, nil); err != nil {
		t.Fatal(err)
	}
	if err := repo.Sync(ctx, []models.NotificationEvent{item}); err != nil {
		t.Fatal(err)
	}
	seen := map[uint64]bool{}
	for range 2 {
		email, err := repo.ClaimEmail(ctx)
		if err != nil || email == nil {
			t.Fatalf("claim=%+v err=%v", email, err)
		}
		if seen[email.UserID] || email.UserID == operator.ID {
			t.Fatalf("unexpected recipient %+v", email)
		}
		seen[email.UserID] = true
		if err := repo.FinishEmail(ctx, *email, true); err != nil {
			t.Fatal(err)
		}
	}
	if !seen[adminA.ID] || !seen[adminB.ID] {
		t.Fatalf("missing admins: %+v", seen)
	}
	if email, err := repo.ClaimEmail(ctx); err != nil || email != nil {
		t.Fatalf("duplicate email=%+v err=%v", email, err)
	}
}
