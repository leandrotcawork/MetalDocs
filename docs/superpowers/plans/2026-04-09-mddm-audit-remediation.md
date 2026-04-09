# MDDM Audit Remediation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix the 6 findings from the Codex/Claude consensus audit so the MDDM foundational branch is merge-ready — safe governance on save, real release wiring, cross-runtime canonicalization parity, complete DOCX export, tightened schema, and proper integration coverage.

**Architecture:** All fixes target existing files. No new modules or packages. Save-path governance wires already-implemented domain primitives (`EnforceLockedBlocks`, `CheckBlockIDContinuity`, `TemplateService.LoadAndVerify`) into the save flow. Release handler calls the existing `ReleaseService`. Canonicalization parity mirrors recent TS fixes in Go. DOCX exporter fills the 4 missing block types. Schema adds `if/then` for Field children by valueMode.

**Tech Stack:** Go 1.24, TypeScript, `docx` npm v9.6.1, JSON Schema 2020-12, Vitest, Go `testing`

**Spec reference:** `docs/superpowers/specs/2026-04-07-mddm-foundational-design.md`

**Worktree:** `.worktrees/mddm-foundational` (branch `feature/mddm-foundational`)

---

## File Structure (locked for this remediation)

### Must Fix
- `internal/modules/documents/application/save_service.go` — wire template governance
- `internal/modules/documents/application/save_service_test.go` — governance rejection tests
- `internal/modules/documents/domain/mddm/rules.go` — add `field` to `isInlineParent`, wire locked-block + ID continuity into Layer 2
- `internal/modules/documents/domain/mddm/canonicalize.go` — add `field` to inline parents, fix unknown-mark sort
- `internal/modules/documents/domain/mddm/canonicalize_test.go` — parity fixtures
- `internal/modules/documents/delivery/http/release_handler.go` — wire to release service
- `internal/modules/documents/delivery/http/release_handler_test.go` — handler wiring test
- `internal/modules/documents/application/release_service.go` — real diff computation

### Should Fix
- `apps/docgen/src/mddm/exporter.ts` — render richBlock, quote, image, hyperlinks
- `apps/docgen/src/mddm/render-image.ts` — already exists, wire into exporter
- `apps/docgen/__tests__/exporter.test.ts` — expand beyond smoke test
- `shared/schemas/mddm.schema.json` — Field children conditional on valueMode
- `shared/schemas/__tests__/schema.test.ts` — new invalid fixtures for field grammar
- `shared/schemas/test-fixtures/canonical/input-field-inline.json` — parity fixture
- `shared/schemas/test-fixtures/canonical/output-field-inline.json` — parity fixture

---

# MUST FIX

## Task 1: Wire template governance into save path

**Files:**
- Modify: `internal/modules/documents/application/save_service.go`
- Modify: `internal/modules/documents/application/save_service_test.go`

The save service has `templateService` and `RulesContext.TemplateBlocks`/`PreviousBlocks` fields but never populates them. The `EnforceLockedBlocks` and `CheckBlockIDContinuity` functions exist in the domain but are never called.

- [ ] **Step 1: Write failing test — save rejects deleted locked block**

Add to `internal/modules/documents/application/save_service_test.go`:

```go
type fakeDraftRepo struct {
	draft *draftRow
}

func (r *fakeDraftRepo) GetActiveDraft(_ context.Context, _ string) (*draftRow, error) {
	return r.draft, nil
}

func (r *fakeDraftRepo) UpdateDraftContent(_ context.Context, _ uuid.UUID, _ json.RawMessage, _ string) error {
	return nil
}

type fakeImageRecon struct{}

func (r *fakeImageRecon) Reconcile(_ context.Context, _ uuid.UUID, _ []uuid.UUID) error {
	return nil
}

type fakeTemplateRepo struct {
	row *templateRow
}

func (r *fakeTemplateRepo) Get(_ context.Context, _ uuid.UUID, _ int) (*templateRow, error) {
	if r.row == nil {
		return nil, fmt.Errorf("not found")
	}
	return r.row, nil
}

func TestSaveDraftService_RejectsDeletedLockedBlock(t *testing.T) {
	templateContent := json.RawMessage(`{
		"mddm_version":1,
		"template_ref":null,
		"blocks":[{
			"id":"aaaa0000-0000-0000-0000-000000000001",
			"template_block_id":"bbbb0000-0000-0000-0000-000000000001",
			"type":"section",
			"props":{"title":"S1","color":"#000000","locked":true},
			"children":[]
		}]
	}`)
	templateHash := computeContentHash(templateContent)

	templateRef := json.RawMessage(fmt.Sprintf(`{
		"template_id":"cccc0000-0000-0000-0000-000000000001",
		"template_version":1,
		"template_mddm_version":1,
		"template_content_hash":"%s"
	}`, templateHash))

	draftRepo := &fakeDraftRepo{draft: &draftRow{
		ID:            uuid.MustParse("dddd0000-0000-0000-0000-000000000001"),
		VersionNumber: 1,
		TemplateRef:   templateRef,
	}}

	tmplRepo := &fakeTemplateRepo{row: &templateRow{
		TemplateID:    uuid.MustParse("cccc0000-0000-0000-0000-000000000001"),
		Version:       1,
		MDDMVersion:   1,
		ContentBlocks: templateContent,
		ContentHash:   templateHash,
		IsPublished:   true,
	}}

	ts := NewTemplateService(tmplRepo)
	svc := NewSaveDraftService(draftRepo, ts, &fakeImageRecon{}, mddm.RulesContext{})

	// Envelope that deletes the locked section block entirely
	envelope := json.RawMessage(`{
		"mddm_version":1,
		"template_ref":{"template_id":"cccc0000-0000-0000-0000-000000000001","template_version":1,"template_mddm_version":1,"template_content_hash":"` + templateHash + `"},
		"blocks":[]
	}`)

	_, err := svc.SaveDraft(context.Background(), SaveDraftInput{
		DocumentID:   "PO-118",
		BaseVersion:  1,
		EnvelopeJSON: envelope,
		UserID:       "user-1",
	})
	if err == nil {
		t.Fatal("expected LOCKED_BLOCK_DELETED error, got nil")
	}
	if !strings.Contains(err.Error(), "LOCKED_BLOCK_DELETED") {
		t.Fatalf("expected LOCKED_BLOCK_DELETED, got: %v", err)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd .worktrees/mddm-foundational && go test ./internal/modules/documents/application/... -run TestSaveDraftService_RejectsDeletedLockedBlock -v`

