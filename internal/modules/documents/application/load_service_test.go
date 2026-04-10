package application

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"metaldocs/internal/modules/documents/domain"
)

type fakeLoadRepo struct {
	draft    *LoadVersion
	released *LoadVersion
	draftErr error
	relErr   error
}

func (f *fakeLoadRepo) GetActiveDraft(ctx context.Context, documentID, userID string) (*LoadVersion, error) {
	return f.draft, f.draftErr
}

func (f *fakeLoadRepo) GetCurrentReleased(ctx context.Context, documentID string) (*LoadVersion, error) {
	return f.released, f.relErr
}

func TestLoadService_PrefersUserDraft(t *testing.T) {
	repo := &fakeLoadRepo{
		draft: &LoadVersion{
			DocumentID:  "PO-118",
			Version:     2,
			Status:      "draft",
			Content:     json.RawMessage(`{"x":"draft"}`),
			TemplateKey: "po",
			ContentHash: "hash-draft",
		},
		released: &LoadVersion{
			DocumentID:  "PO-118",
			Version:     1,
			Status:      "released",
			Content:     json.RawMessage(`{"x":"released"}`),
			TemplateKey: "po",
			ContentHash: "hash-released",
		},
	}
	svc := NewLoadService(repo)

	out, err := svc.LoadForEdit(context.Background(), "PO-118", "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != "draft" {
		t.Fatalf("status = %q, want %q", out.Status, "draft")
	}
	if out.Version != 2 {
		t.Fatalf("version = %d, want %d", out.Version, 2)
	}
	if out.ContentHash != "hash-draft" {
		t.Fatalf("content hash = %q, want %q", out.ContentHash, "hash-draft")
	}
	if out.TemplateKey != "po" {
		t.Fatalf("template key = %q, want %q", out.TemplateKey, "po")
	}
}

func TestLoadService_FallsBackToReleased(t *testing.T) {
	repo := &fakeLoadRepo{
		draft: nil,
		released: &LoadVersion{
			DocumentID:  "PO-118",
			Version:     3,
			Status:      "released",
			Content:     json.RawMessage(`{"x":"released"}`),
			TemplateKey: "po",
			ContentHash: "hash-released",
		},
	}
	svc := NewLoadService(repo)

	out, err := svc.LoadForEdit(context.Background(), "PO-118", "user-1")
	if err != nil {
		t.Fatal(err)
	}
	if out.Status != "released" {
		t.Fatalf("status = %q, want %q", out.Status, "released")
	}
	if out.Version != 3 {
		t.Fatalf("version = %d, want %d", out.Version, 3)
	}
	if out.TemplateKey != "po" {
		t.Fatalf("template key = %q, want %q", out.TemplateKey, "po")
	}
}

func TestLoadService_NotFoundWhenNeitherExists(t *testing.T) {
	repo := &fakeLoadRepo{draft: nil, released: nil}
	svc := NewLoadService(repo)

	_, err := svc.LoadForEdit(context.Background(), "PO-118", "user-1")
	if !errors.Is(err, domain.ErrDocumentNotFound) {
		t.Fatalf("err = %v, want %v", err, domain.ErrDocumentNotFound)
	}
}

func TestLoadService_InvalidInput(t *testing.T) {
	svc := NewLoadService(&fakeLoadRepo{})

	_, err := svc.LoadForEdit(context.Background(), "", "user-1")
	if !errors.Is(err, domain.ErrInvalidCommand) {
		t.Fatalf("err = %v, want %v", err, domain.ErrInvalidCommand)
	}
}

func TestLoadService_PropagatesRepositoryErrors(t *testing.T) {
	wantErr := errors.New("repo failed")
	svc := NewLoadService(&fakeLoadRepo{draftErr: wantErr})

	_, err := svc.LoadForEdit(context.Background(), "PO-118", "user-1")
	if !errors.Is(err, wantErr) {
		t.Fatalf("err = %v, want %v", err, wantErr)
	}
}
