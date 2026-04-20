package domain

import (
	"errors"
	"time"
)

type VersionStatus string

const (
	VersionStatusDraft     VersionStatus = "draft"
	VersionStatusInReview  VersionStatus = "in_review"
	VersionStatusApproved  VersionStatus = "approved"
	VersionStatusPublished VersionStatus = "published"
	VersionStatusObsolete  VersionStatus = "obsolete"
)

type TemplateVersion struct {
	ID                  string
	TemplateID          string
	VersionNumber       int
	Status              VersionStatus
	DocxStorageKey      string
	ContentHash         string
	MetadataSchema      MetadataSchema
	PlaceholderSchema   []Placeholder
	EditableZones       []EditableZone
	AuthorID            string
	PendingReviewerRole *string
	PendingApproverRole string
	ReviewerID          *string
	ApproverID          *string
	SubmittedAt         *time.Time
	ReviewedAt          *time.Time
	ApprovedAt          *time.Time
	PublishedAt         *time.Time
	ObsoletedAt         *time.Time
	CreatedAt           time.Time
}

func (v *TemplateVersion) CanTransition(next VersionStatus, hasReviewer bool) error {
	switch v.Status {
	case VersionStatusDraft:
		if next == VersionStatusInReview {
			return nil
		}
	case VersionStatusInReview:
		if next == VersionStatusDraft {
			return nil
		}
		if next == VersionStatusApproved && hasReviewer {
			return nil
		}
		if next == VersionStatusPublished && !hasReviewer {
			return nil
		}
	case VersionStatusApproved:
		if next == VersionStatusPublished || next == VersionStatusDraft {
			return nil
		}
	case VersionStatusPublished:
		if next == VersionStatusObsolete {
			return nil
		}
	}
	return ErrInvalidStateTransition
}

var (
	ErrInvalidStateTransition = errors.New("invalid_state_transition")
	ErrContentHashMismatch    = errors.New("content_hash_mismatch")
	ErrStaleBase              = errors.New("stale_base")
)
