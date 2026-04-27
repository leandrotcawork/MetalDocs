package fanout

import (
	"context"
	"errors"
	"testing"
)

type fakeDocxKeyReader struct {
	key string
	err error
}

func (f *fakeDocxKeyReader) ReadFinalDocxS3Key(_ context.Context, _, _ string) (string, error) {
	return f.key, f.err
}

func TestPDFDispatchAdapter_Dispatch_Success(t *testing.T) {
	pub := &fakePublisher{}
	dispatcher := NewPDFDispatcher(pub)
	reader := &fakeDocxKeyReader{key: "tenants/t1/revisions/r1/frozen.docx"}

	adapter := NewPDFDispatchAdapter(dispatcher, reader)
	err := adapter.Dispatch(context.Background(), "t1", "r1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(pub.events) != 1 {
		t.Fatalf("events = %d, want 1", len(pub.events))
	}
	e := pub.events[0]
	if e.EventType != "docgen_v2_pdf" {
		t.Errorf("event_type = %q, want docgen_v2_pdf", e.EventType)
	}
	if e.Payload["final_docx_s3_key"] != "tenants/t1/revisions/r1/frozen.docx" {
		t.Errorf("payload final_docx_s3_key = %v", e.Payload["final_docx_s3_key"])
	}
	if e.Payload["tenant_id"] != "t1" || e.Payload["revision_id"] != "r1" {
		t.Errorf("payload tenant/revision mismatch: %v", e.Payload)
	}
}

func TestPDFDispatchAdapter_Dispatch_ReaderError(t *testing.T) {
	pub := &fakePublisher{}
	dispatcher := NewPDFDispatcher(pub)
	reader := &fakeDocxKeyReader{err: errors.New("not frozen")}

	adapter := NewPDFDispatchAdapter(dispatcher, reader)
	err := adapter.Dispatch(context.Background(), "t1", "r1")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if len(pub.events) != 0 {
		t.Error("should not publish when reader fails")
	}
}

func TestPDFDispatchAdapter_Dispatch_PublishError(t *testing.T) {
	pub := &fakePublisher{err: errors.New("bus down")}
	dispatcher := NewPDFDispatcher(pub)
	reader := &fakeDocxKeyReader{key: "some/key.docx"}

	adapter := NewPDFDispatchAdapter(dispatcher, reader)
	err := adapter.Dispatch(context.Background(), "t1", "r1")
	if err == nil {
		t.Fatal("expected error from publisher, got nil")
	}
}
