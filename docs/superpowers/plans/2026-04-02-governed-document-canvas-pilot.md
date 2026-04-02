# Governed Document Canvas Pilot Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship the first governed document canvas for `PO` documents with server-created drafts, versioned template snapshots, MetalDocs-owned rich content envelopes, single-writer draft safety, and end-to-end save -> DOCX -> PDF using the existing docgen/Gotenberg pipeline.

**Architecture:** This plan deliberately implements one vertical slice, not the whole future platform. The backend remains authoritative for template resolution, rich-content validation, draft freshness, lock enforcement, and export projection. The frontend replaces the current form-like runtime editor with a template-driven document canvas for one seeded `PO` template while keeping generated PDF as the final visual truth.

**Tech Stack:** Go 1.24, PostgreSQL migrations, OpenAPI v1 YAML, React 18 + TypeScript, TipTap, CSS Modules, docgen, Gotenberg, Playwright smoke coverage.

---

## Scope Check

This spec could explode into multiple subsystems if implemented naively. This plan intentionally covers only the pilot slice:

- 1 profile: `po`
- 1 active schema version: current runtime `PO` schema
- 1 template version: seeded default governed canvas template
- 1 editor surface: governed canvas inside the existing content builder
- 1 save path: existing native content path, upgraded for in-place `DRAFT` mutation and rich-envelope handling

This plan explicitly does **not** add:

- template management UI
- collaborative co-editing
- slot-level ACL
- conditional template rendering
- generalized multi-profile rollout

Those become follow-up plans after this pilot is stable.

---

## File Structure

| File | Action | Responsibility |
|------|--------|----------------|
| `docs/adr/0021-governed-document-canvas-pilot.md` | Create | Record the architecture and v1 constraints for the pilot slice |
| `api/openapi/v1/openapi.yaml` | Modify | Extend editor-bundle and native save contracts with template snapshot + draft token |
| `migrations/0054_init_document_templates.sql` | Create | Add template storage and revision snapshot columns |
| `migrations/0055_seed_po_document_canvas_template.sql` | Create | Seed the first `PO` default template version |
| `internal/modules/documents/domain/template.go` | Create | Template snapshot, default binding, document override, template DSL structs |
| `internal/modules/documents/domain/errors.go` | Modify | Add template/rich/draft conflict errors |
| `internal/modules/documents/domain/model.go` | Modify | Extend `Version` with template snapshot metadata and rich-envelope-aware values |
| `internal/modules/documents/domain/port.go` | Modify | Add template lookup/binding methods and compare-and-set draft update method |
| `internal/modules/documents/application/service_templates.go` | Create | Resolve profile default vs document override template |
| `internal/modules/documents/application/service_templates_test.go` | Create | Verify template resolution and snapshot rules |
| `internal/modules/documents/application/service_editor_bundle.go` | Modify | Include resolved template snapshot and draft token in editor bundle |
| `internal/modules/documents/application/service_content_native.go` | Modify | Mutate current `DRAFT` version in place with stale-write rejection and lock enforcement |
| `internal/modules/documents/application/service_runtime_validation.go` | Modify | Validate MetalDocs rich envelope and reject legacy naked HTML for pilot saves |
| `internal/modules/documents/application/service_rich_content.go` | Create | Convert TipTap envelope -> docgen rich blocks, resolve governed image assets |
| `internal/modules/documents/application/service_rich_content_test.go` | Create | Validate envelope parsing, asset resolution, and docgen projection |
| `internal/modules/documents/application/service_document_runtime.go` | Modify | Project envelope-backed rich fields before calling docgen |
| `internal/modules/documents/infrastructure/postgres/repository.go` | Modify | Persist templates, template snapshots, and compare-and-set draft updates |
| `internal/modules/documents/infrastructure/memory/repository.go` | Modify | Keep memory repository aligned with new repository contract |
| `internal/modules/documents/delivery/http/handler.go` | Modify | Thread new editor bundle/save fields through HTTP responses |
| `frontend/apps/web/src/lib.types.ts` | Modify | Extend editor bundle/save response types with template snapshot + draft token |
| `frontend/apps/web/src/api/documents.ts` | Modify | Normalize template snapshot and send `draftToken` on native saves |
| `frontend/apps/web/src/features/documents/canvas/templateTypes.ts` | Create | Frontend template DSL types |
| `frontend/apps/web/src/features/documents/canvas/templateAdapters.ts` | Create | Normalize backend template snapshot into frontend types |
| `frontend/apps/web/src/features/documents/canvas/DocumentCanvas.tsx` | Create | Render the governed document canvas |
| `frontend/apps/web/src/features/documents/canvas/TemplateNodeRenderer.tsx` | Create | Recursive renderer for template nodes |
| `frontend/apps/web/src/features/documents/canvas/slots/FieldSlot.tsx` | Create | Inline scalar slot editor |
| `frontend/apps/web/src/features/documents/canvas/slots/RichSlot.tsx` | Create | TipTap-powered rich slot editor backed by MetalDocs envelope |
| `frontend/apps/web/src/features/documents/canvas/slots/TableSlot.tsx` | Create | Governed table slot editor |
| `frontend/apps/web/src/features/documents/canvas/slots/RepeatSlot.tsx` | Create | Governed repeat slot editor |
| `frontend/apps/web/src/features/documents/canvas/rich/metaldocsRich.ts` | Create | Convert TipTap JSON <-> MetalDocs rich envelope |
| `frontend/apps/web/src/features/documents/canvas/DocumentCanvas.module.css` | Create | Canvas styling for the authoring surface |
| `frontend/apps/web/src/components/content-builder/ContentBuilderView.tsx` | Modify | Replace `DynamicEditor` with `DocumentCanvas`, manage draft token + lock state |
| `frontend/apps/web/tests/e2e/governed-canvas-po.spec.ts` | Create | Smoke the pilot flow end to end |

