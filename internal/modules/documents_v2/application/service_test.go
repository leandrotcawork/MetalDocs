package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"metaldocs/internal/modules/documents_v2/application"
	"metaldocs/internal/modules/documents_v2/domain"
)

type fakeRepo struct {
	createDocErr     error
	createDocIDs     [3]string
	setStorageKeyErr error
	updateStatusErr  error
	acquireSess      *domain.Session
	acquireErr       error
	pendingMeta      *application.PendingCommitMeta
	pendingErr       error
	commitResult     *application.CommitResult
	commitErr        error
	restoreResult    *application.RestoreResult
	restoreErr       error
	ownerReturn      bool
	ownerErr         error
	checkpointResult *domain.Checkpoint
	checkpointErr    error

	docReturn   *domain.Document
	listReturn  []domain.Document
	checkpoints []domain.Checkpoint

	setStorageRevID string
	setStorageKey   string

	statusCalls int
	statusCur   domain.DocumentStatus
	statusNext  domain.DocumentStatus
	statusStamp bool
}

var _ application.Repository = (*fakeRepo)(nil)

func (f *fakeRepo) CreateDocument(_ context.Context, _ *domain.Document, _ string) (string, string, string, error) {
	if f.createDocErr != nil {
		return "", "", "", f.createDocErr
	}
	return f.createDocIDs[0], f.createDocIDs[1], f.createDocIDs[2], nil
}

func (f *fakeRepo) SetRevisionStorageKey(_ context.Context, revID, storageKey string) error {
	f.setStorageRevID = revID
	f.setStorageKey = storageKey
	return f.setStorageKeyErr
}

func (f *fakeRepo) GetDocument(_ context.Context, _, _ string) (*domain.Document, error) {
	if f.docReturn == nil {
		return nil, errors.New("document not configured")
	}
	return f.docReturn, nil
}

func (f *fakeRepo) ListDocuments(_ context.Context, _ string) ([]domain.Document, error) {
	return f.listReturn, nil
}

func (f *fakeRepo) ListDocumentsForUser(_ context.Context, _, _ string) ([]domain.Document, error) {
	return f.listReturn, nil
}

func (f *fakeRepo) UpdateDocumentStatus(_ context.Context, _, _ string, cur, next domain.DocumentStatus, stampTime bool) error {
	f.statusCalls++
	f.statusCur = cur
	f.statusNext = next
	f.statusStamp = stampTime
	return f.updateStatusErr
}

func (f *fakeRepo) AcquireSession(_ context.Context, _, _, _ string) (*domain.Session, error) {
	return f.acquireSess, f.acquireErr
}

func (f *fakeRepo) HeartbeatSession(_ context.Context, _, _ string) error { return nil }

func (f *fakeRepo) ReleaseSession(_ context.Context, _, _ string) error { return nil }

func (f *fakeRepo) ForceReleaseSession(_ context.Context, _ string) error { return nil }

func (f *fakeRepo) PresignReserve(_ context.Context, _, _, _, _, _, _ string, _ time.Time) (string, error) {
	return "pending_1", nil
}

func (f *fakeRepo) GetPendingForCommit(_ context.Context, _ string) (*application.PendingCommitMeta, error) {
	if f.pendingErr != nil {
		return nil, f.pendingErr
	}
	return f.pendingMeta, nil
}

func (f *fakeRepo) CommitUpload(_ context.Context, _, _, _, _, _ string, _ []byte) (*application.CommitResult, error) {
	if f.commitErr != nil {
		return nil, f.commitErr
	}
	if f.commitResult == nil {
		f.commitResult = &application.CommitResult{RevisionID: "rev_1", RevisionNum: 1}
	}
	return f.commitResult, nil
}

func (f *fakeRepo) CreateCheckpoint(_ context.Context, _, _, _ string) (*domain.Checkpoint, error) {
	if f.checkpointErr != nil {
		return nil, f.checkpointErr
	}
	return f.checkpointResult, nil
}

func (f *fakeRepo) ListCheckpoints(_ context.Context, _ string) ([]domain.Checkpoint, error) {
	return f.checkpoints, nil
}

