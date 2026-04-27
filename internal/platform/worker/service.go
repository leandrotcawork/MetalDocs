package worker

import (
	"context"
	"fmt"
	"log"
	"time"

	notificationapp "metaldocs/internal/modules/notifications/application"
	"metaldocs/internal/platform/config"
	"metaldocs/internal/platform/messaging"
)

type Service struct {
	consumer      messaging.Consumer
	notifications *notificationapp.Service
	pdfRunner     *PDFJobRunner
	cfg           config.WorkerConfig
}

func NewService(consumer messaging.Consumer, notifications *notificationapp.Service, cfg config.WorkerConfig) *Service {
	return &Service{
		consumer:      consumer,
		notifications: notifications,
		cfg:           cfg,
	}
}

// WithPDFRunner attaches a PDFJobRunner that handles docgen_v2_pdf events.
func (s *Service) WithPDFRunner(r *PDFJobRunner) *Service {
	s.pdfRunner = r
	return s
}

func (s *Service) RunOnce(ctx context.Context, batchSize int) error {
	if s.consumer == nil || s.notifications == nil {
		return fmt.Errorf("worker dependencies not configured")
	}

	start := time.Now()
	events, err := s.consumer.ClaimUnpublished(ctx, batchSize)
	if err != nil {
		return err
	}

	processed := 0
	failed := 0
	deadLettered := 0
	for _, event := range events {
		var handleErr error
		switch event.EventType {
		case "docgen_v2_pdf":
			if s.pdfRunner != nil {
				handleErr = s.pdfRunner.Handle(ctx, event)
			}
		default:
			handleErr = s.notifications.HandleEvent(ctx, event)
		}
		if handleErr != nil {
			failed++
			markedDLQ, markErr := s.markFailure(ctx, event, handleErr)
			if markErr != nil {
				return markErr
			}
			if markedDLQ {
				deadLettered++
			}
			continue
		}

		if err := s.consumer.MarkPublished(ctx, []string{event.EventID}); err != nil {
			return err
		}
		processed++
		log.Printf("worker_event event_id=%s event_type=%s attempt_count=%d result=published trace_id=%s",
			event.EventID, event.EventType, event.AttemptCount, event.TraceID)
	}

	if s.cfg.ReviewReminderDays > 0 {
		if err := s.notifications.EmitReviewReminders(ctx, s.cfg.ReviewReminderDays); err != nil {
			return err
		}
	}

	log.Printf("worker_batch result=completed processed=%d failed=%d dead_lettered=%d duration_ms=%d",
		processed, failed, deadLettered, time.Since(start).Milliseconds())
	return nil
}

func (s *Service) markFailure(ctx context.Context, event messaging.Event, handleErr error) (bool, error) {
	now := time.Now().UTC()
	attempt := event.AttemptCount
	if attempt < 1 {
		attempt = 1
	}

	failure := messaging.FailedEvent{
		EventID:   event.EventID,
		LastError: truncateError(handleErr),
	}

	if attempt >= s.cfg.MaxAttempts {
		failure.DeadLetteredAt = &now
		log.Printf("worker_event event_id=%s event_type=%s attempt_count=%d result=dead_lettered trace_id=%s error=%q",
			event.EventID, event.EventType, attempt, event.TraceID, failure.LastError)
	} else {
		nextAttempt := now.Add(backoffDuration(attempt, s.cfg.RetryBaseSeconds, s.cfg.RetryMaxSeconds))
		failure.NextAttemptAt = &nextAttempt
		log.Printf("worker_event event_id=%s event_type=%s attempt_count=%d result=retry_scheduled trace_id=%s next_attempt_at=%s error=%q",
			event.EventID, event.EventType, attempt, event.TraceID, nextAttempt.Format(time.RFC3339), failure.LastError)
	}

	if err := s.consumer.MarkFailed(ctx, failure); err != nil {
		return false, err
	}
	return failure.DeadLetteredAt != nil, nil
}

func backoffDuration(attempt, baseSeconds, maxSeconds int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	if baseSeconds < 1 {
		baseSeconds = 10
	}
	if maxSeconds < baseSeconds {
		maxSeconds = baseSeconds
	}

	multiplier := 1 << (attempt - 1)
	delaySeconds := baseSeconds * multiplier
	if delaySeconds > maxSeconds {
		delaySeconds = maxSeconds
	}
	return time.Duration(delaySeconds) * time.Second
}

func truncateError(err error) string {
	if err == nil {
		return ""
	}
	message := err.Error()
	if len(message) > 512 {
		return message[:512]
	}
	return message
}