---

### Task 1: Record the architecture decision and publish the pilot API contract

**Files:**
- Create: `docs/adr/0021-governed-document-canvas-pilot.md`
- Modify: `api/openapi/v1/openapi.yaml`

- [ ] **Step 1: Write the ADR before touching runtime code**

Create `docs/adr/0021-governed-document-canvas-pilot.md` with this structure:

```md
# ADR 0021: Governed Document Canvas Pilot

## Status
Accepted

## Context
- `docs/superpowers/specs/2026-04-02-governed-document-canvas-design.md` defines the target model.
- The current repo already has runtime schema, editor bundle, docgen, and Gotenberg.
- The pilot must stay narrow: one `PO` template, no template admin UI, no collaborative editing.

## Decision
- Reuse the existing `/documents/{documentId}/editor-bundle` endpoint and extend it with:
  - `templateSnapshot`
  - `draftToken`
- Reuse the existing native save path and extend it with `draftToken`.
- Persist template snapshot metadata on `document_versions`.
- Treat MetalDocs rich envelope as the canonical persisted shape for `rich` fields.
- Keep PDF as the final visual truth.

## Consequences
- Existing runtime editor code becomes transitional and will be bypassed by the pilot canvas.
- Saves to `DRAFT` become compare-and-set mutations instead of always creating a new version.
- Template management UI is deferred.
```

- [ ] **Step 2: Extend the editor-bundle and native-save schemas in OpenAPI**

In `api/openapi/v1/openapi.yaml`, update the existing editor-bundle and native-content components instead of adding parallel endpoints.

Add fields like:

```yaml
    DocumentEditorBundleResponse:
      type: object
      required: [document, versions, schema, governance, presence]
      properties:
        draftToken:
          type: string
        templateSnapshot:
          $ref: '#/components/schemas/DocumentTemplateSnapshotResponse'
```

These fields are additive and pilot-slice only: generic editor bundles may omit them, but governed-canvas-supported documents/profiles should return them, starting with the PO pilot slice.

Add the new snapshot schema:

```yaml
    DocumentTemplateSnapshotResponse:
      type: object
      required: [templateKey, version, profileCode, schemaVersion, definition]
      description: Snapshot da template DSL pilotada, limitada ao subconjunto MetalDocs do pilot.
      properties:
        templateKey:
          type: string
        version:
          type: integer
        profileCode:
          type: string
        schemaVersion:
          type: integer
        definition:
          $ref: '#/components/schemas/DocumentTemplateNodeResponse'

    DocumentTemplateNodeResponse:
      oneOf:
        - $ref: '#/components/schemas/DocumentTemplatePageNodeResponse'
        - $ref: '#/components/schemas/DocumentTemplateSectionFrameNodeResponse'
        - $ref: '#/components/schemas/DocumentTemplateLabelNodeResponse'
        - $ref: '#/components/schemas/DocumentTemplateFieldSlotNodeResponse'
        - $ref: '#/components/schemas/DocumentTemplateRichSlotNodeResponse'
        - $ref: '#/components/schemas/DocumentTemplateRepeatSlotNodeResponse'
        - $ref: '#/components/schemas/DocumentTemplateTableSlotNodeResponse'
```

Update the native save request/response:

```yaml
    DocumentContentSaveRequest:
      type: object
      required: [content]
      properties:
        draftToken:
          type: string
        content:
          type: object
          additionalProperties: true
```

Legacy native-save callers may omit `draftToken`; governed canvas clients must send it and receive a refreshed token on success.

- [ ] **Step 3: Verify the contract diff is limited to the pilot path**

Run:

```bash
rg -n "draftToken|templateSnapshot|DocumentTemplateSnapshotResponse" api/openapi/v1/openapi.yaml
```

Expected: only the existing editor bundle/native save schemas and the new pilot-subset template node schema appear.

- [ ] **Step 4: Commit**

```bash
git add docs/adr/0021-governed-document-canvas-pilot.md api/openapi/v1/openapi.yaml
git commit -m "docs(canvas): record pilot adr and api contract"
```

---

