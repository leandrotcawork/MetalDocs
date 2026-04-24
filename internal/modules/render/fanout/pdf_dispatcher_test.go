package fanout

import (
	"context"
	"errors"
	"testing"

	"metaldocs/internal/platform/messaging"
)

type fakePublisher struct {
	events []messaging.Event
	err    error
}

func (f *fakePublisher) Publish(_ context.Context, e messaging.Event) error {
	if f.err != nil {
		return f.err
	}
	f.events = append(f.events, e)
	return nil
}

func TestPDFDispatcher_Dispatch_PublishesEvent(t *testing.T) {
	pub := &fakePublisher{}
	d := NewPDFDispatcher(pub)

	err := d.Dispatch(context.Background(), DispatchInput{
		TenantID:       "tenant-1",
		RevisionID:     "rev-1",
		FinalDocxS3Key: "final/rev-1.docx",
	})
	if err != nil {
		t.Fatalf("dispatch err: %v", err)
	}
	if len(pub.events) != 1 {
		t.Fatalf("events = %d, want 1", len(pub.events))
	}
	e := pub.events[0]
	if e.EventType != "docgen_v2_pdf" {
		t.Errorf("event_type = %q, want docgen_v2_pdf", e.EventType)
	}
	if e.Payload["tenant_id"] != "tenant-1" {
		t.Errorf("payload tenant_id = %v", e.Payload["tenant_id"])
	}
	if e.Payload["revision_id"] != "rev-1" {
		t.Errorf("payload revision_id = %v", e.Payload["revision_id"])
	}
	if e.Payload["final_docx_s3_key"] != "final/rev-1.docx" {
		t.Errorf("payload final_docx_s3_key = %v", e.Payload["final_docx_s3_key"])
	}
}

func TestPDFDispatcher_Dispatch_PropagatesErr(t *testing.T) {
	pub := &fakePublisher{err: errors.New("bus down")}
	d := NewPDFDispatcher(pub)

	err := d.Dispatch(context.Background(), DispatchInput{
		TenantID:       "t",
		RevisionID:     "r",
		FinalDocxS3Key: "k",
	})
	if err == nil {
		t.Fatal("expected err")
	}
}
