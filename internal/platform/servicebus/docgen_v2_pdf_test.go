package servicebus_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"metaldocs/internal/platform/servicebus"
)

func TestDocgenV2Client_ConvertPDF_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/convert/pdf" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Fatalf("unexpected method %s", r.Method)
		}
		if got := r.Header.Get("X-Service-Token"); got != "tok" {
			t.Fatalf("unexpected x-service-token %q", got)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"output_key":"out.pdf","content_hash":"abc123","size_bytes":1024,"docgen_v2_version":"docgen-v2@0.4.0"}`))
	}))
	defer srv.Close()

	c := servicebus.NewDocgenV2Client(srv.URL, "tok", 5*time.Second)
	res, err := c.ConvertPDF(context.Background(), servicebus.ConvertPDFRequest{
		DocxKey:   "in.docx",
		OutputKey: "out.pdf",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.OutputKey != "out.pdf" {
		t.Fatalf("expected output key out.pdf, got %q", res.OutputKey)
	}
	if res.ContentHash != "abc123" {
		t.Fatalf("expected content hash abc123, got %q", res.ContentHash)
	}
}

func TestDocgenV2Client_ConvertPDF_Non200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error":"gotenberg_failed"}`))
	}))
	defer srv.Close()

	c := servicebus.NewDocgenV2Client(srv.URL, "tok", 5*time.Second)
	_, err := c.ConvertPDF(context.Background(), servicebus.ConvertPDFRequest{
		DocxKey:   "in.docx",
		OutputKey: "out.pdf",
	})
	if err == nil {
		t.Fatal("expected error on 502")
	}
}
