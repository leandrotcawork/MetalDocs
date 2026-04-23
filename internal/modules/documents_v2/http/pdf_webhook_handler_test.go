package documentshttp

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type fakePDFWriter struct {
	calls       int
	tenant      string
	docID       string
	s3Key       string
	hash        []byte
	generatedAt time.Time
	err         error
}

func (f *fakePDFWriter) WritePDF(_ context.Context, tenant, docID, s3Key string, pdfHash []byte, generatedAt time.Time) error {
	if f.err != nil {
		return f.err
	}
	f.calls++
	f.tenant = tenant
	f.docID = docID
	f.s3Key = s3Key
	f.hash = append([]byte(nil), pdfHash...)
	f.generatedAt = generatedAt
	return nil
}

func sign(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return hex.EncodeToString(mac.Sum(nil))
}

func TestPDFWebhookHandler_ValidSignaturePersists(t *testing.T) {
	writer := &fakePDFWriter{}
	h := NewPDFWebhookHandler(writer, "shh")

	body := `{"tenant_id":"t-1","final_pdf_s3_key":"final/r.docx.pdf","pdf_hash":"abcd","pdf_generated_at":"2026-04-23T19:00:00Z"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v2/documents/doc-1/pdf-complete", strings.NewReader(body))
	req.SetPathValue("id", "doc-1")
	req.Header.Set("X-Docgen-Signature", sign([]byte(body), "shh"))
	rec := httptest.NewRecorder()

	h.HandlePDFComplete(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rec.Code, rec.Body.String())
	}
	if writer.calls != 1 {
		t.Fatalf("writer calls = %d", writer.calls)
	}
	if writer.tenant != "t-1" || writer.docID != "doc-1" || writer.s3Key != "final/r.docx.pdf" {
		t.Fatalf("wrong fields: %+v", writer)
	}
	if hex.EncodeToString(writer.hash) != "abcd" {
		t.Fatalf("hash = %x", writer.hash)
	}
	if writer.generatedAt.IsZero() {
		t.Fatal("generatedAt not set")
	}
}

func TestPDFWebhookHandler_MissingSignatureRejected401(t *testing.T) {
	h := NewPDFWebhookHandler(&fakePDFWriter{}, "shh")

	req := httptest.NewRequest(http.MethodPost, "/api/v2/documents/doc-1/pdf-complete",
		bytes.NewReader([]byte(`{"tenant_id":"t-1"}`)))
	req.SetPathValue("id", "doc-1")
	rec := httptest.NewRecorder()
	h.HandlePDFComplete(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want 401", rec.Code)
	}
}

func TestPDFWebhookHandler_WrongSignatureRejected401(t *testing.T) {
	h := NewPDFWebhookHandler(&fakePDFWriter{}, "shh")

	body := `{"tenant_id":"t-1"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v2/documents/doc-1/pdf-complete",
		strings.NewReader(body))
	req.SetPathValue("id", "doc-1")
	req.Header.Set("X-Docgen-Signature", sign([]byte(body), "other-secret"))
	rec := httptest.NewRecorder()
	h.HandlePDFComplete(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want 401", rec.Code)
	}
}

func TestPDFWebhookHandler_MalformedBodyRejected400(t *testing.T) {
	h := NewPDFWebhookHandler(&fakePDFWriter{}, "shh")

	body := `not-json`
	req := httptest.NewRequest(http.MethodPost, "/api/v2/documents/doc-1/pdf-complete",
		strings.NewReader(body))
	req.SetPathValue("id", "doc-1")
	req.Header.Set("X-Docgen-Signature", sign([]byte(body), "shh"))
	rec := httptest.NewRecorder()
	h.HandlePDFComplete(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", rec.Code)
	}
}