Expected: FAIL — locked block check is not wired, save succeeds.

- [ ] **Step 3: Wire template governance into SaveDraft**

In `internal/modules/documents/application/save_service.go`, replace the section between steps 2 and 3 (after canonicalization, before Layer 2) with template loading and governance:

```go
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

	// 2b. Load existing draft to get template_ref and previous content
	draft, err := s.repo.GetActiveDraft(ctx, in.DocumentID)
	if err != nil {
		return nil, err
	}
	if draft == nil {
		return nil, fmt.Errorf("no active draft for document %s", in.DocumentID)
	}

	// 2c. Template governance: verify snapshot, enforce locked blocks, enforce ID continuity
	rctx := s.rulesDeps
	rctx.Ctx = ctx
	rctx.DocumentID = in.DocumentID
	rctx.UserID = in.UserID

	if s.templateService != nil && len(draft.TemplateRef) > 0 {
		var ref TemplateRef
		if err := json.Unmarshal(draft.TemplateRef, &ref); err == nil && ref.TemplateID != uuid.Nil {
			templateContent, err := s.templateService.LoadAndVerify(ctx, ref)
			if err != nil {
				return nil, err
			}

			var tmplEnvelope map[string]any
			if err := json.Unmarshal(templateContent, &tmplEnvelope); err == nil {
				tmplBlocks, _ := tmplEnvelope["blocks"].([]any)
				rctx.TemplateBlocks = tmplBlocks

				// Locked-block enforcement
				docBlocks, _ := canonical["blocks"].([]any)
				if err := mddm.EnforceLockedBlocks(tmplBlocks, docBlocks); err != nil {
					return nil, err
				}
			}
		}
	}

	// 2d. Block ID continuity (previous version vs current)
	if draft.PreviousContent != nil {
		var prevEnvelope map[string]any
		if err := json.Unmarshal(draft.PreviousContent, &prevEnvelope); err == nil {
			prevBlocks, _ := prevEnvelope["blocks"].([]any)
			rctx.PreviousBlocks = prevBlocks
			docBlocks, _ := canonical["blocks"].([]any)
			if err := mddm.CheckBlockIDContinuity(prevBlocks, docBlocks); err != nil {
				return nil, err
			}
		}
	}

	// 3. Layer 2: business rules
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
```

Also add `PreviousContent` to the `draftRow` struct:

```go
type draftRow struct {
	ID              uuid.UUID
	VersionNumber   int
	TemplateRef     json.RawMessage
	PreviousContent json.RawMessage // content_blocks from the most recent saved state (for ID continuity)
}
```

Update `DraftRepository.GetActiveDraft` to also return `PreviousContent` (the current content_blocks of the draft row before the new save overwrites it).

- [ ] **Step 4: Run test to verify it passes**

Run: `cd .worktrees/mddm-foundational && go test ./internal/modules/documents/application/... -run TestSaveDraftService -v`

Expected: PASS — both old and new tests pass.

- [ ] **Step 5: Write failing test — save rejects block ID rewrite**

Add to `internal/modules/documents/application/save_service_test.go`:

