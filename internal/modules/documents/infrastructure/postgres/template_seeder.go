package postgres

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"

	"metaldocs/internal/modules/documents/domain/mddm"
)

type TemplateSeeder struct {
	db *sql.DB
}

func NewTemplateSeeder(db *sql.DB) *TemplateSeeder {
	return &TemplateSeeder{db: db}
}

func (s *TemplateSeeder) SeedPOTemplate(ctx context.Context, templateID uuid.UUID) error {
	envelope := mddm.POTemplateMDDM()
	canonicalEnvelope, err := mddm.CanonicalizeMDDM(envelope)
	if err != nil {
		return fmt.Errorf("canonicalize po template: %w", err)
	}

	canonicalBytes, err := mddm.MarshalCanonical(canonicalEnvelope)
	if err != nil {
		return fmt.Errorf("marshal canonical po template: %w", err)
	}

	hashBytes := sha256.Sum256(canonicalBytes)
	contentHash := hex.EncodeToString(hashBytes[:])

	mddmVersion, ok := canonicalEnvelope["mddm_version"].(int)
	if !ok {
		return fmt.Errorf("canonical po template missing integer mddm_version")
	}

	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO metaldocs.document_template_versions_mddm
		  (template_id, version, mddm_version, content_blocks, content_hash, is_published)
		VALUES ($1, $2, $3, $4::jsonb, $5, false)
		ON CONFLICT (template_id, version) DO NOTHING
	`, templateID, 1, mddmVersion, canonicalBytes, contentHash); err != nil {
		return fmt.Errorf("insert canonical po template seed: %w", err)
	}

	return nil
}
