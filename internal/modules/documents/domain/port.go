package domain

import (
	"context"
	"io"
	"time"

	workflowdomain "metaldocs/internal/modules/workflow/domain"
)

// Repository defines persistence operations for the documents module.
type Repository interface {
	CreateDocument(ctx context.Context, document Document) error
	GetDocument(ctx context.Context, documentID string) (Document, error)
	ListDocuments(ctx context.Context) ([]Document, error)
	ListDocumentFamilies(ctx context.Context) ([]DocumentFamily, error)
	ListDocumentProfiles(ctx context.Context) ([]DocumentProfile, error)
	ListProcessAreas(ctx context.Context) ([]ProcessArea, error)
	ListSubjects(ctx context.Context) ([]Subject, error)
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
	CreateWorkflowApproval(ctx context.Context, approval workflowdomain.Approval) error
	GetLatestWorkflowApproval(ctx context.Context, documentID string) (workflowdomain.Approval, error)
	UpdateWorkflowApprovalDecision(ctx context.Context, approvalID, status, decisionBy, decisionReason string, decidedAt time.Time) error
	SaveWorkflowApprovalState(ctx context.Context, approval workflowdomain.Approval) error
	DeleteWorkflowApproval(ctx context.Context, approvalID string) error
	ListWorkflowApprovals(ctx context.Context, documentID string) ([]workflowdomain.Approval, error)
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
