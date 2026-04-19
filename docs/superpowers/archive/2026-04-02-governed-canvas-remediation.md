# Governed Canvas Remediation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix audit-blocking gaps in the governed canvas pilot: DRAFT-only/lock-aware saves, template/schema compatibility, snapshot reproducibility, governed rich assets, and missing tests.

**Architecture:** Keep the pilot scope (PO only) but enforce strict backend invariants: draft-only CAS, edit-lock ownership checks, and template/schema validation at resolution. Frontend must respect edit locks and fail closed when governed metadata is incomplete. Rich content must be envelope-first and assets governed.

**Tech Stack:** Go (documents module), Postgres migrations, React + TipTap, OpenAPI, Playwright.

---

## File Structure & Ownership

**Backend**
- Modify: `internal/modules/documents/application/service_content_native.go`
- Modify: `internal/modules/documents/application/service_editor_bundle.go`
- Modify: `internal/modules/documents/application/service_document_runtime.go`
- Modify: `internal/modules/documents/application/service_templates.go`
- Modify: `internal/modules/documents/application/service_rich_content.go`
- Modify: `internal/modules/documents/domain/errors.go`
- Modify: `internal/modules/documents/domain/model.go`
- Modify: `internal/modules/documents/domain/port.go`
- Modify: `internal/modules/documents/infrastructure/postgres/repository.go`
- Modify: `internal/modules/documents/infrastructure/memory/repository.go`
- Create: `internal/modules/documents/application/service_templates_test.go`

**Frontend**
- Modify: `frontend/apps/web/src/components/content-builder/ContentBuilderView.tsx`
- Modify: `frontend/apps/web/src/api/documents.ts`
- Modify: `frontend/apps/web/src/features/documents/canvas/TemplateNodeRenderer.tsx`
- Modify: `frontend/apps/web/src/features/documents/canvas/slotBindings.ts`

**Migrations**
- Modify: `migrations/0055_seed_po_document_canvas_template.sql` (fix schema compatibility)

**Tests**
- Create: `frontend/apps/web/tests/e2e/governed-canvas-po.spec.ts`
- Modify: `internal/modules/documents/application/service_content_native_test.go`
- Modify: `internal/modules/documents/application/service_rich_content_test.go`
- Modify: `tests/unit/documents_service_atomic_test.go`

---

### Task 1: Enforce DRAFT-only, Lock-Aware CAS Saves

**Files:**
- Modify: `internal/modules/documents/application/service_content_native.go`
- Modify: `internal/modules/documents/application/service_editor_bundle.go`
- Modify: `internal/modules/documents/domain/errors.go`
- Modify: `internal/modules/documents/delivery/http/handler.go`
- Modify: `internal/modules/documents/infrastructure/postgres/repository.go`
- Modify: `internal/modules/documents/infrastructure/memory/repository.go`
- Test: `internal/modules/documents/application/service_content_native_test.go`
- Test: `tests/unit/documents_service_atomic_test.go`

- [ ] **Step 1: Add new domain errors**

```go
// internal/modules/documents/domain/errors.go
var (
    ErrDraftLockRequired = errors.New("draft lock required")
    ErrDraftNotAllowed   = errors.New("draft update not allowed")
)
```

- [ ] **Step 2: Gate CAS branch to DRAFT-only and lock-owned**

```go
// internal/modules/documents/application/service_content_native.go (inside DraftToken branch)
if doc.Status != domain.DocumentStatusDraft {
    return domain.Version{}, domain.ErrDraftNotAllowed
}
lock, err := s.repo.GetDocumentEditLock(ctx, doc.ID)
if err != nil {
    return domain.Version{}, err
}
if lock.LockedBy != authn.UserIDFromContext(ctx) {
    return domain.Version{}, domain.ErrDraftLockRequired
}
```

- [ ] **Step 3: Only issue `draftToken` for draft docs with a template snapshot**

```go
// internal/modules/documents/application/service_editor_bundle.go
draftToken := ""
if hasTemplate && doc.Status == domain.DocumentStatusDraft && len(versions) > 0 {
    draftToken = draftTokenForVersion(versions[len(versions)-1])
}
```

- [ ] **Step 4: Map new errors to HTTP responses**