### Task 2: Add template persistence and revision snapshot storage

**Files:**
- Create: `migrations/0054_init_document_templates.sql`
- Create: `migrations/0055_seed_po_document_canvas_template.sql`
- Create: `internal/modules/documents/domain/template.go`
- Modify: `internal/modules/documents/domain/model.go`
- Modify: `internal/modules/documents/domain/errors.go`
- Modify: `internal/modules/documents/domain/port.go`
- Modify: `internal/modules/documents/infrastructure/postgres/repository.go`
- Modify: `internal/modules/documents/infrastructure/memory/repository.go`
- Test: `internal/modules/documents/application/service_templates_test.go`

- [ ] **Step 1: Create the migration for template tables and revision snapshot columns**

Create `migrations/0054_init_document_templates.sql`:

```sql
CREATE TABLE IF NOT EXISTS metaldocs.document_template_versions (
  template_key TEXT NOT NULL,
  version INTEGER NOT NULL,
  profile_code TEXT NOT NULL,
  schema_version INTEGER NOT NULL,
  name TEXT NOT NULL,
  definition_json JSONB NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
  PRIMARY KEY (template_key, version)
);

CREATE TABLE IF NOT EXISTS metaldocs.document_profile_template_defaults (
  profile_code TEXT PRIMARY KEY,
  template_key TEXT NOT NULL,
  template_version INTEGER NOT NULL,
  assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS metaldocs.document_template_assignments (
  document_id TEXT PRIMARY KEY REFERENCES metaldocs.documents(id) ON DELETE CASCADE,
  template_key TEXT NOT NULL,
  template_version INTEGER NOT NULL,
  assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE metaldocs.document_versions
  ADD COLUMN IF NOT EXISTS template_key TEXT,
  ADD COLUMN IF NOT EXISTS template_version INTEGER;
```

- [ ] **Step 2: Seed the first `PO` default template**

Create `migrations/0055_seed_po_document_canvas_template.sql`:

```sql
INSERT INTO metaldocs.document_template_versions (
  template_key, version, profile_code, schema_version, name, definition_json
)
VALUES (
  'po-default-canvas',
  1,
  'po',
  3,
  'PO Governed Canvas v1',
  '{
    "type": "page",
    "id": "po-root",
    "children": [
      {
        "type": "section-frame",
        "id": "identificacao-processo",
        "title": "Identificacao do Processo",
        "children": [
          { "type": "label", "id": "lbl-objetivo", "text": "Objetivo" },
          { "type": "field-slot", "id": "slot-objetivo", "path": "identificacaoProcesso.objetivo", "fieldKind": "scalar" },
          { "type": "label", "id": "lbl-descricao", "text": "Descricao do processo" },
          { "type": "rich-slot", "id": "slot-descricao", "path": "visaoGeral.descricaoProcesso", "fieldKind": "rich" }
        ]
      }
    ]
  }'::jsonb
);

INSERT INTO metaldocs.document_profile_template_defaults (profile_code, template_key, template_version)
VALUES ('po', 'po-default-canvas', 1)
ON CONFLICT (profile_code) DO UPDATE
SET template_key = EXCLUDED.template_key,
    template_version = EXCLUDED.template_version,
    assigned_at = NOW();
```

If the active `PO` schema version differs in the target database, update `schema_version` in this migration before applying it.

- [ ] **Step 3: Add domain models and repository methods**

Create `internal/modules/documents/domain/template.go`:

```go
package domain

import "time"

type DocumentTemplateVersion struct {
	TemplateKey   string
	Version       int
	ProfileCode   string
	SchemaVersion int
	Name          string
	Definition    map[string]any
	CreatedAt     time.Time
}

type DocumentTemplateAssignment struct {
	DocumentID      string
	TemplateKey     string
	TemplateVersion int
	AssignedAt      time.Time
}

type DocumentTemplateSnapshot struct {
	TemplateKey   string
	Version       int
	ProfileCode   string
	SchemaVersion int
	Definition    map[string]any
}
```

Then extend `internal/modules/documents/domain/model.go`:

```go
type Version struct {
	DocumentID       string
	Number           int
	Content          string
	ContentHash      string
	ChangeSummary    string
	ContentSource    string
	NativeContent    DocumentValues
	Values           DocumentValues
	BodyBlocks       []EtapaBody
	DocxStorageKey   string
	PdfStorageKey    string
	TextContent      string
	FileSizeBytes    int64
	OriginalFilename string
	PageCount        int
	TemplateKey      string
	TemplateVersion  int
	CreatedAt        time.Time
}

type SaveNativeContentCommand struct {
	DocumentID string
	DraftToken string
	Content    map[string]any
	TraceID    string
}
```

And extend `internal/modules/documents/domain/port.go` with:

