package http_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"metaldocs/internal/modules/documents_v2/application"
	httphandler "metaldocs/internal/modules/documents_v2/delivery/http"
	"metaldocs/internal/modules/documents_v2/domain"
)

type fakeSvc struct {
	createResult *application.CreateDocumentResult
	createErr    error

	listDocs       []domain.Document
	listForUser    []domain.Document
	listErr        error
	listForUserErr error

	acquireSession *domain.Session
	acquireRO      bool
	acquireErr     error

	commitResult *application.CommitResult
	commitErr    error

	renameErr  error
	renameName string
}

var _ httphandler.Service = (*fakeSvc)(nil)

func (f *fakeSvc) CreateDocument(_ context.Context, _ application.CreateDocumentInput) (*application.CreateDocumentResult, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	if f.createResult == nil {
		return &application.CreateDocumentResult{DocumentID: "doc_1", InitialRevisionID: "rev_1", SessionID: "sess_1"}, nil
	}
	return f.createResult, nil
}

func (f *fakeSvc) GetDocument(_ context.Context, _, _ string) (*domain.Document, error) {
	return &domain.Document{ID: "doc_1", Name: "Doc"}, nil
}

func (f *fakeSvc) RenameDocument(_ context.Context, _, _, _, newName string) error {
	f.renameName = newName
	return f.renameErr
}

func (f *fakeSvc) ListDocuments(_ context.Context, _ string) ([]domain.Document, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	if f.listDocs == nil {
		return []domain.Document{{ID: "doc_1"}}, nil
	}
	return f.listDocs, nil
}

func (f *fakeSvc) ListDocumentsForUser(_ context.Context, _, _ string) ([]domain.Document, error) {
	if f.listForUserErr != nil {
		return nil, f.listForUserErr
	}
	if f.listForUser == nil {
		return []domain.Document{{ID: "doc_1"}}, nil
	}
	return f.listForUser, nil
}

func (f *fakeSvc) IsDocumentOwner(_ context.Context, _, _, _ string) (bool, error) {
	return true, nil
}

func (f *fakeSvc) AcquireSession(_ context.Context, _, _, _ string) (*domain.Session, bool, error) {
	if f.acquireErr != nil {
		return nil, false, f.acquireErr
	}
	if f.acquireSession == nil {
		return &domain.Session{ID: "sess_1", DocumentID: "doc_1", UserID: "user_1", Status: domain.SessionActive}, f.acquireRO, nil
	}
	return f.acquireSession, f.acquireRO, nil
}

func (f *fakeSvc) HeartbeatSession(_ context.Context, _, _ string) error { return nil }

func (f *fakeSvc) ReleaseSession(_ context.Context, _, _, _, _ string) error { return nil }

func (f *fakeSvc) ForceReleaseSession(_ context.Context, _, _, _, _ string) error { return nil }

func (f *fakeSvc) PresignAutosave(_ context.Context, _ application.PresignAutosaveCmd) (*application.PresignAutosaveResult, error) {
	return &application.PresignAutosaveResult{UploadURL: "https://example/upload", PendingUploadID: "pending_1", ExpiresAt: time.Now().Add(time.Minute)}, nil
}

func (f *fakeSvc) CommitAutosave(_ context.Context, _ application.CommitAutosaveCmd) (*application.CommitResult, error) {
	if f.commitErr != nil {
		return nil, f.commitErr
	}
	if f.commitResult == nil {
		return &application.CommitResult{RevisionID: "rev_2", RevisionNum: 2}, nil
	}
	return f.commitResult, nil
}

func (f *fakeSvc) CreateCheckpoint(_ context.Context, _, _, _, _ string) (*domain.Checkpoint, error) {
	return &domain.Checkpoint{ID: "cp_1", VersionNum: 1}, nil
}

func (f *fakeSvc) ListCheckpoints(_ context.Context, _, _ string) ([]domain.Checkpoint, error) {
	return []domain.Checkpoint{{ID: "cp_1", VersionNum: 1}}, nil
}

func (f *fakeSvc) RestoreCheckpoint(_ context.Context, _, _, _ string, _ int) (*application.RestoreResult, error) {
	return &application.RestoreResult{NewRevisionID: "rev_3", NewRevisionNum: 3}, nil
}

func (f *fakeSvc) Finalize(_ context.Context, _, _, _ string) error { return nil }

func (f *fakeSvc) Archive(_ context.Context, _, _, _ string, _ bool) error { return nil }

func (f *fakeSvc) SignedRevisionURL(_ context.Context, _, _, _ string) (string, error) {
	return "https://example/rev", nil
}

func (f *fakeSvc) ListDocumentComments(_ context.Context, _, _, _ string) ([]domain.Comment, error) {
	return nil, nil
}

func (f *fakeSvc) AddDocumentComment(_ context.Context, _, _, _, _ string, _ domain.CommentCreateInput) (*domain.Comment, error) {
	return &domain.Comment{}, nil
}

