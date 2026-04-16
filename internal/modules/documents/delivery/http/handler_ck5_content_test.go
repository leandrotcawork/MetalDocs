package httpdelivery

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"metaldocs/internal/modules/documents/application"
	"metaldocs/internal/modules/documents/domain"
	"metaldocs/internal/modules/documents/infrastructure/memory"
)

func setupCK5ContentHandler(t *testing.T) (*Handler, *memory.Repository) {
	t.Helper()
	repo := memory.NewRepository()
	svc := application.NewService(repo, nil, nil)
	h := NewHandler(svc)
	return h, repo
}

func seedCK5Doc(t *testing.T, repo *memory.Repository, docID, html string) {
	t.Helper()
	ctx := context.Background()
	err := repo.CreateDocumentWithInitialVersion(ctx,
		domain.Document{ID: docID, DocumentProfile: "po", Status: domain.StatusDraft},
		domain.Version{
			DocumentID:    docID,
			Number:        1,
			Content:       html,
			ContentHash:   "h1",
			ContentSource: domain.ContentSourceNative,
		},
	)
	if err != nil {
		t.Fatalf("seed document: %v", err)
	}
}

func TestGetCK5Content_200(t *testing.T) {
	h, repo := setupCK5ContentHandler(t)
	seedCK5Doc(t, repo, "d1", "<p>CK5 content</p>")

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/documents/d1/content/ck5", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200; body: %s", w.Code, w.Body.String())
	}
	var resp map[string]string
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp["body"] != "<p>CK5 content</p>" {
		t.Errorf("got body=%q, want <p>CK5 content</p>", resp["body"])
	}
}

func TestGetCK5Content_404(t *testing.T) {
	h, _ := setupCK5ContentHandler(t)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/documents/missing/content/ck5", nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("got %d, want 404", w.Code)
	}
}

func TestPostCK5Content_201(t *testing.T) {
	h, repo := setupCK5ContentHandler(t)
	seedCK5Doc(t, repo, "d2", "<p>Old</p>")

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	body, _ := json.Marshal(map[string]string{"body": "<p>New</p>"})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/documents/d2/content/ck5",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("got %d, want 201; body: %s", w.Code, w.Body.String())
	}
}

func TestPostCK5Content_MissingBody(t *testing.T) {
	h, repo := setupCK5ContentHandler(t)
	seedCK5Doc(t, repo, "d3", "<p>x</p>")

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	body, _ := json.Marshal(map[string]string{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/documents/d3/content/ck5",
		bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("got %d, want 400", w.Code)
	}
}