```go
	GetDocumentTemplateVersion(ctx context.Context, templateKey string, version int) (DocumentTemplateVersion, error)
	GetDefaultDocumentTemplate(ctx context.Context, profileCode string) (DocumentTemplateVersion, error)
	GetDocumentTemplateAssignment(ctx context.Context, documentID string) (DocumentTemplateAssignment, error)
	UpsertDocumentTemplateAssignment(ctx context.Context, item DocumentTemplateAssignment) error
	UpdateDraftVersionContentCAS(ctx context.Context, version Version, expectedContentHash string) error
```

- [ ] **Step 4: Write the failing backend test for template resolution precedence**

Create `internal/modules/documents/application/service_templates_test.go`:

```go
func TestResolveDocumentTemplatePrefersDocumentAssignmentOverProfileDefault(t *testing.T) {
	repo := memory.NewRepository()
	svc := application.NewService(repo, nil, nil, nil, clockwork.NewFakeClock())

	if err := repo.UpsertDocumentTemplateAssignment(context.Background(), domain.DocumentTemplateAssignment{
		DocumentID:      "doc-1",
		TemplateKey:     "po-doc-special",
		TemplateVersion: 2,
	}); err != nil {
		t.Fatal(err)
	}

	got, err := svc.ResolveDocumentTemplate(context.Background(), "doc-1", "po")
	if err != nil {
		t.Fatal(err)
	}
	if got.TemplateKey != "po-doc-special" || got.Version != 2 {
		t.Fatalf("resolved template = %+v, want po-doc-special v2", got)
	}
}
```

- [ ] **Step 5: Implement repository persistence and run the targeted tests**

Run:

```bash
go test ./internal/modules/documents/application -run "TestResolveDocumentTemplatePrefersDocumentAssignmentOverProfileDefault" -count=1
```

Expected: pass after the domain/repository/service wiring is complete.

- [ ] **Step 6: Commit**

```bash
git add migrations/0054_init_document_templates.sql migrations/0055_seed_po_document_canvas_template.sql internal/modules/documents/domain/template.go internal/modules/documents/domain/model.go internal/modules/documents/domain/errors.go internal/modules/documents/domain/port.go internal/modules/documents/infrastructure/postgres/repository.go internal/modules/documents/infrastructure/memory/repository.go internal/modules/documents/application/service_templates_test.go
git commit -m "feat(documents): add template persistence and snapshot storage"
```

---

### Task 3: Resolve templates in the editor bundle and make `DRAFT` saves conflict-safe

**Files:**
- Create: `internal/modules/documents/application/service_templates.go`
- Modify: `internal/modules/documents/application/service_editor_bundle.go`
- Modify: `internal/modules/documents/application/service_content_native.go`
- Modify: `internal/modules/documents/delivery/http/handler.go`
- Test: `internal/modules/documents/application/service_content_native_test.go`

- [ ] **Step 1: Add a template resolver service**

Create `internal/modules/documents/application/service_templates.go`:

```go
package application

import (
	"context"
	"metaldocs/internal/modules/documents/domain"
)

func (s *Service) ResolveDocumentTemplate(ctx context.Context, documentID, profileCode string) (domain.DocumentTemplateVersion, error) {
	if assignment, err := s.repo.GetDocumentTemplateAssignment(ctx, documentID); err == nil {
		return s.repo.GetDocumentTemplateVersion(ctx, assignment.TemplateKey, assignment.TemplateVersion)
	}
	return s.repo.GetDefaultDocumentTemplate(ctx, profileCode)
}
```

- [ ] **Step 2: Extend `GetDocumentEditorBundle` with template snapshot and draft token**

Update `internal/modules/documents/application/service_editor_bundle.go`:

```go
type DocumentEditorBundle struct {
	Document         domain.Document
	Versions         []domain.Version
	Schema           domain.DocumentProfileSchemaVersion
	Governance       domain.DocumentProfileGovernance
	TemplateSnapshot domain.DocumentTemplateSnapshot
	DraftToken       string
	Presence         []domain.CollaborationPresence
	EditLock         *domain.DocumentEditLock
}
```

When loading the bundle:

```go
templateVersion, err := s.ResolveDocumentTemplate(ctx, doc.ID, doc.DocumentProfile)
if err != nil {
	return DocumentEditorBundle{}, err
}

latest := versions[len(versions)-1]
draftToken := fmt.Sprintf("v%d:%s", latest.Number, latest.ContentHash)
```

For `templateSnapshot`, copy the template version data into the bundle without lookup-by-latest behavior.

- [ ] **Step 3: Rewrite native save to mutate the current `DRAFT` version in place**

In `internal/modules/documents/application/service_content_native.go`, stop calling `NextVersionNumber` for every autosave. Instead:

```go
current, err := s.latestVersion(ctx, doc.ID)
if err != nil {
	return domain.Version{}, err
}
if doc.Status != domain.StatusDraft {
	return domain.Version{}, domain.ErrVersioningNotAllowed
}
expectedHash := current.ContentHash
if !matchesDraftToken(cmd.DraftToken, current) {
	return domain.Version{}, domain.ErrDraftConflict
}

current.Content = contentText
current.ContentHash = contentHash(contentText)
current.NativeContent = contentPayload
current.Values = contentPayload
current.TextContent = contentText
current.TemplateKey = templateVersion.TemplateKey
current.TemplateVersion = templateVersion.Version
```

