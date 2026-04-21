package application_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"metaldocs/internal/modules/documents_v2/application"
	"metaldocs/internal/modules/documents_v2/domain"
	registrydomain "metaldocs/internal/modules/registry/domain"
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

	revisionReturn *domain.Revision
	renameErr      error
	renameName     string
	renameDocID    string
	renameTenantID string
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

func (f *fakeRepo) UpdateDocumentName(_ context.Context, tenantID, docID, name string) error {
	f.renameTenantID = tenantID
	f.renameDocID = docID
	f.renameName = name
	return f.renameErr
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

func (f *fakeRepo) ExpireStaleSessions(_ context.Context, _ time.Time) (int, error) { return 0, nil }

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

func (f *fakeRepo) GetRevision(_ context.Context, _, _ string) (*domain.Revision, error) {
	if f.revisionReturn == nil {
		return nil, errors.New("revision not configured")
	}
	return f.revisionReturn, nil
}

func (f *fakeRepo) DeleteExpiredPending(_ context.Context, _ time.Time) (int, error) { return 0, nil }

func (f *fakeRepo) CreateComment(_ context.Context, _, _, _ string, _ domain.CommentCreateInput) (*domain.Comment, error) {
	return &domain.Comment{}, nil
}

func (f *fakeRepo) ListComments(_ context.Context, _, _ string) ([]domain.Comment, error) {
	return nil, nil
}

func (f *fakeRepo) UpdateComment(_ context.Context, _, _ string, _ int, _ string, _ domain.CommentUpdateInput) (*domain.Comment, error) {
	return &domain.Comment{}, nil
}

func (f *fakeRepo) DeleteComment(_ context.Context, _, _ string, _ int) error {
	return nil
}

type fakePresigner struct {
	hashReturn  string
	hashErr     error
	adoptErr    error
	deleteCalls int
	deleteErr   error
}

func (f *fakePresigner) PresignRevisionPUT(_ context.Context, _, _, _ string) (string, string, error) {
	return "https://example/upload", "documents/doc_1/revisions/rev_1.docx", nil
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

type fakeDocgen struct {
	hashReturn string
	err        error
}

func (f fakeDocgen) RenderDocx(_ context.Context, _, _, _ string, _ json.RawMessage) (string, int64, []string, error) {
	if f.err != nil {
		return "", 0, nil, f.err
	}
	if f.hashReturn != "" {
		return f.hashReturn, 100, nil, nil
	}
	return "h_initial", 100, nil, nil
}

type fakeTplReader struct {
	err error
}

func (f fakeTplReader) GetPublishedVersion(_ context.Context, _, _ string) (string, string, string, error) {
	if f.err != nil {
		return "", "", "", f.err
	}
	return "tpl/docx/key.docx", "tpl/schema/key.json", `{"type":"object"}`, nil
}

type fakeFormVal struct {
	valid bool
	errs  []string
	err   error
}

func (f fakeFormVal) Validate(_ string, _ json.RawMessage) (bool, []string, error) {
	if f.err != nil {
		return false, nil, f.err
	}
	if !f.valid {
		return false, f.errs, nil
	}
	return true, nil, nil
}

type noopAudit struct {
	calls      int
	lastAction string
}

func (n *noopAudit) Write(_ context.Context, _, _, action, _ string, _ any) {
	n.calls++
	n.lastAction = action
}

func TestCreateDocument_OK(t *testing.T) {
	repo := &fakeRepo{createDocIDs: [3]string{"doc_1", "rev_1", "sess_1"}}
	audit := &noopAudit{}
	svc := application.NewService(
		repo,
		fakeDocgen{},
		&fakePresigner{hashReturn: "h_initial"},
		fakeTplReader{},
		fakeFormVal{valid: true},
		audit,
		&fakeRegistryReader{cd: &registrydomain.ControlledDocument{
			ID:              "cd_1",
			ProfileCode:     "PROC",
			ProcessAreaCode: "AREA-01",
			Status:          registrydomain.CDStatusActive,
		}},
		&fakeAuthzChecker{},
		&fakeProfileDefaultTemplateReader{id: strptr("tpl_ver_1"), status: strptr("published")},
	)

	res, err := svc.CreateDocument(context.Background(), application.CreateDocumentCmd{
		TenantID:             "tenant_1",
		ActorUserID:          "user_1",
		ControlledDocumentID: "cd_1",
		TemplateVersionID:    "tpl_ver_1",
		Name:                 "Contract",
		FormData:             []byte(`{"a":1}`),
	})
	if err != nil {
		t.Fatalf("CreateDocument() error = %v", err)
	}
	if res.DocumentID != "doc_1" || res.InitialRevisionID != "rev_1" || res.SessionID != "sess_1" {
		t.Fatalf("unexpected ids: %+v", res)
	}
	if repo.setStorageRevID != "rev_1" {
		t.Fatalf("expected storage key to be set for rev_1, got %q", repo.setStorageRevID)
	}
}

func TestCreateDocument_InvalidFormData_Rejects(t *testing.T) {
	repo := &fakeRepo{createDocIDs: [3]string{"doc_1", "rev_1", "sess_1"}}
	svc := application.NewService(
		repo,
		fakeDocgen{},
		&fakePresigner{},
		fakeTplReader{},
		fakeFormVal{valid: false, errs: []string{"invalid"}},
		&noopAudit{},
		&fakeRegistryReader{cd: &registrydomain.ControlledDocument{
			ID:              "cd_1",
			ProfileCode:     "PROC",
			ProcessAreaCode: "AREA-01",
			Status:          registrydomain.CDStatusActive,
		}},
		&fakeAuthzChecker{},
		&fakeProfileDefaultTemplateReader{id: strptr("tpl_ver_1"), status: strptr("published")},
	)

	_, err := svc.CreateDocument(context.Background(), application.CreateDocumentCmd{
		TenantID:             "tenant_1",
		ActorUserID:          "user_1",
		ControlledDocumentID: "cd_1",
		TemplateVersionID:    "tpl_ver_1",
		Name:                 "Contract",
		FormData:             []byte(`{"a":1}`),
	})
	if err == nil {
		t.Fatalf("expected validation error")
	}
}

func TestAcquireSession_Readonly_WhenTaken(t *testing.T) {
	repo := &fakeRepo{
		acquireSess: &domain.Session{ID: "sess_taken", DocumentID: "doc_1", UserID: "other"},
		acquireErr:  domain.ErrSessionTaken,
	}
	svc := application.New(repo, fakeDocgen{}, &fakePresigner{}, fakeTplReader{}, fakeFormVal{valid: true}, &noopAudit{})

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
	svc := application.New(repo, fakeDocgen{}, &fakePresigner{}, fakeTplReader{}, fakeFormVal{valid: true}, audit)

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
	if audit.calls == 0 || audit.lastAction != "session.acquired" {
		t.Fatalf("expected audit record for acquire, got calls=%d action=%q", audit.calls, audit.lastAction)
	}
}

func TestCreateCheckpoint_OK(t *testing.T) {
	repo := &fakeRepo{checkpointResult: &domain.Checkpoint{ID: "cp_1", DocumentID: "doc_1", VersionNum: 3}}
	svc := application.New(repo, fakeDocgen{}, &fakePresigner{}, fakeTplReader{}, fakeFormVal{valid: true}, &noopAudit{})

	cp, err := svc.CreateCheckpoint(context.Background(), "tenant_1", "doc_1", "user_1", "Milestone")
	if err != nil {
		t.Fatalf("CreateCheckpoint() error = %v", err)
	}
	if cp == nil || cp.ID != "cp_1" {
		t.Fatalf("unexpected checkpoint: %+v", cp)
	}
}

func TestFinalize_FromDraft_OK(t *testing.T) {
	repo := &fakeRepo{docReturn: &domain.Document{ID: "doc_1", Status: domain.DocStatusDraft}}
	svc := application.New(repo, fakeDocgen{}, &fakePresigner{}, fakeTplReader{}, fakeFormVal{valid: true}, &noopAudit{})

	err := svc.Finalize(context.Background(), "tenant_1", "doc_1", "user_1")
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
	repo := &fakeRepo{updateStatusErr: domain.ErrInvalidStateTransition}
	svc := application.New(repo, fakeDocgen{}, &fakePresigner{}, fakeTplReader{}, fakeFormVal{valid: true}, &noopAudit{})

	err := svc.Finalize(context.Background(), "tenant_1", "doc_1", "user_1")
	if !errors.Is(err, domain.ErrInvalidStateTransition) {
		t.Fatalf("expected ErrInvalidStateTransition, got %v", err)
	}
	if repo.statusCalls != 1 {
		t.Fatalf("expected one status update call, got %d", repo.statusCalls)
	}
}

func TestRenameDocument_OK(t *testing.T) {
	repo := &fakeRepo{docReturn: &domain.Document{ID: "doc_1", TenantID: "tenant_1", Status: domain.DocStatusDraft}}
	svc := application.New(repo, fakeDocgen{}, &fakePresigner{}, fakeTplReader{}, fakeFormVal{valid: true}, &noopAudit{})

	err := svc.RenameDocument(context.Background(), "tenant_1", "user_1", "doc_1", "  New Name  ")
	if err != nil {
		t.Fatalf("RenameDocument() error = %v", err)
	}
	if repo.renameTenantID != "tenant_1" || repo.renameDocID != "doc_1" || repo.renameName != "New Name" {
		t.Fatalf("unexpected rename args: tenant=%q doc=%q name=%q", repo.renameTenantID, repo.renameDocID, repo.renameName)
	}
}

func TestRenameDocument_InvalidName(t *testing.T) {
	repo := &fakeRepo{}
	svc := application.New(repo, fakeDocgen{}, &fakePresigner{}, fakeTplReader{}, fakeFormVal{valid: true}, &noopAudit{})

	err := svc.RenameDocument(context.Background(), "tenant_1", "user_1", "doc_1", "   ")
	if !errors.Is(err, domain.ErrInvalidName) {
		t.Fatalf("expected ErrInvalidName, got %v", err)
	}
}
