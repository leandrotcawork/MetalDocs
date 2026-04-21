package application_test

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"metaldocs/internal/modules/documents_v2/application"
	"metaldocs/internal/modules/documents_v2/domain"
	iamapp "metaldocs/internal/modules/iam/application"
	iamdomain "metaldocs/internal/modules/iam/domain"
	registrydomain "metaldocs/internal/modules/registry/domain"
)

type fakeRegistryReader struct {
	cd  *registrydomain.ControlledDocument
	err error
}

func (f *fakeRegistryReader) GetByID(_ context.Context, _, _ string) (*registrydomain.ControlledDocument, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.cd, nil
}

type fakeAuthzChecker struct {
	err error
}

func (f *fakeAuthzChecker) Check(_ context.Context, _, _ string, _ iamdomain.Capability, _ iamapp.ResourceCtx) error {
	return f.err
}

type fakeProfileDefaultTemplateReader struct {
	id     *string
	status *string
	err    error
}

func (f *fakeProfileDefaultTemplateReader) GetDefaultTemplateVersionID(_ context.Context, _, _ string) (*string, *string, error) {
	return f.id, f.status, f.err
}

type captureRepo struct {
	*fakeRepo
	createdDoc  *domain.Document
	initialHash string
}

func (r *captureRepo) CreateDocument(_ context.Context, d *domain.Document, initialContentHash string) (string, string, string, error) {
	r.createdDoc = d
	r.initialHash = initialContentHash
	return r.fakeRepo.CreateDocument(context.Background(), d, initialContentHash)
}

func strptr(v string) *string { return &v }

func TestCreate_FromRegistry_Happy(t *testing.T) {
	repo := &captureRepo{fakeRepo: &fakeRepo{createDocIDs: [3]string{"doc_1", "rev_1", "sess_1"}}}
	cd := &registrydomain.ControlledDocument{
		ID:              "cd_1",
		TenantID:        "tenant_1",
		ProfileCode:     "PROC",
		ProcessAreaCode: "AREA-01",
		Status:          registrydomain.CDStatusActive,
	}
	svc := application.NewService(
		repo,
		fakeDocgen{},
		&fakePresigner{hashReturn: "h_initial"},
		fakeTplReader{},
		fakeFormVal{valid: true},
		&noopAudit{},
		&fakeRegistryReader{cd: cd},
		&fakeAuthzChecker{},
		&fakeProfileDefaultTemplateReader{id: strptr("tpl_ver_default"), status: strptr("published")},
	)

	res, err := svc.CreateDocument(context.Background(), application.CreateDocumentInput{
		TenantID:             "tenant_1",
		ActorUserID:          "user_1",
		ControlledDocumentID: "cd_1",
		Name:                 "Contract",
		FormData:             json.RawMessage(`{"a":1}`),
	})
	if err != nil {
		t.Fatalf("CreateDocument() error = %v", err)
	}
	if res.DocumentID != "doc_1" {
		t.Fatalf("expected doc_1, got %q", res.DocumentID)
	}
	if repo.createdDoc == nil {
		t.Fatalf("expected repo.CreateDocument to receive document")
	}
	if repo.createdDoc.ControlledDocumentID == nil || *repo.createdDoc.ControlledDocumentID != "cd_1" {
		t.Fatalf("expected ControlledDocumentID snapshot")
	}
	if repo.createdDoc.ProfileCodeSnapshot == nil || *repo.createdDoc.ProfileCodeSnapshot != "PROC" {
		t.Fatalf("expected ProfileCodeSnapshot snapshot")
	}
	if repo.createdDoc.ProcessAreaCodeSnapshot == nil || *repo.createdDoc.ProcessAreaCodeSnapshot != "AREA-01" {
		t.Fatalf("expected ProcessAreaCodeSnapshot snapshot")
	}
	if repo.createdDoc.TemplateVersionID != "tpl_ver_default" {
		t.Fatalf("expected resolved template version, got %q", repo.createdDoc.TemplateVersionID)
	}
}