Add the helper in the same file:

```go
func matchesDraftToken(token string, version domain.Version) bool {
	return strings.TrimSpace(token) == fmt.Sprintf("v%d:%s", version.Number, version.ContentHash)
}
```

Persist with a compare-and-set repository method:

```go
if err := s.repo.UpdateDraftVersionContentCAS(ctx, current, expectedHash); err != nil {
	return domain.Version{}, err
}
```

Keep the existing docx/pdf regeneration behavior, but overwrite the current draft version’s storage keys instead of creating a new version row.

- [ ] **Step 4: Add the failing stale-write test before implementing the compare-and-set update**

Create `internal/modules/documents/application/service_content_native_test.go`:

```go
func TestSaveNativeContentRejectsStaleDraftToken(t *testing.T) {
	repo := memory.NewRepository()
	svc := newDocumentsServiceForTests(repo)

	_, err := svc.SaveNativeContentAuthorized(ctxWithEditor(), domain.SaveNativeContentCommand{
		DocumentID: "doc-1",
		DraftToken: "v1:stale",
		Content: map[string]any{
			"identificacaoProcesso": map[string]any{"objetivo": "novo objetivo"},
		},
		TraceID: "trace-stale",
	})

	if !errors.Is(err, domain.ErrDraftConflict) {
		t.Fatalf("err = %v, want ErrDraftConflict", err)
	}
}
```

- [ ] **Step 5: Thread the new fields through HTTP and run the targeted backend tests**

Update `internal/modules/documents/delivery/http/handler.go` response mapping to include:

```go
TemplateSnapshot: DocumentTemplateSnapshotResponse{
	TemplateKey:   bundle.TemplateSnapshot.TemplateKey,
	Version:       bundle.TemplateSnapshot.Version,
	ProfileCode:   bundle.TemplateSnapshot.ProfileCode,
	SchemaVersion: bundle.TemplateSnapshot.SchemaVersion,
	Definition:    bundle.TemplateSnapshot.Definition,
},
DraftToken: bundle.DraftToken,
```

Run:

```bash
go test ./internal/modules/documents/application -run "Test(SaveNativeContentRejectsStaleDraftToken|ResolveDocumentTemplatePrefersDocumentAssignmentOverProfileDefault)" -count=1
```

Expected: both tests pass.

- [ ] **Step 6: Commit**

```bash
git add internal/modules/documents/application/service_templates.go internal/modules/documents/application/service_editor_bundle.go internal/modules/documents/application/service_content_native.go internal/modules/documents/delivery/http/handler.go internal/modules/documents/application/service_content_native_test.go
git commit -m "feat(editor): resolve template snapshots and protect draft saves"
```

---

### Task 4: Standardize rich content as a MetalDocs envelope and project governed assets to docgen

**Files:**
- Create: `internal/modules/documents/application/service_rich_content.go`
- Create: `internal/modules/documents/application/service_rich_content_test.go`
- Modify: `internal/modules/documents/application/service_runtime_validation.go`
- Modify: `internal/modules/documents/application/service_document_runtime.go`

- [ ] **Step 1: Define the backend envelope contract and parser**

Create `internal/modules/documents/application/service_rich_content.go`:

```go
package application

type richEnvelope struct {
	Format  string         `json:"format"`
	Version int            `json:"version"`
	Content map[string]any `json:"content"`
}

func parseRichEnvelope(value any) (richEnvelope, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return richEnvelope{}, domain.ErrInvalidNativeContent
	}
	var env richEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return richEnvelope{}, domain.ErrInvalidNativeContent
	}
	if env.Format != "metaldocs.rich.tiptap" || env.Version != 1 {
		return richEnvelope{}, domain.ErrInvalidNativeContent
	}
	return env, nil
}
```

- [ ] **Step 2: Tighten runtime validation to require the MetalDocs envelope for `rich` fields**

In `internal/modules/documents/application/service_runtime_validation.go`, replace the permissive `case "rich":` with:

```go
case "rich":
	if _, err := parseRichEnvelope(value); err != nil {
		return domain.ErrInvalidNativeContent
	}
```

This intentionally rejects raw HTML strings for the governed canvas pilot.

- [ ] **Step 3: Add the envelope-to-docgen converter with image asset resolution**

In `service_rich_content.go`, add:

