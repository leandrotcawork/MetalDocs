package application

import (
	"context"
	"errors"
	"time"

	"metaldocs/internal/modules/templates_v2/domain"
)

const autosaveUploadTTL = 10 * time.Minute

type PresignAutosaveCmd struct {
	TenantID, ActorUserID, TemplateID string
	VersionNumber                     int
}

type PresignAutosaveResult struct {
	UploadURL  string
	StorageKey string
	ExpiresAt  time.Time
}

func (s *Service) PresignAutosave(ctx context.Context, cmd PresignAutosaveCmd) (*PresignAutosaveResult, error) {
	if _, err := s.repo.GetTemplate(ctx, cmd.TenantID, cmd.TemplateID); err != nil {
		return nil, err
	}

	version, err := s.repo.GetVersion(ctx, cmd.TemplateID, cmd.VersionNumber)
	if err != nil {
		return nil, err
	}
	if version.Status != domain.VersionStatusDraft {
		return nil, domain.ErrInvalidStateTransition
	}

	url, err := s.presign.PresignPUT(ctx, version.DocxStorageKey, autosaveUploadTTL)
	if err != nil {
		return nil, err
	}

	return &PresignAutosaveResult{
		UploadURL:  url,
		StorageKey: version.DocxStorageKey,
		ExpiresAt:  s.clock.Now().Add(autosaveUploadTTL),
	}, nil
}

type CommitAutosaveCmd struct {
	TenantID, ActorUserID, TemplateID string
	VersionNumber                     int
	ExpectedContentHash               string
}

func (s *Service) CommitAutosave(ctx context.Context, cmd CommitAutosaveCmd) (*domain.TemplateVersion, error) {
	if _, err := s.repo.GetTemplate(ctx, cmd.TenantID, cmd.TemplateID); err != nil {
		return nil, err
	}

	version, err := s.repo.GetVersion(ctx, cmd.TemplateID, cmd.VersionNumber)
	if err != nil {
		return nil, err
	}
	if version.Status != domain.VersionStatusDraft {
		return nil, domain.ErrInvalidStateTransition
	}

	actualHash, err := s.presign.HeadContentHash(ctx, version.DocxStorageKey)
	if err != nil {
		if errors.Is(err, domain.ErrUploadMissing) {
			return nil, domain.ErrUploadMissing
		}
		return nil, err
	}
	if actualHash != cmd.ExpectedContentHash {
		_ = s.presign.Delete(ctx, version.DocxStorageKey)
		return nil, domain.ErrContentHashMismatch
	}

	version.ContentHash = actualHash
	if err := s.repo.UpdateVersion(ctx, version); err != nil {
		return nil, err
	}
	if err := s.repo.AppendAudit(ctx, &domain.AuditEvent{
		TenantID:   cmd.TenantID,
		TemplateID: cmd.TemplateID,
		VersionID:  &version.ID,
		ActorID:    cmd.ActorUserID,
		Action:     domain.AuditSaved,
		Details:    map[string]any{"content_hash": actualHash},
		OccurredAt: s.clock.Now(),
	}); err != nil {
		return nil, err
	}

	return version, nil
}
