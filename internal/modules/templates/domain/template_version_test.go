package domain_test

import (
	"testing"

	"metaldocs/internal/modules/templates/domain"
)

func TestTemplateVersion_TransitionDraftToPublished(t *testing.T) {
	v := domain.NewTemplateVersion("tpl1", 1)
	if v.Status != domain.StatusDraft {
		t.Fatalf("new version should be draft")
	}
	if err := v.Publish("user1"); err != nil {
		t.Fatalf("publish: %v", err)
	}
	if v.Status != domain.StatusPublished {
		t.Fatalf("expected published")
	}
	if v.PublishedAt == nil || v.PublishedBy == nil {
		t.Fatalf("published metadata missing")
	}
}

func TestTemplateVersion_CannotPublishTwice(t *testing.T) {
	v := domain.NewTemplateVersion("tpl1", 1)
	_ = v.Publish("u1")
	if err := v.Publish("u1"); err != domain.ErrInvalidStateTransition {
		t.Fatalf("expected ErrInvalidStateTransition, got %v", err)
	}
}

func TestTemplateVersion_Deprecate(t *testing.T) {
	v := domain.NewTemplateVersion("tpl1", 1)
	_ = v.Publish("u1")
	if err := v.Deprecate(); err != nil {
		t.Fatalf("deprecate: %v", err)
	}
	if v.Status != domain.StatusDeprecated {
		t.Fatalf("expected deprecated")
	}
}

func TestTemplateVersion_IncrementLockVersionOnDraftEdit(t *testing.T) {
	v := domain.NewTemplateVersion("tpl1", 1)
	before := v.LockVersion
	if err := v.ApplyDraftEdit(before); err != nil {
		t.Fatalf("apply: %v", err)
	}
	if v.LockVersion != before+1 {
		t.Fatalf("expected lock bump")
	}
}

func TestTemplateVersion_OptimisticLockMismatch(t *testing.T) {
	v := domain.NewTemplateVersion("tpl1", 1)
	if err := v.ApplyDraftEdit(99); err != domain.ErrLockVersionMismatch {
		t.Fatalf("expected ErrLockVersionMismatch, got %v", err)
	}
}
