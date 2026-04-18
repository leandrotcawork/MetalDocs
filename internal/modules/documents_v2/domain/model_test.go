package domain_test

import (
	"testing"

	"metaldocs/internal/modules/documents_v2/domain"
)

func TestCanTransitionDocument(t *testing.T) {
	cases := []struct {
		cur, next domain.DocumentStatus
		ok        bool
	}{
		{domain.DocStatusDraft, domain.DocStatusFinalized, true},
		{domain.DocStatusDraft, domain.DocStatusArchived, true},
		{domain.DocStatusFinalized, domain.DocStatusArchived, true},
		{domain.DocStatusArchived, domain.DocStatusDraft, false},
		{domain.DocStatusArchived, domain.DocStatusFinalized, false},
		{domain.DocStatusFinalized, domain.DocStatusDraft, false},
	}
	for _, c := range cases {
		if got := domain.CanTransitionDocument(c.cur, c.next); got != c.ok {
			t.Errorf("CanTransitionDocument(%s, %s) = %v, want %v", c.cur, c.next, got, c.ok)
		}
	}
}
