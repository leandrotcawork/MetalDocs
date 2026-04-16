package application_test

import (
	"context"
	"testing"

	"metaldocs/internal/modules/documents/application"
	"metaldocs/internal/modules/documents/domain"
	"metaldocs/internal/modules/documents/infrastructure/memory"
)

func makeCK5Service(t *testing.T) (*application.Service, *memory.Repository) {
	t.Helper()
	repo := memory.NewRepository()
	svc := application.NewService(repo, nil, nil)
	return svc, repo
}

func seedDocWithVersion(t *testing.T, repo *memory.Repository, docID, html string) {
	t.Helper()
	ctx := context.Background()
	doc := domain.Document{
		ID:              docID,
		DocumentProfile: "po",
	}
	ver := domain.Version{
		DocumentID:    docID,
		Number:        1,
		Content:       html,
		ContentHash:   "abc123",
		ContentSource: domain.ContentSourceNative,
	}
	if err := repo.CreateDocumentWithInitialVersion(ctx, doc, ver); err != nil {
		t.Fatalf("seed doc+version: %v", err)
	}
}

func TestGetCK5DocumentContentAuthorized(t *testing.T) {
	svc, repo := makeCK5Service(t)
	seedDocWithVersion(t, repo, "doc-1", "<p>Hello CK5</p>")

	ctx := context.Background()
	html, err := svc.GetCK5DocumentContentAuthorized(ctx, "doc-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if html != "<p>Hello CK5</p>" {
		t.Errorf("got %q, want %q", html, "<p>Hello CK5</p>")
	}
}

func TestGetCK5DocumentContentAuthorized_NotFound(t *testing.T) {
	svc, _ := makeCK5Service(t)
	_, err := svc.GetCK5DocumentContentAuthorized(context.Background(), "missing")
	if err == nil {
		t.Fatal("expected error for missing document, got nil")
	}
}

func TestSaveCK5DocumentContentAuthorized(t *testing.T) {
	svc, repo := makeCK5Service(t)
	seedDocWithVersion(t, repo, "doc-2", "<p>Old</p>")

	ctx := context.Background()
	if err := svc.SaveCK5DocumentContentAuthorized(ctx, "doc-2", "<p>New</p>"); err != nil {
		t.Fatalf("save: %v", err)
	}

	html, err := svc.GetCK5DocumentContentAuthorized(ctx, "doc-2")
	if err != nil {
		t.Fatalf("get after save: %v", err)
	}
	if html != "<p>New</p>" {
		t.Errorf("got %q, want %q", html, "<p>New</p>")
	}
}
