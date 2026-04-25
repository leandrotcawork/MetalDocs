package resolvers

import (
	"context"
	"time"
)

type ResolveInput struct {
	TenantID, RevisionID, ControlledDocumentID string
	ProfileCodeSnapshot, AreaCodeSnapshot      string
	RegistryReader                             RegistryReader
	RevisionReader                             RevisionReader
	WorkflowReader                             WorkflowReader
}

type ResolvedValue struct {
	Value       any
	ResolverKey string
	ResolverVer int
	InputsHash  []byte
	ComputedAt  time.Time
}

type ComputedResolver interface {
	Key() string
	Version() int
	Resolve(ctx context.Context, in ResolveInput) (ResolvedValue, error)
}

type ControlledDocumentInfo struct {
	DocCode string
}

type AuthorInfo struct {
	UserID      string
	DisplayName string
}

type ApproverInfo struct {
	UserID      string
	DisplayName string
	SignedAt    time.Time
}

type RegistryReader interface {
	GetControlledDocument(ctx context.Context, tenantID, controlledDocumentID string) (ControlledDocumentInfo, error)
}

type RevisionReader interface {
	GetRevisionNumber(ctx context.Context, tenantID, revisionID string) (int64, error)
	GetEffectiveFrom(ctx context.Context, tenantID, revisionID string) (time.Time, error)
	GetAuthor(ctx context.Context, tenantID, revisionID string) (AuthorInfo, error)
}

type WorkflowReader interface {
	GetApprovers(ctx context.Context, tenantID, revisionID string) ([]ApproverInfo, error)
	GetFinalApprovalDate(ctx context.Context, tenantID, revisionID string) (time.Time, error)
}
