package domain_test

import (
	"testing"
	"time"

	"metaldocs/internal/modules/templates_v2/domain"
)

func TestTemplate_IsArchived_TrueWhenSet(t *testing.T) {
	now := time.Now().UTC()
	tpl := domain.Template{ArchivedAt: &now}

	if !tpl.IsArchived() {
		t.Fatal("expected template to be archived when ArchivedAt is set")
	}
}

func TestTemplate_IsArchived_FalseWhenNil(t *testing.T) {
	tpl := domain.Template{}

	if tpl.IsArchived() {
		t.Fatal("expected template to not be archived when ArchivedAt is nil")
	}
}
