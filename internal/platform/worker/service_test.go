package worker

import (
	"context"
	"testing"

	notificationapp "metaldocs/internal/modules/notifications/application"
	"metaldocs/internal/platform/config"
	"metaldocs/internal/platform/messaging"
	"metaldocs/internal/platform/servicebus"
)

type fakeConsumer struct {
	events []messaging.Event
}

func (f *fakeConsumer) ClaimUnpublished(_ context.Context, _ int) ([]messaging.Event, error) {
	return f.events, nil
}

func (f *fakeConsumer) MarkPublished(_ context.Context, _ []string) error { return nil }

func (f *fakeConsumer) MarkFailed(_ context.Context, _ messaging.FailedEvent) error { return nil }

func TestWorkerService_RoutesPDFEventToPDFRunner(t *testing.T) {
	consumer := &fakeConsumer{events: []messaging.Event{
		{
			EventID:   "e1",
			EventType: "docgen_v2_pdf",
			Payload: map[string]any{
				"tenant_id":         "t1",
				"revision_id":       "r1",
				"final_docx_s3_key": "tenants/t1/revisions/r1/frozen.docx",
			},
		},
	}}

	converter := &fakePDFConverter{result: servicebus.ConvertPDFResult{
		OutputKey:   "tenants/t1/revisions/r1/final.pdf",
		ContentHash: "deadbeef",
	}}
	persister := &fakePDFPersister{}
	runner := NewPDFJobRunner(converter, persister)

	cfg := config.WorkerConfig{MaxAttempts: 3, RetryBaseSeconds: 10, RetryMaxSeconds: 300}
	svc := NewService(consumer, new(notificationapp.Service), cfg).WithPDFRunner(runner)

	if err := svc.RunOnce(context.Background(), 10); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if len(persister.calls) != 1 {
		t.Errorf("PDFJobRunner.Handle called %d times, want 1", len(persister.calls))
	}
}
