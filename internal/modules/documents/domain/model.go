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
	OwnerID        string
	Classification string
	Status         string
	CreatedAt      time.Time
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
	OwnerID        string
	Classification string
	InitialContent string
	TraceID        string
}

type AddVersionCommand struct {
	DocumentID string
	Content    string
	TraceID    string
}
