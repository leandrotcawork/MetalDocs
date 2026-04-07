package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

var (
	ErrTemplateSnapshotMismatch = errors.New("TEMPLATE_SNAPSHOT_MISMATCH")
	ErrTemplateSnapshotMissing  = errors.New("TEMPLATE_SNAPSHOT_MISSING")
)

type TemplateRef struct {
	TemplateID          uuid.UUID `json:"template_id"`
	TemplateVersion     int       `json:"template_version"`
	TemplateMDDMVersion int       `json:"template_mddm_version"`
	TemplateContentHash string    `json:"template_content_hash"`
}

// templateRow is the raw template record loaded from a template repository.
// Package-private because the public boundary is LoadAndVerify returning json.RawMessage.
type templateRow struct {
	ID            uuid.UUID
	TemplateID    uuid.UUID
	Version       int
	MDDMVersion   int
	ContentBlocks json.RawMessage
	ContentHash   string
	IsPublished   bool
}

type TemplateRepository interface {
	Get(ctx context.Context, templateID uuid.UUID, version int) (*templateRow, error)
}

type TemplateService struct {
	repo TemplateRepository
}

func NewTemplateService(repo TemplateRepository) *TemplateService {
	return &TemplateService{repo: repo}
}

// LoadAndVerify loads the template snapshot and verifies its hash matches the ref.
// Returns the verified content_blocks (still at the template's mddm_version, NOT migrated).
func (s *TemplateService) LoadAndVerify(ctx context.Context, ref TemplateRef) (json.RawMessage, error) {
	row, err := s.repo.Get(ctx, ref.TemplateID, ref.TemplateVersion)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrTemplateSnapshotMissing, err)
	}

	// Compute hash of the content bytes (in later tasks we'll canonicalize first as defense-in-depth)
	computed := computeContentHash(row.ContentBlocks)
	if computed != ref.TemplateContentHash {
		return nil, fmt.Errorf("%w: stored=%s ref=%s", ErrTemplateSnapshotMismatch, computed, ref.TemplateContentHash)
	}
	return row.ContentBlocks, nil
}

func computeContentHash(content json.RawMessage) string {
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}
