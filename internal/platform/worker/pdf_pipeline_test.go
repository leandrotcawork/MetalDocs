package worker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	notificationapp "metaldocs/internal/modules/notifications/application"
	"metaldocs/internal/platform/config"
	"metaldocs/internal/platform/messaging"
	"metaldocs/internal/platform/servicebus"
)

// TestPDFPipeline_RealClient exercises the real DocgenV2Client.ConvertPDF against
// a fake HTTP server, then verifies PDFJobRunner writes the result via the persister.
// Tests the actual HTTP client path — not mocked at the interface level.
func TestPDFPipeline_RealClient(t *testing.T) {
	const (
		tenantID   = "tenant-abc"
		revisionID = "rev-xyz"
		docxKey    = "tenants/tenant-abc/revisions/rev-xyz/frozen.docx"
		outputKey  = "tenants/tenant-abc/revisions/rev-xyz/final.pdf"
		hexHash    = "deadbeef01234567"
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/convert/pdf" || r.Method != http.MethodPost {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		var req servicebus.ConvertPDFRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad body", http.StatusBadRequest)
			return
		}
		if req.DocxKey != docxKey {
			http.Error(w, "wrong docx_key: "+req.DocxKey, http.StatusBadRequest)
			return
		}
		if req.OutputKey != outputKey {
			http.Error(w, "wrong output_key: "+req.OutputKey, http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(servicebus.ConvertPDFResult{
			OutputKey:   outputKey,
			ContentHash: hexHash,
			SizeBytes:   12345,
		})
	}))
	defer srv.Close()

	client := servicebus.NewDocgenV2Client(srv.URL, "test-token", 10*time.Second)
	persister := &fakePDFPersister{}
	runner := NewPDFJobRunner(client, persister)

	event := makePDFEvent(map[string]any{
		"tenant_id":         tenantID,
		"revision_id":       revisionID,
		"final_docx_s3_key": docxKey,
	})
	if err := runner.Handle(context.Background(), event); err != nil {
		t.Fatalf("Handle: %v", err)
	}

	if len(persister.calls) != 1 {
		t.Fatalf("WritePDF called %d times, want 1", len(persister.calls))
	}
	call := persister.calls[0]
	if call.tenant != tenantID {
		t.Errorf("tenant = %q, want %q", call.tenant, tenantID)
	}
	if call.docID != revisionID {
		t.Errorf("docID = %q, want %q", call.docID, revisionID)
	}
	if call.s3Key != outputKey {
		t.Errorf("s3Key = %q, want %q", call.s3Key, outputKey)
	}
	if len(call.pdfHash) == 0 {
		t.Error("pdfHash empty")
	}
}

func TestPDFPipeline_RealClient_DocgenError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"NoSuchKey"}`, http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := servicebus.NewDocgenV2Client(srv.URL, "", 10*time.Second)
	persister := &fakePDFPersister{}
	runner := NewPDFJobRunner(client, persister)

	err := runner.Handle(context.Background(), makePDFEvent(map[string]any{
		"tenant_id":         "t1",
		"revision_id":       "r1",
		"final_docx_s3_key": "some/key.docx",
	}))
	if err == nil {
		t.Fatal("expected error from docgen, got nil")
	}
	if len(persister.calls) != 0 {
		t.Error("WritePDF must not be called when convert fails")
	}
}

// TestPDFPipeline_WorkerLoop verifies the full Service.RunOnce → PDFJobRunner →
// DocgenV2Client → WritePDF chain using a fake HTTP server for docgen-v2.
func TestPDFPipeline_WorkerLoop(t *testing.T) {
	const (
		tenantID   = "t-loop"
		revisionID = "r-loop"
		docxKey    = "tenants/t-loop/revisions/r-loop/frozen.docx"
		outputKey  = "tenants/t-loop/revisions/r-loop/final.pdf"
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(servicebus.ConvertPDFResult{
			OutputKey:   outputKey,
			ContentHash: "cafebabe",
		})
	}))
	defer srv.Close()

	consumer := &fakeConsumer{events: []messaging.Event{
		makePDFEvent(map[string]any{
			"tenant_id":         tenantID,
			"revision_id":       revisionID,
			"final_docx_s3_key": docxKey,
		}),
	}}
	client := servicebus.NewDocgenV2Client(srv.URL, "", 10*time.Second)
	persister := &fakePDFPersister{}
	runner := NewPDFJobRunner(client, persister)

	cfg := config.WorkerConfig{MaxAttempts: 3, RetryBaseSeconds: 10, RetryMaxSeconds: 300}
	svc := NewService(consumer, new(notificationapp.Service), cfg).WithPDFRunner(runner)

	if err := svc.RunOnce(context.Background(), 10); err != nil {
		t.Fatalf("RunOnce: %v", err)
	}
	if len(persister.calls) != 1 {
		t.Errorf("WritePDF called %d times, want 1", len(persister.calls))
	}
}
