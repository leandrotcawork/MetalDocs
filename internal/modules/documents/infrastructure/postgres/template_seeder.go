package postgres

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"encoding/hex"
	"fmt"

	"github.com/google/uuid"

	"metaldocs/internal/modules/documents/domain/mddm"
)

var DefaultPOTemplateID = uuid.MustParse("00000000-0000-0000-0000-0000000000a1")

type TemplateSeeder struct {
	db *sql.DB
}

func NewTemplateSeeder(db *sql.DB) *TemplateSeeder {
	return &TemplateSeeder{db: db}
}

func (s *TemplateSeeder) SeedPOTemplate(ctx context.Context, templateID uuid.UUID) error {
	return s.seedTemplateVersion(ctx, templateID, mddm.POTemplateMDDM())
}

func (s *TemplateSeeder) seedTemplateVersion(ctx context.Context, templateID uuid.UUID, envelope map[string]any) error {
	normalizedEnvelope, err := normalizeTemplateEnvelope(envelope)
	if err != nil {
		return fmt.Errorf("normalize po template envelope: %w", err)
	}

	canonicalEnvelope, err := mddm.CanonicalizeMDDM(normalizedEnvelope)
	if err != nil {
		return fmt.Errorf("canonicalize po template: %w", err)
	}

	canonicalBlocks, ok := canonicalEnvelope["blocks"].([]any)
	if !ok || len(canonicalBlocks) == 0 {
		return fmt.Errorf("canonical po template malformed: blocks must be non-empty")
	}

	canonicalBytes, err := mddm.MarshalCanonical(canonicalEnvelope)
	if err != nil {
		return fmt.Errorf("marshal canonical po template: %w", err)
	}

	hashBytes := sha256.Sum256(canonicalBytes)
	contentHash := hex.EncodeToString(hashBytes[:])

	mddmVersion, err := canonicalEnvelopeVersion(canonicalEnvelope["mddm_version"])
	if err != nil {
		return err
	}

	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO metaldocs.document_template_versions_mddm
		  (template_id, version, mddm_version, content_blocks, content_hash, is_published)
		VALUES ($1, $2, $3, $4::jsonb, $5, true)
		ON CONFLICT (template_id, version) DO NOTHING
	`, templateID, 1, mddmVersion, canonicalBytes, contentHash); err != nil {
		return fmt.Errorf("insert canonical po template seed: %w", err)
	}

	return nil
}

func normalizeTemplateEnvelope(envelope map[string]any) (map[string]any, error) {
	raw, err := json.Marshal(envelope)
	if err != nil {
		return nil, err
	}

	var normalized map[string]any
	if err := json.Unmarshal(raw, &normalized); err != nil {
		return nil, err
	}
	return normalized, nil
}

func canonicalEnvelopeVersion(value any) (int, error) {
	switch typed := value.(type) {
	case int:
		return typed, nil
	case int32:
		return int(typed), nil
	case int64:
		return int(typed), nil
	case float64:
		if typed != float64(int(typed)) {
			return 0, fmt.Errorf("canonical po template missing integer mddm_version")
		}
		return int(typed), nil
	default:
		return 0, fmt.Errorf("canonical po template missing integer mddm_version")
	}
}
