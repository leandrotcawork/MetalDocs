package httpdelivery

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"metaldocs/internal/modules/documents/domain"
)

func TestHandleGetCK5Draft_200_AuthorMode(t *testing.T) {
	h, repo := newTemplateTestHandler(t)
	upsertCK5TemplateDraft(t, repo, "tpl-ck5-author", domain.TemplateStatusDraft, "<p>live</p>")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/templates/tpl-ck5-author/ck5-draft", nil)
	req = withAdminCtx(req)
	rec := httptest.NewRecorder()

	h.handleGetCK5Draft(rec, req, "tpl-ck5-author")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}

	var resp ck5DraftResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.HTML != "<p>live</p>" {
		t.Fatalf("html = %q, want %q", resp.HTML, "<p>live</p>")
	}
}

func TestHandleGetCK5Draft_200_FillMode_Published(t *testing.T) {
	h, repo := newTemplateTestHandler(t)
	upsertCK5TemplateDraft(t, repo, "tpl-ck5-fill-published", domain.TemplateStatusDraft, "<p>live</p>")

	draft, err := repo.GetTemplateDraft(ctxAdmin().Context(), "tpl-ck5-fill-published")
	if err != nil {
		t.Fatalf("GetTemplateDraft() error = %v", err)
	}
	published := "<p>published</p>"
	draft.DraftStatus = domain.TemplateStatusPublished
	draft.PublishedHTML = &published
	if _, err := repo.UpsertTemplateDraftCAS(ctxAdmin().Context(), draft, draft.LockVersion); err != nil {
		t.Fatalf("UpsertTemplateDraftCAS() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/templates/tpl-ck5-fill-published/ck5-draft?mode=fill", nil)
	req = withAdminCtx(req)
	rec := httptest.NewRecorder()

	h.handleGetCK5Draft(rec, req, "tpl-ck5-fill-published")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}

	var resp ck5DraftResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.HTML != "<p>published</p>" {
		t.Fatalf("html = %q, want %q", resp.HTML, "<p>published</p>")
	}
}

func TestHandleGetCK5Draft_200_FillMode_NotPublished(t *testing.T) {
	h, repo := newTemplateTestHandler(t)
	upsertCK5TemplateDraft(t, repo, "tpl-ck5-fill-draft", domain.TemplateStatusDraft, "<p>live</p>")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/templates/tpl-ck5-fill-draft/ck5-draft?mode=fill", nil)
	req = withAdminCtx(req)
	rec := httptest.NewRecorder()

	h.handleGetCK5Draft(rec, req, "tpl-ck5-fill-draft")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}

	var resp ck5DraftResponse
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.HTML != "<p>live</p>" {
		t.Fatalf("html = %q, want %q", resp.HTML, "<p>live</p>")
	}
}

func TestHandleGetCK5Draft_401_NoUser(t *testing.T) {
	h, repo := newTemplateTestHandler(t)
	upsertCK5TemplateDraft(t, repo, "tpl-ck5-401", domain.TemplateStatusDraft, "<p>live</p>")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/templates/tpl-ck5-401/ck5-draft", nil)
	rec := httptest.NewRecorder()

	h.handleGetCK5Draft(rec, req, "tpl-ck5-401")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleGetCK5Draft_404_NotFound(t *testing.T) {
	h, _ := newTemplateTestHandler(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/templates/no-such/ck5-draft", nil)
	req = withAdminCtx(req)
	rec := httptest.NewRecorder()

	h.handleGetCK5Draft(rec, req, "no-such")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body: %s", rec.Code, rec.Body.String())
	}
}