```go
func TestSaveDraftService_RejectsBlockIDRewrite(t *testing.T) {
	templateContent := json.RawMessage(`{
		"mddm_version":1,"template_ref":null,
		"blocks":[{
			"id":"aaaa0000-0000-0000-0000-000000000001",
			"template_block_id":"bbbb0000-0000-0000-0000-000000000001",
			"type":"section",
			"props":{"title":"S1","color":"#000000","locked":true},
			"children":[]
		}]
	}`)
	templateHash := computeContentHash(templateContent)

	// Previous content has the original block ID
	previousContent := templateContent

	draftRepo := &fakeDraftRepo{draft: &draftRow{
		ID:              uuid.MustParse("dddd0000-0000-0000-0000-000000000001"),
		VersionNumber:   1,
		TemplateRef:     json.RawMessage(fmt.Sprintf(`{"template_id":"cccc0000-0000-0000-0000-000000000001","template_version":1,"template_mddm_version":1,"template_content_hash":"%s"}`, templateHash)),
		PreviousContent: previousContent,
	}}

	tmplRepo := &fakeTemplateRepo{row: &templateRow{
		TemplateID: uuid.MustParse("cccc0000-0000-0000-0000-000000000001"), Version: 1, MDDMVersion: 1,
		ContentBlocks: templateContent, ContentHash: templateHash, IsPublished: true,
	}}

	ts := NewTemplateService(tmplRepo)
	svc := NewSaveDraftService(draftRepo, ts, &fakeImageRecon{}, mddm.RulesContext{})

	// Same template_block_id but DIFFERENT block id
	envelope := json.RawMessage(`{
		"mddm_version":1,
		"template_ref":{"template_id":"cccc0000-0000-0000-0000-000000000001","template_version":1,"template_mddm_version":1,"template_content_hash":"` + templateHash + `"},
		"blocks":[{
			"id":"eeee0000-0000-0000-0000-999999999999",
			"template_block_id":"bbbb0000-0000-0000-0000-000000000001",
			"type":"section",
			"props":{"title":"S1","color":"#000000","locked":true},
			"children":[]
		}]
	}`)

	_, err := svc.SaveDraft(context.Background(), SaveDraftInput{
		DocumentID: "PO-118", BaseVersion: 1, EnvelopeJSON: envelope, UserID: "user-1",
	})
	if err == nil {
		t.Fatal("expected BLOCK_ID_REWRITE_FORBIDDEN, got nil")
	}
	if !strings.Contains(err.Error(), "BLOCK_ID_REWRITE_FORBIDDEN") {
		t.Fatalf("expected BLOCK_ID_REWRITE_FORBIDDEN, got: %v", err)
	}
}
```

- [ ] **Step 6: Run test to verify it passes**

Run: `cd .worktrees/mddm-foundational && go test ./internal/modules/documents/application/... -run TestSaveDraftService_RejectsBlockIDRewrite -v`

Expected: PASS (the wiring from Step 3 already handles this).

- [ ] **Step 7: Commit**

```bash
git add internal/modules/documents/application/save_service.go internal/modules/documents/application/save_service_test.go
git commit -m "fix(mddm): wire template governance into save path (locked-block, ID continuity, snapshot verify)"
```

---

## Task 2: Fix Go canonicalization parity with TypeScript

**Files:**
- Modify: `internal/modules/documents/domain/mddm/canonicalize.go`
- Modify: `internal/modules/documents/domain/mddm/canonicalize_test.go`
- Create: `shared/schemas/test-fixtures/canonical/input-field-inline.json`
- Create: `shared/schemas/test-fixtures/canonical/output-field-inline.json`

Two mismatches exist: (a) Go does not include `field` in inline parents, (b) Go unknown-mark sorting is unstable.

- [ ] **Step 1: Create parity fixtures for `field` inline canonicalization**

Create `shared/schemas/test-fixtures/canonical/input-field-inline.json`:

```json
{
  "mddm_version": 1,
  "template_ref": null,
  "blocks": [
    {
      "id": "11111111-1111-1111-1111-111111111111",
      "type": "field",
      "props": { "label": "Name", "valueMode": "inline", "locked": true },
      "children": [
        { "text": "Hello ", "marks": [{ "type": "bold" }] },
        { "text": "world", "marks": [{ "type": "bold" }] }
      ]
    }
  ]
}
```

Create `shared/schemas/test-fixtures/canonical/output-field-inline.json`:

```json
{
  "blocks": [
    {
      "children": [
        {
          "marks": [{ "type": "bold" }],
          "text": "Hello world"
        }
      ],
      "id": "11111111-1111-1111-1111-111111111111",
      "props": { "label": "Name", "locked": true, "valueMode": "inline" },
      "type": "field"
    }
  ],
  "mddm_version": 1,
  "template_ref": null
}
```

- [ ] **Step 2: Write failing parity test in Go**

Add to `internal/modules/documents/domain/mddm/canonicalize_test.go`:

