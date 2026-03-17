package domain

import "context"

// Repository defines persistence operations for the documents module.
type Repository interface {
	CreateDocument(ctx context.Context, document Document) error
	GetDocument(ctx context.Context, documentID string) (Document, error)
	ListDocuments(ctx context.Context) ([]Document, error)
	ListDocumentTypes(ctx context.Context) ([]DocumentType, error)
	UpdateDocumentStatus(ctx context.Context, documentID, status string) error
	SaveVersion(ctx context.Context, version Version) error
	ListVersions(ctx context.Context, documentID string) ([]Version, error)
	NextVersionNumber(ctx context.Context, documentID string) (int, error)
}

// AtomicCreateRepository is an optional capability for strong consistency on create flow.
// If implemented, service can persist document + initial version in a single atomic operation.
type AtomicCreateRepository interface {
	CreateDocumentWithInitialVersion(ctx context.Context, document Document, version Version) error
}
