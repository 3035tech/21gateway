package services

import (
	"context"
	"log/slog"

	"github.com/21mobile/sms-gateway/internal/models"
)

// Completion is separate from HTTP evidence: a successful response does not
// establish a committed job state. Never include the underlying storage error.
func recordCallbackCompletion(ctx context.Context, kind models.CallbackKind, jobID, eventID uint64, attempts int, delivered bool, code string, persistErr error) {
	// Most messages succeed on the first attempt: avoid attributes/log allocation.
	if persistErr == nil && delivered && attempts <= 1 {
		return
	}

	attrs := []slog.Attr{
		slog.String("callback_kind", string(kind)),
		slog.Uint64("job_id", jobID),
		slog.Uint64("event_id", eventID),
		slog.Int("attempts", attempts),
		slog.Bool("delivered", delivered),
		slog.String("failure_code", code),
		slog.Bool("outcome_persisted", persistErr == nil),
	}
	level, message := slog.LevelWarn, "Callback attempt failed"
	if persistErr != nil {
		level, message = slog.LevelError, "Callback outcome persistence failed"
	} else if delivered {
		level, message = slog.LevelInfo, "Callback delivery recovered"
	}
	slog.LogAttrs(ctx, level, message, attrs...)
}
