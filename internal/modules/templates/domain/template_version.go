package domain

import "time"

type Status string

const (
	StatusDraft      Status = "draft"
	StatusPublished  Status = "published"
	StatusDeprecated Status = "deprecated"
)

type TemplateVersion struct {
	ID                string
	TemplateID        string
	VersionNum        int
	Status            Status
	GrammarVersion    int
	DocxStorageKey    string
	SchemaStorageKey  string
	DocxContentHash   string
	SchemaContentHash string
	PublishedAt       *time.Time
	PublishedBy       *string
	DeprecatedAt      *time.Time
	LockVersion       int
	CreatedAt         time.Time
	UpdatedAt         time.Time
	CreatedBy         string
}

func NewTemplateVersion(templateID string, versionNum int) *TemplateVersion {
	return &TemplateVersion{
		TemplateID:     templateID,
		VersionNum:     versionNum,
		Status:         StatusDraft,
		GrammarVersion: 1,
		LockVersion:    0,
	}
}

func (v *TemplateVersion) Publish(by string) error {
	if v.Status != StatusDraft {
		return ErrInvalidStateTransition
	}
	now := time.Now().UTC()
	v.Status = StatusPublished
	v.PublishedAt = &now
	v.PublishedBy = &by
	return nil
}

func (v *TemplateVersion) Deprecate() error {
	if v.Status != StatusPublished {
		return ErrInvalidStateTransition
	}
	now := time.Now().UTC()
	v.Status = StatusDeprecated
	v.DeprecatedAt = &now
	return nil
}

func (v *TemplateVersion) ApplyDraftEdit(expectedLockVersion int) error {
	if v.Status != StatusDraft {
		return ErrInvalidStateTransition
	}
	if v.LockVersion != expectedLockVersion {
		return ErrLockVersionMismatch
	}
	v.LockVersion++
	v.UpdatedAt = time.Now().UTC()
	return nil
}
