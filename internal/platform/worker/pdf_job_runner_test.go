package worker

import (
	"context"
	"encoding/hex"
	"errors"
	"testing"
	"time"

	"metaldocs/internal/platform/messaging"
	"metaldocs/internal/platform/servicebus"
)

type fakePDFConverter struct {
	calls  int
	req    servicebus.ConvertPDFRequest
	result servicebus.ConvertPDFResult
	err    error
}

func (f *fakePDFConverter) ConvertPDF(_ context.Context, req servicebus.ConvertPDFRequest) (servicebus.ConvertPDFResult, error) {
	f.calls++
	f.req = req
	if f.err != nil {
		return servicebus.ConvertPDFResult{}, f.err
	}
	return f.result, nil
}

type fakePDFPersister struct {
	calls []pdfPersistCall
	err   error
}

type pdfPersistCall struct {
	tenant      string
	docID       string
	s3Key       string
	pdfHash     []byte
	generatedAt time.Time
}

func (f *fakePDFPersister) WritePDF(_ context.Context, tenant, docID, s3Key string, pdfHash []byte, generatedAt time.Time) error {
	f.calls = append(f.calls, pdfPersistCall{
		tenant:      tenant,
		docID:       docID,
		s3Key:       s3Key,
		pdfHash:     append([]byte(nil), pdfHash...),
		generatedAt: generatedAt,
	})
	return f.err
}

func makePDFEvent(payload map[string]any) messaging.Event {
	return messaging.Event{
		EventID:   "event-1",
		EventType: "docgen_v2_pdf",
		Payload:   payload,
	}
}

func TestPDFJobRunner_Handle_Success(t *testing.T) {
	hash := "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc"
	converter := &fakePDFConverter{
		result: servicebus.ConvertPDFResult{
			OutputKey:   "tenants/tenant-1/revisions/rev-1/final.pdf",
			ContentHash: hash,
		},
	}
	persister := &fakePDFPersister{}
	runner := NewPDFJobRunner(converter, persister)

	err := runner.Handle(context.Background(), makePDFEvent(map[string]any{
		"tenant_id":         "tenant-1",
		"revision_id":       "rev-1",
		"final_docx_s3_key": "tenants/tenant-1/revisions/rev-1/final.docx",
	}))
	if err != nil {
		t.Fatalf("Handle: %v", err)
	}

	if converter.calls != 1 {
		t.Fatalf("ConvertPDF calls = %d, want 1", converter.calls)
	}
	if converter.req.DocxKey != "tenants/tenant-1/revisions/rev-1/final.docx" {
		t.Fatalf("DocxKey = %q", converter.req.DocxKey)
	}
	if converter.req.OutputKey != "tenants/tenant-1/revisions/rev-1/final.pdf" {
		t.Fatalf("OutputKey = %q", converter.req.OutputKey)
	}

	if len(persister.calls) != 1 {
		t.Fatalf("WritePDF calls = %d, want 1", len(persister.calls))
	}
	call := persister.calls[0]
	if call.tenant != "tenant-1" || call.docID != "rev-1" || call.s3Key != converter.result.OutputKey {
		t.Fatalf("WritePDF args = tenant %q docID %q s3Key %q", call.tenant, call.docID, call.s3Key)
	}
	wantHash, err := hex.DecodeString(hash)
	if err != nil {
		t.Fatalf("decode hash: %v", err)
	}
	if hex.EncodeToString(call.pdfHash) != hex.EncodeToString(wantHash) {
		t.Fatalf("pdfHash = %x, want %x", call.pdfHash, wantHash)
	}
	if call.generatedAt.IsZero() {
		t.Fatalf("generatedAt is zero")
	}
	if call.generatedAt.Location() != time.UTC {
		t.Fatalf("generatedAt location = %v, want UTC", call.generatedAt.Location())
	}
}

func TestPDFJobRunner_Handle_MissingPayloadFields(t *testing.T) {
	converter := &fakePDFConverter{}
	persister := &fakePDFPersister{}
	runner := NewPDFJobRunner(converter, persister)

	err := runner.Handle(context.Background(), makePDFEvent(map[string]any{
		"tenant_id":         "tenant-1",
		"revision_id":       "",
		"final_docx_s3_key": "tenants/tenant-1/revisions/rev-1/final.docx",
	}))
	if err == nil {
		t.Fatalf("Handle error = nil, want error")
	}
	if converter.calls != 0 {
		t.Fatalf("ConvertPDF calls = %d, want 0", converter.calls)
	}
	if len(persister.calls) != 0 {
		t.Fatalf("WritePDF calls = %d, want 0", len(persister.calls))
	}
}

func TestPDFJobRunner_Handle_ConvertError(t *testing.T) {
	convertErr := errors.New("convert failed")
	converter := &fakePDFConverter{err: convertErr}
	persister := &fakePDFPersister{}
	runner := NewPDFJobRunner(converter, persister)

	err := runner.Handle(context.Background(), makePDFEvent(map[string]any{
		"tenant_id":         "tenant-1",
		"revision_id":       "rev-1",
		"final_docx_s3_key": "tenants/tenant-1/revisions/rev-1/final.docx",
	}))
	if !errors.Is(err, convertErr) {
		t.Fatalf("Handle error = %v, want wrapped convert error", err)
	}
	if len(persister.calls) != 0 {
		t.Fatalf("WritePDF calls = %d, want 0", len(persister.calls))
	}
}

func TestPDFJobRunner_Handle_PersistError(t *testing.T) {
	persistErr := errors.New("persist failed")
	converter := &fakePDFConverter{
		result: servicebus.ConvertPDFResult{
			OutputKey:   "tenants/tenant-1/revisions/rev-1/final.pdf",
			ContentHash: "cccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccccc",
		},
	}
	persister := &fakePDFPersister{err: persistErr}
	runner := NewPDFJobRunner(converter, persister)

	err := runner.Handle(context.Background(), makePDFEvent(map[string]any{
		"tenant_id":         "tenant-1",
		"revision_id":       "rev-1",
		"final_docx_s3_key": "tenants/tenant-1/revisions/rev-1/final.docx",
	}))
	if !errors.Is(err, persistErr) {
		t.Fatalf("Handle error = %v, want wrapped persist error", err)
	}
	if len(persister.calls) != 1 {
		t.Fatalf("WritePDF calls = %d, want 1", len(persister.calls))
	}
}