```go
func (s *Service) projectRichEnvelopeToDocgenBlocks(ctx context.Context, value any) ([]any, error) {
	env, err := parseRichEnvelope(value)
	if err != nil {
		return nil, err
	}
	return projectTipTapDocument(ctx, env.Content, s.resolveRichAsset)
}

func projectTipTapDocument(ctx context.Context, doc map[string]any, resolveAsset func(context.Context, string) (resolvedRichAsset, error)) ([]any, error) {
	// Walk TipTap JSON and emit docgen-compatible rich blocks:
	// paragraph -> {type:"text", runs:[...]}
	// bulletList / orderedList -> {type:"list", ...}
	// image -> {type:"image", ...}
	return nil, nil
}

type resolvedRichAsset struct {
	Base64Data string
	MimeType   string
	AltText    string
}

func (s *Service) resolveRichAsset(ctx context.Context, assetID string) (resolvedRichAsset, error) {
	// Look up the attachment, open content from attachmentStore, normalize WEBP -> PNG when needed, then base64 encode.
	return resolvedRichAsset{}, nil
}
```

For image nodes, resolve `assetId` through the existing attachment repository/store and emit docgen image blocks shaped like:

```go
map[string]any{
	"type":     "image",
	"data":     base64Data,
	"mimeType": "image/png",
	"altText":  altText,
	"width":    320,
	"height":   180,
}
```

If a referenced image is `image/webp`, decode it with `golang.org/x/image/webp` and re-encode to PNG before building the block.

- [ ] **Step 4: Write the failing projection test before wiring it into payload building**

Create `internal/modules/documents/application/service_rich_content_test.go`:

```go
func TestProjectRichEnvelopeToDocgenBlocks_TextAndList(t *testing.T) {
	svc := newDocumentsServiceForTests(memory.NewRepository())

	value := map[string]any{
		"format": "metaldocs.rich.tiptap",
		"version": 1,
		"content": map[string]any{
			"type": "doc",
			"content": []any{
				map[string]any{
					"type": "paragraph",
					"content": []any{map[string]any{"type": "text", "text": "Teste rich"}},
				},
				map[string]any{
					"type": "bulletList",
					"content": []any{
						map[string]any{"type": "listItem", "content": []any{map[string]any{"type": "paragraph", "content": []any{map[string]any{"type": "text", "text": "Item 1"}}}}},
					},
				},
			},
		},
	}

	blocks, err := svc.projectRichEnvelopeToDocgenBlocks(context.Background(), value)
	if err != nil {
		t.Fatal(err)
	}
	if len(blocks) != 2 {
		t.Fatalf("len(blocks) = %d, want 2", len(blocks))
	}
}
```

- [ ] **Step 5: Wire the converter into docgen payload building and run targeted tests**

In `internal/modules/documents/application/service_document_runtime.go`, before sending values to docgen, walk the schema/value tree and replace rich envelopes with projected block arrays.

Run:

```bash
go test ./internal/modules/documents/application -run "TestProjectRichEnvelopeToDocgenBlocks_TextAndList" -count=1
```

Expected: pass.

- [ ] **Step 6: Commit**

```bash
git add internal/modules/documents/application/service_rich_content.go internal/modules/documents/application/service_rich_content_test.go internal/modules/documents/application/service_runtime_validation.go internal/modules/documents/application/service_document_runtime.go
git commit -m "feat(rich): standardize rich envelope validation and docgen projection"
```

---

### Task 5: Add the frontend template/canvas layer and rich-envelope adapters

**Files:**
- Modify: `frontend/apps/web/src/lib.types.ts`
- Modify: `frontend/apps/web/src/api/documents.ts`
- Create: `frontend/apps/web/src/features/documents/canvas/templateTypes.ts`
- Create: `frontend/apps/web/src/features/documents/canvas/templateAdapters.ts`
- Create: `frontend/apps/web/src/features/documents/canvas/rich/metaldocsRich.ts`
- Create: `frontend/apps/web/src/features/documents/canvas/DocumentCanvas.tsx`
- Create: `frontend/apps/web/src/features/documents/canvas/TemplateNodeRenderer.tsx`
- Create: `frontend/apps/web/src/features/documents/canvas/slots/FieldSlot.tsx`
- Create: `frontend/apps/web/src/features/documents/canvas/slots/RichSlot.tsx`
- Create: `frontend/apps/web/src/features/documents/canvas/slots/TableSlot.tsx`
- Create: `frontend/apps/web/src/features/documents/canvas/slots/RepeatSlot.tsx`
- Create: `frontend/apps/web/src/features/documents/canvas/DocumentCanvas.module.css`

- [ ] **Step 1: Extend the frontend API types for the enriched editor bundle**

In `frontend/apps/web/src/lib.types.ts`, extend the existing bundle/save types:

```ts
export interface DocumentTemplateSnapshotItem {
  templateKey: string;
  version: number;
  profileCode: string;
  schemaVersion: number;
  definition: Record<string, unknown>;
}

export interface DocumentEditorBundleResponse {
  document: DocumentListItem;
  versions: VersionListItem[];
  schema: DocumentProfileSchemaItem;
  governance: DocumentProfileGovernanceItem;
  templateSnapshot: DocumentTemplateSnapshotItem;
  draftToken: string;
  presence: CollaborationPresenceItem[];
  editLock?: DocumentEditLockItem;
}
```

Normalize these new fields in `frontend/apps/web/src/api/documents.ts`.

