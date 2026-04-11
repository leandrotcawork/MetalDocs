package application

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"metaldocs/internal/modules/documents/domain"
)

type fakeReleaseRepo struct {
	steps []string
}

func (f *fakeReleaseRepo) ArchivePreviousReleased(ctx context.Context, documentID string) (uuid.UUID, []byte, []byte, error) {
	f.steps = append(f.steps, "archive_previous")
	return uuid.New(), []byte(`{"mddm_version":1,"blocks":[],"template_ref":null}`), []byte("rendered"), nil
}

func (f *fakeReleaseRepo) PromoteDraftToReleased(ctx context.Context, draftID uuid.UUID, docxBytes []byte, approvedBy string) error {
	f.steps = append(f.steps, "promote_draft")
	return nil
}

func (f *fakeReleaseRepo) StoreRevisionDiff(ctx context.Context, versionID uuid.UUID, diff json.RawMessage) error {
	f.steps = append(f.steps, "store_diff")
	return nil
}

func (f *fakeReleaseRepo) DeleteImageRefs(ctx context.Context, versionID uuid.UUID) error {
	f.steps = append(f.steps, "delete_image_refs")
	return nil
}

func (f *fakeReleaseRepo) CleanupOrphanImages(ctx context.Context) error {
	f.steps = append(f.steps, "cleanup_orphans")
	return nil
}

func (f *fakeReleaseRepo) GetDraft(ctx context.Context, id uuid.UUID) (*DraftSnapshot, error) {
	return &DraftSnapshot{ID: id, ContentBlocks: []byte(`{"mddm_version":1,"blocks":[],"template_ref":null}`)}, nil
}

type fakeDocxRenderer struct{}

func (r *fakeDocxRenderer) RenderDocx(ctx context.Context, content []byte) ([]byte, error) {
	return []byte("docx-bytes"), nil
}

func TestReleaseService_AtomicSequence(t *testing.T) {
	repo := &fakeReleaseRepo{}
	renderer := &fakeDocxRenderer{}
	svc := NewReleaseService(repo, renderer)

	err := svc.ReleaseDraft(context.Background(), ReleaseInput{
		DocumentID: "PO-118",
		DraftID:    uuid.New(),
		ApprovedBy: "user-1",
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{"archive_previous", "promote_draft", "store_diff", "delete_image_refs", "cleanup_orphans"}
	if len(repo.steps) != len(expected) {
		t.Fatalf("step count mismatch: %v", repo.steps)
	}
	for i, s := range expected {
		if repo.steps[i] != s {
			t.Errorf("step %d: expected %s, got %s", i, s, repo.steps[i])
		}
	}
}

// fakeReleaseRepoWithBrowserEditor overrides GetDraft to return a browser_editor
// draft with valid template metadata, so the pin capturer fires.
type fakeReleaseRepoWithBrowserEditor struct {
	fakeReleaseRepo
}

func (f *fakeReleaseRepoWithBrowserEditor) GetDraft(ctx context.Context, id uuid.UUID) (*DraftSnapshot, error) {
	return &DraftSnapshot{
		ID:              id,
		ContentBlocks:   []byte(`{"mddm_version":1,"blocks":[],"template_ref":null}`),
		DocumentID:      "PO-118",
		VersionNumber:   3,
		ContentSource:   domain.ContentSourceBrowserEditor,
		TemplateKey:     "po-default-canvas",
		TemplateVersion: 1,
	}, nil
}

// failingPinRepo is a RendererPinRepo whose SetVersionRendererPin always fails.
type failingPinRepo struct {
	pinErr error
}

func (r *failingPinRepo) SetVersionRendererPin(_ context.Context, _ string, _ int, _ *domain.RendererPin) error {
	return r.pinErr
}

// TestReleaseVersion_FailsWhenPinCaptureFails verifies that release is aborted
// when the renderer pin capturer returns an error (fail-loud guarantee).
// The PromoteDraftToReleased step must NOT be reached.
func TestReleaseVersion_FailsWhenPinCaptureFails(t *testing.T) {
	pinErr := errors.New("storage unavailable")
	repo := &fakeReleaseRepoWithBrowserEditor{}
	renderer := &fakeDocxRenderer{}

	capturer := NewRendererPinCapturer(RendererPinCapturerConfig{
		CurrentRendererVersion: "1.0.0",
		CurrentLayoutIRHash:    "abc123def456abc123def456abc123def456abc123def456abc123def456abcd",
		Repo:                   &failingPinRepo{pinErr: pinErr},
		Clock:                  func() time.Time { return time.Now() },
	})

	svc := NewReleaseService(repo, renderer).WithRendererPinCapturer(capturer)

	err := svc.ReleaseDraft(context.Background(), ReleaseInput{
		DocumentID: "PO-118",
		DraftID:    uuid.New(),
		ApprovedBy: "approver-1",
	})

	if err == nil {
		t.Fatal("expected release to fail when pin capture fails, got nil error")
	}
	if !errors.Is(err, pinErr) {
		t.Errorf("expected wrapped pinErr, got: %v", err)
	}

	// PromoteDraftToReleased must NOT have been called — abort before status write.
	for _, step := range repo.steps {
		if step == "promote_draft" {
			t.Errorf("promote_draft was called despite pin capture failure; steps: %v", repo.steps)
		}
	}
}
