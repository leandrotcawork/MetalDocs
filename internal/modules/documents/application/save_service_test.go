package application

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/uuid"

	"metaldocs/internal/modules/documents/domain/mddm"
)

func TestSaveDraftService_RejectsInvalidEnvelope(t *testing.T) {
	svc := NewSaveDraftService(nil, nil, nil, mddm.RulesContext{})
	envelope := json.RawMessage(`{"mddm_version":1}`) // missing blocks/template_ref

	_, err := svc.SaveDraft(context.Background(), SaveDraftInput{
		DocumentID:   "PO-118",
		BaseVersion:  1,
		EnvelopeJSON: envelope,
		UserID:       "user-1",
	})
	if err == nil {
		t.Error("expected validation error, got nil")
	}
}

// ---------------------------------------------------------------------------
// Test A: locked block deleted from document
// ---------------------------------------------------------------------------

func TestSaveDraftService_RejectsDeletedLockedBlock(t *testing.T) {
	templateID := uuid.New()
	templateVersion := 1

	// Template content: one locked section block with template_block_id "tpl-sec-1"
	templateBlocks := json.RawMessage(`[{
		"id": "tpl-id-001",
		"type": "section",
		"template_block_id": "tpl-sec-1",
		"props": {"title": "Header", "color": "#000000", "locked": true},
		"children": []
	}]`)
	templateContent := json.RawMessage(fmt.Sprintf(`{"mddm_version":1,"blocks":%s,"template_ref":null}`, templateBlocks))
	templateHash := computeContentHash(templateContent)

	templateRefJSON, _ := json.Marshal(TemplateRef{
		TemplateID:          templateID,
		TemplateVersion:     templateVersion,
		TemplateMDDMVersion: 1,
		TemplateContentHash: templateHash,
	})

	draftID := uuid.New()
	fakeRepo := &fakeDocDraftRepository{
		draft: &draftRow{
			ID:              draftID,
			VersionNumber:   1,
			TemplateRef:     templateRefJSON,
			PreviousContent: nil,
		},
	}

	fakeTmplRepo := &fakeTemplateRepository{
		row: &templateRow{
			ID:            uuid.New(),
			TemplateID:    templateID,
			Version:       templateVersion,
			MDDMVersion:   1,
			ContentBlocks: templateContent,
			ContentHash:   templateHash,
			IsPublished:   true,
		},
	}

	ts := NewTemplateService(fakeTmplRepo)
	svc := NewSaveDraftService(fakeRepo, ts, &noopImageReconciler{}, mddm.RulesContext{})

	// Envelope with the locked block REMOVED
	envelope := json.RawMessage(`{"mddm_version":1,"blocks":[],"template_ref":null}`)

	_, err := svc.SaveDraft(context.Background(), SaveDraftInput{
		DocumentID:   "doc-1",
		BaseVersion:  1,
		EnvelopeJSON: envelope,
		UserID:       "user-1",
	})
	if err == nil {
		t.Fatal("expected LOCKED_BLOCK_DELETED error, got nil")
	}
	if !containsCode(err.Error(), "LOCKED_BLOCK_DELETED") {
		t.Errorf("expected LOCKED_BLOCK_DELETED in error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Test B: block ID rewrite for a templated block
// ---------------------------------------------------------------------------

func TestSaveDraftService_RejectsBlockIDRewrite(t *testing.T) {
	templateID := uuid.New()
	templateVersion := 1

	// Template content: one section with template_block_id (not locked, so no prop check)
	templateBlocks := json.RawMessage(`[{
		"id": "tpl-id-001",
		"type": "section",
		"template_block_id": "tpl-sec-1",
		"props": {"title": "Header", "color": "#000000", "locked": false},
		"children": []
	}]`)
	templateContent := json.RawMessage(fmt.Sprintf(`{"mddm_version":1,"blocks":%s,"template_ref":null}`, templateBlocks))
	templateHash := computeContentHash(templateContent)

	templateRefJSON, _ := json.Marshal(TemplateRef{
		TemplateID:          templateID,
		TemplateVersion:     templateVersion,
		TemplateMDDMVersion: 1,
		TemplateContentHash: templateHash,
	})

	// Previous saved content: block with id "original-id" and template_block_id "tpl-sec-1"
	previousContent := json.RawMessage(`{"mddm_version":1,"blocks":[{
		"id": "original-id",
		"type": "section",
		"template_block_id": "tpl-sec-1",
		"props": {"title": "Header", "color": "#000000", "locked": false},
		"children": []
	}],"template_ref":null}`)

	draftID := uuid.New()
	fakeRepo := &fakeDocDraftRepository{
		draft: &draftRow{
			ID:              draftID,
			VersionNumber:   2,
			TemplateRef:     templateRefJSON,
			PreviousContent: previousContent,
		},
	}

	fakeTmplRepo := &fakeTemplateRepository{
		row: &templateRow{
			ID:            uuid.New(),
			TemplateID:    templateID,
			Version:       templateVersion,
			MDDMVersion:   1,
			ContentBlocks: templateContent,
			ContentHash:   templateHash,
			IsPublished:   true,
		},
	}

	ts := NewTemplateService(fakeTmplRepo)
	svc := NewSaveDraftService(fakeRepo, ts, &noopImageReconciler{}, mddm.RulesContext{})

	// Envelope: same template_block_id "tpl-sec-1" but DIFFERENT block id "rewritten-id"
	envelope := json.RawMessage(`{"mddm_version":1,"blocks":[{
		"id": "rewritten-id",
		"type": "section",
		"template_block_id": "tpl-sec-1",
		"props": {"title": "Header", "color": "#000000", "locked": false},
		"children": []
	}],"template_ref":null}`)

	_, err := svc.SaveDraft(context.Background(), SaveDraftInput{
		DocumentID:   "doc-2",
		BaseVersion:  2,
		EnvelopeJSON: envelope,
		UserID:       "user-1",
	})
	if err == nil {
		t.Fatal("expected BLOCK_ID_REWRITE_FORBIDDEN error, got nil")
	}
	if !containsCode(err.Error(), "BLOCK_ID_REWRITE_FORBIDDEN") {
		t.Errorf("expected BLOCK_ID_REWRITE_FORBIDDEN in error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Test C: locked block prop mutation
// ---------------------------------------------------------------------------

func TestSaveDraftService_RejectsLockedPropMutation(t *testing.T) {
	templateID := uuid.New()
	templateVersion := 1

	// Template: one locked section with template_block_id "tpl-sec-1", title "S1"
	templateBlocks := json.RawMessage(`[{
		"id": "tpl-id-001",
		"type": "section",
		"template_block_id": "tpl-sec-1",
		"props": {"title": "S1", "color": "#000000", "locked": true},
		"children": []
	}]`)
	templateContent := json.RawMessage(fmt.Sprintf(`{"mddm_version":1,"blocks":%s,"template_ref":null}`, templateBlocks))
	templateHash := computeContentHash(templateContent)

	templateRefJSON, _ := json.Marshal(TemplateRef{
		TemplateID:          templateID,
		TemplateVersion:     templateVersion,
		TemplateMDDMVersion: 1,
		TemplateContentHash: templateHash,
	})

	draftID := uuid.New()
	fakeRepo := &fakeDocDraftRepository{
		draft: &draftRow{
			ID:              draftID,
			VersionNumber:   1,
			TemplateRef:     templateRefJSON,
			PreviousContent: nil,
		},
	}

	fakeTmplRepo := &fakeTemplateRepository{
		row: &templateRow{
			ID:            uuid.New(),
			TemplateID:    templateID,
			Version:       templateVersion,
			MDDMVersion:   1,
			ContentBlocks: templateContent,
			ContentHash:   templateHash,
			IsPublished:   true,
		},
	}

	ts := NewTemplateService(fakeTmplRepo)
	svc := NewSaveDraftService(fakeRepo, ts, &noopImageReconciler{}, mddm.RulesContext{})

	// Envelope: same template_block_id and id but title changed from "S1" to "S2"
	envelope := json.RawMessage(`{"mddm_version":1,"blocks":[{
		"id": "tpl-id-001",
		"type": "section",
		"template_block_id": "tpl-sec-1",
		"props": {"title": "S2", "color": "#000000", "locked": true},
		"children": []
	}],"template_ref":null}`)

	_, err := svc.SaveDraft(context.Background(), SaveDraftInput{
		DocumentID:   "doc-3",
		BaseVersion:  1,
		EnvelopeJSON: envelope,
		UserID:       "user-1",
	})
	if err == nil {
		t.Fatal("expected LOCKED_BLOCK_PROP_MUTATED error, got nil")
	}
	if !containsCode(err.Error(), "LOCKED_BLOCK_PROP_MUTATED") {
		t.Errorf("expected LOCKED_BLOCK_PROP_MUTATED in error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Test D: locked block reparented
// ---------------------------------------------------------------------------

func TestSaveDraftService_RejectsLockedBlockReparenting(t *testing.T) {
	templateID := uuid.New()
	templateVersion := 1

	// Template: a section (tpl-sec-1) containing a locked fieldGroup child (tpl-fg-1)
	templateBlocks := json.RawMessage(`[{
		"id": "tpl-id-001",
		"type": "section",
		"template_block_id": "tpl-sec-1",
		"props": {"title": "Section", "color": "#000000", "locked": false},
		"children": [{
			"id": "tpl-id-002",
			"type": "fieldGroup",
			"template_block_id": "tpl-fg-1",
			"props": {"columns": 1, "locked": true},
			"children": []
		}]
	}]`)
	templateContent := json.RawMessage(fmt.Sprintf(`{"mddm_version":1,"blocks":%s,"template_ref":null}`, templateBlocks))
	templateHash := computeContentHash(templateContent)

	templateRefJSON, _ := json.Marshal(TemplateRef{
		TemplateID:          templateID,
		TemplateVersion:     templateVersion,
		TemplateMDDMVersion: 1,
		TemplateContentHash: templateHash,
	})

	draftID := uuid.New()
	fakeRepo := &fakeDocDraftRepository{
		draft: &draftRow{
			ID:              draftID,
			VersionNumber:   1,
			TemplateRef:     templateRefJSON,
			PreviousContent: nil,
		},
	}

	fakeTmplRepo := &fakeTemplateRepository{
		row: &templateRow{
			ID:            uuid.New(),
			TemplateID:    templateID,
			Version:       templateVersion,
			MDDMVersion:   1,
			ContentBlocks: templateContent,
			ContentHash:   templateHash,
			IsPublished:   true,
		},
	}

	ts := NewTemplateService(fakeTmplRepo)
	svc := NewSaveDraftService(fakeRepo, ts, &noopImageReconciler{}, mddm.RulesContext{})

	// Envelope: section present, but fieldGroup moved to top level (different parent)
	envelope := json.RawMessage(`{"mddm_version":1,"blocks":[{
		"id": "tpl-id-001",
		"type": "section",
		"template_block_id": "tpl-sec-1",
		"props": {"title": "Section", "color": "#000000", "locked": false},
		"children": []
	},{
		"id": "tpl-id-002",
		"type": "fieldGroup",
		"template_block_id": "tpl-fg-1",
		"props": {"columns": 1, "locked": true},
		"children": []
	}],"template_ref":null}`)

	_, err := svc.SaveDraft(context.Background(), SaveDraftInput{
		DocumentID:   "doc-4",
		BaseVersion:  1,
		EnvelopeJSON: envelope,
		UserID:       "user-1",
	})
	if err == nil {
		t.Fatal("expected LOCKED_BLOCK_REPARENTED error, got nil")
	}
	if !containsCode(err.Error(), "LOCKED_BLOCK_REPARENTED") {
		t.Errorf("expected LOCKED_BLOCK_REPARENTED in error, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Fake implementations
// ---------------------------------------------------------------------------

type fakeDocDraftRepository struct {
	draft *draftRow
}

func (r *fakeDocDraftRepository) GetActiveDraft(_ context.Context, _ string) (*draftRow, error) {
	return r.draft, nil
}

func (r *fakeDocDraftRepository) UpdateDraftContent(_ context.Context, _ uuid.UUID, _ json.RawMessage, _ string) error {
	return nil
}

type fakeTemplateRepository struct {
	row *templateRow
}

func (r *fakeTemplateRepository) Get(_ context.Context, _ uuid.UUID, _ int) (*templateRow, error) {
	return r.row, nil
}

type noopImageReconciler struct{}

func (n *noopImageReconciler) Reconcile(_ context.Context, _ uuid.UUID, _ []uuid.UUID) error {
	return nil
}

// containsCode is a helper that checks if a substring appears in s.
func containsCode(s, code string) bool {
	return len(s) >= len(code) && (s == code || len(s) > 0 && containsSubstring(s, code))
}

func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