func TestCreate_CD_NotActive(t *testing.T) {
	repo := &captureRepo{fakeRepo: &fakeRepo{}}
	svc := application.NewService(
		repo,
		fakeDocgen{},
		&fakePresigner{},
		fakeTplReader{},
		fakeFormVal{valid: true},
		&noopAudit{},
		&fakeRegistryReader{cd: &registrydomain.ControlledDocument{
			ID:              "cd_1",
			ProfileCode:     "PROC",
			ProcessAreaCode: "AREA-01",
			Status:          registrydomain.CDStatusObsolete,
		}},
		&fakeAuthzChecker{},
		&fakeProfileDefaultTemplateReader{},
	)

	_, err := svc.CreateDocument(context.Background(), application.CreateDocumentInput{
		TenantID:             "tenant_1",
		ActorUserID:          "user_1",
		ControlledDocumentID: "cd_1",
		Name:                 "Contract",
		FormData:             json.RawMessage(`{"a":1}`),
	})
	if !errors.Is(err, registrydomain.ErrCDNotActive) {
		t.Fatalf("expected ErrCDNotActive, got %v", err)
	}
}

func TestCreate_NoDefaultTemplate(t *testing.T) {
	repo := &captureRepo{fakeRepo: &fakeRepo{}}
	svc := application.NewService(
		repo,
		fakeDocgen{},
		&fakePresigner{},
		fakeTplReader{},
		fakeFormVal{valid: true},
		&noopAudit{},
		&fakeRegistryReader{cd: &registrydomain.ControlledDocument{
			ID:              "cd_1",
			ProfileCode:     "PROC",
			ProcessAreaCode: "AREA-01",
			Status:          registrydomain.CDStatusActive,
		}},
		&fakeAuthzChecker{},
		&fakeProfileDefaultTemplateReader{},
	)

	_, err := svc.CreateDocument(context.Background(), application.CreateDocumentInput{
		TenantID:             "tenant_1",
		ActorUserID:          "user_1",
		ControlledDocumentID: "cd_1",
		Name:                 "Contract",
		FormData:             json.RawMessage(`{"a":1}`),
	})
	if !errors.Is(err, registrydomain.ErrProfileHasNoDefaultTemplate) {
		t.Fatalf("expected ErrProfileHasNoDefaultTemplate, got %v", err)
	}
}

func TestCreate_AuthzFail(t *testing.T) {
	repo := &captureRepo{fakeRepo: &fakeRepo{}}
	svc := application.NewService(
		repo,
		fakeDocgen{},
		&fakePresigner{},
		fakeTplReader{},
		fakeFormVal{valid: true},
		&noopAudit{},
		&fakeRegistryReader{cd: &registrydomain.ControlledDocument{
			ID:              "cd_1",
			ProfileCode:     "PROC",
			ProcessAreaCode: "AREA-01",
			Status:          registrydomain.CDStatusActive,
		}},
		&fakeAuthzChecker{err: iamapp.ErrAccessDenied},
		&fakeProfileDefaultTemplateReader{id: strptr("tpl_ver_default"), status: strptr("published")},
	)

	_, err := svc.CreateDocument(context.Background(), application.CreateDocumentInput{
		TenantID:             "tenant_1",
		ActorUserID:          "user_1",
		ControlledDocumentID: "cd_1",
		Name:                 "Contract",
		FormData:             json.RawMessage(`{"a":1}`),
	})
	if !errors.Is(err, iamapp.ErrAccessDenied) {
		t.Fatalf("expected ErrAccessDenied, got %v", err)
	}
}

func TestCreate_NoControlledDocID(t *testing.T) {
	repo := &captureRepo{fakeRepo: &fakeRepo{}}
	svc := application.NewService(
		repo,
		fakeDocgen{},
		&fakePresigner{},
		fakeTplReader{},
		fakeFormVal{valid: true},
		&noopAudit{},
		&fakeRegistryReader{},
		&fakeAuthzChecker{},
		&fakeProfileDefaultTemplateReader{},
	)

	_, err := svc.CreateDocument(context.Background(), application.CreateDocumentInput{
		TenantID:    "tenant_1",
		ActorUserID: "user_1",
		Name:        "Contract",
		FormData:    json.RawMessage(`{"a":1}`),
	})
	if !errors.Is(err, application.ErrControlledDocumentRequired) {
		t.Fatalf("expected ErrControlledDocumentRequired, got %v", err)
	}
}