func (f *fakeRepo) RestoreCheckpoint(_ context.Context, _, _ string, _ int) (*application.RestoreResult, error) {
	if f.restoreErr != nil {
		return nil, f.restoreErr
	}
	return f.restoreResult, nil
}

func (f *fakeRepo) IsDocumentOwner(_ context.Context, _, _, _ string) (bool, error) {
	if f.ownerErr != nil {
		return false, f.ownerErr
	}
	return f.ownerReturn, nil
}

type fakePresigner struct {
	hashReturn  string
	hashErr     error
	adoptErr    error
	deleteCalls int
	deleteErr   error
}

func (f *fakePresigner) PresignAutosavePUT(_ context.Context, _, _, _, _ string, _ time.Time) (string, error) {
	return "https://example/upload", nil
}

func (f *fakePresigner) HashObject(_ context.Context, _ string) (string, error) {
	if f.hashErr != nil {
		return "", f.hashErr
	}
	return f.hashReturn, nil
}

func (f *fakePresigner) AdoptTempObject(_ context.Context, _, _ string) error {
	return f.adoptErr
}

func (f *fakePresigner) DeleteObject(_ context.Context, _ string) error {
	f.deleteCalls++
	return f.deleteErr
}

func (f *fakePresigner) PresignObjectGET(_ context.Context, storageKey string) (string, error) {
	return "https://example/get/" + storageKey, nil
}

type fakeDocgen struct{}

func (fakeDocgen) Render(_ context.Context, _ []byte, _ []byte) (string, string, error) {
	return "tmp/rendered.docx", "h_initial", nil
}

type fakeTplReader struct {
	err error
}

func (f fakeTplReader) ReadTemplateDocx(_ context.Context, _, _ string) ([]byte, error) {
	if f.err != nil {
		return nil, f.err
	}
	return []byte("docx-template"), nil
}

type fakeFormVal struct {
	err error
}

func (f fakeFormVal) Validate(_ context.Context, _, _ string, _ []byte) error {
	return f.err
}

type noopAudit struct {
	calls      int
	lastAction string
}

func (n *noopAudit) Record(_ context.Context, action string, _ map[string]string) error {
	n.calls++
	n.lastAction = action
	return nil
}

func TestCreateDocument_OK(t *testing.T) {
	repo := &fakeRepo{createDocIDs: [3]string{"doc_1", "rev_1", "sess_1"}}
	audit := &noopAudit{}
	svc := application.New(repo, fakeDocgen{}, &fakePresigner{hashReturn: "h_expected"}, fakeTplReader{}, fakeFormVal{}, audit)

	doc, err := svc.CreateDocument(context.Background(), "tenant_1", "tpl_ver_1", "Contract", []byte(`{"a":1}`), "user_1")
	if err != nil {
		t.Fatalf("CreateDocument() error = %v", err)
	}
	if doc.ID != "doc_1" || doc.CurrentRevisionID != "rev_1" || doc.ActiveSessionID != "sess_1" {
		t.Fatalf("unexpected ids: %+v", doc)
	}
	if repo.setStorageRevID != "rev_1" {
		t.Fatalf("expected storage key to be set for rev_1, got %q", repo.setStorageRevID)
	}
}

func TestCreateDocument_InvalidFormData_Rejects(t *testing.T) {
	expectedErr := errors.New("invalid form data")
	repo := &fakeRepo{createDocIDs: [3]string{"doc_1", "rev_1", "sess_1"}}
	svc := application.New(repo, fakeDocgen{}, &fakePresigner{}, fakeTplReader{}, fakeFormVal{err: expectedErr}, &noopAudit{})

	_, err := svc.CreateDocument(context.Background(), "tenant_1", "tpl_ver_1", "Contract", []byte(`{"a":1}`), "user_1")
	if !errors.Is(err, expectedErr) {
		t.Fatalf("expected %v, got %v", expectedErr, err)
	}
}

