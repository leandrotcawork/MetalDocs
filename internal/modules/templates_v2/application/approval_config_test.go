package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"metaldocs/internal/modules/templates_v2/application"
	"metaldocs/internal/modules/templates_v2/domain"
)

func TestUpsertApprovalConfig_Happy_NeverPublished_AsAuthor(t *testing.T) {
	repo := newFakeRepo()
	tpl := &domain.Template{ID: "tpl-1", TenantID: "tenant-a", CreatedBy: "author-1"}
	repo.templates[tpl.ID] = tpl
	reviewerRole := "reviewer"

	svc := application.New(repo, &fakePresigner{}, fakeClock{}, &fakeUUID{})

	got, err := svc.UpsertApprovalConfig(context.Background(), application.UpsertApprovalConfigCmd{
		TenantID:     "tenant-a",
		ActorUserID:  "author-1",
		TemplateID:   tpl.ID,
		ActorRoles:   []string{"editor"},
		ReviewerRole: &reviewerRole,
		ApproverRole: "approver",
	})
	if err != nil {
		t.Fatalf("UpsertApprovalConfig returned error: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil config")
	}
	if got.TemplateID != tpl.ID {
		t.Fatalf("expected TemplateID %q, got %q", tpl.ID, got.TemplateID)
	}
	if got.ReviewerRole == nil || *got.ReviewerRole != reviewerRole {
		t.Fatalf("expected ReviewerRole %q, got %v", reviewerRole, got.ReviewerRole)
	}
	if got.ApproverRole != "approver" {
		t.Fatalf("expected ApproverRole %q, got %q", "approver", got.ApproverRole)
	}
	stored := repo.approvalConfigs[tpl.ID]
	if stored == nil || stored.ApproverRole != "approver" {
		t.Fatalf("expected stored approval config with approver role %q", "approver")
	}
	if len(repo.audit) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(repo.audit))
	}
	if repo.audit[0].Action != domain.AuditApprovalConfigUpdated {
		t.Fatalf("expected audit action %q, got %q", domain.AuditApprovalConfigUpdated, repo.audit[0].Action)
	}
	if repo.audit[0].Details["approver_role"] != "approver" {
		t.Fatalf("expected approver_role %q, got %v", "approver", repo.audit[0].Details["approver_role"])
	}
	reviewerDetail, ok := repo.audit[0].Details["reviewer_role"].(*string)
	if !ok || reviewerDetail == nil || *reviewerDetail != reviewerRole {
		t.Fatalf("expected reviewer_role detail %q, got %v", reviewerRole, repo.audit[0].Details["reviewer_role"])
	}
}

func TestUpsertApprovalConfig_Happy_NeverPublished_AsAdmin(t *testing.T) {
	repo := newFakeRepo()
	tpl := &domain.Template{ID: "tpl-1", TenantID: "tenant-a", CreatedBy: "author-1"}
	repo.templates[tpl.ID] = tpl

	svc := application.New(repo, &fakePresigner{}, fakeClock{}, &fakeUUID{})

	got, err := svc.UpsertApprovalConfig(context.Background(), application.UpsertApprovalConfigCmd{
		TenantID:     "tenant-a",
		ActorUserID:  "admin-1",
		TemplateID:   tpl.ID,
		ActorRoles:   []string{"admin"},
		ReviewerRole: nil,
		ApproverRole: "approver",
	})
	if err != nil {
		t.Fatalf("UpsertApprovalConfig returned error: %v", err)
	}
	if got.ReviewerRole != nil {
		t.Fatalf("expected nil ReviewerRole, got %v", got.ReviewerRole)
	}
}

