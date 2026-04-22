package domain

import (
	"errors"
	"time"
)

type DocumentProfile struct {
	Code                     string     `json:"code"`
	TenantID                 string     `json:"tenantId"`
	FamilyCode               string     `json:"familyCode"`
	Name                     string     `json:"name"`
	Description              string     `json:"description"`
	Alias                    string     `json:"alias"`
	ReviewIntervalDays       int        `json:"reviewIntervalDays"`
	DefaultTemplateVersionID *string    `json:"defaultTemplateVersionId"`
	OwnerUserID              *string    `json:"ownerUserId"`
	EditableByRole           string     `json:"editableByRole"`
	ArchivedAt               *time.Time `json:"archivedAt"`
	CreatedAt                time.Time  `json:"createdAt"`
}

var (
	ErrProfileNotFound         = errors.New("profile not found")
	ErrProfileCodeImmutable    = errors.New("profile code is immutable")
	ErrProfileArchived         = errors.New("profile is archived")
	ErrTemplateNotPublished    = errors.New("template version is not published")
	ErrTemplateProfileMismatch = errors.New("template version belongs to different profile")
)

func (p *DocumentProfile) IsActive() bool {
	return p.ArchivedAt == nil
}

func (p *DocumentProfile) Archive(now time.Time) error {
	if !p.IsActive() {
		return ErrProfileArchived
	}
	p.ArchivedAt = &now
	return nil
}