- [ ] **Step 2: Add the frontend template DSL and rich-envelope helpers**

Create `frontend/apps/web/src/features/documents/canvas/templateTypes.ts`:

```ts
export type CanvasTemplateNode =
  | { type: "page"; id: string; children: CanvasTemplateNode[] }
  | { type: "section-frame"; id: string; title?: string; children: CanvasTemplateNode[] }
  | { type: "label"; id: string; text: string }
  | { type: "field-slot"; id: string; path: string; fieldKind: "scalar" }
  | { type: "rich-slot"; id: string; path: string; fieldKind: "rich" }
  | { type: "table-slot"; id: string; path: string; fieldKind: "table" }
  | { type: "repeat-slot"; id: string; path: string; fieldKind: "repeat" };
```

Create `frontend/apps/web/src/features/documents/canvas/rich/metaldocsRich.ts`:

```ts
export function toEnvelope(editorJSON: Record<string, unknown>) {
  return {
    format: "metaldocs.rich.tiptap",
    version: 1,
    content: editorJSON,
  };
}

export function fromEnvelope(value: unknown) {
  if (!value || typeof value !== "object") return { type: "doc", content: [] };
  const record = value as Record<string, unknown>;
  return record.format === "metaldocs.rich.tiptap" && record.content && typeof record.content === "object"
    ? (record.content as Record<string, unknown>)
    : { type: "doc", content: [] };
}
```

- [ ] **Step 3: Write the canvas renderer skeleton**

Create `frontend/apps/web/src/features/documents/canvas/DocumentCanvas.tsx`:

```tsx
import type { CanvasTemplateNode } from "./templateTypes";
import { TemplateNodeRenderer } from "./TemplateNodeRenderer";
import styles from "./DocumentCanvas.module.css";

type DocumentCanvasProps = {
  template: CanvasTemplateNode;
  values: Record<string, unknown>;
  onChange: (next: Record<string, unknown>) => void;
};

export function DocumentCanvas({ template, values, onChange }: DocumentCanvasProps) {
  return (
    <div className={styles.canvasPage}>
      <TemplateNodeRenderer node={template} values={values} onChange={onChange} />
    </div>
  );
}
```

Create the recursive renderer and slot files, keeping each slot focused on one field kind.

- [ ] **Step 4: Make rich slots persist envelopes instead of HTML**

In `frontend/apps/web/src/features/documents/canvas/slots/RichSlot.tsx`, use `editor.getJSON()` and the helper above:

```tsx
onUpdate({ editor }) {
  onChange(toEnvelope(editor.getJSON() as Record<string, unknown>));
}
```

Use `fromEnvelope(value)` for initial editor content. Do not call `getHTML()` in the new canvas path.

- [ ] **Step 5: Build the frontend bundle after the canvas layer exists**

Run:

```bash
cd frontend/apps/web
npm run build
```

Expected: build passes with the new canvas files present and unused warnings resolved.

- [ ] **Step 6: Commit**

```bash
git add frontend/apps/web/src/lib.types.ts frontend/apps/web/src/api/documents.ts frontend/apps/web/src/features/documents/canvas
git commit -m "feat(frontend-canvas): add template renderer and rich envelope adapters"
```

---

### Task 6: Integrate the governed canvas into the content builder and enforce lock-aware saves

**Files:**
- Modify: `frontend/apps/web/src/components/content-builder/ContentBuilderView.tsx`
- Create: `frontend/apps/web/tests/e2e/governed-canvas-po.spec.ts`

- [ ] **Step 1: Replace `DynamicEditor` with the new `DocumentCanvas`**

In `frontend/apps/web/src/components/content-builder/ContentBuilderView.tsx`, replace:

```tsx
<DynamicEditor
  schema={schema}
  value={contentDraft}
  activeSectionKey={currentSectionKey}
  onChange={(next) => {
    dispatch({ type: "set_draft", payload: { contentDraft: next } });
    dispatch({ type: "set_status", payload: { status: "dirty" } });
  }}
/>
```

with:

```tsx
<DocumentCanvas
  template={activeTemplate}
  values={contentDraft}
  onChange={(next) => {
    dispatch({ type: "set_draft", payload: { contentDraft: next } });
    dispatch({ type: "set_status", payload: { status: "dirty" } });
  }}
/>
```

`activeTemplate` should come from `bundle.templateSnapshot.definition` normalized through `templateAdapters.ts`.

- [ ] **Step 2: Store and send the `draftToken` on every native save**

Extend the builder state:

```ts
type BuilderState = {
  status: BuilderStatus;
  error: string;
  pdfUrl: string;
  version: number | null;
  contentDraft: Record<string, unknown>;
  schema: DocumentProfileSchemaItem | null;
  previewCollapsed: boolean;
  sidebarCollapsed: boolean;
  draftToken: string;
  templateSnapshot: DocumentTemplateSnapshotItem | null;
};
```

And update the native save call:

