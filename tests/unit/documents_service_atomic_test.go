package unit

import (
	"context"
	"testing"
	"time"

	"metaldocs/internal/modules/documents/application"
	"metaldocs/internal/modules/documents/domain"
)

type atomicRepoSpy struct {
	createCalled bool
	saveCalled   bool
	atomicCalled bool
	versions     map[int]domain.Version
}

func (r *atomicRepoSpy) CreateDocument(context.Context, domain.Document) error {
	r.createCalled = true
	return nil
}

func (r *atomicRepoSpy) CreateDocumentWithInitialVersion(context.Context, domain.Document, domain.Version) error {
	r.atomicCalled = true
	return nil
}

func (r *atomicRepoSpy) GetDocument(context.Context, string) (domain.Document, error) {
	return domain.Document{}, nil
}

func (r *atomicRepoSpy) ListDocuments(context.Context) ([]domain.Document, error) {
	return nil, nil
}

func (r *atomicRepoSpy) ListDocumentsForReviewReminder(context.Context, time.Time, time.Time) ([]domain.Document, error) {
	return nil, nil
}

func (r *atomicRepoSpy) ListDocumentTypes(context.Context) ([]domain.DocumentType, error) {
	return domain.DefaultDocumentTypes(), nil
}

func (r *atomicRepoSpy) ListDocumentFamilies(context.Context) ([]domain.DocumentFamily, error) {
	return domain.DefaultDocumentFamilies(), nil
}

func (r *atomicRepoSpy) ListDocumentProfiles(context.Context) ([]domain.DocumentProfile, error) {
	return domain.DefaultDocumentProfiles(), nil
}

func (r *atomicRepoSpy) UpsertDocumentProfile(context.Context, domain.DocumentProfile) error {
	return nil
}

func (r *atomicRepoSpy) DeactivateDocumentProfile(context.Context, string) error {
	return nil
}

func (r *atomicRepoSpy) ListDocumentProfileSchemas(_ context.Context, profileCode string) ([]domain.DocumentProfileSchemaVersion, error) {
	items := domain.DefaultDocumentProfileSchemas()
	if profileCode == "" {
		return items, nil
	}
	filtered := make([]domain.DocumentProfileSchemaVersion, 0, len(items))
	for _, item := range items {
		if item.ProfileCode == profileCode {
			filtered = append(filtered, item)
		}
	}
	return filtered, nil
}

func (r *atomicRepoSpy) UpsertDocumentProfileSchemaVersion(context.Context, domain.DocumentProfileSchemaVersion) error {
	return nil
}

func (r *atomicRepoSpy) ActivateDocumentProfileSchemaVersion(context.Context, string, int) error {
	return nil
}

func (r *atomicRepoSpy) GetDocumentProfileGovernance(_ context.Context, profileCode string) (domain.DocumentProfileGovernance, error) {
	for _, item := range domain.DefaultDocumentProfileGovernance() {
		if item.ProfileCode == profileCode {
			return item, nil
		}
	}
	return domain.DocumentProfileGovernance{}, domain.ErrInvalidCommand
}

func (r *atomicRepoSpy) UpsertDocumentProfileGovernance(context.Context, domain.DocumentProfileGovernance) error {
	return nil
}

func (r *atomicRepoSpy) ListProcessAreas(context.Context) ([]domain.ProcessArea, error) {
	return domain.DefaultProcessAreas(), nil
}

func (r *atomicRepoSpy) UpsertProcessArea(context.Context, domain.ProcessArea) error {
	return nil
}

func (r *atomicRepoSpy) DeactivateProcessArea(context.Context, string) error {
	return nil
}

func (r *atomicRepoSpy) ListDocumentDepartments(context.Context) ([]domain.DocumentDepartment, error) {
	return domain.DefaultDocumentDepartments(), nil
}

func (r *atomicRepoSpy) UpsertDocumentDepartment(context.Context, domain.DocumentDepartment) error {
	return nil
}

