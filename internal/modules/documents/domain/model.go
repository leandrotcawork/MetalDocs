package domain

import "time"

const (
	StatusDraft     = "DRAFT"
	StatusInReview  = "IN_REVIEW"
	StatusApproved  = "APPROVED"
	StatusPublished = "PUBLISHED"
	StatusArchived  = "ARCHIVED"
)

const (
	ClassificationPublic       = "PUBLIC"
	ClassificationInternal     = "INTERNAL"
	ClassificationConfidential = "CONFIDENTIAL"
	ClassificationRestricted   = "RESTRICTED"
)

type Document struct {
	ID             string
	Title          string
	DocumentType   string
	OwnerID        string
	BusinessUnit   string
	Department     string
	Classification string
	Status         string
	Tags           []string
	EffectiveAt    *time.Time
	ExpiryAt       *time.Time
	MetadataJSON   map[string]any
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type Version struct {
	DocumentID string
	Number     int
	Content    string
	CreatedAt  time.Time
}

type CreateDocumentCommand struct {
	DocumentID     string
	Title          string
	DocumentType   string
	OwnerID        string
	BusinessUnit   string
	Department     string
	Classification string
	Tags           []string
	EffectiveAt    *time.Time
	ExpiryAt       *time.Time
	MetadataJSON   map[string]any
	InitialContent string
	TraceID        string
}

type AddVersionCommand struct {
	DocumentID string
	Content    string
	TraceID    string
}

type DocumentType struct {
	Code               string
	Name               string
	Description        string
	ReviewIntervalDays int
}

type AccessPolicy struct {
	SubjectType   string
	SubjectID     string
	ResourceScope string
	ResourceID    string
	Capability    string
	Effect        string
}

const (
	SubjectTypeUser  = "user"
	SubjectTypeRole  = "role"
	SubjectTypeGroup = "group"
)

const (
	ResourceScopeDocument     = "document"
	ResourceScopeDocumentType = "document_type"
	ResourceScopeArea         = "area"
)

const (
	CapabilityDocumentCreate            = "document.create"
	CapabilityDocumentView              = "document.view"
	CapabilityDocumentEdit              = "document.edit"
	CapabilityDocumentUploadAttachment  = "document.upload_attachment"
	CapabilityDocumentChangeWorkflow    = "document.change_workflow"
	CapabilityDocumentManagePermissions = "document.manage_permissions"
)

const (
	PolicyEffectAllow = "allow"
	PolicyEffectDeny  = "deny"
)

func DefaultDocumentTypes() []DocumentType {
	return []DocumentType{
		{Code: "policy", Name: "Policy", Description: "High-level governance and policy document", ReviewIntervalDays: 365},
		{Code: "procedure", Name: "Procedure", Description: "Operational procedure with controlled steps", ReviewIntervalDays: 365},
		{Code: "work_instruction", Name: "Work Instruction", Description: "Detailed execution instruction", ReviewIntervalDays: 180},
		{Code: "contract", Name: "Contract", Description: "Commercial or legal agreement", ReviewIntervalDays: 365},
		{Code: "supplier_document", Name: "Supplier Document", Description: "Document received from supplier", ReviewIntervalDays: 180},
		{Code: "technical_drawing", Name: "Technical Drawing", Description: "Engineering drawing or technical artifact", ReviewIntervalDays: 180},
		{Code: "certificate", Name: "Certificate", Description: "Certificate with issuer and validity context", ReviewIntervalDays: 365},
		{Code: "report", Name: "Report", Description: "Periodic or ad-hoc report", ReviewIntervalDays: 365},
		{Code: "form", Name: "Form", Description: "Structured business form", ReviewIntervalDays: 180},
		{Code: "manual", Name: "Manual", Description: "Reference or guidance manual", ReviewIntervalDays: 365},
	}
}
