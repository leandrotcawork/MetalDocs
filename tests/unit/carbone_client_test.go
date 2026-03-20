package unit

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"metaldocs/internal/platform/config"
	"metaldocs/internal/platform/render/carbone"
)

func TestCarboneClientRegisterRenderDownload(t *testing.T) {
	t.Helper()

	var traceSeen string
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/template", func(w http.ResponseWriter, r *http.Request) {
		traceSeen = r.Header.Get("X-Trace-Id")
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		if err := r.ParseMultipartForm(8 << 20); err != nil {
			t.Fatalf("parse multipart: %v", err)
		}
		file, _, err := r.FormFile("template")
		if err != nil {
			t.Fatalf("missing template form file: %v", err)
		}
		defer func() {
			_ = file.Close()
		}()
		_, _ = io.ReadAll(file)
		respondJSON(w, map[string]any{
			"success": true,
			"data": map[string]any{
				"templateId": "template-123",
			},
		})
	})
	mux.HandleFunc("/render/template-123", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("expected POST, got %s", r.Method)
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), `"convertTo":"pdf"`) {
			t.Fatalf("expected convertTo in render payload")
		}
		respondJSON(w, map[string]any{
			"success": true,
			"data": map[string]any{
				"renderId": "render-123",
			},
		})
	})
	mux.HandleFunc("/render/render-123", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("%PDF-1.4"))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	cfg := config.CarboneConfig{
		Enabled:               true,
		APIURL:                server.URL,
		RequestTimeoutSeconds: 2,
	}
	client := carbone.NewClient(cfg)
	if client == nil {
		t.Fatalf("expected client")
	}

	if err := client.Ping(context.Background(), "trace-health"); err != nil {
		t.Fatalf("ping: %v", err)
	}

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "template.docx")
	if err := os.WriteFile(tmpFile, []byte("docx"), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}

	templateID, err := client.RegisterTemplate(context.Background(), "trace-123", tmpFile)
	if err != nil {
		t.Fatalf("register template: %v", err)
	}
	if templateID != "template-123" {
		t.Fatalf("unexpected template id: %s", templateID)
	}
	if traceSeen != "trace-123" {
		t.Fatalf("expected trace header")
	}

	renderID, err := client.RenderTemplate(context.Background(), "trace-234", "template-123", map[string]any{"name": "test"}, "pdf")
	if err != nil {
		t.Fatalf("render template: %v", err)
	}
	if renderID != "render-123" {
		t.Fatalf("unexpected render id: %s", renderID)
	}

	payload, err := client.DownloadRender(context.Background(), "trace-345", "render-123")
	if err != nil {
		t.Fatalf("download render: %v", err)
	}
	if !strings.HasPrefix(string(payload), "%PDF") {
		t.Fatalf("unexpected payload")
	}
}

func TestCarboneClientRegisterErrorsOnBadResponse(t *testing.T) {
	t.Helper()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/template" {
			respondJSON(w, map[string]any{"success": false, "error": "nope"})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	cfg := config.CarboneConfig{
		Enabled:               true,
		APIURL:                server.URL,
		RequestTimeoutSeconds: 2,
	}
	client := carbone.NewClient(cfg)

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "template.docx")
	if err := os.WriteFile(tmpFile, []byte("docx"), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	if _, err := client.RegisterTemplate(context.Background(), "trace-err", tmpFile); err == nil {
		t.Fatalf("expected error")
	}
}

func respondJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}
