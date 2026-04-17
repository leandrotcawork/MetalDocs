package httpdelivery

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"metaldocs/internal/modules/documents/domain"
	documentmemory "metaldocs/internal/modules/documents/infrastructure/memory"
	iamdomain "metaldocs/internal/modules/iam/domain"
)

func TestHandleCK5SubmitReview_200(t *testing.T) {
	h, repo := newTemplateTestHandler(t)
	upsertCK5TemplateDraft(t, repo, "tpl-submit-200", domain.TemplateStatusDraft, "<p>ok</p>")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates/tpl-submit-200/submit-review", nil)
	req = withAdminCtx(req)
	rec := httptest.NewRecorder()

	h.handleTemplateCK5SubmitReview(rec, req, "tpl-submit-200")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"status":"pending_review"`) {
		t.Fatalf("body = %s, want pending_review status", rec.Body.String())
	}
}

func TestHandleCK5SubmitReview_401_NoUser(t *testing.T) {
	h, repo := newTemplateTestHandler(t)
	upsertCK5TemplateDraft(t, repo, "tpl-submit-401", domain.TemplateStatusDraft, "<p>ok</p>")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates/tpl-submit-401/submit-review", nil)
	rec := httptest.NewRecorder()

	h.handleTemplateCK5SubmitReview(rec, req, "tpl-submit-401")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestHandleCK5SubmitReview_403_NoCapability(t *testing.T) {
	h, repo := newTemplateTestHandler(t)
	upsertCK5TemplateDraft(t, repo, "tpl-submit-403", domain.TemplateStatusDraft, "<p>ok</p>")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates/tpl-submit-403/submit-review", nil)
	req = withNoTemplateEditCapability(req)
	rec := httptest.NewRecorder()

	h.handleTemplateCK5SubmitReview(rec, req, "tpl-submit-403")

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleCK5SubmitReview_404_NotFound(t *testing.T) {
	h, _ := newTemplateTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates/no-such/submit-review", nil)
	req = withAdminCtx(req)
	rec := httptest.NewRecorder()

	h.handleTemplateCK5SubmitReview(rec, req, "no-such")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleCK5SubmitReview_409_WrongStatus(t *testing.T) {
	h, repo := newTemplateTestHandler(t)
	upsertCK5TemplateDraft(t, repo, "tpl-submit-409", domain.TemplateStatusPendingReview, "<p>ok</p>")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates/tpl-submit-409/submit-review", nil)
	req = withAdminCtx(req)
	rec := httptest.NewRecorder()

	h.handleTemplateCK5SubmitReview(rec, req, "tpl-submit-409")

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleCK5SubmitReview_400_EmptyContent(t *testing.T) {
	h, repo := newTemplateTestHandler(t)
	upsertCK5TemplateDraft(t, repo, "tpl-submit-400", domain.TemplateStatusDraft, "  ")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates/tpl-submit-400/submit-review", nil)
	req = withAdminCtx(req)
	rec := httptest.NewRecorder()

	h.handleTemplateCK5SubmitReview(rec, req, "tpl-submit-400")

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleCK5Approve_200(t *testing.T) {
	h, repo := newTemplateTestHandler(t)
	upsertCK5TemplateDraft(t, repo, "tpl-approve-200", domain.TemplateStatusPendingReview, "<p>ok</p>")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates/tpl-approve-200/approve", nil)
	req = withAdminCtx(req)
	rec := httptest.NewRecorder()

	h.handleTemplateCK5Approve(rec, req, "tpl-approve-200")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body: %s", rec.Code, rec.Body.String())
	}
	if !strings.Contains(rec.Body.String(), `"status":"published"`) {
		t.Fatalf("body = %s, want published status", rec.Body.String())
	}
}

func TestHandleCK5Approve_401_NoUser(t *testing.T) {
	h, repo := newTemplateTestHandler(t)
	upsertCK5TemplateDraft(t, repo, "tpl-approve-401", domain.TemplateStatusPendingReview, "<p>ok</p>")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates/tpl-approve-401/approve", nil)
	rec := httptest.NewRecorder()

	h.handleTemplateCK5Approve(rec, req, "tpl-approve-401")

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestHandleCK5Approve_403_NoCapability(t *testing.T) {
	h, repo := newTemplateTestHandler(t)
	upsertCK5TemplateDraft(t, repo, "tpl-approve-403", domain.TemplateStatusPendingReview, "<p>ok</p>")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates/tpl-approve-403/approve", nil)
	req = withNoTemplatePublishCapability(req)
	rec := httptest.NewRecorder()

	h.handleTemplateCK5Approve(rec, req, "tpl-approve-403")

	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleCK5Approve_404_NotFound(t *testing.T) {
	h, _ := newTemplateTestHandler(t)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates/no-such/approve", nil)
	req = withAdminCtx(req)
	rec := httptest.NewRecorder()

	h.handleTemplateCK5Approve(rec, req, "no-such")

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404; body: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleCK5Approve_409_WrongStatus(t *testing.T) {
	h, repo := newTemplateTestHandler(t)
	upsertCK5TemplateDraft(t, repo, "tpl-approve-409", domain.TemplateStatusDraft, "<p>ok</p>")

	req := httptest.NewRequest(http.MethodPost, "/api/v1/templates/tpl-approve-409/approve", nil)
	req = withAdminCtx(req)
	rec := httptest.NewRecorder()

	h.handleTemplateCK5Approve(rec, req, "tpl-approve-409")

	if rec.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409; body: %s", rec.Code, rec.Body.String())
	}
}

func withNoTemplateEditCapability(req *http.Request) *http.Request {
	return req.WithContext(iamdomain.WithAuthContext(req.Context(), "actor-editor", []iamdomain.Role{iamdomain.RoleEditor}))
}

func withNoTemplatePublishCapability(req *http.Request) *http.Request {
	return req.WithContext(iamdomain.WithAuthContext(req.Context(), "actor-editor", []iamdomain.Role{iamdomain.RoleEditor}))
}

func upsertCK5TemplateDraft(t *testing.T, repo *documentmemory.Repository, key string, status domain.TemplateStatus, html string) {
	t.Helper()

	draft := &domain.TemplateDraft{
		TemplateKey: key,
		ProfileCode: "po",
		Name:        "Template",
		DraftStatus: status,
		BlocksJSON:  json.RawMessage(`{"_ck5":{"contentHtml":` + marshalJSONString(t, html) + `}}`),
	}
	if _, err := repo.UpsertTemplateDraftCAS(ctxAdmin().Context(), draft, 0); err != nil {
		t.Fatalf("UpsertTemplateDraftCAS() error = %v", err)
	}
}

func marshalJSONString(t *testing.T, value string) string {
	t.Helper()
	b, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("json.Marshal(%q) error = %v", value, err)
	}
	return string(b)
}
