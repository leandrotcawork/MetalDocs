package application_test

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"

	"metaldocs/internal/modules/templates_v2/application"
	"metaldocs/internal/modules/templates_v2/domain"
)

func TestCreateTemplate_Happy(t *testing.T) {
	repo := newFakeRepo()
	svc := application.New(repo, &fakePresigner{}, fakeClock{}, &fakeUUID{})

	cmd := application.CreateTemplateCmd{
		TenantID:      "tenant-a",
		ActorUserID:   "user-a",
		DocTypeCode:   "CONTRACT",
		Key:           "contract-default",
		Name:          "Contract Template",
		Description:   "Default contract",
		Areas:         []string{"legal"},
		Visibility:    domain.VisibilityPublic,
		SpecificAreas: nil,
		ApproverRole:  "approver",
		ReviewerRole:  strPtr("reviewer"),
	}

	got, err := svc.CreateTemplate(context.Background(), cmd)
	if err != nil {
		t.Fatalf("CreateTemplate returned error: %v", err)
	}
	if got == nil || got.Template == nil || got.Version == nil {
		t.Fatal("expected non-nil result with template and version")
	}
	if got.Template.ID != "id_1" {
		t.Fatalf("expected template id id_1, got %q", got.Template.ID)
	}
	if got.Version.ID != "id_2" {
		t.Fatalf("expected version id id_2, got %q", got.Version.ID)
	}
	if got.Template.LatestVersion != 1 {
		t.Fatalf("expected LatestVersion 1, got %d", got.Template.LatestVersion)
	}
	if got.Version.VersionNumber != 1 {
		t.Fatalf("expected VersionNumber 1, got %d", got.Version.VersionNumber)
	}
	if got.Version.TemplateID != got.Template.ID {
		t.Fatalf("expected version.TemplateID %q, got %q", got.Template.ID, got.Version.TemplateID)
	}
	if len(repo.audit) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(repo.audit))
	}
	audit := repo.audit[0]
	if audit.Action != domain.AuditCreated {
		t.Fatalf("expected audit action %q, got %q", domain.AuditCreated, audit.Action)
	}
	if audit.VersionID == nil || *audit.VersionID != got.Version.ID {
		t.Fatalf("expected audit version id %q, got %v", got.Version.ID, audit.VersionID)
	}
	cfg, ok := repo.approvalConfigs[got.Template.ID]
	if !ok {
		t.Fatalf("expected approval config for template %q", got.Template.ID)
	}
	if cfg.TemplateID != got.Template.ID {
		t.Fatalf("expected config template id %q, got %q", got.Template.ID, cfg.TemplateID)
	}
	if cfg.ApproverRole != cmd.ApproverRole {
		t.Fatalf("expected approver role %q, got %q", cmd.ApproverRole, cfg.ApproverRole)
	}
	if cfg.ReviewerRole == nil || *cfg.ReviewerRole != *cmd.ReviewerRole {
		t.Fatalf("expected reviewer role %q, got %v", *cmd.ReviewerRole, cfg.ReviewerRole)
	}
}

func TestCreateTemplate_KeyConflict(t *testing.T) {
	repo := newFakeRepo()
	repo.templates["t1"] = &domain.Template{
		ID:       "t1",
		TenantID: "tenant-a",
		Key:      "contract-default",
	}
	svc := application.New(repo, &fakePresigner{}, fakeClock{}, &fakeUUID{})

	_, err := svc.CreateTemplate(context.Background(), application.CreateTemplateCmd{
		TenantID:     "tenant-a",
		ActorUserID:  "user-a",
		DocTypeCode:  "CONTRACT",
		Key:          "contract-default",
		Name:         "Contract Template",
		Visibility:   domain.VisibilityPublic,
		ApproverRole: "approver",
	})
	if !errors.Is(err, domain.ErrKeyConflict) {
		t.Fatalf("expected ErrKeyConflict, got %v", err)
	}
}

func TestCreateTemplate_InvalidVisibility(t *testing.T) {
	repo := newFakeRepo()
	svc := application.New(repo, &fakePresigner{}, fakeClock{}, &fakeUUID{})

	_, err := svc.CreateTemplate(context.Background(), application.CreateTemplateCmd{
		TenantID:     "tenant-a",
		ActorUserID:  "user-a",
		DocTypeCode:  "CONTRACT",
		Key:          "contract-default",
		Name:         "Contract Template",
		Visibility:   domain.Visibility("weird"),
		ApproverRole: "approver",
	})
	if !errors.Is(err, domain.ErrInvalidVisibility) {
		t.Fatalf("expected ErrInvalidVisibility, got %v", err)
	}
}

func TestCreateTemplate_SpecificNoAreas(t *testing.T) {
	repo := newFakeRepo()
	svc := application.New(repo, &fakePresigner{}, fakeClock{}, &fakeUUID{})

	_, err := svc.CreateTemplate(context.Background(), application.CreateTemplateCmd{
		TenantID:      "tenant-a",
		ActorUserID:   "user-a",
		DocTypeCode:   "CONTRACT",
		Key:           "contract-default",
		Name:          "Contract Template",
		Visibility:    domain.VisibilitySpecific,
		SpecificAreas: nil,
		ApproverRole:  "approver",
	})
	if !errors.Is(err, domain.ErrInvalidVisibility) {
		t.Fatalf("expected ErrInvalidVisibility, got %v", err)
	}
}

