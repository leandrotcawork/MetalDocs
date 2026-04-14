package domain

import (
	"encoding/json"
	"errors"
	"time"
)

// TemplateStatus represents the lifecycle state of a published template version.
type TemplateStatus string

const (
	TemplateStatusDraft      TemplateStatus = "draft"
	TemplateStatusPublished  TemplateStatus = "published"
	TemplateStatusDeprecated TemplateStatus = "deprecated"
)

// TemplateDraftKey identifies a template draft aggregate.
type TemplateDraftKey string

// TemplateDraft is the admin scratch-pad for a template being authored or edited.
// It is keyed by template_key and only one draft can exist per key at a time.
type TemplateDraft struct {
	TemplateKey        string
	ProfileCode        string
	BaseVersion        int
	Name               string
	ThemeJSON          json.RawMessage
	MetaJSON           json.RawMessage
	BlocksJSON         json.RawMessage
	LockVersion        int
	HasStrippedFields  bool
	StrippedFieldsJSON json.RawMessage // nullable
	CreatedBy          string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// TemplateAuditEvent is an append-only record describing an action taken on a template.
type TemplateAuditEvent struct {
	TemplateKey string
	Version     *int // nullable — absent before first publish
	Action      string
	ActorID     string
	DiffSummary string
	TraceID     string
}

// StrippedField describes a block field removed from the draft because it violated
// schema rules. The editor must acknowledge all stripped fields before publishing.
type StrippedField struct {
	BlockID   string
	BlockType string
	Field     string
	Reason    string
}

// PublishError describes a validation failure that prevents publishing a draft.
type PublishError struct {
	BlockID   string
	BlockType string
	Field     string
	Reason    string
}

var (
	ErrTemplateLockConflict      = errors.New("template: lock version conflict")
	ErrTemplateHasStrippedFields = errors.New("template: has stripped fields, acknowledge before publishing")
	ErrTemplateAlreadyPublished  = errors.New("template: already published")
	ErrTemplateNotDraft          = errors.New("template: not in draft status")
	ErrTemplatePublishValidation = errors.New("template: publish validation failed")
	ErrTemplateDraftNotFound     = errors.New("template: draft not found")
	ErrTemplateNotFound          = errors.New("template: not found")
)