```go
func TestCanonicalizeMDDM_FieldInlineParity(t *testing.T) {
	canonicalDir := filepath.Join("..", "..", "..", "..", "..", "shared", "schemas", "test-fixtures", "canonical")
	inputBytes, err := os.ReadFile(filepath.Join(canonicalDir, "input-field-inline.json"))
	if err != nil {
		t.Fatal(err)
	}
	expectedBytes, err := os.ReadFile(filepath.Join(canonicalDir, "output-field-inline.json"))
	if err != nil {
		t.Fatal(err)
	}

	var input map[string]any
	if err := json.Unmarshal(inputBytes, &input); err != nil {
		t.Fatal(err)
	}

	canonical, err := CanonicalizeMDDM(input)
	if err != nil {
		t.Fatalf("canonicalize: %v", err)
	}

	actualBytes, err := MarshalCanonical(canonical)
	if err != nil {
		t.Fatal(err)
	}

	var expected map[string]any
	if err := json.Unmarshal(expectedBytes, &expected); err != nil {
		t.Fatal(err)
	}
	expectedNormalized, err := MarshalCanonical(expected)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(actualBytes, expectedNormalized) {
		t.Errorf("field inline parity mismatch:\nexpected: %s\nactual:   %s", expectedNormalized, actualBytes)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `cd .worktrees/mddm-foundational && go test ./internal/modules/documents/domain/mddm/... -run TestCanonicalizeMDDM_FieldInlineParity -v`

Expected: FAIL — `field` not in inline parents, so adjacent bold runs are not merged.

- [ ] **Step 4: Fix Go canonicalize — add `field` to inline parents and fix unknown-mark sort**

In `internal/modules/documents/domain/mddm/canonicalize.go`, replace the `inlineParents` map in `canonicalizeBlock`:

```go
	inlineParents := map[string]bool{
		"paragraph":        true,
		"heading":          true,
		"bulletListItem":   true,
		"numberedListItem": true,
		"dataTableCell":    true,
		"field":            true,
	}
```

Replace the mark sort in `canonicalizeInlineContent` (around line 108):

```go
			sort.SliceStable(sortedMarks, func(i, j int) bool {
				aType := sortedMarks[i].(map[string]any)["type"].(string)
				bType := sortedMarks[j].(map[string]any)["type"].(string)
				aIdx, aKnown := markOrder[aType]
				bIdx, bKnown := markOrder[bType]
				if !aKnown && !bKnown {
					return aType < bType
				}
				if !aKnown {
					return false
				}
				if !bKnown {
					return true
				}
				return aIdx < bIdx
			})
```

- [ ] **Step 5: Run tests to verify parity passes**

Run: `cd .worktrees/mddm-foundational && go test ./internal/modules/documents/domain/mddm/... -v`

Expected: ALL PASS — both existing and new parity fixtures.

- [ ] **Step 6: Also run TS canonicalize tests for sanity**

Run: `cd .worktrees/mddm-foundational/shared/schemas && npx vitest run`

Expected: ALL PASS (9 tests).

- [ ] **Step 7: Commit**

```bash
git add internal/modules/documents/domain/mddm/canonicalize.go internal/modules/documents/domain/mddm/canonicalize_test.go shared/schemas/test-fixtures/canonical/
git commit -m "fix(mddm): align Go canonicalization with TS (field inline parents, unknown-mark sort)"
```

---

## Task 3: Wire release handler to release service with real diff

**Files:**
- Modify: `internal/modules/documents/delivery/http/release_handler.go`
- Modify: `internal/modules/documents/application/release_service.go`
- Modify: `internal/modules/documents/delivery/http/release_handler_test.go` (create if not exists)

The release handler currently auth-checks and returns 200 without calling the service. The service computes a placeholder diff instead of using `ComputeDiff`.

- [ ] **Step 1: Write failing test — release handler calls service**

Create or modify `internal/modules/documents/delivery/http/release_handler_test.go`:

```go
package httpdelivery

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeReleaseAuth struct{ allow bool }

func (f *fakeReleaseAuth) CanApprove(_, _ string) bool { return f.allow }

type spyReleaseService struct {
	called bool
	err    error
}

func (s *spyReleaseService) ReleaseDraft(_ context.Context, _ application.ReleaseInput) error {
	s.called = true
	return s.err
}

func TestReleaseHandler_CallsService(t *testing.T) {
	spy := &spyReleaseService{}
	h := NewReleaseHandler(&fakeReleaseAuth{allow: true})
	h.releaseService = spy

	req := httptest.NewRequest(http.MethodPost, "/api/documents/PO-118/release", strings.NewReader(`{"draft_id":"dddd0000-0000-0000-0000-000000000001"}`))
	req = req.WithContext(withUserID(req.Context(), "user-1"))
	w := httptest.NewRecorder()

	h.Release(w, req)

	if !spy.called {
		t.Fatal("expected release service to be called")
	}
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd .worktrees/mddm-foundational && go test ./internal/modules/documents/delivery/http/... -run TestReleaseHandler_CallsService -v`

Expected: FAIL — handler does not have `releaseService` field or call it.

- [ ] **Step 3: Wire release handler to service**

Replace `internal/modules/documents/delivery/http/release_handler.go`:

```go
package httpdelivery

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"github.com/google/uuid"

	"metaldocs/internal/modules/documents/application"
)

