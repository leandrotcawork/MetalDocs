package application

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

type fakeReleaseRepo struct {
	steps []string
}

func (f *fakeReleaseRepo) ArchivePreviousReleased(ctx context.Context, documentID string) (uuid.UUID, []byte, error) {
	f.steps = append(f.steps, "archive_previous")
	return uuid.New(), []byte("rendered"), nil
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

func (f *fakeReleaseRepo) GetDraft(ctx context.Context, id uuid.UUID) (*draftSnapshot, error) {
	return &draftSnapshot{ID: id, ContentBlocks: []byte(`{"mddm_version":1,"blocks":[],"template_ref":null}`)}, nil
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
