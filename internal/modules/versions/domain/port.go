package domain

import "context"

type VersionSummary struct {
	DocumentID string
	Number     int
}

// ReadService is the stable read contract exposed by the versions module.
type ReadService interface {
	ListByDocumentID(ctx context.Context, documentID string) ([]VersionSummary, error)
}
