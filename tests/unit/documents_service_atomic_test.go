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

func (r *atomicRepoSpy) ListDocumentTypes(context.Context) ([]domain.DocumentType, error) {
	return domain.DefaultDocumentTypes(), nil
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

func TestCreateDocumentPrefersAtomicRepositoryWhenAvailable(t *testing.T) {
	repo := &atomicRepoSpy{}
	svc := application.NewService(repo, nil, fixedClock{now: time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)})

	_, err := svc.CreateDocument(context.Background(), domain.CreateDocumentCommand{
		DocumentID:   "doc-atomic",
		Title:        "Atomic",
		DocumentType: "manual",
		OwnerID:      "owner-atomic",
		BusinessUnit: "ops",
		Department:   "general",
		MetadataJSON: map[string]any{
			"manual_code": "MAN-ATOMIC",
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
