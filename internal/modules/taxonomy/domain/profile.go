package domain

import (
	"errors"
	"time"
)

type DocumentProfile struct {
	Code                     string
	TenantID                 string
	FamilyCode               string
	Name                     string
	Description              string
	ReviewIntervalDays       int
	DefaultTemplateVersionID *string
	OwnerUserID              *string
	EditableByRole           string
	ArchivedAt               *time.Time
	CreatedAt                time.Time
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