type ReleaseServiceInterface interface {
	ReleaseDraft(ctx context.Context, in application.ReleaseInput) error
}

type ReleaseHandler struct {
	authChecker    ReleaseAuthChecker
	releaseService ReleaseServiceInterface
}

type ReleaseAuthChecker interface {
	CanApprove(userID, documentID string) bool
}

func NewReleaseHandler(auth ReleaseAuthChecker) *ReleaseHandler {
	return &ReleaseHandler{authChecker: auth}
}

func (h *ReleaseHandler) WithReleaseService(svc ReleaseServiceInterface) *ReleaseHandler {
	h.releaseService = svc
	return h
}

type releaseRequest struct {
	DraftID string `json:"draft_id"`
}

func (h *ReleaseHandler) Release(w http.ResponseWriter, r *http.Request) {
	userID := userIDFromContext(r.Context())
	if userID == "" {
		writeAPIError(w, http.StatusUnauthorized, "AUTH_UNAUTHORIZED", "Authentication required", requestTraceID(r))
		return
	}

	documentID := extractDocIDFromPath(r.URL.Path)
	if h.authChecker == nil || !h.authChecker.CanApprove(userID, documentID) {
		writeAPIError(w, http.StatusForbidden, "AUTH_FORBIDDEN", "Approval permission required", requestTraceID(r))
		return
	}

	if h.releaseService == nil {
		writeAPIError(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "Release service not configured", requestTraceID(r))
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1024))
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "BAD_REQUEST", "Failed to read body", requestTraceID(r))
		return
	}
	defer r.Body.Close()

	var req releaseRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeAPIError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid JSON body", requestTraceID(r))
		return
	}

	draftID, err := uuid.Parse(req.DraftID)
	if err != nil {
		writeAPIError(w, http.StatusBadRequest, "BAD_REQUEST", "Invalid draft_id", requestTraceID(r))
		return
	}

	if err := h.releaseService.ReleaseDraft(r.Context(), application.ReleaseInput{
		DocumentID: documentID,
		DraftID:    draftID,
		ApprovedBy: userID,
	}); err != nil {
		writeAPIError(w, http.StatusInternalServerError, "RELEASE_FAILED", err.Error(), requestTraceID(r))
		return
	}

	w.WriteHeader(http.StatusOK)
}
```

- [ ] **Step 4: Replace placeholder diff in release service with real ComputeDiff**

In `internal/modules/documents/application/release_service.go`, replace the placeholder diff line (around line 69):

```go
	// Compute real diff if there was a previous released version
	var diffJSON json.RawMessage
	if prevVersionID != uuid.Nil && len(prevDocx) > 0 {
		// Load previous canonical content for diff (archived version has content_blocks = NULL,
		// but prevDocx was fetched before archival). We need the canonical blocks.
		// For now, use the archived content that was returned from ArchivePreviousReleased.
		// The repo method should also return the previous content_blocks before nulling them.
		diffJSON, _ = json.Marshal(mddm.ComputeDiff(nil, docBlocks))
	} else {
		diffJSON, _ = json.Marshal(mddm.ComputeDiff(nil, docBlocks))
	}
