package services

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/21mobile/sms-gateway/internal/dtos"
	apperrors "github.com/21mobile/sms-gateway/internal/errors"
	"github.com/21mobile/sms-gateway/internal/models"
	"github.com/21mobile/sms-gateway/internal/repositories"
)

type NotificationMailer interface {
	SendCriticalAlert(context.Context, string, string, string, string) error
}

type notificationDefinition struct {
	Code  models.NotificationType
	Title string
	Path  string
}

var notificationDefinitions = []notificationDefinition{
	{models.NotificationUnknownSends, "Envios com resultado incerto", "/envios"},
	{models.NotificationExpiredLeases, "Processamento com prazo expirado", "/callbacks"},
	{models.NotificationDLRPending, "Notificações DLR atrasadas", "/callbacks"},
	{models.NotificationMOPending, "Notificações MO atrasadas", "/callbacks"},
	{models.NotificationDLRUnmatched, "DLR sem envio associado", "/callbacks"},
	{models.NotificationMOUnmatched, "MO sem envio associado", "/callbacks"},
	{models.NotificationScheduleDue, "SMS agendados atrasados", "/agendamentos"},
	{models.NotificationCallbackFailed, "Falhas definitivas de notificação", "/callbacks"},
}

type NotificationService struct {
	repo       *repositories.Notifications
	operations *OperationsService
	mailer     NotificationMailer
}

func NewNotificationService(repo *repositories.Notifications, operations *OperationsService, mailer NotificationMailer) *NotificationService {
	return &NotificationService{repo: repo, operations: operations, mailer: mailer}
}
func (s *NotificationService) EmailAvailable() bool { return s.mailer != nil }

func (s *NotificationService) List(ctx context.Context, userID uint64, q dtos.NotificationListInput) (dtos.NotificationList, error) {
	result, err := s.repo.List(ctx, userID, q)
	return result, dbError(err)
}
func (s *NotificationService) MarkRead(ctx context.Context, userID, eventID uint64) error {
	ok, err := s.repo.MarkRead(ctx, userID, eventID)
	if err != nil {
		return dbError(err)
	}
	if !ok {
		return apperrors.Wrap(apperrors.CodeNotFound, nil)
	}
	return nil
}
func (s *NotificationService) Preferences(ctx context.Context) ([]dtos.NotificationPreference, error) {
	values, err := s.repo.Preferences(ctx)
	if err != nil {
		return nil, dbError(err)
	}
	result := make([]dtos.NotificationPreference, 0, len(notificationDefinitions))
	for _, definition := range notificationDefinitions {
		result = append(result, dtos.NotificationPreference{Code: string(definition.Code), Title: definition.Title, EmailEnabled: values[definition.Code]})
	}
	return result, nil
}
func (s *NotificationService) SetPreference(ctx context.Context, code string, enabled bool) (dtos.NotificationPreference, error) {
	if enabled && s.mailer == nil {
		return dtos.NotificationPreference{}, apperrors.Wrap(apperrors.CodeServiceUnavailable, nil)
	}
	for _, definition := range notificationDefinitions {
		if code == string(definition.Code) {
			ok, err := s.repo.SetPreference(ctx, definition.Code, enabled)
			if err != nil {
				return dtos.NotificationPreference{}, dbError(err)
			}
			if !ok {
				return dtos.NotificationPreference{}, apperrors.Wrap(apperrors.CodeNotFound, nil)
			}
			return dtos.NotificationPreference{Code: code, Title: definition.Title, EmailEnabled: enabled}, nil
		}
	}
	return dtos.NotificationPreference{}, apperrors.Wrap(apperrors.CodeNotFound, nil)
}

func olderThanNotification(value *time.Time, now time.Time, age time.Duration) bool {
	return value != nil && !value.After(now.Add(-age))
}

func notificationEvents(snapshot dtos.OperationsSnapshot) []models.NotificationEvent {
	const callbackAge = 15 * time.Minute
	const scheduleAge = models.OperationScheduleGrace
	var result []models.NotificationEvent
	add := func(code models.NotificationType, count int64, detail string) {
		for _, definition := range notificationDefinitions {
			if definition.Code == code {
				result = append(result, models.NotificationEvent{TypeCode: code, Title: definition.Title, Description: fmt.Sprintf("%d %s", count, detail), Path: definition.Path, OpenedAt: snapshot.ObservedAt, UpdatedAt: snapshot.ObservedAt})
				return
			}
		}
	}
	if snapshot.UnresolvedSends > 0 {
		add(models.NotificationUnknownSends, snapshot.UnresolvedSends, "envios sem resultado conclusivo; investigue antes de reenviar.")
	}
	if n := snapshot.DLR.ExpiredLeases + snapshot.MO.ExpiredLeases; n > 0 {
		add(models.NotificationExpiredLeases, n, "jobs com lease expirado; confira os workers.")
	}
	if olderThanNotification(snapshot.DLR.OldestPendingAt, snapshot.ObservedAt, callbackAge) {
		add(models.NotificationDLRPending, snapshot.DLR.Pending, "notificações DLR pendentes há mais de 15 minutos.")
	}
	if olderThanNotification(snapshot.MO.OldestPendingAt, snapshot.ObservedAt, callbackAge) {
		add(models.NotificationMOPending, snapshot.MO.Pending, "notificações MO pendentes há mais de 15 minutos.")
	}
	if olderThanNotification(snapshot.DLR.OldestUnmatchedAt, snapshot.ObservedAt, callbackAge) {
		add(models.NotificationDLRUnmatched, snapshot.DLR.UnmatchedEvents, "eventos DLR sem envio associado há mais de 15 minutos.")
	}
	if olderThanNotification(snapshot.MO.OldestUnmatchedAt, snapshot.ObservedAt, callbackAge) {
		add(models.NotificationMOUnmatched, snapshot.MO.UnmatchedEvents, "eventos MO sem envio associado há mais de 15 minutos.")
	}
	if olderThanNotification(snapshot.Scheduler.OldestDueAt, snapshot.ObservedAt, scheduleAge) {
		add(models.NotificationScheduleDue, snapshot.Scheduler.Due, "SMS agendados atrasados há mais de 5 minutos.")
	}
	if n := snapshot.DLR.FailedLastHour + snapshot.MO.FailedLastHour; n > 0 {
		add(models.NotificationCallbackFailed, n, "notificações ao cliente com falha definitiva na última hora.")
	}
	return result
}

// Cycle runs on a worker. A snapshot failure leaves current incidents untouched.
func (s *NotificationService) Cycle(ctx context.Context) error {
	snapshot, err := s.operations.Snapshot(ctx)
	if err != nil {
		return err
	}
	if err := s.repo.Sync(ctx, notificationEvents(snapshot)); err != nil {
		return dbError(err)
	}
	if s.mailer == nil {
		return nil
	}
	for i := 0; i < 8; i++ {
		email, err := s.repo.ClaimEmail(ctx)
		if err != nil {
			return dbError(err)
		}
		if email == nil {
			return nil
		}
		mailCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
		err = s.mailer.SendCriticalAlert(mailCtx, email.To, email.Title, email.Description, email.Path)
		cancel()
		if finishErr := s.repo.FinishEmail(ctx, *email, err == nil); finishErr != nil {
			return dbError(finishErr)
		}
		if err != nil {
			slog.Error("Notification email delivery failed", "event_id", email.EventID, "user_id", email.UserID)
		}
	}
	return nil
}

func (s *NotificationService) Run(ctx context.Context) {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		cycleCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
		if err := s.Cycle(cycleCtx); err != nil {
			slog.Error("Notification cycle failed")
		}
		cancel()
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
