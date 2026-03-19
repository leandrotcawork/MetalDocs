package domain

import (
	"strings"
	"time"
)

const (
	DefaultPresenceWindowSeconds = 90
	DefaultLockTTLSeconds        = 300
)

type CollaborationPresence struct {
	DocumentID  string
	UserID      string
	DisplayName string
	LastSeenAt  time.Time
}

type DocumentEditLock struct {
	DocumentID  string
	LockedBy    string
	DisplayName string
	LockReason  string
	AcquiredAt  time.Time
	ExpiresAt   time.Time
}

func NormalizeCollaborationPresence(item CollaborationPresence) (CollaborationPresence, error) {
	item.DocumentID = strings.TrimSpace(item.DocumentID)
	item.UserID = strings.TrimSpace(item.UserID)
	item.DisplayName = strings.TrimSpace(item.DisplayName)
	if item.DocumentID == "" || item.UserID == "" {
		return CollaborationPresence{}, ErrInvalidCommand
	}
	if item.DisplayName == "" {
		item.DisplayName = item.UserID
	}
	if item.LastSeenAt.IsZero() {
		return CollaborationPresence{}, ErrInvalidCommand
	}
	return item, nil
}

func NormalizeDocumentEditLock(item DocumentEditLock) (DocumentEditLock, error) {
	item.DocumentID = strings.TrimSpace(item.DocumentID)
	item.LockedBy = strings.TrimSpace(item.LockedBy)
	item.DisplayName = strings.TrimSpace(item.DisplayName)
	item.LockReason = strings.TrimSpace(item.LockReason)
	if item.DocumentID == "" || item.LockedBy == "" {
		return DocumentEditLock{}, ErrInvalidCommand
	}
	if item.DisplayName == "" {
		item.DisplayName = item.LockedBy
	}
	if item.AcquiredAt.IsZero() || item.ExpiresAt.IsZero() {
		return DocumentEditLock{}, ErrInvalidCommand
	}
	if !item.ExpiresAt.After(item.AcquiredAt) {
		return DocumentEditLock{}, ErrInvalidCommand
	}
	return item, nil
}
