package application

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"metaldocs/internal/modules/documents/domain/mddm"
)

type SaveDraftInput struct {
	DocumentID   string
	BaseVersion  int
	EnvelopeJSON json.RawMessage
	UserID       string
}

type SaveDraftOutput struct {
	VersionID   uuid.UUID
	ContentHash string
	NewVersion  int
}

// SaveDraftService coordinates: normalize → Layer 1 → load template → verify hash → Layer 2 →
// transactionally update draft row + reconcile image references.
type SaveDraftService struct {
	repo            DraftRepository
	templateService *TemplateService
	imageRecon      ImageReconciler
	rulesDeps       mddm.RulesContext // partially populated; per-call fields filled in SaveDraft
}

type DraftRepository interface {
	GetActiveDraft(ctx context.Context, documentID string) (*draftRow, error)
	UpdateDraftContent(ctx context.Context, id uuid.UUID, content json.RawMessage, hash string) error
}

type ImageReconciler interface {
	Reconcile(ctx context.Context, versionID uuid.UUID, imageIDs []uuid.UUID) error
}

type draftRow struct {
	ID              uuid.UUID
	VersionNumber   int
	TemplateRef     json.RawMessage
	PreviousContent json.RawMessage
}

func NewSaveDraftService(repo DraftRepository, ts *TemplateService, recon ImageReconciler, rulesDeps mddm.RulesContext) *SaveDraftService {
	return &SaveDraftService{repo: repo, templateService: ts, imageRecon: recon, rulesDeps: rulesDeps}
}

func (s *SaveDraftService) SaveDraft(ctx context.Context, in SaveDraftInput) (*SaveDraftOutput, error) {
	// 1. Layer 1: schema validation
	if err := mddm.ValidateMDDMBytes(in.EnvelopeJSON); err != nil {
		return nil, fmt.Errorf("validation_failed: %w", err)
	}

	// 2. Parse + canonicalize
	var envelope map[string]any
	if err := json.Unmarshal(in.EnvelopeJSON, &envelope); err != nil {
		return nil, fmt.Errorf("parse: %w", err)
	}
	canonical, err := mddm.CanonicalizeMDDM(envelope)
	if err != nil {
		return nil, fmt.Errorf("canonicalize: %w", err)
	}

	// (moved up from step 5) Load existing draft row so TemplateRef and PreviousContent
	// are available for governance checks before Layer 2 runs.
	draft, err := s.repo.GetActiveDraft(ctx, in.DocumentID)
	if err != nil {
		return nil, err
	}
	if draft == nil {
		return nil, fmt.Errorf("no active draft for document %s", in.DocumentID)
	}

	docBlocks, _ := canonical["blocks"].([]any)

	// 2a. Template governance: snapshot verify + locked-block enforcement
	var templateBlocks []any
	if s.templateService != nil && len(draft.TemplateRef) > 0 {
		var ref TemplateRef
		if err := json.Unmarshal(draft.TemplateRef, &ref); err != nil {
			return nil, fmt.Errorf("template_ref parse: %w", err)
		}
		templateContent, err := s.templateService.LoadAndVerify(ctx, ref)
		if err != nil {
			return nil, err
		}
		var tmplEnvelope map[string]any
		if err := json.Unmarshal(templateContent, &tmplEnvelope); err != nil {
			return nil, fmt.Errorf("template content parse: %w", err)
		}
		templateBlocks, _ = tmplEnvelope["blocks"].([]any)
		if err := mddm.EnforceLockedBlocks(templateBlocks, docBlocks); err != nil {
			return nil, err
		}
	}

	// 2b. Block ID continuity: reject rewrites of templated block IDs across saves
	if len(draft.PreviousContent) > 0 {
		var prevEnvelope map[string]any
		if err := json.Unmarshal(draft.PreviousContent, &prevEnvelope); err != nil {
			return nil, fmt.Errorf("previous content parse: %w", err)
		}
		prevBlocks, _ := prevEnvelope["blocks"].([]any)
		if err := mddm.CheckBlockIDContinuity(prevBlocks, docBlocks); err != nil {
			return nil, err
		}
	}

	// 3. Layer 2: business rules (with template/previous context populated)
	rctx := s.rulesDeps
	rctx.Ctx = ctx
	rctx.DocumentID = in.DocumentID
	rctx.UserID = in.UserID
	rctx.TemplateBlocks = templateBlocks
	if len(draft.PreviousContent) > 0 {
		var prevEnvelope map[string]any
		if err := json.Unmarshal(draft.PreviousContent, &prevEnvelope); err == nil {
			rctx.PreviousBlocks, _ = prevEnvelope["blocks"].([]any)
		}
	}
	if err := mddm.EnforceLayer2(rctx, canonical); err != nil {
		return nil, err
	}

	// 4. Marshal canonical, compute hash
	canonicalBytes, err := mddm.MarshalCanonical(canonical)
	if err != nil {
		return nil, err
	}
	hash := computeContentHash(canonicalBytes)

	// 5. Update draft content (in-place)
	if err := s.repo.UpdateDraftContent(ctx, draft.ID, canonicalBytes, hash); err != nil {
		return nil, err
	}

	// 6. Reconcile image references
	imageIDs := extractImageIDs(canonical)
	if err := s.imageRecon.Reconcile(ctx, draft.ID, imageIDs); err != nil {
		return nil, err
	}

	return &SaveDraftOutput{VersionID: draft.ID, ContentHash: hash, NewVersion: draft.VersionNumber}, nil
}

func extractImageIDs(envelope map[string]any) []uuid.UUID {
	out := []uuid.UUID{}
	blocks, _ := envelope["blocks"].([]any)
	var walk func([]any)
	walk = func(bs []any) {
		for _, b := range bs {
			bm, ok := b.(map[string]any)
			if !ok {
				continue
			}
			if t, _ := bm["type"].(string); t == "image" {
				if props, ok := bm["props"].(map[string]any); ok {
					if src, ok := props["src"].(string); ok && len(src) > len("/api/images/") {
						idStr := src[len("/api/images/"):]
						if id, err := uuid.Parse(idStr); err == nil {
							out = append(out, id)
						}
					}
				}
			}
			if children, ok := bm["children"].([]any); ok {
				walk(children)
			}
		}
	}
	walk(blocks)
	return out
}
