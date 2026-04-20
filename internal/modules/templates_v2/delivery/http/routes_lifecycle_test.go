package http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"metaldocs/internal/modules/templates_v2/domain"
)

func strPtr(v string) *string { return &v }

func withActorRoles(req *http.Request, roles string) {
	req.Header.Set("X-Actor-Roles", roles)
}

func TestSubmitForReview_Happy(t *testing.T) {
	repo := newFakeRepo()
	reviewerRole := "reviewer"
	repo.templates["tpl-1"] = &domain.Template{ID: "tpl-1", TenantID: "tenant-a"}
	repo.versions["ver-1"] = &domain.TemplateVersion{
		ID:            "ver-1",
		TemplateID:    "tpl-1",
		VersionNumber: 1,
		Status:        domain.VersionStatusDraft,
		AuthorID:      "author-1",
	}
	repo.approvalConfigs["tpl-1"] = &domain.ApprovalConfig{
		TemplateID:   "tpl-1",
		ReviewerRole: &reviewerRole,
		ApproverRole: "approver",
	}
	mux := newMux(t, func(_ *http.Request, _, _, _ string) error { return nil }, repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v2/templates/tpl-1/versions/1/submit", nil)
	withHeaders(req)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var out struct {
		Data struct {
			Version struct {
				Status string `json:"status"`
			} `json:"version"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if out.Data.Version.Status != string(domain.VersionStatusInReview) {
		t.Fatalf("expected status=in_review, got %q", out.Data.Version.Status)
	}
}

func TestSubmitForReview_NonDraft(t *testing.T) {
	repo := newFakeRepo()
	repo.templates["tpl-1"] = &domain.Template{ID: "tpl-1", TenantID: "tenant-a"}
	repo.versions["ver-1"] = &domain.TemplateVersion{
		ID:            "ver-1",
		TemplateID:    "tpl-1",
		VersionNumber: 1,
		Status:        domain.VersionStatusInReview,
	}
	repo.approvalConfigs["tpl-1"] = &domain.ApprovalConfig{TemplateID: "tpl-1", ApproverRole: "approver"}
	mux := newMux(t, func(_ *http.Request, _, _, _ string) error { return nil }, repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v2/templates/tpl-1/versions/1/submit", nil)
	withHeaders(req)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusConflict {
		t.Fatalf("expected 409, got %d body=%s", rr.Code, rr.Body.String())
	}

	var out struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if out.Error.Code != "invalid_state_transition" {
		t.Fatalf("expected error.code=invalid_state_transition, got %q", out.Error.Code)
	}
}

func TestReview_Accept_Happy(t *testing.T) {
	repo := newFakeRepo()
	reviewerRole := "reviewer"
	repo.templates["tpl-1"] = &domain.Template{ID: "tpl-1", TenantID: "tenant-a"}
	repo.versions["ver-1"] = &domain.TemplateVersion{
		ID:                  "ver-1",
		TemplateID:          "tpl-1",
		VersionNumber:       1,
		Status:              domain.VersionStatusInReview,
		AuthorID:            "author-1",
		PendingReviewerRole: &reviewerRole,
	}
	mux := newMux(t, func(_ *http.Request, _, _, _ string) error { return nil }, repo)

	raw, _ := json.Marshal(map[string]any{"accept": true})
	req := httptest.NewRequest(http.MethodPost, "/api/v2/templates/tpl-1/versions/1/review", bytes.NewReader(raw))
	withHeaders(req)
	req.Header.Set("X-User-ID", "reviewer-1")
	withActorRoles(req, "reviewer")
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var out struct {
		Data struct {
			Version struct {
				Status string `json:"status"`
			} `json:"version"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if out.Data.Version.Status != string(domain.VersionStatusApproved) {
		t.Fatalf("expected status=approved, got %q", out.Data.Version.Status)
	}
}

func TestApprove_Accept_Happy(t *testing.T) {
	repo := newFakeRepo()
	reviewerRole := "reviewer"
	repo.templates["tpl-1"] = &domain.Template{ID: "tpl-1", TenantID: "tenant-a"}
	repo.versions["ver-1"] = &domain.TemplateVersion{
		ID:                  "ver-1",
		TemplateID:          "tpl-1",
		VersionNumber:       1,
		Status:              domain.VersionStatusApproved,
		AuthorID:            "author-1",
		PendingReviewerRole: &reviewerRole,
		PendingApproverRole: "approver",
		ReviewerID:          strPtr("reviewer-1"),
	}
	mux := newMux(t, func(_ *http.Request, _, _, _ string) error { return nil }, repo)

	raw, _ := json.Marshal(map[string]any{"accept": true})
	req := httptest.NewRequest(http.MethodPost, "/api/v2/templates/tpl-1/versions/1/approve", bytes.NewReader(raw))
	withHeaders(req)
	req.Header.Set("X-User-ID", "approver-1")
	withActorRoles(req, "approver")
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var out struct {
		Data struct {
			Version struct {
				Status string `json:"status"`
			} `json:"version"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if out.Data.Version.Status != string(domain.VersionStatusPublished) {
		t.Fatalf("expected status=published, got %q", out.Data.Version.Status)
	}
}

func TestArchiveTemplate_Happy(t *testing.T) {
	repo := newFakeRepo()
	repo.templates["tpl-1"] = &domain.Template{ID: "tpl-1", TenantID: "tenant-a"}
	mux := newMux(t, func(_ *http.Request, _, _, _ string) error { return nil }, repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v2/templates/tpl-1/archive", nil)
	withHeaders(req)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var out struct {
		Data struct {
			Template struct {
				ArchivedAt *string `json:"archived_at"`
			} `json:"template"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if out.Data.Template.ArchivedAt == nil || *out.Data.Template.ArchivedAt == "" {
		t.Fatal("expected data.template.archived_at to be set")
	}
}

func TestUpsertApprovalConfig_Happy(t *testing.T) {
	repo := newFakeRepo()
	repo.templates["tpl-1"] = &domain.Template{ID: "tpl-1", TenantID: "tenant-a", CreatedBy: "user-a"}
	mux := newMux(t, func(_ *http.Request, _, _, _ string) error { return nil }, repo)

	raw, _ := json.Marshal(map[string]any{
		"reviewer_role": "reviewer",
		"approver_role": "approver",
	})
	req := httptest.NewRequest(http.MethodPut, "/api/v2/templates/tpl-1/approval-config", bytes.NewReader(raw))
	withHeaders(req)
	withActorRoles(req, "editor")
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var out struct {
		Data struct {
			ApprovalConfig struct {
				ApproverRole string `json:"approver_role"`
			} `json:"approval_config"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if out.Data.ApprovalConfig.ApproverRole != "approver" {
		t.Fatalf("expected data.approval_config.approver_role=approver, got %q", out.Data.ApprovalConfig.ApproverRole)
	}
}
