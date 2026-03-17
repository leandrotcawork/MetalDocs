package unit

import (
	"context"
	"testing"
	"time"

	docapp "metaldocs/internal/modules/documents/application"
	docdomain "metaldocs/internal/modules/documents/domain"
	docmemory "metaldocs/internal/modules/documents/infrastructure/memory"
	searchapp "metaldocs/internal/modules/search/application"
	searchdomain "metaldocs/internal/modules/search/domain"
	searchdocs "metaldocs/internal/modules/search/infrastructure/documents"
)

func TestSearchDocumentsFiltersAndLimits(t *testing.T) {
	repo := docmemory.NewRepository()
	docSvc := docapp.NewService(repo, nil, fixedClock{now: time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)})
	searchSvc := searchapp.NewService(searchdocs.NewReader(repo))

	_, _ = docSvc.CreateDocument(context.Background(), docdomain.CreateDocumentCommand{
		DocumentID:     "search-1",
		Title:          "Contract Alpha",
		DocumentType:   "contract",
		OwnerID:        "owner-a",
		BusinessUnit:   "legal",
		Department:     "contracts",
		Classification: docdomain.ClassificationInternal,
		InitialContent: "v1",
	})
	_, _ = docSvc.CreateDocument(context.Background(), docdomain.CreateDocumentCommand{
		DocumentID:     "search-2",
		Title:          "Contract Beta",
		DocumentType:   "contract",
		OwnerID:        "owner-a",
		BusinessUnit:   "legal",
		Department:     "contracts",
		Classification: docdomain.ClassificationConfidential,
		InitialContent: "v1",
	})
	_, _ = docSvc.CreateDocument(context.Background(), docdomain.CreateDocumentCommand{
		DocumentID:     "search-3",
		Title:          "Policy Public",
		DocumentType:   "policy",
		OwnerID:        "owner-b",
		BusinessUnit:   "quality",
		Department:     "qa",
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

func TestSearchDocumentsByStatus(t *testing.T) {
	repo := docmemory.NewRepository()
	docSvc := docapp.NewService(repo, nil, fixedClock{now: time.Date(2026, 3, 16, 10, 0, 0, 0, time.UTC)})
	searchSvc := searchapp.NewService(searchdocs.NewReader(repo))

	doc, err := docSvc.CreateDocument(context.Background(), docdomain.CreateDocumentCommand{
		DocumentID:     "search-status-1",
		Title:          "Status Doc",
		DocumentType:   "manual",
		OwnerID:        "owner-x",
		BusinessUnit:   "ops",
		Department:     "general",
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
		DocumentID:     "search-area-1",
		Title:          "Quality Procedure",
		DocumentType:   "procedure",
		OwnerID:        "owner-qa",
		BusinessUnit:   "quality",
		Department:     "qa",
		Classification: docdomain.ClassificationInternal,
		InitialContent: "v1",
	})
	_, _ = docSvc.CreateDocument(context.Background(), docdomain.CreateDocumentCommand{
		DocumentID:     "search-area-2",
		Title:          "Engineering Manual",
		DocumentType:   "manual",
		OwnerID:        "owner-eng",
		BusinessUnit:   "engineering",
		Department:     "projects",
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
