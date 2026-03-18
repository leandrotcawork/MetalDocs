package unit

import (
	"context"
	"fmt"
	"testing"
	"time"

	docapp "metaldocs/internal/modules/documents/application"
	docdomain "metaldocs/internal/modules/documents/domain"
	docmemory "metaldocs/internal/modules/documents/infrastructure/memory"
	notificationapp "metaldocs/internal/modules/notifications/application"
	notificationdomain "metaldocs/internal/modules/notifications/domain"
	notificationmemory "metaldocs/internal/modules/notifications/infrastructure/memory"
	"metaldocs/internal/platform/config"
	"metaldocs/internal/platform/messaging"
	workerapp "metaldocs/internal/platform/worker"
)

type fakeConsumer struct {
	events          []messaging.Event
	markedPublished []string
	failed          []messaging.FailedEvent
}

func (f *fakeConsumer) ClaimUnpublished(context.Context, int) ([]messaging.Event, error) {
	return append([]messaging.Event(nil), f.events...), nil
}

func (f *fakeConsumer) MarkPublished(_ context.Context, eventIDs []string) error {
	f.markedPublished = append([]string(nil), eventIDs...)
	return nil
}

type failingNotificationRepo struct{}

func (f failingNotificationRepo) Create(context.Context, notificationdomain.Notification) error {
	return fmt.Errorf("forced notification failure")
}

func (f *fakeConsumer) MarkFailed(_ context.Context, failure messaging.FailedEvent) error {
	f.failed = append(f.failed, failure)
	return nil
}

func TestNotificationServiceHandlesApprovalRequested(t *testing.T) {
	repo := notificationmemory.NewRepository()
	docRepo := docmemory.NewRepository()
	svc := notificationapp.NewService(repo, docRepo, fixedClock{now: time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)})

	err := svc.HandleEvent(context.Background(), messaging.Event{
		EventID:        "evt-1",
		EventType:      "workflow.approval.requested",
		IdempotencyKey: "approval-request-1",
		Payload: map[string]any{
			"document_id":       "doc-1",
			"assigned_reviewer": "reviewer-user",
		},
	})
	if err != nil {
		t.Fatalf("unexpected handle event error: %v", err)
	}

	items := repo.Items()
	if len(items) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(items))
	}
	if items[0].RecipientUserID != "reviewer-user" {
		t.Fatalf("expected reviewer-user, got %s", items[0].RecipientUserID)
	}
}

func TestNotificationServiceEmitsReviewReminderByExpiry(t *testing.T) {
	docRepo := docmemory.NewRepository()
	notifRepo := notificationmemory.NewRepository()
	docSvc := docapp.NewService(docRepo, nil, fixedClock{now: time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)})
	notifSvc := notificationapp.NewService(notifRepo, docRepo, fixedClock{now: time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)})

	expirySoon := time.Date(2026, 3, 25, 0, 0, 0, 0, time.UTC)
	doc, err := docSvc.CreateDocument(context.Background(), docdomain.CreateDocumentCommand{
		DocumentID:     "doc-reminder",
		Title:          "Published Manual",
		DocumentType:   "manual",
		OwnerID:        "owner-reminder",
		BusinessUnit:   "ops",
		Department:     "general",
		ExpiryAt:       &expirySoon,
		MetadataJSON:   map[string]any{"manual_code": "MAN-REM"},
		InitialContent: "v1",
	})
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}
	if err := docRepo.UpdateDocumentStatus(context.Background(), doc.ID, docdomain.StatusPublished); err != nil {
		t.Fatalf("unexpected update error: %v", err)
	}

	if err := notifSvc.EmitReviewReminders(context.Background(), 14); err != nil {
		t.Fatalf("unexpected reminder error: %v", err)
	}

	items := notifRepo.Items()
	if len(items) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(items))
	}
	if items[0].EventType != "document.review.reminder" {
		t.Fatalf("expected review reminder, got %s", items[0].EventType)
	}
}

func TestWorkerRunOnceProcessesOutboxAndMarksPublished(t *testing.T) {
	docRepo := docmemory.NewRepository()
	notifRepo := notificationmemory.NewRepository()
	notifSvc := notificationapp.NewService(notifRepo, docRepo, fixedClock{now: time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)})
	consumer := &fakeConsumer{
		events: []messaging.Event{
			{
				EventID:        "evt-2",
				EventType:      "workflow.approval.requested",
				IdempotencyKey: "approval-request-2",
				Payload: map[string]any{
					"document_id":       "doc-2",
					"assigned_reviewer": "reviewer-worker",
				},
			},
		},
	}
	workerSvc := workerapp.NewService(consumer, notifSvc, config.WorkerConfig{})

	if err := workerSvc.RunOnce(context.Background(), 10); err != nil {
		t.Fatalf("unexpected worker run error: %v", err)
	}

	if len(consumer.markedPublished) != 1 || consumer.markedPublished[0] != "evt-2" {
		t.Fatalf("expected evt-2 to be marked published, got %+v", consumer.markedPublished)
	}
	if len(notifRepo.Items()) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(notifRepo.Items()))
	}
}

func TestWorkerRunOnceMarksDeadLetterAfterMaxAttempts(t *testing.T) {
	docRepo := docmemory.NewRepository()
	notifRepo := failingNotificationRepo{}
	notifSvc := notificationapp.NewService(notifRepo, docRepo, fixedClock{now: time.Date(2026, 3, 20, 10, 0, 0, 0, time.UTC)})
	consumer := &fakeConsumer{
		events: []messaging.Event{
			{
				EventID:      "evt-dead",
				EventType:    "workflow.approval.requested",
				AttemptCount: 5,
				Payload: map[string]any{
					"document_id":       "doc-dead",
					"assigned_reviewer": "reviewer-dead",
				},
			},
		},
	}
	workerSvc := workerapp.NewService(consumer, notifSvc, config.WorkerConfig{
		MaxAttempts:      5,
		RetryBaseSeconds: 10,
		RetryMaxSeconds:  300,
	})

	if err := workerSvc.RunOnce(context.Background(), 10); err != nil {
		t.Fatalf("unexpected worker run error: %v", err)
	}

	if len(consumer.failed) != 1 {
		t.Fatalf("expected 1 failed event, got %d", len(consumer.failed))
	}
	if consumer.failed[0].DeadLetteredAt == nil {
		t.Fatalf("expected event to be dead-lettered")
	}
}
