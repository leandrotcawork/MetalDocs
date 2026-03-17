package unit

import (
	"context"
	"testing"
	"time"

	docapp "metaldocs/internal/modules/documents/application"
	docdomain "metaldocs/internal/modules/documents/domain"
	docmemory "metaldocs/internal/modules/documents/infrastructure/memory"
	iamdomain "metaldocs/internal/modules/iam/domain"
	searchapp "metaldocs/internal/modules/search/application"
	searchdomain "metaldocs/internal/modules/search/domain"
	searchdocs "metaldocs/internal/modules/search/infrastructure/documents"
)

func TestSearchDocumentsFiltersAndLimits(t *testing.T) {
	repo := docmemory.NewRepository()
	docSvc := docapp.NewService(repo, nil, fixedClock{now: time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)})
	searchSvc := searchapp.NewService(searchdocs.NewReader(repo))

	_, _ = docSvc.CreateDocument(context.Background(), docdomain.CreateDocumentCommand{
		DocumentID:   "search-1",
		Title:        "Contract Alpha",
		DocumentType: "contract",
		OwnerID:      "owner-a",
		BusinessUnit: "legal",
		Department:   "contracts",
		Tags:         []string{"vendor", "critical"},
		MetadataJSON: map[string]any{
			"counterparty":    "Alpha",
			"contract_number": "CNT-A",
			"start_date":      "2026-01-01",
			"end_date":        "2026-12-31",
		},
		Classification: docdomain.ClassificationInternal,
		InitialContent: "v1",
	})
	_, _ = docSvc.CreateDocument(context.Background(), docdomain.CreateDocumentCommand{
		DocumentID:   "search-2",
		Title:        "Contract Beta",
		DocumentType: "contract",
		OwnerID:      "owner-a",
		BusinessUnit: "legal",
		Department:   "contracts",
		Tags:         []string{"vendor"},
		MetadataJSON: map[string]any{
			"counterparty":    "Beta",
			"contract_number": "CNT-B",
			"start_date":      "2026-01-01",
			"end_date":        "2026-12-31",
		},
		Classification: docdomain.ClassificationConfidential,
		InitialContent: "v1",
	})
	_, _ = docSvc.CreateDocument(context.Background(), docdomain.CreateDocumentCommand{
		DocumentID:   "search-3",
		Title:        "Policy Public",
		DocumentType: "policy",
		OwnerID:      "owner-b",
		BusinessUnit: "quality",
		Department:   "qa",
		Tags:         []string{"public"},
		MetadataJSON: map[string]any{
			"policy_code": "POL-PUBLIC",
		},
		Classification: docdomain.ClassificationPublic,
		InitialContent: "v1",
	})

	items, err := searchSvc.SearchDocuments(context.Background(), searchdomain.Query{
		Text:    "contract",
		OwnerID: "owner-a",
		Limit:   1,
	})
	if err != nil {
		t.Fatalf("unexpected search error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item due limit, got %d", len(items))
	}
	if items[0].OwnerID != "owner-a" {
		t.Fatalf("expected owner-a, got %s", items[0].OwnerID)
	}
}

func TestSearchDocumentsFiltersByTagAndExpiry(t *testing.T) {
	repo := docmemory.NewRepository()
	docSvc := docapp.NewService(repo, nil, fixedClock{now: time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)})
	searchSvc := searchapp.NewService(searchdocs.NewReader(repo))

	expiringSoon := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)
	expiringLater := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)

	_, _ = docSvc.CreateDocument(context.Background(), docdomain.CreateDocumentCommand{
		DocumentID:   "search-expiry-1",
		Title:        "Supplier Contract",
		DocumentType: "contract",
		OwnerID:      "owner-supplier",
		BusinessUnit: "procurement",
		Department:   "buyers",
		Tags:         []string{"supplier", "critical"},
		ExpiryAt:     &expiringSoon,
		MetadataJSON: map[string]any{
			"counterparty":    "Supplier A",
			"contract_number": "CNT-S1",
			"start_date":      "2026-01-01",
			"end_date":        "2026-04-01",
		},
		InitialContent: "v1",
	})
	_, _ = docSvc.CreateDocument(context.Background(), docdomain.CreateDocumentCommand{
		DocumentID:   "search-expiry-2",
		Title:        "Long Term Contract",
		DocumentType: "contract",
		OwnerID:      "owner-supplier",
		BusinessUnit: "procurement",
		Department:   "buyers",
		Tags:         []string{"supplier"},
		ExpiryAt:     &expiringLater,
		MetadataJSON: map[string]any{
			"counterparty":    "Supplier B",
			"contract_number": "CNT-S2",
			"start_date":      "2026-01-01",
			"end_date":        "2026-08-01",
		},
		InitialContent: "v1",
	})

	items, err := searchSvc.SearchDocuments(context.Background(), searchdomain.Query{
		Tag:          "critical",
		ExpiryBefore: ptrTime(time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)),
	})
	if err != nil {
		t.Fatalf("unexpected search error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].ID != "search-expiry-1" {
		t.Fatalf("expected search-expiry-1, got %s", items[0].ID)
	}
}

