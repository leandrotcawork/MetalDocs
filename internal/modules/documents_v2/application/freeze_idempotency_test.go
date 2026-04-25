package application

import (
	"context"
	"testing"
	"time"

	v2dom "metaldocs/internal/modules/documents_v2/domain"
	"metaldocs/internal/modules/render/resolvers"
)

func TestFreezeService_Freeze_IdempotentWhenAlreadyFrozen(t *testing.T) {
	frozenAt := time.Date(2026, 4, 23, 10, 0, 0, 0, time.UTC)
	valuesRead := &fakeValuesReader{}
	finalize := &fakeFreezeFinalizer{}
	finalDocx := &fakeFinalDocxWriter{}

	svc := NewFreezeService(
		fakeSchemaReader{},
		&fakeFillInWriter{},
		valuesRead,
		resolvers.NewRegistry(),
		finalize,
		&fakeResolverContextBuilder{},
		fakeSnapshotReader{
			snap:           v2dom.TemplateSnapshot{BodyDocxS3Key: "body.docx"},
			valuesFrozenAt: &frozenAt,
		},
		finalDocx,
		&fakeFanoutClient{},
	)

	if err := svc.Freeze(context.Background(), nil, "tenant-1", "doc-1", ApproverContext{}); err != nil {
		t.Fatalf("Freeze() error = %v", err)
	}
	if valuesRead.calls != 0 {
		t.Fatalf("ListValues should not be called, got %d call(s)", valuesRead.calls)
	}
	if finalize.calls != 0 {
		t.Fatalf("WriteFreeze should not be called, got %d call(s)", finalize.calls)
	}
	if finalDocx.calls != 0 {
		t.Fatalf("WriteFinalDocx should not be called, got %d call(s)", finalDocx.calls)
	}
}
