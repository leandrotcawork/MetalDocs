package domain

import (
	"errors"
	"testing"
	"time"
)

func TestProcessAreaIsActiveWhenNotArchived(t *testing.T) {
	a := ProcessArea{}

	if !a.IsActive() {
		t.Fatal("expected process area to be active when archived_at is nil")
	}
}

func TestProcessAreaArchiveSetsArchivedAt(t *testing.T) {
	a := ProcessArea{}
	now := time.Date(2026, 4, 21, 12, 0, 0, 0, time.UTC)

	if err := a.Archive(now); err != nil {
		t.Fatalf("unexpected error archiving area: %v", err)
	}
	if a.ArchivedAt == nil {
		t.Fatal("expected archived_at to be set")
	}
	if !a.ArchivedAt.Equal(now) {
		t.Fatalf("expected archived_at %s, got %s", now, a.ArchivedAt)
	}
}

func TestProcessAreaArchiveReturnsErrorWhenAlreadyArchived(t *testing.T) {
	archivedAt := time.Date(2026, 4, 20, 10, 0, 0, 0, time.UTC)
	a := ProcessArea{ArchivedAt: &archivedAt}

	err := a.Archive(time.Date(2026, 4, 21, 10, 0, 0, 0, time.UTC))
	if !errors.Is(err, ErrAreaArchived) {
		t.Fatalf("expected ErrAreaArchived, got %v", err)
	}
}