func TestUpsertApprovalConfig_Happy_EverPublished_AsAdmin(t *testing.T) {
	repo := newFakeRepo()
	tpl := &domain.Template{ID: "tpl-1", TenantID: "tenant-a", CreatedBy: "author-1", PublishedVersionID: strPtr("ver-1")}
	repo.templates[tpl.ID] = tpl

	svc := application.New(repo, &fakePresigner{}, fakeClock{}, &fakeUUID{})

	_, err := svc.UpsertApprovalConfig(context.Background(), application.UpsertApprovalConfigCmd{
		TenantID:     "tenant-a",
		ActorUserID:  "admin-1",
		TemplateID:   tpl.ID,
		ActorRoles:   []string{"admin"},
		ApproverRole: "approver",
	})
	if err != nil {
		t.Fatalf("UpsertApprovalConfig returned error: %v", err)
	}
}

func TestUpsertApprovalConfig_Forbidden_NeverPublished_NonAuthor_NonAdmin(t *testing.T) {
	repo := newFakeRepo()
	tpl := &domain.Template{ID: "tpl-1", TenantID: "tenant-a", CreatedBy: "author-1"}
	repo.templates[tpl.ID] = tpl

	svc := application.New(repo, &fakePresigner{}, fakeClock{}, &fakeUUID{})

	_, err := svc.UpsertApprovalConfig(context.Background(), application.UpsertApprovalConfigCmd{
		TenantID:     "tenant-a",
		ActorUserID:  "user-2",
		TemplateID:   tpl.ID,
		ActorRoles:   []string{"editor"},
		ApproverRole: "approver",
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestUpsertApprovalConfig_Forbidden_EverPublished_NonAdmin(t *testing.T) {
	repo := newFakeRepo()
	tpl := &domain.Template{ID: "tpl-1", TenantID: "tenant-a", CreatedBy: "author-1", PublishedVersionID: strPtr("ver-1")}
	repo.templates[tpl.ID] = tpl

	svc := application.New(repo, &fakePresigner{}, fakeClock{}, &fakeUUID{})

	_, err := svc.UpsertApprovalConfig(context.Background(), application.UpsertApprovalConfigCmd{
		TenantID:     "tenant-a",
		ActorUserID:  "author-1",
		TemplateID:   tpl.ID,
		ActorRoles:   []string{"editor"},
		ApproverRole: "approver",
	})
	if !errors.Is(err, domain.ErrForbidden) {
		t.Fatalf("expected ErrForbidden, got %v", err)
	}
}

func TestUpsertApprovalConfig_Archived(t *testing.T) {
	repo := newFakeRepo()
	archivedAt := time.Date(2026, 4, 20, 10, 0, 0, 0, time.UTC)
	tpl := &domain.Template{ID: "tpl-1", TenantID: "tenant-a", CreatedBy: "author-1", ArchivedAt: &archivedAt}
	repo.templates[tpl.ID] = tpl

	svc := application.New(repo, &fakePresigner{}, fakeClock{}, &fakeUUID{})

	_, err := svc.UpsertApprovalConfig(context.Background(), application.UpsertApprovalConfigCmd{
		TenantID:     "tenant-a",
		ActorUserID:  "admin-1",
		TemplateID:   tpl.ID,
		ActorRoles:   []string{"admin"},
		ApproverRole: "approver",
	})
	if !errors.Is(err, domain.ErrArchived) {
		t.Fatalf("expected ErrArchived, got %v", err)
	}
}

func TestUpsertApprovalConfig_EmptyApproverRole(t *testing.T) {
	repo := newFakeRepo()
	tpl := &domain.Template{ID: "tpl-1", TenantID: "tenant-a", CreatedBy: "author-1"}
	repo.templates[tpl.ID] = tpl

	svc := application.New(repo, &fakePresigner{}, fakeClock{}, &fakeUUID{})

	_, err := svc.UpsertApprovalConfig(context.Background(), application.UpsertApprovalConfigCmd{
		TenantID:     "tenant-a",
		ActorUserID:  "author-1",
		TemplateID:   tpl.ID,
		ActorRoles:   []string{"editor"},
		ApproverRole: "",
	})
	if !errors.Is(err, domain.ErrInvalidApprovalConfig) {
		t.Fatalf("expected ErrInvalidApprovalConfig, got %v", err)
	}
}
