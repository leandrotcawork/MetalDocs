package application_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"metaldocs/internal/modules/templates_v2/application"
	"metaldocs/internal/modules/templates_v2/domain"
)

func TestUpdateSchemas_Happy(t *testing.T) {
	repo := newFakeRepo()
	version := &domain.TemplateVersion{
		ID:            "v1",
		TemplateID:    "tpl-1",
		VersionNumber: 1,
		Status:        domain.VersionStatusDraft,
		ContentHash:   "hash-1",
	}
	repo.versions[version.ID] = version

	svc := application.New(repo, &fakePresigner{}, fakeClock{}, &fakeUUID{})
	got, err := svc.UpdateSchemas(context.Background(), application.UpdateSchemasCmd{
		TenantID:      "tenant-a",
		ActorUserID:   "user-a",
		TemplateID:    "tpl-1",
		VersionNumber: 1,
		MetadataSchema: domain.MetadataSchema{
			DocCodePattern: "ABC-###",
		},
		PlaceholderSchema: []domain.Placeholder{
			{ID: "ph-1", Label: "Signer", Type: domain.PHSelect, Options: []string{"a", "b"}},
		},
		EditableZones:       []domain.EditableZone{{ID: "zone-1", Label: "Body"}},
		ExpectedContentHash: "hash-1",
	})
	if err != nil {
		t.Fatalf("UpdateSchemas returned error: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil version")
	}
	if got.MetadataSchema.DocCodePattern != "ABC-###" {
		t.Fatalf("expected metadata schema to be updated, got %q", got.MetadataSchema.DocCodePattern)
	}
	if len(got.PlaceholderSchema) != 1 || got.PlaceholderSchema[0].ID != "ph-1" {
		t.Fatalf("expected placeholder schema to be updated, got %v", got.PlaceholderSchema)
	}
	if len(got.EditableZones) != 1 || got.EditableZones[0].ID != "zone-1" {
		t.Fatalf("expected editable zones to be updated, got %v", got.EditableZones)
	}
	if len(repo.audit) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(repo.audit))
	}
	if repo.audit[0].Action != domain.AuditSaved {
		t.Fatalf("expected audit action %q, got %q", domain.AuditSaved, repo.audit[0].Action)
	}
	kind, ok := repo.audit[0].Details["kind"]
	if !ok || kind != "schema" {
		t.Fatalf("expected audit details kind=schema, got %v", repo.audit[0].Details)
	}
}

func TestUpdateSchemas_NonDraft(t *testing.T) {
	repo := newFakeRepo()
	repo.versions["v1"] = &domain.TemplateVersion{
		ID:            "v1",
		TemplateID:    "tpl-1",
		VersionNumber: 1,
		Status:        domain.VersionStatusPublished,
	}
	svc := application.New(repo, &fakePresigner{}, fakeClock{}, &fakeUUID{})

	_, err := svc.UpdateSchemas(context.Background(), application.UpdateSchemasCmd{
		TemplateID:    "tpl-1",
		VersionNumber: 1,
	})
	if !errors.Is(err, domain.ErrInvalidStateTransition) {
		t.Fatalf("expected ErrInvalidStateTransition, got %v", err)
	}
}

func TestUpdateSchemas_StaleHash(t *testing.T) {
	repo := newFakeRepo()
	repo.versions["v1"] = &domain.TemplateVersion{
		ID:            "v1",
		TemplateID:    "tpl-1",
		VersionNumber: 1,
		Status:        domain.VersionStatusDraft,
		ContentHash:   "hash-1",
	}
	svc := application.New(repo, &fakePresigner{}, fakeClock{}, &fakeUUID{})

	_, err := svc.UpdateSchemas(context.Background(), application.UpdateSchemasCmd{
		TemplateID:           "tpl-1",
		VersionNumber:        1,
		ExpectedContentHash:  "hash-2",
		PlaceholderSchema:    []domain.Placeholder{{ID: "ph-1", Type: domain.PHText}},
	})
	if !errors.Is(err, domain.ErrStaleBase) {
		t.Fatalf("expected ErrStaleBase, got %v", err)
	}
}