func TestSearchDocumentsRespectsViewPolicies(t *testing.T) {
	repo := docmemory.NewRepository()
	docSvc := docapp.NewService(repo, nil, fixedClock{now: time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)})
	searchSvc := searchapp.NewService(searchdocs.NewReader(repo))

	_, _ = docSvc.CreateDocument(context.Background(), docdomain.CreateDocumentCommand{
		DocumentID:   "search-visible",
		Title:        "Visible Contract",
		DocumentType: "contract",
		OwnerID:      "owner-a",
		BusinessUnit: "legal",
		Department:   "contracts",
		MetadataJSON: map[string]any{
			"counterparty":    "Visible",
			"contract_number": "CNT-V",
			"start_date":      "2026-01-01",
			"end_date":        "2026-12-31",
		},
		InitialContent: "v1",
	})
	_, _ = docSvc.CreateDocument(context.Background(), docdomain.CreateDocumentCommand{
		DocumentID:   "search-hidden",
		Title:        "Hidden Contract",
		DocumentType: "contract",
		OwnerID:      "owner-b",
		BusinessUnit: "legal",
		Department:   "contracts",
		MetadataJSON: map[string]any{
			"counterparty":    "Hidden",
			"contract_number": "CNT-H",
			"start_date":      "2026-01-01",
			"end_date":        "2026-12-31",
		},
		InitialContent: "v1",
	})

	if err := docSvc.ReplaceAccessPolicies(context.Background(), "document", "search-visible", []docdomain.AccessPolicy{
		{SubjectType: docdomain.SubjectTypeUser, SubjectID: "viewer-1", Capability: docdomain.CapabilityDocumentView, Effect: docdomain.PolicyEffectAllow},
	}); err != nil {
		t.Fatalf("unexpected replace error: %v", err)
	}
	if err := docSvc.ReplaceAccessPolicies(context.Background(), "document", "search-hidden", []docdomain.AccessPolicy{
		{SubjectType: docdomain.SubjectTypeUser, SubjectID: "other-user", Capability: docdomain.CapabilityDocumentView, Effect: docdomain.PolicyEffectAllow},
	}); err != nil {
		t.Fatalf("unexpected replace error: %v", err)
	}

	ctx := iamdomain.WithAuthContext(context.Background(), "viewer-1", []iamdomain.Role{iamdomain.RoleViewer})
	items, err := searchSvc.SearchDocuments(ctx, searchdomain.Query{DocumentType: "contract"})
	if err != nil {
		t.Fatalf("unexpected search error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].ID != "search-visible" {
		t.Fatalf("expected search-visible, got %s", items[0].ID)
	}
}

func ptrTime(value time.Time) *time.Time {
	return &value
}

func TestSearchDocumentsByStatus(t *testing.T) {
	repo := docmemory.NewRepository()
	docSvc := docapp.NewService(repo, nil, fixedClock{now: time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)})
	searchSvc := searchapp.NewService(searchdocs.NewReader(repo))

	doc, err := docSvc.CreateDocument(context.Background(), docdomain.CreateDocumentCommand{
		DocumentID:   "search-status-1",
		Title:        "Status Doc",
		DocumentType: "manual",
		OwnerID:      "owner-x",
		BusinessUnit: "ops",
		Department:   "general",
		MetadataJSON: map[string]any{
			"manual_code": "MAN-STATUS",
		},
		Classification: docdomain.ClassificationInternal,
		InitialContent: "v1",
	})
	if err != nil {
		t.Fatalf("unexpected create error: %v", err)
	}
	if err := repo.UpdateDocumentStatus(context.Background(), doc.ID, docdomain.StatusInReview); err != nil {
		t.Fatalf("unexpected update error: %v", err)
	}

	items, err := searchSvc.SearchDocuments(context.Background(), searchdomain.Query{
		Status: docdomain.StatusInReview,
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("unexpected search error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Status != docdomain.StatusInReview {
		t.Fatalf("expected status %s, got %s", docdomain.StatusInReview, items[0].Status)
	}
}

func TestSearchDocumentsByDocumentTypeAndArea(t *testing.T) {
	repo := docmemory.NewRepository()
	docSvc := docapp.NewService(repo, nil, fixedClock{now: time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)})
	searchSvc := searchapp.NewService(searchdocs.NewReader(repo))

	_, _ = docSvc.CreateDocument(context.Background(), docdomain.CreateDocumentCommand{
		DocumentID:   "search-area-1",
		Title:        "Quality Procedure",
		DocumentType: "procedure",
		OwnerID:      "owner-qa",
		BusinessUnit: "quality",
		Department:   "qa",
		MetadataJSON: map[string]any{
			"procedure_code": "PROC-QA",
		},
		Classification: docdomain.ClassificationInternal,
		InitialContent: "v1",
	})
	_, _ = docSvc.CreateDocument(context.Background(), docdomain.CreateDocumentCommand{
		DocumentID:   "search-area-2",
		Title:        "Engineering Manual",
		DocumentType: "manual",
		OwnerID:      "owner-eng",
		BusinessUnit: "engineering",
		Department:   "projects",
		MetadataJSON: map[string]any{
			"manual_code": "MAN-ENG",
		},
		Classification: docdomain.ClassificationInternal,
		InitialContent: "v1",
	})

	items, err := searchSvc.SearchDocuments(context.Background(), searchdomain.Query{
		DocumentType: "procedure",
		BusinessUnit: "quality",
		Department:   "qa",
	})
	if err != nil {
		t.Fatalf("unexpected search error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].DocumentType != "procedure" {
		t.Fatalf("expected procedure, got %s", items[0].DocumentType)
	}
}
