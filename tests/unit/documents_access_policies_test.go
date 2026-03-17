package unit

import (
	"context"
	"testing"
	"time"

	"metaldocs/internal/modules/documents/application"
	"metaldocs/internal/modules/documents/domain"
	"metaldocs/internal/modules/documents/infrastructure/memory"
)

func TestReplaceAndListAccessPolicies(t *testing.T) {
	repo := memory.NewRepository()
	svc := application.NewService(repo, nil, fixedClock{now: time.Date(2026, 3, 17, 10, 0, 0, 0, time.UTC)})

	err := svc.ReplaceAccessPolicies(context.Background(), "document", "doc-123", []domain.AccessPolicy{
		{
			SubjectType: domain.SubjectTypeUser,
			SubjectID:   "leandro",
			Capability:  domain.CapabilityDocumentView,
			Effect:      domain.PolicyEffectAllow,
		},
		{
			SubjectType: domain.SubjectTypeRole,
			SubjectID:   "editor",
			Capability:  domain.CapabilityDocumentEdit,
			Effect:      domain.PolicyEffectAllow,
		},
	})
	if err != nil {
		t.Fatalf("unexpected replace error: %v", err)
	}

	items, err := svc.ListAccessPolicies(context.Background(), "document", "doc-123")
	if err != nil {
		t.Fatalf("unexpected list error: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 policies, got %d", len(items))
	}
	if items[0].ResourceScope != domain.ResourceScopeDocument {
		t.Fatalf("expected resource scope document, got %s", items[0].ResourceScope)
	}
}

func TestReplaceAccessPoliciesRejectsInvalidCapability(t *testing.T) {
	repo := memory.NewRepository()
	svc := application.NewService(repo, nil, fixedClock{now: time.Now().UTC()})

	err := svc.ReplaceAccessPolicies(context.Background(), "document", "doc-123", []domain.AccessPolicy{
		{
			SubjectType: domain.SubjectTypeUser,
			SubjectID:   "leandro",
			Capability:  "document.delete",
			Effect:      domain.PolicyEffectAllow,
		},
	})
	if err == nil {
		t.Fatal("expected invalid access policy error")
	}
}
