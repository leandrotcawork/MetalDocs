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
			{ID: "ph-1", Name: "doc_code", Label: "Doc Code", Type: domain.PHComputed, Computed: true, ResolverKey: func() *string { s := "doc_code"; return &s }()},
		},
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
		TemplateID:          "tpl-1",
		VersionNumber:       1,
		ExpectedContentHash: "hash-2",
		PlaceholderSchema:   []domain.Placeholder{{ID: "ph-1", Type: domain.PHText}},
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

func TestValidatePlaceholders_InvalidName_Error(t *testing.T) {
	err := application.ValidatePlaceholders([]domain.Placeholder{
		{ID: "p1", Name: "Bad Name!", Type: domain.PHText},
	})
	if !errors.Is(err, domain.ErrPlaceholderNameInvalid) {
		t.Fatalf("expected ErrPlaceholderNameInvalid, got %v", err)
	}
}

func TestValidatePlaceholders_DuplicateName_Error(t *testing.T) {
	rk := "doc_code"
	err := application.ValidatePlaceholders([]domain.Placeholder{
		{ID: "p1", Name: "doc_code", Type: domain.PHComputed, Computed: true, ResolverKey: &rk},
		{ID: "p2", Name: "doc_code", Type: domain.PHComputed, Computed: true, ResolverKey: &rk},
	})
	if !errors.Is(err, domain.ErrDuplicatePlaceholderName) {
		t.Fatalf("expected ErrDuplicatePlaceholderName, got %v", err)
	}
}

func TestValidatePlaceholders_EmptyName_Allowed(t *testing.T) {
	err := application.ValidatePlaceholders([]domain.Placeholder{
		{ID: "p1", Type: domain.PHText},
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestValidatePlaceholders_ValidName_NoError(t *testing.T) {
	rk1, rk2 := "doc_code", "effective_date"
	err := application.ValidatePlaceholders([]domain.Placeholder{
		{ID: "p1", Name: "doc_code", Type: domain.PHComputed, Computed: true, ResolverKey: &rk1},
		{ID: "p2", Name: "effective_date", Type: domain.PHComputed, Computed: true, ResolverKey: &rk2},
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
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

func TestValidatePlaceholders_NumberRangeInverted_Error(t *testing.T) {
	min := 10.0
	max := 5.0
	err := application.ValidatePlaceholders([]domain.Placeholder{
		{ID: "p1", Type: domain.PHNumber, MinNumber: &min, MaxNumber: &max},
	})
	if !errors.Is(err, domain.ErrInvalidConstraint) {
		t.Fatalf("expected ErrInvalidConstraint, got %v", err)
	}
}

func TestValidatePlaceholders_DateRangeInverted_Error(t *testing.T) {
	minDate := "2026-05-02"
	maxDate := "2026-04-01"
	err := application.ValidatePlaceholders([]domain.Placeholder{
		{ID: "p1", Type: domain.PHDate, MinDate: &minDate, MaxDate: &maxDate},
	})
	if !errors.Is(err, domain.ErrInvalidConstraint) {
		t.Fatalf("expected ErrInvalidConstraint, got %v", err)
	}
}

func TestValidatePlaceholders_ComputedRequiresResolverKey(t *testing.T) {
	err := application.ValidatePlaceholders([]domain.Placeholder{
		{ID: "p1", Type: domain.PHComputed, Computed: true},
	})
	if !errors.Is(err, domain.ErrInvalidConstraint) {
		t.Fatalf("expected ErrInvalidConstraint, got %v", err)
	}
}

func TestUpdateSchemas_UnknownResolverKey_Error(t *testing.T) {
	repo := newFakeRepo()
	repo.versions["v1"] = &domain.TemplateVersion{
		ID:            "v1",
		TemplateID:    "tpl-1",
		VersionNumber: 1,
		Status:        domain.VersionStatusDraft,
	}
	svc := newService(repo, WithKnownResolvers("doc_code"))

	_, err := svc.UpdateSchemas(context.Background(), updateCmdWithComputed("p1", "missing_resolver"))
	if !errors.Is(err, domain.ErrUnknownResolver) {
		t.Fatalf("expected ErrUnknownResolver, got %v", err)
	}
}

func TestValidatePlaceholders_RejectsNonCatalogName(t *testing.T) {
	rk := "customer_name"
	phs := []domain.Placeholder{
		{ID: "p1", Name: "customer_name", Label: "Customer", Type: domain.PHComputed, Computed: true, ResolverKey: &rk},
	}
	err := application.ValidatePlaceholders(phs)
	if !errors.Is(err, domain.ErrPlaceholderNotInCatalog) {
		t.Fatalf("err = %v, want ErrPlaceholderNotInCatalog", err)
	}
}

func TestValidatePlaceholders_AcceptsCatalogName(t *testing.T) {
	rk := "doc_code"
	phs := []domain.Placeholder{
		{ID: "p1", Name: "doc_code", Label: "Codigo", Type: domain.PHComputed, Computed: true, ResolverKey: &rk},
	}
	if err := application.ValidatePlaceholders(phs); err != nil {
		t.Fatalf("err = %v, want nil", err)
	}
}

func TestValidatePlaceholders_RejectsCatalogNameWithWrongShape(t *testing.T) {
	rk := "doc_code"
	phs := []domain.Placeholder{
		{ID: "p1", Name: "doc_code", Label: "X", Type: domain.PHText, ResolverKey: &rk},
	}
	if err := application.ValidatePlaceholders(phs); !errors.Is(err, domain.ErrPlaceholderNotComputed) {
		t.Fatalf("err = %v, want ErrPlaceholderNotComputed", err)
	}
}