func TestAcquireSession_Readonly_WhenTaken(t *testing.T) {
	repo := &fakeRepo{
		acquireSess: &domain.Session{ID: "sess_taken", DocumentID: "doc_1", UserID: "other"},
		acquireErr:  domain.ErrSessionTaken,
	}
	svc := application.New(repo, fakeDocgen{}, &fakePresigner{}, fakeTplReader{}, fakeFormVal{}, &noopAudit{})

	sess, readonly, err := svc.AcquireSession(context.Background(), "tenant_1", "doc_1", "user_1")
	if err != nil {
		t.Fatalf("AcquireSession() error = %v", err)
	}
	if !readonly {
		t.Fatalf("expected readonly session")
	}
	if sess == nil || sess.ID != "sess_taken" {
		t.Fatalf("unexpected session: %+v", sess)
	}
}

func TestAcquireSession_Success_RecordsAudit(t *testing.T) {
	repo := &fakeRepo{acquireSess: &domain.Session{ID: "sess_1", DocumentID: "doc_1", UserID: "user_1"}}
	audit := &noopAudit{}
	svc := application.New(repo, fakeDocgen{}, &fakePresigner{}, fakeTplReader{}, fakeFormVal{}, audit)

	sess, readonly, err := svc.AcquireSession(context.Background(), "tenant_1", "doc_1", "user_1")
	if err != nil {
		t.Fatalf("AcquireSession() error = %v", err)
	}
	if readonly {
		t.Fatalf("expected writable session")
	}
	if sess == nil || sess.ID != "sess_1" {
		t.Fatalf("unexpected session: %+v", sess)
	}
	if audit.calls == 0 || audit.lastAction != "documents_v2.session.acquire" {
		t.Fatalf("expected audit record for acquire, got calls=%d action=%q", audit.calls, audit.lastAction)
	}
}

func TestCreateCheckpoint_OK(t *testing.T) {
	repo := &fakeRepo{checkpointResult: &domain.Checkpoint{ID: "cp_1", DocumentID: "doc_1", VersionNum: 3}}
	svc := application.New(repo, fakeDocgen{}, &fakePresigner{}, fakeTplReader{}, fakeFormVal{}, &noopAudit{})

	cp, err := svc.CreateCheckpoint(context.Background(), "doc_1", "user_1", "Milestone")
	if err != nil {
		t.Fatalf("CreateCheckpoint() error = %v", err)
	}
	if cp == nil || cp.ID != "cp_1" {
		t.Fatalf("unexpected checkpoint: %+v", cp)
	}
}

func TestFinalize_FromDraft_OK(t *testing.T) {
	repo := &fakeRepo{docReturn: &domain.Document{ID: "doc_1", Status: domain.DocStatusDraft}}
	svc := application.New(repo, fakeDocgen{}, &fakePresigner{}, fakeTplReader{}, fakeFormVal{}, &noopAudit{})

	err := svc.Finalize(context.Background(), "tenant_1", "doc_1")
	if err != nil {
		t.Fatalf("Finalize() error = %v", err)
	}
	if repo.statusCalls != 1 {
		t.Fatalf("expected one status update call, got %d", repo.statusCalls)
	}
	if repo.statusCur != domain.DocStatusDraft || repo.statusNext != domain.DocStatusFinalized || !repo.statusStamp {
		t.Fatalf("unexpected status transition cur=%s next=%s stamp=%v", repo.statusCur, repo.statusNext, repo.statusStamp)
	}
}

func TestFinalize_FromFinalized_Rejects(t *testing.T) {
	repo := &fakeRepo{docReturn: &domain.Document{ID: "doc_1", Status: domain.DocStatusFinalized}}
	svc := application.New(repo, fakeDocgen{}, &fakePresigner{}, fakeTplReader{}, fakeFormVal{}, &noopAudit{})

	err := svc.Finalize(context.Background(), "tenant_1", "doc_1")
	if !errors.Is(err, domain.ErrInvalidStateTransition) {
		t.Fatalf("expected ErrInvalidStateTransition, got %v", err)
	}
	if repo.statusCalls != 0 {
		t.Fatalf("expected no status update call, got %d", repo.statusCalls)
	}
}
