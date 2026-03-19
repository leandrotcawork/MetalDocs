package domain

import (
	"context"
	"io"
	"time"
)

// Repository defines persistence operations for the documents module.
type Repository interface {
	CreateDocument(ctx context.Context, document Document) error
	GetDocument(ctx context.Context, documentID string) (Document, error)
	ListDocuments(ctx context.Context) ([]Document, error)
	ListDocumentsForReviewReminder(ctx context.Context, fromInclusive, toInclusive time.Time) ([]Document, error)
	ListDocumentFamilies(ctx context.Context) ([]DocumentFamily, error)
	ListDocumentProfiles(ctx context.Context) ([]DocumentProfile, error)
	ListDocumentProfileSchemas(ctx context.Context, profileCode string) ([]DocumentProfileSchemaVersion, error)
	GetDocumentProfileGovernance(ctx context.Context, profileCode string) (DocumentProfileGovernance, error)
	ListProcessAreas(ctx context.Context) ([]ProcessArea, error)
	UpsertProcessArea(ctx context.Context, item ProcessArea) error
	DeactivateProcessArea(ctx context.Context, code string) error
	ListSubjects(ctx context.Context) ([]Subject, error)
	UpsertSubject(ctx context.Context, item Subject) error
	DeactivateSubject(ctx context.Context, code string) error
	ListDocumentTypes(ctx context.Context) ([]DocumentType, error)
	ListAccessPolicies(ctx context.Context, resourceScope, resourceID string) ([]AccessPolicy, error)
	ReplaceAccessPolicies(ctx context.Context, resourceScope, resourceID string, policies []AccessPolicy) error
	UpdateDocumentStatus(ctx context.Context, documentID, status string) error
	SaveVersion(ctx context.Context, version Version) error
	ListVersions(ctx context.Context, documentID string) ([]Version, error)
	GetVersion(ctx context.Context, documentID string, versionNumber int) (Version, error)
	NextVersionNumber(ctx context.Context, documentID string) (int, error)
	CreateAttachment(ctx context.Context, attachment Attachment) error
	GetAttachment(ctx context.Context, attachmentID string) (Attachment, error)
	ListAttachments(ctx context.Context, documentID string) ([]Attachment, error)
}

// AtomicCreateRepository is an optional capability for strong consistency on create flow.
// If implemented, service can persist document + initial version in a single atomic operation.
type AtomicCreateRepository interface {
	CreateDocumentWithInitialVersion(ctx context.Context, document Document, version Version) error
}

type AttachmentStore interface {
	Save(ctx context.Context, storageKey string, content []byte) error
	Open(ctx context.Context, storageKey string) (io.ReadCloser, error)
	Delete(ctx context.Context, storageKey string) error
}
