package http_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"metaldocs/internal/modules/templates_v2/domain"
)

func TestUpdateSchemas_Happy(t *testing.T) {
	repo := newFakeRepo()
	repo.versions["ver-1"] = &domain.TemplateVersion{
		ID:            "ver-1",
		TemplateID:    "tpl-1",
		VersionNumber: 1,
		Status:        domain.VersionStatusDraft,
		ContentHash:   "hash_abc",
	}

	var gotAction string
	authz := func(_ *http.Request, _, _, action string) error {
		gotAction = action
		return nil
	}
	mux := newMux(t, authz, repo)

	body := map[string]any{
		"metadata_schema": map[string]any{
			"doc_code_pattern": "ABC-###",
		},
		"placeholder_schema": []map[string]any{
			{"id": "ph-1", "label": "Signer", "type": "select", "options": []string{"a", "b"}},
		},
		"editable_zones": []map[string]any{
			{"id": "zone-1", "label": "Body", "required": true},
		},
		"expected_content_hash": "hash_abc",
	}
	raw, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPut, "/api/v2/templates/tpl-1/versions/1/schema", bytes.NewReader(raw))
	withHeaders(req)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	if gotAction != "template.edit" {
		t.Fatalf("expected authz action template.edit, got %q", gotAction)
	}

	var out struct {
		Data struct {
			Version struct {
				VersionNumber  int `json:"version_number"`
				MetadataSchema struct {
					DocCodePattern string `json:"doc_code_pattern"`
				} `json:"metadata_schema"`
			} `json:"version"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if out.Data.Version.VersionNumber != 1 {
		t.Fatalf("expected version_number=1, got %d", out.Data.Version.VersionNumber)
	}
	if out.Data.Version.MetadataSchema.DocCodePattern != "ABC-###" {
		t.Fatalf("expected metadata_schema.doc_code_pattern=ABC-###, got %q", out.Data.Version.MetadataSchema.DocCodePattern)
	}
}

func TestPresignAutosave_Happy(t *testing.T) {
	repo := newFakeRepo()
	repo.templates["tpl-1"] = &domain.Template{ID: "tpl-1", TenantID: "tenant-a"}
	repo.versions["ver-1"] = &domain.TemplateVersion{
		ID:             "ver-1",
		TemplateID:     "tpl-1",
		VersionNumber:  1,
		Status:         domain.VersionStatusDraft,
		DocxStorageKey: "templates/tpl-1/versions/1.docx",
	}

	var gotAction string
	authz := func(_ *http.Request, _, _, action string) error {
		gotAction = action
		return nil
	}
	mux := newMux(t, authz, repo)

	req := httptest.NewRequest(http.MethodPost, "/api/v2/templates/tpl-1/versions/1/autosave/presign", nil)
	withHeaders(req)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d body=%s", rr.Code, rr.Body.String())
	}
	if gotAction != "template.edit" {
		t.Fatalf("expected authz action template.edit, got %q", gotAction)
	}

	var out struct {
		Data struct {
			UploadURL string `json:"upload_url"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if out.Data.UploadURL == "" {
		t.Fatal("expected data.upload_url to be present")
	}
}

func TestCommitAutosave_Happy(t *testing.T) {
	repo := newFakeRepo()
	repo.templates["tpl-1"] = &domain.Template{ID: "tpl-1", TenantID: "tenant-a"}
	repo.versions["ver-1"] = &domain.TemplateVersion{
		ID:             "ver-1",
		TemplateID:     "tpl-1",
		VersionNumber:  1,
		Status:         domain.VersionStatusDraft,
		DocxStorageKey: "templates/tpl-1/versions/1.docx",
	}
	mux := newMux(t, func(_ *http.Request, _, _, _ string) error { return nil }, repo)

	raw, _ := json.Marshal(map[string]any{"expected_content_hash": "hash_abc"})
	req := httptest.NewRequest(http.MethodPost, "/api/v2/templates/tpl-1/versions/1/autosave/commit", bytes.NewReader(raw))
	withHeaders(req)
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var out struct {
		Data struct {
			Version struct {
				ContentHash string `json:"content_hash"`
			} `json:"version"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if out.Data.Version.ContentHash != "hash_abc" {
		t.Fatalf("expected data.version.content_hash=hash_abc, got %q", out.Data.Version.ContentHash)
	}
}

func TestCommitAutosave_HashMismatch(t *testing.T) {
	repo := newFakeRepo()
	repo.templates["tpl-1"] = &domain.Template{ID: "tpl-1", TenantID: "tenant-a"}
	repo.versions["ver-1"] = &domain.TemplateVersion{
		ID:             "ver-1",
		TemplateID:     "tpl-1",
		VersionNumber:  1,
		Status:         domain.VersionStatusDraft,
		DocxStorageKey: "templates/tpl-1/versions/1.docx",
	}
	mux := newMux(t, func(_ *http.Request, _, _, _ string) error { return nil }, repo)

	raw, _ := json.Marshal(map[string]any{"expected_content_hash": "hash_expected"})
	req := httptest.NewRequest(http.MethodPost, "/api/v2/templates/tpl-1/versions/1/autosave/commit", bytes.NewReader(raw))
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
	if out.Error.Code != "content_hash_mismatch" {
		t.Fatalf("expected error.code=content_hash_mismatch, got %q", out.Error.Code)
	}
}