```

Actually, the cleaner approach: update `ArchivePreviousReleased` to also return content_blocks, or add a method. For now, use what we have and compute a diff against empty for first release, or against the previous blocks. Update the repo interface and service:

In `release_service.go`, change `ArchivePreviousReleased` return to include content blocks:

```go
type ReleaseRepo interface {
	GetDraft(ctx context.Context, id uuid.UUID) (*DraftSnapshot, error)
	ArchivePreviousReleased(ctx context.Context, documentID string) (versionID uuid.UUID, prevContentBlocks []byte, docxBytes []byte, err error)
	PromoteDraftToReleased(ctx context.Context, draftID uuid.UUID, docxBytes []byte, approvedBy string) error
	StoreRevisionDiff(ctx context.Context, versionID uuid.UUID, diff json.RawMessage) error
	DeleteImageRefs(ctx context.Context, versionID uuid.UUID) error
	CleanupOrphanImages(ctx context.Context) error
}
```

Then in `ReleaseDraft`:

```go
func (s *ReleaseService) ReleaseDraft(ctx context.Context, in ReleaseInput) error {
	draft, err := s.repo.GetDraft(ctx, in.DraftID)
	if err != nil {
		return err
	}
	docxBytes, err := s.renderer.RenderDocx(ctx, draft.ContentBlocks)
	if err != nil {
		return err
	}

	prevVersionID, prevContentBlocks, _, err := s.repo.ArchivePreviousReleased(ctx, in.DocumentID)
	if err != nil {
		return err
	}

	if err := s.repo.PromoteDraftToReleased(ctx, in.DraftID, docxBytes, in.ApprovedBy); err != nil {
		return err
	}

	// Real diff computation
	var prevBlocks []any
	if len(prevContentBlocks) > 0 {
		var prevEnvelope map[string]any
		if err := json.Unmarshal(prevContentBlocks, &prevEnvelope); err == nil {
			prevBlocks, _ = prevEnvelope["blocks"].([]any)
		}
	}
	var currBlocks []any
	var currEnvelope map[string]any
	if err := json.Unmarshal(draft.ContentBlocks, &currEnvelope); err == nil {
		currBlocks, _ = currEnvelope["blocks"].([]any)
	}
	diff := mddm.ComputeDiff(prevBlocks, currBlocks)
	diffJSON, _ := json.Marshal(diff)
	if err := s.repo.StoreRevisionDiff(ctx, in.DraftID, diffJSON); err != nil {
		return err
	}

	if prevVersionID != uuid.Nil {
		if err := s.repo.DeleteImageRefs(ctx, prevVersionID); err != nil {
			return err
		}
	}

	return s.repo.CleanupOrphanImages(ctx)
}
```

- [ ] **Step 5: Update release_repo.go to return content_blocks from archive**

In `internal/modules/documents/infrastructure/postgres/release_repo.go`, update `ArchivePreviousReleased`:

```go
func (r *ReleaseRepo) ArchivePreviousReleased(ctx context.Context, documentID string) (uuid.UUID, []byte, []byte, error) {
	tx, err := r.beginOrReuseTx(ctx)
	if err != nil {
		return uuid.Nil, nil, nil, err
	}

	var prevID uuid.UUID
	var prevDocx []byte
	var prevContent []byte
	err = tx.QueryRowContext(ctx, `
		SELECT id, content_blocks, docx_bytes
		FROM metaldocs.document_versions_mddm
		WHERE document_id = $1 AND status = 'released'
	`, documentID).Scan(&prevID, &prevContent, &prevDocx)
	if errors.Is(err, sql.ErrNoRows) {
		return uuid.Nil, nil, nil, nil
	}
	if err != nil {
		return uuid.Nil, nil, nil, r.rollbackWithError(ctx, fmt.Errorf("archive previous released lookup for %s: %w", documentID, err))
	}

	if _, err := tx.ExecContext(ctx, `
		UPDATE metaldocs.document_versions_mddm
		SET status = 'archived', content_blocks = NULL
		WHERE id = $1
	`, prevID); err != nil {
		return uuid.Nil, nil, nil, r.rollbackWithError(ctx, fmt.Errorf("archive previous released %s: %w", prevID, err))
	}

	return prevID, prevContent, prevDocx, nil
}
```

- [ ] **Step 6: Update fake release repo in test to match new interface**

Update `internal/modules/documents/application/release_service_test.go`:

```go
func (f *fakeReleaseRepo) ArchivePreviousReleased(ctx context.Context, documentID string) (uuid.UUID, []byte, []byte, error) {
	f.steps = append(f.steps, "archive_previous")
	return uuid.New(), []byte(`{"mddm_version":1,"blocks":[],"template_ref":null}`), []byte("rendered"), nil
}
```

- [ ] **Step 7: Run all tests**

Run: `cd .worktrees/mddm-foundational && go test ./internal/modules/documents/... -v 2>&1 | tail -20`

Expected: ALL PASS.

- [ ] **Step 8: Commit**

```bash
git add internal/modules/documents/delivery/http/release_handler.go internal/modules/documents/delivery/http/release_handler_test.go internal/modules/documents/application/release_service.go internal/modules/documents/application/release_service_test.go internal/modules/documents/infrastructure/postgres/release_repo.go
git commit -m "fix(mddm): wire release handler to service with real diff computation"
```

---

# SHOULD FIX

## Task 4: Complete DOCX exporter for missing block types

**Files:**
- Modify: `apps/docgen/src/mddm/exporter.ts`
- Modify: `apps/docgen/__tests__/exporter.test.ts`

Missing from `renderBlock`: `richBlock`, `quote`, `image`. Also: `runToTextRun` does not render hyperlinks or document_ref.

- [ ] **Step 1: Add richBlock, quote, and image rendering to exporter**

In `apps/docgen/src/mddm/exporter.ts`, add these functions before `renderBlock`:

```typescript
function renderRichBlock(block: MDDMBlock, sectionPath: number[]): RenderedNode[] {
  const label = (block.props.label as string) ?? "";
  const out: RenderedNode[] = [
    new Paragraph({ children: [new TextRun({ text: label, bold: true })] }),
  ];
  for (const child of (block.children as MDDMBlock[]) ?? []) {
    out.push(...renderBlock(child, sectionPath));
  }
  return out;
}

