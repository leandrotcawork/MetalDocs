package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"metaldocs/internal/modules/documents/domain"
	documentmemory "metaldocs/internal/modules/documents/infrastructure/memory"
)

func TestGetCK5DocumentContent_OK(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.April, 16, 10, 0, 0, 0, time.UTC)
	repo := documentmemory.NewRepository()
	service := NewService(repo, nil, fixedClock{now: now})
	doc := seedDraftDocument(t, ctx, repo, now)

	if err := repo.SaveVersion(ctx, domain.Version{
		DocumentID:    doc.ID,
		Number:        1,
		Content:       "native-content",
		ContentHash:   contentHash("native-content"),
		ChangeSummary: "Initial native version",
		ContentSource: domain.ContentSourceNative,
		CreatedAt:     now,
	}); err != nil {
		t.Fatalf("save native version: %v", err)
	}
	if err := repo.SaveVersion(ctx, domain.Version{
		DocumentID:    doc.ID,
		Number:        2,
		Content:       "ck5-html-content",
		ContentHash:   contentHash("ck5-html-content"),
		ChangeSummary: "CK5 browser version",
		ContentSource: domain.ContentSourceCK5Browser,
		CreatedAt:     now.Add(time.Minute),
	}); err != nil {
		t.Fatalf("save ck5 version: %v", err)
	}

	html, title, err := service.GetCK5DocumentContent(ctx, doc.ID)
	if err != nil {
		t.Fatalf("GetCK5DocumentContent() error = %v", err)
	}
	if html != "ck5-html-content" {
		t.Fatalf("html = %q, want %q", html, "ck5-html-content")
	}
	if title != doc.Title {
		t.Fatalf("title = %q, want %q", title, doc.Title)
	}
}

func TestGetCK5DocumentContent_NoCK5Version(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.April, 16, 10, 0, 0, 0, time.UTC)
	repo := documentmemory.NewRepository()
	service := NewService(repo, nil, fixedClock{now: now})
	doc := seedDraftDocument(t, ctx, repo, now)

	if err := repo.SaveVersion(ctx, domain.Version{
		DocumentID:    doc.ID,
		Number:        1,
		Content:       "native-content",
		ContentHash:   contentHash("native-content"),
		ChangeSummary: "Initial native version",
		ContentSource: domain.ContentSourceNative,
		CreatedAt:     now,
	}); err != nil {
		t.Fatalf("save native version: %v", err)
	}

	_, _, err := service.GetCK5DocumentContent(ctx, doc.ID)
	if !errors.Is(err, domain.ErrDocumentNotFound) {
		t.Fatalf("err = %v, want ErrDocumentNotFound", err)
	}
}

func TestGetCK5DocumentContent_MissingDoc(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, time.April, 16, 10, 0, 0, 0, time.UTC)
	repo := documentmemory.NewRepository()
	service := NewService(repo, nil, fixedClock{now: now})

	_, _, err := service.GetCK5DocumentContent(ctx, "missing-doc-id")
	if !errors.Is(err, domain.ErrDocumentNotFound) {
		t.Fatalf("err = %v, want ErrDocumentNotFound", err)
	}
}
