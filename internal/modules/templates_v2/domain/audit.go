package domain

import "time"

type AuditAction string

const (
	AuditCreated               AuditAction = "created"
	AuditSaved                 AuditAction = "saved"
	AuditSubmitted             AuditAction = "submitted"
	AuditReviewed              AuditAction = "reviewed"
	AuditApproved              AuditAction = "approved"
	AuditRejected              AuditAction = "rejected"
	AuditPublished             AuditAction = "published"
	AuditObsoleted             AuditAction = "obsoleted"
	AuditArchived              AuditAction = "archived"
	AuditRestored              AuditAction = "restored"
	AuditApprovalConfigUpdated AuditAction = "approval_config_updated"
)

type AuditEvent struct {
	TenantID   string
	TemplateID string
	VersionID  *string
	ActorID    string
	Action     AuditAction
	Details    map[string]any
	OccurredAt time.Time
}
