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
	UpsertDocumentProfile(ctx context.Context, item DocumentProfile) error
	DeactivateDocumentProfile(ctx context.Context, code string) error
	ListDocumentProfileSchemas(ctx context.Context, profileCode string) ([]DocumentProfileSchemaVersion, error)
	UpsertDocumentProfileSchemaVersion(ctx context.Context, item DocumentProfileSchemaVersion) error
	ActivateDocumentProfileSchemaVersion(ctx context.Context, profileCode string, version int) error
	GetDocumentProfileGovernance(ctx context.Context, profileCode string) (DocumentProfileGovernance, error)
	UpsertDocumentProfileGovernance(ctx context.Context, item DocumentProfileGovernance) error
	ListProcessAreas(ctx context.Context) ([]ProcessArea, error)
	UpsertProcessArea(ctx context.Context, item ProcessArea) error
	DeactivateProcessArea(ctx context.Context, code string) error
	ListDocumentDepartments(ctx context.Context) ([]DocumentDepartment, error)
	UpsertDocumentDepartment(ctx context.Context, item DocumentDepartment) error
	DeactivateDocumentDepartment(ctx context.Context, code string) error
	ListSubjects(ctx context.Context) ([]Subject, error)
	UpsertSubject(ctx context.Context, item Subject) error
	DeactivateSubject(ctx context.Context, code string) error
	ListDocumentTypes(ctx context.Context) ([]DocumentType, error)
	ListDocumentTypeDefinitions(ctx context.Context) ([]DocumentTypeDefinition, error)
	GetDocumentTypeDefinition(ctx context.Context, key string) (DocumentTypeDefinition, error)
	UpsertDocumentTypeDefinition(ctx context.Context, item DocumentTypeDefinition) error
	ReserveNextDocumentSequence(ctx context.Context, profileCode string) (int, error)
	ListAccessPolicies(ctx context.Context, resourceScope, resourceID string) ([]AccessPolicy, error)
	ReplaceAccessPolicies(ctx context.Context, resourceScope, resourceID string, policies []AccessPolicy) error
	UpdateDocumentStatus(ctx context.Context, documentID, status string) error
	SaveVersion(ctx context.Context, version Version) error
	UpdateVersionPDF(ctx context.Context, documentID string, versionNumber int, pdfStorageKey string, pageCount int) error
	UpdateVersionBodyBlocks(ctx context.Context, documentID string, versionNumber int, bodyBlocks []EtapaBody) error
	UpdateVersionValues(ctx context.Context, documentID string, versionNumber int, values DocumentValues) error
	ListVersions(ctx context.Context, documentID string) ([]Version, error)
	GetVersion(ctx context.Context, documentID string, versionNumber int) (Version, error)
	NextVersionNumber(ctx context.Context, documentID string) (int, error)
	CreateAttachment(ctx context.Context, attachment Attachment) error
	GetAttachment(ctx context.Context, attachmentID string) (Attachment, error)
	ListAttachments(ctx context.Context, documentID string) ([]Attachment, error)
	UpsertCollaborationPresence(ctx context.Context, item CollaborationPresence) error
	ListCollaborationPresence(ctx context.Context, documentID string, activeSince time.Time) ([]CollaborationPresence, error)
	AcquireDocumentEditLock(ctx context.Context, item DocumentEditLock, now time.Time) (DocumentEditLock, error)
	GetDocumentEditLock(ctx context.Context, documentID string, now time.Time) (DocumentEditLock, error)
	ReleaseDocumentEditLock(ctx context.Context, documentID, lockedBy string) error
}

// AtomicCreateRepository is an optional capability for strong consistency on create flow.
// If implemented, service can persist document + initial version in a single atomic operation.
type AtomicCreateRepository interface {
	CreateDocumentWithInitialVersion(ctx context.Context, document Document, version Version) error
}

// AtomicCreateRepositoryWithPolicies extends atomic create with access policies.
type AtomicCreateRepositoryWithPolicies interface {
	CreateDocumentWithInitialVersionAndPolicies(ctx context.Context, document Document, version Version, policies []AccessPolicy) error
}

// DocumentTypeDefinitionRepository persists canonical runtime schema definitions.
type DocumentTypeDefinitionRepository interface {
	ListDocumentTypeDefinitions(ctx context.Context) ([]DocumentTypeDefinition, error)
	GetDocumentTypeDefinition(ctx context.Context, key string) (DocumentTypeDefinition, error)
	UpsertDocumentTypeDefinition(ctx context.Context, item DocumentTypeDefinition) error
}

type AttachmentStore interface {
	Save(ctx context.Context, storageKey string, content []byte) error
	Open(ctx context.Context, storageKey string) (io.ReadCloser, error)
	Delete(ctx context.Context, storageKey string) error
}
