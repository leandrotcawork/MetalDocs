package httpdelivery

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"metaldocs/internal/modules/documents/application"
	"metaldocs/internal/modules/documents/domain"
	"metaldocs/internal/modules/documents/infrastructure/memory"
	iamdomain "metaldocs/internal/modules/iam/domain"
)

func setupCK5TemplateHandler(t *testing.T) (*Handler, *memory.Repository) {
	t.Helper()
	repo := memory.NewRepository()
	svc := application.NewService(repo, nil, nil)
	return NewHandler(svc), repo
}

func withTemplateAuth(req *http.Request) *http.Request {
	return req.WithContext(iamdomain.WithAuthContext(req.Context(), "actor-admin", []iamdomain.Role{iamdomain.RoleAdmin}))
}

func seedTemplateDraft(t *testing.T, repo *memory.Repository, key string) {
	t.Helper()
	blocks, err := json.Marshal(map[string]any{
		"type":    "doc",
		"content": []any{},
	})
	if err != nil {
		t.Fatalf("marshal seed blocks: %v", err)
	}

	err = repo.UpsertTemplateDraftForTest(&domain.TemplateDraft{
		TemplateKey: key,
		ProfileCode: "po",
		Name:        "CK5 Draft",
		BlocksJSON:  blocks,
		LockVersion: 1,
		CreatedBy:   "seed",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("seed draft: %v", err)
	}
}

func TestGetCK5TemplateDraft_200_EmptyInitially(t *testing.T) {
	h, repo := setupCK5TemplateHandler(t)
	seedTemplateDraft(t, repo, "k1")

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/templates/k1/ck5-draft", nil)
	req = withTemplateAuth(req)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("got %d, want 200; body: %s", w.Code, w.Body.String())
	}

	var resp struct {
		ContentHTML string         `json:"contentHtml"`
		Manifest    map[string]any `json:"manifest"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.ContentHTML != "" {
		t.Fatalf("contentHtml = %q, want empty", resp.ContentHTML)
	}
	if resp.Manifest == nil {
		t.Fatal("manifest should be non-nil")
	}
}

func TestGetCK5TemplateDraft_404(t *testing.T) {
	h, _ := setupCK5TemplateHandler(t)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/templates/missing/ck5-draft", nil)
	req = withTemplateAuth(req)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("got %d, want 404; body: %s", w.Code, w.Body.String())
	}
}

func TestPutCK5TemplateDraft_200(t *testing.T) {
	h, repo := setupCK5TemplateHandler(t)
	seedTemplateDraft(t, repo, "k2")

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	putBody, _ := json.Marshal(map[string]any{
		"contentHtml": "<p>CK5</p>",
		"manifest":    map[string]any{},
	})
	putReq := httptest.NewRequest(http.MethodPut, "/api/v1/templates/k2/ck5-draft", bytes.NewReader(putBody))
	putReq.Header.Set("Content-Type", "application/json")
	putReq = withTemplateAuth(putReq)
	putW := httptest.NewRecorder()
	mux.ServeHTTP(putW, putReq)

	if putW.Code != http.StatusOK {
		t.Fatalf("PUT got %d, want 200; body: %s", putW.Code, putW.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/templates/k2/ck5-draft", nil)
	getReq = withTemplateAuth(getReq)
	getW := httptest.NewRecorder()
	mux.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Fatalf("GET got %d, want 200; body: %s", getW.Code, getW.Body.String())
	}

	var resp struct {
		ContentHTML string         `json:"contentHtml"`
		Manifest    map[string]any `json:"manifest"`
	}
	if err := json.NewDecoder(getW.Body).Decode(&resp); err != nil {
		t.Fatalf("decode GET response: %v", err)
	}
	if resp.ContentHTML != "<p>CK5</p>" {
		t.Fatalf("contentHtml = %q, want <p>CK5</p>", resp.ContentHTML)
	}
}

func TestPutCK5TemplateDraft_404_NoDraft(t *testing.T) {
	h, _ := setupCK5TemplateHandler(t)

	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	putBody, _ := json.Marshal(map[string]any{
		"contentHtml": "<p>CK5</p>",
		"manifest":    map[string]any{},
	})
	req := httptest.NewRequest(http.MethodPut, "/api/v1/templates/missing/ck5-draft", bytes.NewReader(putBody))
	req = withTemplateAuth(req)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("got %d, want 404; body: %s", w.Code, w.Body.String())
	}
}
