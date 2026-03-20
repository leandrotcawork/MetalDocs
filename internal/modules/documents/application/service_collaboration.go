package application

import (
	"context"
	"strings"
	"time"

	"metaldocs/internal/modules/documents/domain"
	"metaldocs/internal/platform/authn"
)

func (s *Service) HeartbeatCollaborationPresenceAuthorized(ctx context.Context, documentID, userID, displayName string) error {
	doc, err := s.GetDocumentAuthorized(ctx, documentID)
	if err != nil {
		return err
	}
	if strings.TrimSpace(userID) == "" {
		userID = authn.UserIDFromContext(ctx)
	}
	normalized, err := domain.NormalizeCollaborationPresence(domain.CollaborationPresence{
		DocumentID:  doc.ID,
		UserID:      userID,
		DisplayName: strings.TrimSpace(displayName),
		LastSeenAt:  s.clock.Now(),
	})
	if err != nil {
		return err
	}
	return s.repo.UpsertCollaborationPresence(ctx, normalized)
}

func (s *Service) ListCollaborationPresenceAuthorized(ctx context.Context, documentID string) ([]domain.CollaborationPresence, error) {
	doc, err := s.GetDocumentAuthorized(ctx, documentID)
	if err != nil {
		return nil, err
	}
	activeSince := s.clock.Now().Add(-time.Duration(domain.DefaultPresenceWindowSeconds) * time.Second)
	return s.repo.ListCollaborationPresence(ctx, doc.ID, activeSince)
}

func (s *Service) AcquireDocumentEditLockAuthorized(ctx context.Context, documentID, userID, displayName, reason string, ttlSeconds int) (domain.DocumentEditLock, error) {
	doc, err := s.repo.GetDocument(ctx, strings.TrimSpace(documentID))
	if err != nil {
		return domain.DocumentEditLock{}, err
	}
	allowed, err := s.isAllowed(ctx, doc, domain.CapabilityDocumentEdit)
	if err != nil {
		return domain.DocumentEditLock{}, err
	}
	if !allowed {
		return domain.DocumentEditLock{}, domain.ErrDocumentNotFound
	}

	if strings.TrimSpace(userID) == "" {
		userID = authn.UserIDFromContext(ctx)
	}
	if ttlSeconds <= 0 {
		ttlSeconds = domain.DefaultLockTTLSeconds
	}
	now := s.clock.Now()
	normalized, err := domain.NormalizeDocumentEditLock(domain.DocumentEditLock{
		DocumentID:  doc.ID,
		LockedBy:    userID,
		DisplayName: strings.TrimSpace(displayName),
		LockReason:  strings.TrimSpace(reason),
		AcquiredAt:  now,
		ExpiresAt:   now.Add(time.Duration(ttlSeconds) * time.Second),
	})
	if err != nil {
		return domain.DocumentEditLock{}, err
	}
	return s.repo.AcquireDocumentEditLock(ctx, normalized, now)
}

func (s *Service) GetDocumentEditLockAuthorized(ctx context.Context, documentID string) (domain.DocumentEditLock, error) {
	doc, err := s.GetDocumentAuthorized(ctx, documentID)
	if err != nil {
		return domain.DocumentEditLock{}, err
	}
	return s.repo.GetDocumentEditLock(ctx, doc.ID, s.clock.Now())
}

func (s *Service) ReleaseDocumentEditLockAuthorized(ctx context.Context, documentID, userID string) error {
	doc, err := s.repo.GetDocument(ctx, strings.TrimSpace(documentID))
	if err != nil {
		return err
	}
	allowed, err := s.isAllowed(ctx, doc, domain.CapabilityDocumentEdit)
	if err != nil {
		return err
	}
	if !allowed {
		return domain.ErrDocumentNotFound
	}
	if strings.TrimSpace(userID) == "" {
		userID = authn.UserIDFromContext(ctx)
	}
	return s.repo.ReleaseDocumentEditLock(ctx, doc.ID, strings.TrimSpace(userID))
}
