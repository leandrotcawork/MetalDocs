package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	thttp "metaldocs/internal/modules/templates/delivery/http"
	"metaldocs/internal/modules/templates/application"
	"metaldocs/internal/modules/templates/domain"
)

type fakeSvc struct{}

func (f *fakeSvc) CreateTemplate(_ context.Context, _ application.CreateTemplateCmd) (*domain.Template, *domain.TemplateVersion, error) {
	return &domain.Template{ID: "tpl1"}, &domain.TemplateVersion{ID: "ver1"}, nil
}
func (f *fakeSvc) SaveDraft(_ context.Context, _ application.SaveDraftCmd) error { return nil }
func (f *fakeSvc) PublishVersion(_ context.Context, _ application.PublishCmd) (application.PublishResult, error) {
	return application.PublishResult{NewDraftID: "ver2", NewDraftVersion: 2}, nil
}
func (f *fakeSvc) ListTemplates(_ context.Context, _ string) ([]domain.TemplateListItem, error) {
	return []domain.TemplateListItem{{ID: "tpl1", Key: "po", Name: "Purchase Order", LatestVersion: 3}}, nil
}
func (f *fakeSvc) GetVersion(_ context.Context, _ string, _ int) (*domain.Template, *domain.TemplateVersion, error) {
	return &domain.Template{ID: "tpl1", Name: "Purchase Order"}, &domain.TemplateVersion{ID: "ver1", VersionNum: 1, Status: domain.StatusDraft, LockVersion: 0}, nil
}
func (f *fakeSvc) PresignDocxUpload(_ context.Context, _ string, _ int) (string, string, error) {
	return "https://s3.test/put", "tenants/t1/templates/tpl1/v1.docx", nil
}
func (f *fakeSvc) PresignSchemaUpload(_ context.Context, _ string, _ int) (string, string, error) {
	return "https://s3.test/put-schema", "tenants/t1/templates/tpl1/v1.schema.json", nil
}
func (f *fakeSvc) PresignObjectDownload(_ context.Context, _ string) (string, error) {
	return "https://s3.test/get", nil
}

func TestCreateTemplate(t *testing.T) {
	h := thttp.NewHandler(&fakeSvc{})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	body, _ := json.Marshal(map[string]string{"key": "po", "name": "Purchase Order"})
	req := httptest.NewRequest(http.MethodPost, "/api/v2/templates", bytes.NewReader(body))
	req.Header.Set("content-type", "application/json")
	req.Header.Set("X-User-Roles", "template_author")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != 201 {
		t.Fatalf("expected 201, got %d", rr.Code)
	}
	var out map[string]string
	_ = json.Unmarshal(rr.Body.Bytes(), &out)
	if out["id"] != "tpl1" {
		t.Fatalf("id mismatch: %v", out)
	}
}

func TestCreateTemplate_ForbiddenForFiller(t *testing.T) {
	h := thttp.NewHandler(&fakeSvc{})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	body, _ := json.Marshal(map[string]string{"key": "po", "name": "Purchase Order"})
	req := httptest.NewRequest(http.MethodPost, "/api/v2/templates", bytes.NewReader(body))
	req.Header.Set("content-type", "application/json")
	req.Header.Set("X-User-Roles", "document_filler")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != 403 {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}

func TestSaveDraft_ForbiddenForFiller(t *testing.T) {
	h := thttp.NewHandler(&fakeSvc{})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	body, _ := json.Marshal(map[string]any{"expected_lock_version": 0, "docx_storage_key": "k", "schema_storage_key": "k2"})
	req := httptest.NewRequest(http.MethodPut, "/api/v2/templates/tpl1/versions/1/draft", bytes.NewReader(body))
	req.Header.Set("content-type", "application/json")
	req.Header.Set("X-User-Roles", "document_filler")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != 403 {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}

func TestPublish_ForbiddenForFiller(t *testing.T) {
	h := thttp.NewHandler(&fakeSvc{})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	body, _ := json.Marshal(map[string]string{"docx_key": "k", "schema_key": "k2"})
	req := httptest.NewRequest(http.MethodPost, "/api/v2/templates/tpl1/versions/1/publish", bytes.NewReader(body))
	req.Header.Set("content-type", "application/json")
	req.Header.Set("X-User-Roles", "template_author")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != 403 {
		t.Fatalf("expected 403 (author cannot publish), got %d", rr.Code)
	}
}

func TestPresignSchemaUpload_OK_ForAuthor(t *testing.T) {
	h := thttp.NewHandler(&fakeSvc{})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/v2/templates/tpl1/versions/1/schema-upload-url", nil)
	req.Header.Set("X-User-Roles", "template_author")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var out map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if out["url"] == "" || out["storage_key"] == "" {
		t.Fatalf("expected url+storage_key, got %v", out)
	}
}

func TestPresignSchemaUpload_ForbiddenForFiller(t *testing.T) {
	h := thttp.NewHandler(&fakeSvc{})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/v2/templates/tpl1/versions/1/schema-upload-url", nil)
	req.Header.Set("X-User-Roles", "document_filler")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != 403 {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}

func TestListTemplates_ForbiddenForFiller(t *testing.T) {
	h := thttp.NewHandler(&fakeSvc{})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v2/templates", nil)
	req.Header.Set("X-User-Roles", "document_filler")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != 403 {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}

func TestListTemplates_ReturnsLatestVersion(t *testing.T) {
	h := thttp.NewHandler(&fakeSvc{})
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v2/templates", nil)
	req.Header.Set("X-User-Roles", "template_author")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != 200 {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	var out []map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(out) == 0 {
		t.Fatalf("expected at least one row")
	}
	if v, ok := out[0]["latest_version"].(float64); !ok || int(v) < 1 {
		t.Fatalf("expected latest_version >= 1, got %v", out[0]["latest_version"])
	}
}