func (r *atomicRepoSpy) DeactivateDocumentDepartment(context.Context, string) error {
	return nil
}

func (r *atomicRepoSpy) ListSubjects(context.Context) ([]domain.Subject, error) {
	return domain.DefaultSubjects(), nil
}

func (r *atomicRepoSpy) UpsertSubject(context.Context, domain.Subject) error {
	return nil
}

func (r *atomicRepoSpy) DeactivateSubject(context.Context, string) error {
	return nil
}

func (r *atomicRepoSpy) ListAccessPolicies(context.Context, string, string) ([]domain.AccessPolicy, error) {
	return nil, nil
}

func (r *atomicRepoSpy) ReplaceAccessPolicies(context.Context, string, string, []domain.AccessPolicy) error {
	return nil
}

func (r *atomicRepoSpy) UpdateDocumentStatus(context.Context, string, string) error {
	return nil
}

func (r *atomicRepoSpy) SaveVersion(context.Context, domain.Version) error {
	r.saveCalled = true
	return nil
}

func (r *atomicRepoSpy) UpdateVersionPDF(context.Context, string, int, string, int) error {
	return nil
}

func (r *atomicRepoSpy) ListVersions(context.Context, string) ([]domain.Version, error) {
	return nil, nil
}

func (r *atomicRepoSpy) GetVersion(context.Context, string, int) (domain.Version, error) {
	return domain.Version{}, domain.ErrVersionNotFound
}

func (r *atomicRepoSpy) NextVersionNumber(context.Context, string) (int, error) {
	return 1, nil
}

func (r *atomicRepoSpy) CreateAttachment(context.Context, domain.Attachment) error {
	return nil
}

func (r *atomicRepoSpy) GetAttachment(context.Context, string) (domain.Attachment, error) {
	return domain.Attachment{}, domain.ErrAttachmentNotFound
}

func (r *atomicRepoSpy) ListAttachments(context.Context, string) ([]domain.Attachment, error) {
	return nil, nil
}

func (r *atomicRepoSpy) UpsertCollaborationPresence(context.Context, domain.CollaborationPresence) error {
	return nil
}

func (r *atomicRepoSpy) ListCollaborationPresence(context.Context, string, time.Time) ([]domain.CollaborationPresence, error) {
	return nil, nil
}

func (r *atomicRepoSpy) AcquireDocumentEditLock(context.Context, domain.DocumentEditLock, time.Time) (domain.DocumentEditLock, error) {
	return domain.DocumentEditLock{}, domain.ErrEditLockNotFound
}

func (r *atomicRepoSpy) GetDocumentEditLock(context.Context, string, time.Time) (domain.DocumentEditLock, error) {
	return domain.DocumentEditLock{}, domain.ErrEditLockNotFound
}

func (r *atomicRepoSpy) ReleaseDocumentEditLock(context.Context, string, string) error {
	return domain.ErrEditLockNotFound
}

func TestCreateDocumentPrefersAtomicRepositoryWhenAvailable(t *testing.T) {
	repo := &atomicRepoSpy{}
	svc := application.NewService(repo, nil, fixedClock{now: time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)})

	_, err := svc.CreateDocument(context.Background(), domain.CreateDocumentCommand{
		DocumentID:   "doc-atomic",
		Title:        "Atomic",
		DocumentType: "it",
		OwnerID:      "owner-atomic",
		BusinessUnit: "ops",
		Department:   "general",
		MetadataJSON: map[string]any{
			"instruction_code": "IT-ATOMIC",
		},
		InitialContent: "v1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !repo.atomicCalled {
		t.Fatal("expected atomic create to be used")
	}
	if repo.createCalled {
		t.Fatal("did not expect CreateDocument fallback when atomic is available")
	}
	if repo.saveCalled {
		t.Fatal("did not expect SaveVersion fallback when atomic is available")
	}
}