func TestCreateTemplate_NonSpecificWithAreas(t *testing.T) {
	repo := newFakeRepo()
	svc := application.New(repo, &fakePresigner{}, fakeClock{}, &fakeUUID{})

	got, err := svc.CreateTemplate(context.Background(), application.CreateTemplateCmd{
		TenantID:      "tenant-a",
		ActorUserID:   "user-a",
		DocTypeCode:   "CONTRACT",
		Key:           "contract-default",
		Name:          "Contract Template",
		Visibility:    domain.VisibilityPublic,
		SpecificAreas: []string{"restricted"},
		ApproverRole:  "approver",
	})
	if err != nil {
		t.Fatalf("CreateTemplate returned error: %v", err)
	}
	if len(got.Template.SpecificAreas) != 0 {
		t.Fatalf("expected empty SpecificAreas, got %v", got.Template.SpecificAreas)
	}
}

func TestCreateNextVersion_FromPublished(t *testing.T) {
	repo := newFakeRepo()
	publishedID := "v1"
	template := &domain.Template{
		ID:                 "tpl-1",
		TenantID:           "tenant-a",
		LatestVersion:      1,
		PublishedVersionID: &publishedID,
	}
	v1 := &domain.TemplateVersion{
		ID:                publishedID,
		TemplateID:        template.ID,
		VersionNumber:     1,
		Status:            domain.VersionStatusPublished,
		MetadataSchema:    domain.MetadataSchema{DocCodePattern: "ABC-###", RequiredMetadata: []string{"department"}},
		PlaceholderSchema: []domain.Placeholder{{ID: "ph-1", Label: "Signer", Type: domain.PHUser, Required: true}},
	}
	repo.templates[template.ID] = template
	repo.versions[v1.ID] = v1
	svc := application.New(repo, &fakePresigner{}, fakeClock{}, &fakeUUID{})

	got, err := svc.CreateNextVersion(context.Background(), application.CreateVersionCmd{
		TenantID:    "tenant-a",
		ActorUserID: "user-b",
		TemplateID:  template.ID,
	})
	if err != nil {
		t.Fatalf("CreateNextVersion returned error: %v", err)
	}
	if got.VersionNumber != 2 {
		t.Fatalf("expected version number 2, got %d", got.VersionNumber)
	}
	if !reflect.DeepEqual(got.MetadataSchema, v1.MetadataSchema) {
		t.Fatalf("expected metadata schema to be cloned from published version")
	}
	if !reflect.DeepEqual(got.PlaceholderSchema, v1.PlaceholderSchema) {
		t.Fatalf("expected placeholder schema to be cloned from published version")
	}
	v1.PlaceholderSchema[0].Label = "Mutated"
	if got.PlaceholderSchema[0].Label == "Mutated" {
		t.Fatal("expected placeholder schema to be deep-cloned, but got aliasing")
	}
}

func TestCreateNextVersion_NoPublished_ClonesLatest(t *testing.T) {
	repo := newFakeRepo()
	template := &domain.Template{
		ID:            "tpl-1",
		TenantID:      "tenant-a",
		LatestVersion: 1,
	}
	v1 := &domain.TemplateVersion{
		ID:                "v1",
		TemplateID:        template.ID,
		VersionNumber:     1,
		Status:            domain.VersionStatusDraft,
		MetadataSchema:    domain.MetadataSchema{DocCodePattern: "XYZ-###", RequiredMetadata: []string{"site"}},
		PlaceholderSchema: []domain.Placeholder{{ID: "ph-1", Label: "Department", Type: domain.PHText}},
	}
	repo.templates[template.ID] = template
	repo.versions[v1.ID] = v1
	svc := application.New(repo, &fakePresigner{}, fakeClock{}, &fakeUUID{})

	got, err := svc.CreateNextVersion(context.Background(), application.CreateVersionCmd{
		TenantID:    "tenant-a",
		ActorUserID: "user-b",
		TemplateID:  template.ID,
	})
	if err != nil {
		t.Fatalf("CreateNextVersion returned error: %v", err)
	}
	if got.VersionNumber != 2 {
		t.Fatalf("expected version number 2, got %d", got.VersionNumber)
	}
	if !reflect.DeepEqual(got.MetadataSchema, v1.MetadataSchema) {
		t.Fatalf("expected metadata schema to be cloned from latest version")
	}
	if !reflect.DeepEqual(got.PlaceholderSchema, v1.PlaceholderSchema) {
		t.Fatalf("expected placeholder schema to be cloned from latest version")
	}
}

func TestCreateNextVersion_Archived(t *testing.T) {
	repo := newFakeRepo()
	archivedAt := time.Date(2026, 4, 20, 12, 0, 0, 0, time.UTC)
	template := &domain.Template{
		ID:            "tpl-1",
		TenantID:      "tenant-a",
		LatestVersion: 1,
		ArchivedAt:    &archivedAt,
	}
	repo.templates[template.ID] = template
	svc := application.New(repo, &fakePresigner{}, fakeClock{}, &fakeUUID{})

	_, err := svc.CreateNextVersion(context.Background(), application.CreateVersionCmd{
		TenantID:    "tenant-a",
		ActorUserID: "user-b",
		TemplateID:  template.ID,
	})
	if !errors.Is(err, domain.ErrArchived) {
		t.Fatalf("expected ErrArchived, got %v", err)
	}
}

func strPtr(v string) *string {
	return &v
}
