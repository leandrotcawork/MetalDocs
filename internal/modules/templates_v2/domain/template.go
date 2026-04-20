package domain

import (
	"errors"
	"time"
)

type Visibility string

const (
	VisibilityPublic   Visibility = "public"
	VisibilityInternal Visibility = "internal"
	VisibilitySpecific Visibility = "specific"
)

type Template struct {
	ID                 string
	TenantID           string
	DocTypeCode        string
	Key                string
	Name               string
	Description        string
	Areas              []string
	Visibility         Visibility
	SpecificAreas      []string
	LatestVersion      int
	PublishedVersionID *string
	CreatedBy          string
	CreatedAt          time.Time
	ArchivedAt         *time.Time
}

func (t *Template) IsArchived() bool { return t.ArchivedAt != nil }

var (
	ErrNotFound          = errors.New("not_found")
	ErrKeyConflict       = errors.New("key_conflict")
	ErrInvalidVisibility = errors.New("invalid_visibility")
	ErrArchived          = errors.New("archived")
)