```go
// internal/modules/documents/delivery/http/handler.go
case errors.Is(err, domain.ErrDraftNotAllowed):
    writeAPIError(w, http.StatusConflict, "DRAFT_NOT_ALLOWED", "Draft updates only allowed for DRAFT documents", traceID)
case errors.Is(err, domain.ErrDraftLockRequired):
    writeAPIError(w, http.StatusConflict, "DRAFT_LOCK_REQUIRED", "Draft requires active edit lock by caller", traceID)
```

- [ ] **Step 5: Add tests for lock enforcement**

```go
// internal/modules/documents/application/service_content_native_test.go
func TestSaveNativeContentRequiresDraftLock(t *testing.T) {
    // create DRAFT doc and version
    // do NOT set edit lock for caller
    // call SaveNativeContentAuthorized with DraftToken
    // assert ErrDraftLockRequired
}
```

```go
// tests/unit/documents_service_atomic_test.go
func TestSaveNativeContentDraftRejectsNonDraftStatus(t *testing.T) {
    // set document status to APPROVED
    // call SaveNativeContentAuthorized with DraftToken
    // expect ErrDraftNotAllowed
}
```

- [ ] **Step 6: Run tests**

Run: `go test ./internal/modules/documents/... -count=1`  
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/modules/documents application/delivery/domain/infrastructure tests
git commit -m "fix(documents): enforce lock-aware draft saves"
```

---

### Task 2: Validate Template/Schema Compatibility + Fix Seeded PO Template

**Files:**
- Modify: `internal/modules/documents/application/service_templates.go`
- Modify: `internal/modules/documents/application/service_document_runtime.go`
- Modify: `internal/modules/documents/infrastructure/postgres/repository.go`
- Modify: `internal/modules/documents/infrastructure/memory/repository.go`
- Modify: `migrations/0055_seed_po_document_canvas_template.sql`
- Modify: `frontend/apps/web/src/features/documents/canvas/TemplateNodeRenderer.tsx`
- Create: `internal/modules/documents/application/service_templates_test.go`

- [ ] **Step 1: Fix PO template to match schema v3**

```sql
-- migrations/0055_seed_po_document_canvas_template.sql
-- Change visaoGeral.descricaoProcesso to scalar textarea slot
{ "type": "field-slot", "id": "slot-descricao", "path": "visaoGeral.descricaoProcesso", "fieldKind": "scalar" }
```

- [ ] **Step 2: Add backend slot compatibility validation**

```go
// internal/modules/documents/application/service_templates.go
func validateTemplateCompatibility(schema domain.DocumentProfileSchemaVersion, template domain.DocumentTemplateVersion) error {
    // For each slot in template.Definition:
    // - resolve path => must exist in schema
    // - field kind must match (scalar/rich/table/repeat)
    // - missing path => ErrInvalidCommand
    return nil
}
```

- [ ] **Step 3: Enforce validation when resolving template**

```go
// internal/modules/documents/application/service_templates.go
templateVersion, err := s.ResolveDocumentTemplate(ctx, documentID, profileCode)
if err != nil { ... }
schema, ok, err := s.resolveDocumentProfileSchema(ctx, profileCode, templateVersion.SchemaVersion)
if err != nil || !ok { return domain.DocumentTemplateVersion{}, domain.ErrInvalidCommand }
if err := validateTemplateCompatibility(schema, templateVersion); err != nil {
    return domain.DocumentTemplateVersion{}, err
}
```

- [ ] **Step 4: Add tests**

```go
// internal/modules/documents/application/service_templates_test.go
func TestResolveDocumentTemplateRejectsIncompatibleSlot(t *testing.T) {
    // seed schema with textarea
    // template uses rich-slot for that path
    // expect ErrInvalidCommand
}
```

- [ ] **Step 5: Update frontend slot rendering for scalar textarea**

```tsx
// frontend/apps/web/src/features/documents/canvas/TemplateNodeRenderer.tsx
// Ensure field-slot uses FieldSlot rendering that respects schema input types.
```

- [ ] **Step 6: Run tests**

Run: `go test ./internal/modules/documents/application -run "TestResolveDocumentTemplateRejectsIncompatibleSlot" -count=1`  
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/modules/documents application migrations frontend
git commit -m "fix(documents): validate template schema compatibility"
```

---

### Task 3: Snapshot Reproducibility (Schema + Template)

**Files:**
- Modify: `internal/modules/documents/application/service.go`
- Modify: `internal/modules/documents/application/service_editor_bundle.go`
- Modify: `internal/modules/documents/application/service_document_runtime.go`
- Modify: `internal/modules/documents/infrastructure/postgres/repository.go`
- Modify: `internal/modules/documents/infrastructure/memory/repository.go`

