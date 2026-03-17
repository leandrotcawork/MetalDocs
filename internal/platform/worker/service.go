package worker

import (
	"context"
	"fmt"

	notificationapp "metaldocs/internal/modules/notifications/application"
	"metaldocs/internal/platform/messaging"
)

type Service struct {
	consumer           messaging.Consumer
	notifications      *notificationapp.Service
	reviewReminderDays int
}

func NewService(consumer messaging.Consumer, notifications *notificationapp.Service, reviewReminderDays int) *Service {
	return &Service{
		consumer:           consumer,
		notifications:      notifications,
		reviewReminderDays: reviewReminderDays,
	}
}

func (s *Service) RunOnce(ctx context.Context, batchSize int) error {
	if s.consumer == nil || s.notifications == nil {
		return fmt.Errorf("worker dependencies not configured")
	}

	events, err := s.consumer.ClaimUnpublished(ctx, batchSize)
	if err != nil {
		return err
	}
	if len(events) > 0 {
		claimedIDs := collectEventIDs(events)
		for _, event := range events {
			if err := s.notifications.HandleEvent(ctx, event); err != nil {
				_ = s.consumer.Release(ctx, claimedIDs)
				return err
			}
		}
		if err := s.consumer.MarkPublished(ctx, claimedIDs); err != nil {
			_ = s.consumer.Release(ctx, claimedIDs)
			return err
		}
	}

	if s.reviewReminderDays > 0 {
		if err := s.notifications.EmitReviewReminders(ctx, s.reviewReminderDays); err != nil {
			return err
		}
	}
	return nil
}

func collectEventIDs(events []messaging.Event) []string {
	out := make([]string, 0, len(events))
	for _, event := range events {
		out = append(out, event.EventID)
	}
	return out
}
