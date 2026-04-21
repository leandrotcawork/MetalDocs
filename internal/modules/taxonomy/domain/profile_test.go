package domain

import (
	"errors"
	"testing"
	"time"
)

func TestDocumentProfileIsActiveWhenNotArchived(t *testing.T) {
	p := DocumentProfile{}

	if !p.IsActive() {
		t.Fatal("expected profile to be active when archived_at is nil")
	}
}

func TestDocumentProfileArchiveSetsArchivedAt(t *testing.T) {
	p := DocumentProfile{}
	now := time.Date(2026, 4, 21, 12, 0, 0, 0, time.UTC)

	if err := p.Archive(now); err != nil {
		t.Fatalf("unexpected error archiving profile: %v", err)
	}
	if p.ArchivedAt == nil {
		t.Fatal("expected archived_at to be set")
	}
	if !p.ArchivedAt.Equal(now) {
		t.Fatalf("expected archived_at %s, got %s", now, p.ArchivedAt)
	}
}

func TestDocumentProfileArchiveReturnsErrorWhenAlreadyArchived(t *testing.T) {
	archivedAt := time.Date(2026, 4, 20, 10, 0, 0, 0, time.UTC)
	p := DocumentProfile{ArchivedAt: &archivedAt}

	err := p.Archive(time.Date(2026, 4, 21, 10, 0, 0, 0, time.UTC))
	if !errors.Is(err, ErrProfileArchived) {
		t.Fatalf("expected ErrProfileArchived, got %v", err)
	}
}