func TestUpdateSchemas_DuplicatePlaceholderID(t *testing.T) {
	repo := newFakeRepo()
	repo.versions["v1"] = &domain.TemplateVersion{
		ID:            "v1",
		TemplateID:    "tpl-1",
		VersionNumber: 1,
		Status:        domain.VersionStatusDraft,
	}
	svc := application.New(repo, &fakePresigner{}, fakeClock{}, &fakeUUID{})

	_, err := svc.UpdateSchemas(context.Background(), application.UpdateSchemasCmd{
		TemplateID:    "tpl-1",
		VersionNumber: 1,
		PlaceholderSchema: []domain.Placeholder{
			{ID: "ph-1", Type: domain.PHText},
			{ID: "ph-1", Type: domain.PHText},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "duplicate_placeholder_id") {
		t.Fatalf("expected duplicate_placeholder_id error, got %v", err)
	}
}

func TestUpdateSchemas_DuplicateZoneID(t *testing.T) {
	repo := newFakeRepo()
	repo.versions["v1"] = &domain.TemplateVersion{
		ID:            "v1",
		TemplateID:    "tpl-1",
		VersionNumber: 1,
		Status:        domain.VersionStatusDraft,
	}
	svc := application.New(repo, &fakePresigner{}, fakeClock{}, &fakeUUID{})

	_, err := svc.UpdateSchemas(context.Background(), application.UpdateSchemasCmd{
		TemplateID:    "tpl-1",
		VersionNumber: 1,
		EditableZones: []domain.EditableZone{
			{ID: "zone-1"},
			{ID: "zone-1"},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "duplicate_zone_id") {
		t.Fatalf("expected duplicate_zone_id error, got %v", err)
	}
}

func TestUpdateSchemas_SelectNoOptions(t *testing.T) {
	repo := newFakeRepo()
	repo.versions["v1"] = &domain.TemplateVersion{
		ID:            "v1",
		TemplateID:    "tpl-1",
		VersionNumber: 1,
		Status:        domain.VersionStatusDraft,
	}
	svc := application.New(repo, &fakePresigner{}, fakeClock{}, &fakeUUID{})

	_, err := svc.UpdateSchemas(context.Background(), application.UpdateSchemasCmd{
		TemplateID:    "tpl-1",
		VersionNumber: 1,
		PlaceholderSchema: []domain.Placeholder{
			{ID: "ph-1", Type: domain.PHSelect},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "select_placeholder_requires_options") {
		t.Fatalf("expected select_placeholder_requires_options error, got %v", err)
	}
}

func TestUpdateSchemas_OptionsOnNonSelect(t *testing.T) {
	repo := newFakeRepo()
	repo.versions["v1"] = &domain.TemplateVersion{
		ID:            "v1",
		TemplateID:    "tpl-1",
		VersionNumber: 1,
		Status:        domain.VersionStatusDraft,
	}
	svc := application.New(repo, &fakePresigner{}, fakeClock{}, &fakeUUID{})

	_, err := svc.UpdateSchemas(context.Background(), application.UpdateSchemasCmd{
		TemplateID:    "tpl-1",
		VersionNumber: 1,
		PlaceholderSchema: []domain.Placeholder{
			{ID: "ph-1", Type: domain.PHText, Options: []string{"x"}},
		},
	})
	if err == nil || !strings.Contains(err.Error(), "options_allowed_only_for_select") {
		t.Fatalf("expected options_allowed_only_for_select error, got %v", err)
	}
}

func TestValidatePlaceholders_DuplicateID_Error(t *testing.T) {
	err := application.ValidatePlaceholders([]domain.Placeholder{
		{ID: "p1", Type: domain.PHText},
		{ID: "p1", Type: domain.PHText},
	})
	if !errors.Is(err, domain.ErrDuplicatePlaceholderID) {
		t.Fatalf("expected ErrDuplicatePlaceholderID, got %v", err)
	}
}

func TestValidatePlaceholders_InvalidRegex_Error(t *testing.T) {
	regex := "["
	err := application.ValidatePlaceholders([]domain.Placeholder{
		{ID: "p1", Type: domain.PHText, Regex: &regex},
	})
	if !errors.Is(err, domain.ErrInvalidConstraint) {
		t.Fatalf("expected ErrInvalidConstraint, got %v", err)
	}
}
