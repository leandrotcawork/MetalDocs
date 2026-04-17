package application

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCK5ExportClient_RenderDocx_OK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want %s", r.Method, http.MethodPost)
		}
		if r.URL.Path != "/render/docx" {
			t.Fatalf("path = %s, want %s", r.URL.Path, "/render/docx")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("PK\x03\x04stub"))
	}))
	defer server.Close()

	client := NewCK5ExportClient(server.URL)

	got, err := client.RenderDocx(context.Background(), "<p>hello</p>")
	if err != nil {
		t.Fatalf("RenderDocx() error = %v", err)
	}
	if string(got) != "PK\x03\x04stub" {
		t.Fatalf("RenderDocx() = %q, want %q", got, "PK\x03\x04stub")
	}
}

func TestCK5ExportClient_RenderDocx_Non200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("server error"))
	}))
	defer server.Close()

	client := NewCK5ExportClient(server.URL)

	_, err := client.RenderDocx(context.Background(), "<p>hello</p>")
	var exportErr *CK5ExportError
	if !errors.As(err, &exportErr) {
		t.Fatalf("RenderDocx() error = %v, want *CK5ExportError", err)
	}
	if exportErr.Status != http.StatusInternalServerError {
		t.Fatalf("CK5ExportError.Status = %d, want %d", exportErr.Status, http.StatusInternalServerError)
	}
}

func TestCK5ExportClient_RenderDocx_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("late"))
	}))
	defer server.Close()

	client := newCK5ExportClientWithHTTP(server.URL, &http.Client{Timeout: time.Millisecond})

	_, err := client.RenderDocx(context.Background(), "<p>hello</p>")
	if err == nil {
		t.Fatal("RenderDocx() error = nil, want timeout error")
	}
}

func TestCK5ExportClient_RenderPDFHtml_OK(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want %s", r.Method, http.MethodPost)
		}
		if r.URL.Path != "/render/pdf-html" {
			t.Fatalf("path = %s, want %s", r.URL.Path, "/render/pdf-html")
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("<!DOCTYPE html>"))
	}))
	defer server.Close()

	client := NewCK5ExportClient(server.URL)

	got, err := client.RenderPDFHtml(context.Background(), "<p>hello</p>")
	if err != nil {
		t.Fatalf("RenderPDFHtml() error = %v", err)
	}
	if got != "<!DOCTYPE html>" {
		t.Fatalf("RenderPDFHtml() = %q, want %q", got, "<!DOCTYPE html>")
	}
}

func TestCK5ExportClient_RenderPDFHtml_Non200(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		_, _ = w.Write([]byte("unprocessable"))
	}))
	defer server.Close()

	client := NewCK5ExportClient(server.URL)

	_, err := client.RenderPDFHtml(context.Background(), "<p>hello</p>")
	var exportErr *CK5ExportError
	if !errors.As(err, &exportErr) {
		t.Fatalf("RenderPDFHtml() error = %v, want *CK5ExportError", err)
	}
	if exportErr.Status != http.StatusUnprocessableEntity {
		t.Fatalf("CK5ExportError.Status = %d, want %d", exportErr.Status, http.StatusUnprocessableEntity)
	}
}
