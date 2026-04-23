package fanout

import (
	"context"

	"metaldocs/internal/platform/messaging"
)

// DispatchInput carries the payload for the docgen_v2_pdf job.
type DispatchInput struct {
	TenantID       string
	RevisionID     string
	FinalDocxS3Key string
}

// PDFDispatcher enqueues docgen_v2_pdf jobs onto the platform event bus.
// Dispatch is called AFTER the approval transaction commits — failures are
// best-effort and never roll back the freeze.
type PDFDispatcher struct {
	pub messaging.Publisher
}

func NewPDFDispatcher(pub messaging.Publisher) *PDFDispatcher {
	return &PDFDispatcher{pub: pub}
}

func (d *PDFDispatcher) Dispatch(ctx context.Context, in DispatchInput) error {
	return d.pub.Publish(ctx, messaging.Event{
		EventType:     "docgen_v2_pdf",
		AggregateType: "document_revision",
		AggregateID:   in.RevisionID,
		Payload: map[string]any{
			"tenant_id":         in.TenantID,
			"revision_id":       in.RevisionID,
			"final_docx_s3_key": in.FinalDocxS3Key,
		},
	})
}