function renderQuote(block: MDDMBlock): Paragraph[] {
  const paragraphs = (block.children as MDDMBlock[]) ?? [];
  return paragraphs.map((p) => {
    const runs = (p.children as InlineRun[] | undefined) ?? [];
    return new Paragraph({
      style: "Quote",
      children: runs.map(runToTextRun),
    });
  });
}

function renderImagePlaceholder(block: MDDMBlock): Paragraph {
  const alt = (block.props.alt as string) ?? "";
  const caption = (block.props.caption as string) ?? "";
  const text = caption ? `[Image: ${alt}] ${caption}` : `[Image: ${alt}]`;
  return new Paragraph({
    children: [new TextRun({ text, italics: true })],
  });
}
```

Note: Real image embedding requires an async `ImageFetcher` (render-image.ts has this). For the foundational sprint, use a placeholder that shows alt+caption. The async image fetch integration is deferred.

- [ ] **Step 2: Update runToTextRun to render hyperlinks**

Replace `runToTextRun` in `exporter.ts`:

```typescript
import { ExternalHyperlink } from "docx";

function runToDocxElement(run: InlineRun): TextRun | ExternalHyperlink {
  const marks = new Set(
    (run.marks ?? [])
      .filter((mark): mark is { type: string } => isObject(mark) && typeof mark.type === "string")
      .map((mark) => mark.type),
  );
  const textRun = new TextRun({
    text: run.text,
    bold: marks.has("bold"),
    italics: marks.has("italic"),
    underline: marks.has("underline") ? {} : undefined,
    strike: marks.has("strike"),
    style: run.link?.href ? "Hyperlink" : undefined,
  });

  if (run.link?.href) {
    return new ExternalHyperlink({
      children: [textRun],
      link: run.link.href,
    });
  }

  return textRun;
}
```

Update the `RenderedNode` type and any place that called `runToTextRun` to use `runToDocxElement` instead. Note: `ExternalHyperlink` is a valid child of `Paragraph` in the `docx` library.

- [ ] **Step 3: Wire new renderers into renderBlock switch**

In `renderBlock`, add these cases before the `default`:

```typescript
    case "richBlock":
      return renderRichBlock(block, sectionPath);
    case "quote":
      return renderQuote(block);
    case "image":
      return [renderImagePlaceholder(block)];