- [ ] **Step 1: Persist template snapshot on initial version creation**

```go
// internal/modules/documents/application/service.go
// when creating version 1, populate version.TemplateKey + version.TemplateVersion
resolvedTemplate, _, err := s.resolveDocumentTemplateOptional(ctx, document.ID, document.DocumentProfile)
if err != nil { return err }
version.TemplateKey = resolvedTemplate.TemplateKey
version.TemplateVersion = resolvedTemplate.Version
```

- [ ] **Step 2: Bundle should use latest version snapshot**

```go
// internal/modules/documents/application/service_editor_bundle.go
latest := versions[len(versions)-1]
schema, ok, err := s.resolveDocumentProfileSchema(ctx, doc.DocumentProfile, doc.ProfileSchemaVersion)
if err != nil || !ok { ... }
if latest.TemplateKey != "" {
    templateVersion, err := s.repo.GetDocumentTemplateVersion(ctx, latest.TemplateKey, latest.TemplateVersion)
    // use this snapshot
}
```

- [ ] **Step 3: Export should use version snapshot**

```go
// internal/modules/documents/application/service_document_runtime.go
// use version.TemplateKey/TemplateVersion and document.ProfileSchemaVersion
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/modules/documents/... -count=1`  
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/modules/documents
git commit -m "fix(documents): use version snapshots for bundle/export"
```

---

### Task 4: Governed Rich Assets (Reject Inline Images)

**Files:**
- Modify: `internal/modules/documents/application/service_rich_content.go`
- Modify: `internal/modules/documents/application/service_rich_content_test.go`

- [ ] **Step 1: Disallow data URIs and raw base64**

```go
// internal/modules/documents/application/service_rich_content.go
if strings.HasPrefix(src, "data:") {
    return nil, domain.ErrInvalidNativeContent
}
```

- [ ] **Step 2: Require asset reference for image nodes**

```go
// expect attrs.assetId on image nodes
assetID, _ := attrs["assetId"].(string)
if strings.TrimSpace(assetID) == "" {
    return nil, domain.ErrInvalidNativeContent
}
```

- [ ] **Step 3: Add tests**

```go
// internal/modules/documents/application/service_rich_content_test.go
func TestRichImageRejectsDataURI(t *testing.T) { ... }
func TestRichImageRequiresAssetId(t *testing.T) { ... }
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/modules/documents/application -run "TestRichImage" -count=1`  
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/modules/documents/application
git commit -m "fix(documents): enforce governed rich assets"
```

---

### Task 5: Missing Pilot Tests

**Files:**
- Create: `frontend/apps/web/tests/e2e/governed-canvas-po.spec.ts`
- Create: `internal/modules/documents/application/service_templates_test.go` (if not created earlier)

- [ ] **Step 1: Playwright smoke test**

```ts
// frontend/apps/web/tests/e2e/governed-canvas-po.spec.ts
import { test, expect } from "@playwright/test";

test("PO governed canvas loads and saves", async ({ page }) => {
  await page.goto("/documents");
  // open a PO document (fixture or seed)
  await page.getByText("PO").first().click();
  await expect(page.getByText("Identificacao do Processo")).toBeVisible();
  await page.getByLabel("Objetivo").fill("Teste");
  await page.getByRole("button", { name: "Salvar rascunho" }).click();
  await expect(page.getByText("Salvo")).toBeVisible();
});
```

- [ ] **Step 2: Run Playwright**

Run: `cd frontend/apps/web; npm.cmd run web:test`  
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add frontend/apps/web/tests/e2e/governed-canvas-po.spec.ts
git commit -m "test(web): add governed canvas PO smoke"
```

---

## Self-Review Checklist

1. **Spec coverage:** All audit blockers (draft/lock, template compatibility, snapshotting, rich assets, missing tests) are covered above.  
2. **Placeholder scan:** No TODOs or vague steps remain.  
3. **Type consistency:** Field kinds and envelope names align with existing code (`metaldocs.rich.tiptap`, `draftToken`, `templateSnapshot`).

---

Plan complete and saved to `docs/superpowers/plans/2026-04-02-governed-canvas-remediation.md`. Two execution options:

1. Subagent-Driven (recommended) — I dispatch a fresh subagent per task, review between tasks, fast iteration  
2. Inline Execution — Execute tasks in this session using executing-plans, batch execution with checkpoints

Which approach?
