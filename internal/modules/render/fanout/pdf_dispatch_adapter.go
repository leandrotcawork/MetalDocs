package fanout

import (
	"context"
	"fmt"
)

// DocxKeyReader reads the frozen DOCX S3 key for a document from storage.
type DocxKeyReader interface {
	ReadFinalDocxS3Key(ctx context.Context, tenantID, docID string) (string, error)
}

// PDFDispatchAdapter implements approval/application.PDFDispatchInvoker.
// It reads the frozen DOCX S3 key from the DB and delegates to PDFDispatcher.
type PDFDispatchAdapter struct {
	dispatcher *PDFDispatcher
	reader     DocxKeyReader
}

func NewPDFDispatchAdapter(dispatcher *PDFDispatcher, reader DocxKeyReader) *PDFDispatchAdapter {
	return &PDFDispatchAdapter{dispatcher: dispatcher, reader: reader}
}

func (a *PDFDispatchAdapter) Dispatch(ctx context.Context, tenantID, revisionID string) error {
	key, err := a.reader.ReadFinalDocxS3Key(ctx, tenantID, revisionID)
	if err != nil {
		return fmt.Errorf("pdf dispatch adapter: read docx key: %w", err)
	}
	return a.dispatcher.Dispatch(ctx, DispatchInput{
		TenantID:       tenantID,
		RevisionID:     revisionID,
		FinalDocxS3Key: key,
	})
}
