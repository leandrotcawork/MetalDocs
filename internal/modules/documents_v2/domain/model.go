package domain

import (
	"errors"
	"time"
)

type DocumentStatus string

const (
	DocStatusDraft     DocumentStatus = "draft"
	DocStatusFinalized DocumentStatus = "finalized"
	DocStatusArchived  DocumentStatus = "archived"
)

type SessionStatus string

const (
	SessionActive        SessionStatus = "active"
	SessionExpired       SessionStatus = "expired"
	SessionReleased      SessionStatus = "released"
	SessionForceReleased SessionStatus = "force_released"
)

type Document struct {
	ID                string
	TenantID          string
	TemplateVersionID string
	Name              string
	Status            DocumentStatus
	FormDataJSON      []byte
	CurrentRevisionID string
	ActiveSessionID   string
	FinalizedAt       *time.Time
	ArchivedAt        *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
	CreatedBy         string
}

type Session struct {
	ID                         string
	DocumentID                 string
	UserID                     string
	AcquiredAt                 time.Time
	ExpiresAt                  time.Time
	ReleasedAt                 *time.Time
	LastAcknowledgedRevisionID string
	Status                     SessionStatus
}

type Revision struct {
	ID               string
	DocumentID       string
	RevisionNum      int64
	ParentRevisionID string
	SessionID        string
	StorageKey       string
	ContentHash      string
	FormDataSnapshot []byte
	CreatedAt        time.Time
}

type PendingUpload struct {
	ID             string
	SessionID      string
	DocumentID     string
	BaseRevisionID string
	ContentHash    string
	StorageKey     string
	PresignedAt    time.Time
	ExpiresAt      time.Time
	ConsumedAt     *time.Time
}

type Checkpoint struct {
	ID         string
	DocumentID string
	RevisionID string
	VersionNum int
	Label      string
	CreatedAt  time.Time
	CreatedBy  string
}

var (
	ErrInvalidStateTransition = errors.New("invalid_state_transition")
	ErrSessionInactive        = errors.New("session_inactive")
	ErrSessionNotHolder       = errors.New("session_not_holder")
	ErrStaleBase              = errors.New("stale_base")
	ErrMisbound               = errors.New("misbound")
	ErrExpiredUpload          = errors.New("expired_upload")
	ErrContentHashMismatch    = errors.New("content_hash_mismatch")
	ErrPendingNotFound        = errors.New("pending_not_found")
	ErrAlreadyConsumed        = errors.New("already_consumed")
	ErrSessionTaken           = errors.New("session_taken")
	ErrForbidden              = errors.New("forbidden")
	ErrUploadMissing          = errors.New("upload_missing")
	ErrCheckpointNotFound     = errors.New("checkpoint_not_found")
	ErrDocumentNotOwner       = errors.New("document_not_owner")
	ErrNotFound               = errors.New("not_found")
	ErrInvalidName            = errors.New("invalid_name")
)