func (f *fakeSvc) UpdateDocumentComment(_ context.Context, _, _, _ string, _ int, _ domain.CommentUpdateInput) (*domain.Comment, error) {
	return &domain.Comment{}, nil
}

func (f *fakeSvc) DeleteDocumentComment(_ context.Context, _, _, _ string, _ int) error {
	return nil
}

func newMux(t *testing.T, svc *fakeSvc) *http.ServeMux {
	t.Helper()
	h := httphandler.NewHandler(svc)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)
	return mux
}

func withAuthHeaders(req *http.Request, roles string) {
	req.Header.Set("content-type", "application/json")
	req.Header.Set("X-Tenant-ID", "tenant_1")
	req.Header.Set("X-User-ID", "user_1")
	req.Header.Set("X-User-Roles", roles)
}

func TestCreateDocument_Happy(t *testing.T) {
	mux := newMux(t, &fakeSvc{})

	body := []byte(`{"controlled_document_id":"cd_1","template_version_id":"tpl_ver_1","name":"Contract","form_data":{"a":1}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v2/documents", bytes.NewReader(body))
	withAuthHeaders(req, "document_filler")
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rr.Code)
	}
}

func TestCreateDocument_Forbidden(t *testing.T) {
	mux := newMux(t, &fakeSvc{})

	body := []byte(`{"controlled_document_id":"cd_1","template_version_id":"tpl_ver_1","name":"Contract","form_data":{"a":1}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v2/documents", bytes.NewReader(body))
	withAuthHeaders(req, "template_author")
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}

func TestListDocuments_Happy(t *testing.T) {
	mux := newMux(t, &fakeSvc{})

	req := httptest.NewRequest(http.MethodGet, "/api/v2/documents", nil)
	withAuthHeaders(req, "document_filler")
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
}

func TestListDocuments_Forbidden(t *testing.T) {
	mux := newMux(t, &fakeSvc{})

	req := httptest.NewRequest(http.MethodGet, "/api/v2/documents", nil)
	withAuthHeaders(req, "template_author")
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}

func TestAcquireSession_Happy(t *testing.T) {
	mux := newMux(t, &fakeSvc{acquireSession: &domain.Session{ID: "sess_1"}})

	req := httptest.NewRequest(http.MethodPost, "/api/v2/documents/doc_1/session/acquire", bytes.NewReader([]byte(`{}`)))
	withAuthHeaders(req, "document_filler")
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", rr.Code)
	}
}

func TestAcquireSession_Forbidden(t *testing.T) {
	mux := newMux(t, &fakeSvc{})

	req := httptest.NewRequest(http.MethodPost, "/api/v2/documents/doc_1/session/acquire", bytes.NewReader([]byte(`{}`)))
	withAuthHeaders(req, "template_author")
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}

func TestCommitAutosave_IdempotentReplay_Returns200(t *testing.T) {
	mux := newMux(t, &fakeSvc{commitResult: &application.CommitResult{RevisionID: "rev_2", RevisionNum: 2, AlreadyConsumed: true}})

	body := []byte(`{"session_id":"sess_1","pending_upload_id":"pending_1","form_data_snapshot":{"a":1}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v2/documents/doc_1/autosave/commit", bytes.NewReader(body))
	withAuthHeaders(req, "document_filler")
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}

	var out map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if ok, _ := out["idempotent_replay"].(bool); !ok {
		t.Fatalf("expected idempotent_replay=true, got %v", out["idempotent_replay"])
	}
}

func TestForceReleaseSession_RequiresAdmin(t *testing.T) {
	mux := newMux(t, &fakeSvc{})

	body := []byte(`{"session_id":"sess_1"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v2/documents/doc_1/session/force-release", bytes.NewReader(body))
	withAuthHeaders(req, "document_filler")
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", rr.Code)
	}
}

func TestRenameDocument_Happy(t *testing.T) {
	svc := &fakeSvc{}
	mux := newMux(t, svc)

	body := []byte(`{"name":"Updated Name"}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v2/documents/doc_1", bytes.NewReader(body))
	withAuthHeaders(req, "document_filler")
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rr.Code)
	}
	if svc.renameName != "Updated Name" {
		t.Fatalf("expected rename name to be passed to service, got %q", svc.renameName)
	}
}

func TestRenameDocument_EmptyName_Returns400(t *testing.T) {
	svc := &fakeSvc{renameErr: domain.ErrInvalidName}
	mux := newMux(t, svc)

	body := []byte(`{"name":"   "}`)
	req := httptest.NewRequest(http.MethodPatch, "/api/v2/documents/doc_1", bytes.NewReader(body))
	withAuthHeaders(req, "document_filler")
	rr := httptest.NewRecorder()

	mux.ServeHTTP(rr, req)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}