```

- [ ] **Step 4: Expand exporter test with all block types**

In `apps/docgen/__tests__/exporter.test.ts`, add a test that uses the full-block-types fixture:

```typescript
it("renders all 17 block types without throwing", async () => {
  const fixture = JSON.parse(
    readFileSync(join(__dirname, "../../shared/schemas/test-fixtures/valid/full-block-types.json"), "utf8"),
  );
  const result = await exportMDDMToDocx({
    envelope: fixture,
    metadata: {
      document_code: "PO-TEST",
      title: "Test Document",
      revision_label: "REV01",
      mode: "debug",
    },
  });
  expect(result).toBeInstanceOf(Uint8Array);
  expect(result.length).toBeGreaterThan(0);
});
```

- [ ] **Step 5: Run tests**

Run: `cd .worktrees/mddm-foundational/apps/docgen && npx vitest run`

Expected: ALL PASS.

- [ ] **Step 6: Commit**

```bash
git add apps/docgen/src/mddm/exporter.ts apps/docgen/__tests__/exporter.test.ts
git commit -m "fix(docgen): render richBlock, quote, image placeholder, and hyperlinks in MDDM exporter"
```

---

## Task 5: Tighten JSON Schema for Field children by valueMode

**Files:**
- Modify: `shared/schemas/mddm.schema.json`
- Create: `shared/schemas/test-fixtures/invalid/field-inline-with-blocks.json`
- Modify: `shared/schemas/__tests__/schema.test.ts` (exercised automatically via fixture dirs)

- [ ] **Step 1: Create invalid fixture — inline field with block children**

Create `shared/schemas/test-fixtures/invalid/field-inline-with-blocks.json`:

```json
{
  "mddm_version": 1,
  "template_ref": null,
  "blocks": [
    {
      "id": "11111111-1111-1111-1111-111111111111",
      "type": "field",
      "props": { "label": "Name", "valueMode": "inline", "locked": true },
      "children": [
        {
          "id": "22222222-2222-2222-2222-222222222222",
          "type": "paragraph",
          "props": {},
          "children": [{ "text": "not allowed inside inline field" }]
        }
      ]
    }
  ]
}
```

- [ ] **Step 2: Run tests to confirm it currently passes (incorrectly)**

Run: `cd .worktrees/mddm-foundational/shared/schemas && npx vitest run`

Expected: FAIL — the invalid fixture is incorrectly accepted.

- [ ] **Step 3: Add if/then/else to Field schema for valueMode discrimination**

In `shared/schemas/mddm.schema.json`, replace the `Field` definition:

```json
"Field": {
  "type": "object",
  "additionalProperties": false,
  "required": ["id", "type", "props", "children"],
  "properties": {
    "id": { "type": "string", "format": "uuid" },
    "template_block_id": { "type": "string", "format": "uuid" },
    "type": { "const": "field" },
    "props": {
      "type": "object",
      "additionalProperties": false,
      "required": ["label", "valueMode", "locked"],
      "properties": {
        "label": { "type": "string", "minLength": 1, "maxLength": 100 },
        "valueMode": { "enum": ["inline", "multiParagraph"] },
        "locked": { "type": "boolean" }
      }
    },
    "children": true
  },
  "if": {
    "properties": { "props": { "properties": { "valueMode": { "const": "inline" } } } }
  },
  "then": {
    "properties": { "children": { "$ref": "#/$defs/InlineContent" } }
  },
  "else": {
    "properties": {
      "children": {
        "type": "array",
        "items": {
          "oneOf": [
            { "$ref": "#/$defs/Paragraph" },
            { "$ref": "#/$defs/BulletListItem" },
            { "$ref": "#/$defs/NumberedListItem" },
            { "$ref": "#/$defs/Quote" },
            { "$ref": "#/$defs/Divider" }
          ]
        }
      }
    }
  }
}
```

- [ ] **Step 4: Run TS schema tests**

Run: `cd .worktrees/mddm-foundational/shared/schemas && npx vitest run`

Expected: ALL PASS — the invalid fixture is now correctly rejected.

- [ ] **Step 5: Run Go schema tests for parity**

Run: `cd .worktrees/mddm-foundational && go test -count=1 ./internal/modules/documents/domain/mddm/... -v`

Expected: ALL PASS — Go embeds the same schema file.

- [ ] **Step 6: Commit**

```bash
git add shared/schemas/mddm.schema.json shared/schemas/test-fixtures/invalid/field-inline-with-blocks.json
git commit -m "fix(mddm): tighten Field schema to constrain children by valueMode"
```

---

## Task 6: Add `field` to Go Layer 2 `isInlineParent` for cross-doc ref checking

**Files:**
- Modify: `internal/modules/documents/domain/mddm/rules.go`

The `isInlineParent` function in `rules.go:323` is missing `"field"`, which means cross-doc refs inside inline fields are not validated.

- [ ] **Step 1: Fix isInlineParent**

In `internal/modules/documents/domain/mddm/rules.go`, replace `isInlineParent`:

```go
func isInlineParent(t string) bool {
	switch t {
	case "paragraph", "heading", "bulletListItem", "numberedListItem", "dataTableCell", "field":
		return true
	}
	return false
}
```

- [ ] **Step 2: Run Go tests**

Run: `cd .worktrees/mddm-foundational && go test ./internal/modules/documents/domain/mddm/... -v`

Expected: ALL PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/modules/documents/domain/mddm/rules.go
git commit -m "fix(mddm): add field to isInlineParent for cross-doc ref validation"
```

---

## Task 7: Run full verification matrix

**Files:** None (verification only)

- [ ] **Step 1: Run frontend build**

Run: `cd .worktrees/mddm-foundational/frontend/apps/web && npx vite build`

Expected: PASS (no build errors).

- [ ] **Step 2: Run adapter tests**

Run: `cd .worktrees/mddm-foundational/frontend/apps/web && npx vitest run src/features/documents/mddm-editor/__tests__/adapter.test.ts`

Expected: 7/7 PASS.

- [ ] **Step 3: Run shared schema tests**

Run: `cd .worktrees/mddm-foundational/shared/schemas && npx vitest run`

Expected: ALL PASS (should be 10+ tests now).

- [ ] **Step 4: Run Go MDDM domain tests**

Run: `cd .worktrees/mddm-foundational && go test -count=1 ./internal/modules/documents/domain/mddm/... -v`

Expected: ALL PASS.

- [ ] **Step 5: Run Go application layer tests**

Run: `cd .worktrees/mddm-foundational && go test -count=1 ./internal/modules/documents/application/... -v`

Expected: ALL PASS.

- [ ] **Step 6: Run docgen tests**

Run: `cd .worktrees/mddm-foundational/apps/docgen && npx vitest run`

Expected: ALL PASS.

- [ ] **Step 7: Commit all uncommitted audit remediation fixes (from prior session)**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/adapter.ts frontend/apps/web/src/features/documents/mddm-editor/blocks/Repeatable.tsx frontend/apps/web/src/features/documents/mddm-editor/__tests__/adapter.test.ts frontend/apps/web/src/features/documents/mddm-editor/schema.ts shared/schemas/canonicalize.ts shared/schemas/mddm.schema.json
git commit -m "fix(mddm): audit remediation — heading clamp, maxItems default, canonicalize parity, schema registration"
```
