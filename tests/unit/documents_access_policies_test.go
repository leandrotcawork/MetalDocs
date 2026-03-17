package unit

import (
	"context"
	"testing"
	"time"

	"metaldocs/internal/modules/documents/application"
	"metaldocs/internal/modules/documents/domain"
	"metaldocs/internal/modules/documents/infrastructure/memory"
	iamdomain "metaldocs/internal/modules/iam/domain"
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

func TestCreateDocumentAuthorizedBlockedByTypePolicy(t *testing.T) {
	repo := memory.NewRepository()
	svc := application.NewService(repo, nil, fixedClock{now: time.Date(2026, 3, 17, 10, 0, 0, 0, time.UTC)})

	err := svc.ReplaceAccessPolicies(context.Background(), "document_type", "contract", []domain.AccessPolicy{
		{
			SubjectType: domain.SubjectTypeRole,
			SubjectID:   "reviewer",
			Capability:  domain.CapabilityDocumentCreate,
			Effect:      domain.PolicyEffectAllow,
		},
	})
	if err != nil {
		t.Fatalf("unexpected replace error: %v", err)
	}

	ctx := iamdomain.WithAuthContext(context.Background(), "editor-user", []iamdomain.Role{iamdomain.RoleEditor})
	_, err = svc.CreateDocumentAuthorized(ctx, domain.CreateDocumentCommand{
		DocumentID:   "doc-blocked",
		Title:        "Blocked Contract",
		DocumentType: "contract",
		OwnerID:      "editor-user",
		BusinessUnit: "legal",
		Department:   "contracts",
	})
	if err == nil {
		t.Fatal("expected create to be blocked by policy")
	}
}

func TestListDocumentsAuthorizedFiltersByViewPolicy(t *testing.T) {
	repo := memory.NewRepository()
	svc := application.NewService(repo, nil, fixedClock{now: time.Date(2026, 3, 17, 10, 0, 0, 0, time.UTC)})

	_, err := svc.CreateDocument(context.Background(), domain.CreateDocumentCommand{
		DocumentID:   "doc-visible",
		Title:        "Visible Manual",
		DocumentType: "manual",
		OwnerID:      "owner-a",
		BusinessUnit: "ops",
		Department:   "general",
	})
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}
	_, err = svc.CreateDocument(context.Background(), domain.CreateDocumentCommand{
		DocumentID:   "doc-hidden",
		Title:        "Hidden Manual",
		DocumentType: "manual",
		OwnerID:      "owner-b",
		BusinessUnit: "ops",
		Department:   "general",
	})
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}

	err = svc.ReplaceAccessPolicies(context.Background(), "document", "doc-visible", []domain.AccessPolicy{
		{
			SubjectType: domain.SubjectTypeUser,
			SubjectID:   "viewer-1",
			Capability:  domain.CapabilityDocumentView,
			Effect:      domain.PolicyEffectAllow,
		},
	})
	if err != nil {
		t.Fatalf("unexpected replace error: %v", err)
	}
	err = svc.ReplaceAccessPolicies(context.Background(), "document", "doc-hidden", []domain.AccessPolicy{
		{
			SubjectType: domain.SubjectTypeUser,
			SubjectID:   "other-user",
			Capability:  domain.CapabilityDocumentView,
			Effect:      domain.PolicyEffectAllow,
		},
	})
	if err != nil {
		t.Fatalf("unexpected replace error: %v", err)
	}

	ctx := iamdomain.WithAuthContext(context.Background(), "viewer-1", []iamdomain.Role{iamdomain.RoleViewer})
	items, err := svc.ListDocumentsAuthorized(ctx)
	if err != nil {
		t.Fatalf("unexpected list error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 visible document, got %d", len(items))
	}
	if items[0].ID != "doc-visible" {
		t.Fatalf("expected doc-visible, got %s", items[0].ID)
	}
}