```ts
const response = await api.saveDocumentContentNative(docId, {
  draftToken,
  content: body.content,
});
```

When the backend returns a new `draftToken`, store it immediately in reducer state.

- [ ] **Step 3: Respect the existing edit lock in the UI**

Use the bundle’s `editLock` data to disable editing if another user holds the lock:

```tsx
const lockedByOtherUser = editLock && editLock.lockedBy && editLock.lockedBy !== currentUserId;
```

If `lockedByOtherUser` is true:

- render the canvas read-only
- disable save/export buttons
- show a clear banner above the canvas

- [ ] **Step 4: Add a Playwright smoke for the pilot flow**

Create `frontend/apps/web/tests/e2e/governed-canvas-po.spec.ts`:

```ts
import { test, expect } from "@playwright/test";

test("po governed canvas saves and keeps preview/export alive", async ({ page }) => {
  await page.goto("/");
  await page.getByLabel("Identificador").fill("admin");
  await page.getByLabel("Senha").fill("admin123");
  await page.getByRole("button", { name: /entrar/i }).click();

  await page.getByRole("button", { name: /novo documento/i }).click();
  await page.getByLabel(/titulo/i).fill("PO teste canvas");
  await page.getByRole("button", { name: /ir para editor/i }).click();

  await page.getByText("Objetivo").click();
  await page.keyboard.type("Objetivo preenchido no canvas");
  await page.getByRole("button", { name: /salvar rascunho/i }).click();

  await expect(page.getByText(/salvo/i)).toBeVisible();
  await expect(page.getByRole("button", { name: /gerar pdf/i })).toBeEnabled();
});
```

- [ ] **Step 5: Run the frontend build and the new smoke test**

Run:

```bash
cd frontend/apps/web
npm run build
npx playwright test tests/e2e/governed-canvas-po.spec.ts --project=chromium
```

Expected: build passes, and the pilot smoke reaches save without stale-write errors.

- [ ] **Step 6: Commit**

```bash
git add frontend/apps/web/src/components/content-builder/ContentBuilderView.tsx frontend/apps/web/tests/e2e/governed-canvas-po.spec.ts
git commit -m "feat(content-builder): switch po pilot to governed document canvas"
```

---

### Task 7: Final end-to-end verification and cleanup

**Files:**
- Modify: `docs/superpowers/plans/2026-04-02-governed-document-canvas-pilot.md` (check off completed steps during execution)

- [ ] **Step 1: Run the backend test suites touched by the pilot**

```bash
go test ./internal/modules/documents/application ./internal/modules/documents/delivery/http ./internal/modules/documents/infrastructure/... -count=1
```

Expected: all touched documents-module suites pass.

- [ ] **Step 2: Run the docgen harness**

```bash
powershell -ExecutionPolicy Bypass -File apps/docgen/scripts/harness.ps1
```

Expected: harness passes with no rich-block regressions.

- [ ] **Step 3: Run the frontend build and smoke one more time**

```bash
cd frontend/apps/web
npm run build
npx playwright test tests/e2e/auth-smoke.spec.ts tests/e2e/governed-canvas-po.spec.ts --project=chromium
```

Expected: both smoke tests pass.

- [ ] **Step 4: Manual pilot verification**

Verify these flows manually:

1. `New Document` for `PO` creates the server draft before editor open.
2. The editor surface is the governed canvas, not the old form cards.
3. Rich text saves as structured content and survives reload.
4. Losing the lock or using a stale tab rejects save cleanly.
5. PDF preview still reflects the generated output.
6. A document-specific template assignment in the database overrides the profile default for new drafts of that lineage.

- [ ] **Step 5: Commit final cleanup**

```bash
git add docs/superpowers/plans/2026-04-02-governed-document-canvas-pilot.md
git commit -m "chore(canvas): verify governed canvas pilot end to end"
```

---

## Self-Review Checklist

- [x] **Spec coverage:** The plan covers template snapshotting, profile default vs document override, MetalDocs rich envelope, server-created drafts, edit-lock/stale-write protection, governed canvas authoring, and generated PDF truth.
- [x] **Scope discipline:** The plan stays on one `PO` pilot slice and explicitly excludes template admin UI, collaborative editing, slot ACL, and conditional templates.
- [x] **No placeholders:** Each task includes concrete files, commands, and starter code.
- [x] **Repo alignment:** The plan reuses the existing editor bundle, native save flow, attachment system, docgen, and Gotenberg rather than inventing parallel infrastructure.
- [x] **Type consistency:** `templateSnapshot`, `draftToken`, and the `metaldocs.rich.tiptap` envelope use the same names across ADR, OpenAPI, backend, and frontend tasks.

---

Plan complete and saved to `docs/superpowers/plans/2026-04-02-governed-document-canvas-pilot.md`. Two execution options:

**1. Subagent-Driven (recommended)** - I dispatch a fresh subagent per task, review between tasks, fast iteration

**2. Inline Execution** - Execute tasks in this session using executing-plans, batch execution with checkpoints

**Which approach?**
