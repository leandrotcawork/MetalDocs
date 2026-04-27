package worker

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"metaldocs/internal/platform/messaging"
	"metaldocs/internal/platform/servicebus"
)

type PDFConverter interface {
	ConvertPDF(ctx context.Context, req servicebus.ConvertPDFRequest) (servicebus.ConvertPDFResult, error)
}

type PDFPersister interface {
	WritePDF(ctx context.Context, tenant, docID, s3Key string, pdfHash []byte, generatedAt time.Time) error
}

type PDFJobRunner struct {
	converter PDFConverter
	persister PDFPersister
}

func NewPDFJobRunner(converter PDFConverter, persister PDFPersister) *PDFJobRunner {
	return &PDFJobRunner{
		converter: converter,
		persister: persister,
	}
}

func (r *PDFJobRunner) Handle(ctx context.Context, event messaging.Event) error {
	tenantID, _ := event.Payload["tenant_id"].(string)
	revisionID, _ := event.Payload["revision_id"].(string)
	docxKey, _ := event.Payload["final_docx_s3_key"].(string)
	if tenantID == "" || revisionID == "" || docxKey == "" {
		return fmt.Errorf("pdf job runner: missing payload fields")
	}

	outputKey := fmt.Sprintf("tenants/%s/revisions/%s/final.pdf", tenantID, revisionID)
	result, err := r.converter.ConvertPDF(ctx, servicebus.ConvertPDFRequest{
		DocxKey:   docxKey,
		OutputKey: outputKey,
	})
	if err != nil {
		return fmt.Errorf("pdf job runner: convert pdf: %w", err)
	}

	hashBytes, err := hex.DecodeString(result.ContentHash)
	if err != nil {
		return fmt.Errorf("pdf job runner: decode content hash: %w", err)
	}

	if err := r.persister.WritePDF(ctx, tenantID, revisionID, result.OutputKey, hashBytes, time.Now().UTC()); err != nil {
		return fmt.Errorf("pdf job runner: persist pdf: %w", err)
	}
	return nil
}
