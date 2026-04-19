# PO Browser Template Visual Polish — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add colored section header bars, Section 3 side-by-side layout, and configurable page margins to the PO browser template — both in the browser editor and DOCX export.

**Architecture:** Three independent layers: (1) CSS design language with reusable `md-section-header` and `md-two-col` classes, (2) template HTML body with inline styles for DOCX compatibility, (3) export config pipeline carrying per-template margins from domain through docgen to `HTMLtoDOCX` options.

**Tech Stack:** Go 1.23, CKEditor 5 RestrictedEditingMode, `@turbodocx/html-to-docx` v1.14+, PostgreSQL JSONB

**Spec:** `docs/superpowers/specs/2026-04-06-po-browser-template-visual-polish.md`

---

## File Structure

| File | Responsibility | Action |
|------|---------------|--------|
| `internal/modules/documents/domain/template.go` | `TemplateExportConfig` struct, field on `DocumentTemplateVersion` / `DocumentTemplateSnapshot`, PO template body + ExportConfig seed | Modify |
| `internal/modules/documents/domain/template_test.go` | ExportConfig assertion, parity tests | Modify |
| `internal/platform/render/docgen/types.go` | `BrowserRenderMargins` struct, field on `BrowserRenderPayload` | Modify |
| `internal/modules/documents/application/service_content_docx.go` | Load template ExportConfig, map to margins on payload | Modify |
| `internal/modules/documents/application/service_content_docx_test.go` | Test margin extraction helper | Modify |
| `internal/modules/documents/application/service_browser_editor.go` | Copy ExportConfig in snapshot builder | Modify |
| `internal/modules/documents/infrastructure/postgres/repository.go` | Scan `export_config` JSONB column | Modify |
| `apps/docgen/src/generateBrowser.ts` | Parse margins, convert inches→twips, pass to `HTMLtoDOCX` options | Modify |
| `frontend/apps/web/src/styles/document-content.css` | Add `md-section-header`, `md-two-col` classes; replace `h2` rule; adjust shell padding | Modify |
| `migrations/0059_add_template_export_config.sql` | ALTER TABLE add `export_config` column | Create |
| `migrations/0060_update_po_browser_template_visual_polish.sql` | UPDATE body + export_config for PO template | Create |

---

## Task 1: Add TemplateExportConfig domain type

**Files:**
- Modify: `internal/modules/documents/domain/template.go:1-19`

- [ ] **Step 1: Write the test — verify PO template has ExportConfig**

Add to `internal/modules/documents/domain/template_test.go`:

```go
func TestPOBrowserTemplateHasExportConfig(t *testing.T) {
	var template *DocumentTemplateVersion
	for _, tmpl := range DefaultDocumentTemplateVersions() {
		if tmpl.TemplateKey == "po-default-browser" {
			found := tmpl
			template = &found
			break
		}
	}
	if template == nil {
		t.Fatal("po-default-browser template not found")
	}
	if template.ExportConfig == nil {
		t.Fatal("po-default-browser template must have ExportConfig")
	}
	if template.ExportConfig.MarginTop != 0.625 {
		t.Fatalf("MarginTop = %f, want 0.625", template.ExportConfig.MarginTop)
	}
	if template.ExportConfig.MarginRight != 0.625 {
		t.Fatalf("MarginRight = %f, want 0.625", template.ExportConfig.MarginRight)
	}
	if template.ExportConfig.MarginBottom != 0.625 {
		t.Fatalf("MarginBottom = %f, want 0.625", template.ExportConfig.MarginBottom)
	}
	if template.ExportConfig.MarginLeft != 0.625 {
		t.Fatalf("MarginLeft = %f, want 0.625", template.ExportConfig.MarginLeft)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd internal/modules/documents/domain && go test -run TestPOBrowserTemplateHasExportConfig -v`
Expected: FAIL — `ExportConfig` field does not exist yet.

- [ ] **Step 3: Add TemplateExportConfig struct and fields**

In `internal/modules/documents/domain/template.go`, add the struct after the imports and before `DocumentTemplateVersion`:

```go
type TemplateExportConfig struct {
	MarginTop    float64 `json:"marginTop"`
	MarginRight  float64 `json:"marginRight"`
	MarginBottom float64 `json:"marginBottom"`
	MarginLeft   float64 `json:"marginLeft"`
}
```

Add `ExportConfig *TemplateExportConfig` field to `DocumentTemplateVersion` (after `CreatedAt`):

```go
type DocumentTemplateVersion struct {
	TemplateKey   string
	Version       int
	ProfileCode   string
	SchemaVersion int
	Name          string
	Editor        string
	ContentFormat string
	Body          string
	Definition    map[string]any
	CreatedAt     time.Time
	ExportConfig  *TemplateExportConfig
}
```

Add `ExportConfig *TemplateExportConfig` field to `DocumentTemplateSnapshot` (after `Definition`):

```go
type DocumentTemplateSnapshot struct {
	TemplateKey   string
	Version       int
	ProfileCode   string
	SchemaVersion int
	Editor        string
	ContentFormat string
	Body          string
	Definition    map[string]any
	ExportConfig  *TemplateExportConfig
}
```

Set ExportConfig on the `po-default-browser` seed in `DefaultDocumentTemplateVersions()` — add after `CreatedAt: time.Unix(0, 0).UTC(),`:

```go
ExportConfig: &TemplateExportConfig{
	MarginTop:    0.625,
	MarginRight:  0.625,
	MarginBottom: 0.625,
	MarginLeft:   0.625,
},
```

- [ ] **Step 4: Update snapshot builder**

In `internal/modules/documents/application/service_browser_editor.go`, update `documentTemplateSnapshotFromVersion` (line 173) to copy `ExportConfig`:

```go
func documentTemplateSnapshotFromVersion(item domain.DocumentTemplateVersion) domain.DocumentTemplateSnapshot {
	return domain.DocumentTemplateSnapshot{
		TemplateKey:   item.TemplateKey,
		Version:       item.Version,
		ProfileCode:   item.ProfileCode,
		SchemaVersion: item.SchemaVersion,
		Editor:        item.Editor,
		ContentFormat: item.ContentFormat,
		Body:          item.Body,
		Definition:    item.Definition,
		ExportConfig:  item.ExportConfig,
	}
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `cd internal/modules/documents/domain && go test -run TestPOBrowserTemplateHasExportConfig -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add internal/modules/documents/domain/template.go internal/modules/documents/domain/template_test.go internal/modules/documents/application/service_browser_editor.go
git commit -m "feat(domain): add TemplateExportConfig with margins to DocumentTemplateVersion"
```

---

## Task 2: Add BrowserRenderMargins to docgen types

**Files:**
- Modify: `internal/platform/render/docgen/types.go:50-55`

- [ ] **Step 1: Add BrowserRenderMargins struct and field**

In `internal/platform/render/docgen/types.go`, add before `BrowserRenderPayload`:

```go
type BrowserRenderMargins struct {
	Top    float64 `json:"top"`
	Right  float64 `json:"right"`
	Bottom float64 `json:"bottom"`
	Left   float64 `json:"left"`
}
```

Add `Margins` field to `BrowserRenderPayload`:

```go
type BrowserRenderPayload struct {
	DocumentCode string                `json:"documentCode"`
	Title        string                `json:"title"`
	Version      string                `json:"version,omitempty"`
	HTML         string                `json:"html"`
	Margins      *BrowserRenderMargins `json:"margins,omitempty"`
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd internal/platform/render/docgen && go build ./...`
Expected: success (no errors)

- [ ] **Step 3: Commit**

```bash
git add internal/platform/render/docgen/types.go
git commit -m "feat(docgen): add BrowserRenderMargins to BrowserRenderPayload"
```

---

## Task 3: Migration 0059 — add export_config column

**Files:**
- Create: `migrations/0059_add_template_export_config.sql`

- [ ] **Step 1: Create migration file**

Create `migrations/0059_add_template_export_config.sql`:

```sql
-- 0059_add_template_export_config.sql
-- Adds export_config JSONB column to document_template_versions.
-- Stores per-template rendering configuration (margins, etc.) used by docgen.
-- NULL means "use docgen defaults" (backward compatible).

ALTER TABLE metaldocs.document_template_versions
  ADD COLUMN IF NOT EXISTS export_config JSONB;
```

- [ ] **Step 2: Commit**

```bash
git add migrations/0059_add_template_export_config.sql
git commit -m "feat(migrations): add export_config JSONB column to document_template_versions"
```

---

## Task 4: Repository — scan export_config JSONB column

**Files:**
- Modify: `internal/modules/documents/infrastructure/postgres/repository.go:1199-1281`

- [ ] **Step 1: Update GetDocumentTemplateVersion**

In `internal/modules/documents/infrastructure/postgres/repository.go`, update `GetDocumentTemplateVersion` (line 1199).

Change the SQL query to include `export_config`:

```go
func (r *Repository) GetDocumentTemplateVersion(ctx context.Context, templateKey string, version int) (domain.DocumentTemplateVersion, error) {
	const q = `
SELECT template_key, version, profile_code, schema_version, name, editor, content_format, body_html, definition_json, created_at, export_config
FROM metaldocs.document_template_versions
WHERE template_key = $1 AND version = $2
`
	var item domain.DocumentTemplateVersion
	var definitionJSON []byte
	var exportConfigJSON []byte
	if err := r.db.QueryRowContext(ctx, q, strings.TrimSpace(templateKey), version).Scan(
		&item.TemplateKey,
		&item.Version,
		&item.ProfileCode,
		&item.SchemaVersion,
		&item.Name,
		&item.Editor,
		&item.ContentFormat,
		&item.Body,
		&definitionJSON,
		&item.CreatedAt,
		&exportConfigJSON,
	); err != nil {
		if err == sql.ErrNoRows {
			return domain.DocumentTemplateVersion{}, domain.ErrDocumentTemplateNotFound
		}
		return domain.DocumentTemplateVersion{}, fmt.Errorf("get document template version: %w", err)
	}
	if len(definitionJSON) > 0 {
		if err := json.Unmarshal(definitionJSON, &item.Definition); err != nil {
			return domain.DocumentTemplateVersion{}, fmt.Errorf("unmarshal document template version definition: %w", err)
		}
	}
	if item.Definition == nil {
		item.Definition = map[string]any{}
	}
	if len(exportConfigJSON) > 0 {
		var cfg domain.TemplateExportConfig
		if err := json.Unmarshal(exportConfigJSON, &cfg); err != nil {
			return domain.DocumentTemplateVersion{}, fmt.Errorf("unmarshal template export config: %w", err)
		}
		item.ExportConfig = &cfg
	}
	return item, nil
}
```

- [ ] **Step 2: Update ListDocumentTemplateVersions**

Same pattern for `ListDocumentTemplateVersions` (line 1235). Update the SQL query and scan:

```go
func (r *Repository) ListDocumentTemplateVersions(ctx context.Context, profileCode string) ([]domain.DocumentTemplateVersion, error) {
	const q = `
SELECT template_key, version, profile_code, schema_version, name, editor, content_format, body_html, definition_json, created_at, export_config
FROM metaldocs.document_template_versions
WHERE ($1 = '' OR profile_code = $1)
ORDER BY profile_code ASC, template_key ASC, version DESC
`
	rows, err := r.db.QueryContext(ctx, q, strings.TrimSpace(profileCode))
	if err != nil {
		return nil, fmt.Errorf("list document template versions: %w", err)
	}
	defer rows.Close()

	items := make([]domain.DocumentTemplateVersion, 0)
	for rows.Next() {
		var item domain.DocumentTemplateVersion
		var definitionJSON []byte
		var exportConfigJSON []byte
		if err := rows.Scan(
			&item.TemplateKey,
			&item.Version,
			&item.ProfileCode,
			&item.SchemaVersion,
			&item.Name,
			&item.Editor,
			&item.ContentFormat,
			&item.Body,
			&definitionJSON,
			&item.CreatedAt,
			&exportConfigJSON,
		); err != nil {
			return nil, fmt.Errorf("scan document template version: %w", err)
		}
		if len(definitionJSON) > 0 {
			if err := json.Unmarshal(definitionJSON, &item.Definition); err != nil {
				return nil, fmt.Errorf("unmarshal document template version definition: %w", err)
			}
		}
		if item.Definition == nil {
			item.Definition = map[string]any{}
		}
		if len(exportConfigJSON) > 0 {
			var cfg domain.TemplateExportConfig
			if err := json.Unmarshal(exportConfigJSON, &cfg); err != nil {
				return nil, fmt.Errorf("unmarshal template export config: %w", err)
			}
			item.ExportConfig = &cfg
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list document template versions rows: %w", err)
	}

	return items, nil
}
```

- [ ] **Step 3: Verify compilation**

Run: `cd internal/modules/documents/infrastructure/postgres && go build ./...`
Expected: success

- [ ] **Step 4: Commit**

```bash
git add internal/modules/documents/infrastructure/postgres/repository.go
git commit -m "feat(postgres): scan export_config JSONB from document_template_versions"
```

---

## Task 5: Wire margins in generateBrowserDocxBytes

**Design note:** Template versions are only written via SQL migrations in this codebase — there is no Go runtime INSERT/UPDATE path for `document_template_versions`. The in-memory test repository stores full `DocumentTemplateVersion` structs, so `ExportConfig` flows automatically. No write-path changes are needed.

**Architecture note:** Margins are extracted from the template loaded by the caller (`ExportDocumentDocxAuthorized`), not via a fresh repo lookup inside `generateBrowserDocxBytes`. This keeps the export path consistent — the template is resolved once.

**Files:**
- Modify: `internal/modules/documents/application/service_content_docx.go:166-188`
- Modify: `internal/modules/documents/application/service_content_docx_test.go`
- Modify: `internal/modules/documents/application/service_document_runtime.go:230-245`

- [ ] **Step 1: Write the test — margin extraction helper**

Add to `internal/modules/documents/application/service_content_docx_test.go`:

```go
func TestBrowserRenderMarginsFromExportConfig(t *testing.T) {
	t.Run("nil config returns nil", func(t *testing.T) {
		got := browserRenderMarginsFromExportConfig(nil)
		if got != nil {
			t.Fatalf("expected nil, got %+v", got)
		}
	})

	t.Run("populated config returns margins", func(t *testing.T) {
		cfg := &domain.TemplateExportConfig{
			MarginTop:    0.625,
			MarginRight:  0.625,
			MarginBottom: 0.625,
			MarginLeft:   0.625,
		}
		got := browserRenderMarginsFromExportConfig(cfg)
		if got == nil {
			t.Fatal("expected non-nil margins")
		}
		if got.Top != 0.625 {
			t.Fatalf("Top = %f, want 0.625", got.Top)
		}
		if got.Right != 0.625 {
			t.Fatalf("Right = %f, want 0.625", got.Right)
		}
		if got.Bottom != 0.625 {
			t.Fatalf("Bottom = %f, want 0.625", got.Bottom)
		}
		if got.Left != 0.625 {
			t.Fatalf("Left = %f, want 0.625", got.Left)
		}
	})
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `cd internal/modules/documents/application && go test -run TestBrowserRenderMarginsFromExportConfig -v`
Expected: FAIL — `browserRenderMarginsFromExportConfig` does not exist yet.

- [ ] **Step 3: Implement helper and update generateBrowserDocxBytes signature**

In `internal/modules/documents/application/service_content_docx.go`, add the helper function after `convertDocxToPDF`:

```go
func browserRenderMarginsFromExportConfig(cfg *domain.TemplateExportConfig) *docgen.BrowserRenderMargins {
	if cfg == nil {
		return nil
	}
	return &docgen.BrowserRenderMargins{
		Top:    cfg.MarginTop,
		Right:  cfg.MarginRight,
		Bottom: cfg.MarginBottom,
		Left:   cfg.MarginLeft,
	}
}
```

Update `generateBrowserDocxBytes` to accept `exportConfig` as a parameter instead of doing its own repo lookup. Replace the existing function (lines 166-188):

```go
func (s *Service) generateBrowserDocxBytes(ctx context.Context, doc domain.Document, version domain.Version, exportConfig *domain.TemplateExportConfig, traceID string) ([]byte, error) {
	if s.docgenClient == nil {
		return nil, domain.ErrRenderUnavailable
	}
	if strings.TrimSpace(version.Content) == "" {
		return nil, domain.ErrInvalidCommand
	}
	headerHTML := buildBrowserDocumentHeaderHTML(doc, version)
	payload := docgen.BrowserRenderPayload{
		DocumentCode: doc.DocumentCode,
		Title:        doc.Title,
		Version:      fmt.Sprintf("%d", version.Number),
		HTML:         headerHTML + substituteTemplateTokens(version.Content, doc, version),
		Margins:      browserRenderMarginsFromExportConfig(exportConfig),
	}
	rendered, err := s.docgenClient.GenerateBrowser(ctx, payload, traceID)
	if err != nil {
		if errors.Is(err, docgen.ErrUnavailable) {
			return nil, domain.ErrRenderUnavailable
		}
		return nil, err
	}
	return rendered, nil
}
```

- [ ] **Step 4: Update ExportDocumentDocxAuthorized caller**

In `internal/modules/documents/application/service_document_runtime.go`, update `ExportDocumentDocxAuthorized` (line 230). The caller now loads the template version and passes its `ExportConfig`:

```go
if strings.TrimSpace(version.ContentSource) == domain.ContentSourceBrowserEditor {
	var exportConfig *domain.TemplateExportConfig
	if version.TemplateKey != "" && version.TemplateVersion > 0 {
		tmpl, err := s.repo.GetDocumentTemplateVersion(ctx, version.TemplateKey, version.TemplateVersion)
		if err != nil {
			return nil, err
		}
		exportConfig = tmpl.ExportConfig
	}
	return s.generateBrowserDocxBytes(ctx, doc, version, exportConfig, traceID)
}
```

Replace the existing single-line call `return s.generateBrowserDocxBytes(ctx, doc, version, traceID)` with the block above. Note: if the template cannot be resolved, the error propagates (template binding failure). If the template is found but has no `ExportConfig` (nil), margins default to docgen library defaults — this is expected for templates without margin configuration.

- [ ] **Step 5: Run test to verify it passes**

Run: `cd internal/modules/documents/application && go test -run TestBrowserRenderMarginsFromExportConfig -v`
Expected: PASS

- [ ] **Step 6: Run all existing tests to verify no regression**

Run: `cd internal/modules/documents/application && go test ./... -v -count=1`
Expected: All existing tests pass.

- [ ] **Step 7: Commit**

```bash
git add internal/modules/documents/application/service_content_docx.go internal/modules/documents/application/service_content_docx_test.go internal/modules/documents/application/service_document_runtime.go
git commit -m "feat(docx): pass template ExportConfig to generateBrowserDocxBytes for margin support"
```

---

## Task 6: Update generateBrowser.ts — parse margins, pass to HTMLtoDOCX

**Files:**
- Modify: `apps/docgen/src/generateBrowser.ts`

- [ ] **Step 1: Update the type and normalization**

Replace the full content of `apps/docgen/src/generateBrowser.ts`:

```typescript
import HTMLtoDOCX from "@turbodocx/html-to-docx";

type BrowserDocumentMargins = {
  top: number;
  right: number;
  bottom: number;
  left: number;
};

type BrowserDocumentPayload = {
  documentCode?: string;
  title?: string;
  version?: string;
  html: string;
  margins?: BrowserDocumentMargins;
};

function invalid(code: string): never {
  throw new Error(code);
}

function normalizeBrowserPayload(input: unknown): BrowserDocumentPayload {
  if (typeof input !== "object" || input === null || Array.isArray(input)) {
    invalid("DOCGEN_INVALID_PAYLOAD");
  }

  const payload = input as Partial<BrowserDocumentPayload>;
  if (typeof payload.html !== "string" || !payload.html.trim()) {
    invalid("DOCGEN_INVALID_PAYLOAD");
  }

  let margins: BrowserDocumentMargins | undefined;
  if (
    payload.margins &&
    typeof payload.margins === "object" &&
    typeof payload.margins.top === "number" &&
    typeof payload.margins.right === "number" &&
    typeof payload.margins.bottom === "number" &&
    typeof payload.margins.left === "number"
  ) {
    margins = payload.margins;
  }

  return {
    documentCode: typeof payload.documentCode === "string" ? payload.documentCode.trim() : undefined,
    title: typeof payload.title === "string" ? payload.title.trim() : undefined,
    version: typeof payload.version === "string" ? payload.version.trim() : undefined,
    html: payload.html,
    margins,
  };
}

/** Convert inches to twips (1 inch = 1440 twips). */
function inchesToTwips(inches: number): number {
  return Math.round(inches * 1440);
}

export async function generateBrowserDocx(payload: unknown): Promise<Uint8Array> {
  const browserPayload = normalizeBrowserPayload(payload);

  const options: HTMLtoDOCX.DocumentOptions | undefined = browserPayload.margins
    ? {
        margins: {
          top: inchesToTwips(browserPayload.margins.top),
          right: inchesToTwips(browserPayload.margins.right),
          bottom: inchesToTwips(browserPayload.margins.bottom),
          left: inchesToTwips(browserPayload.margins.left),
        },
      }
    : undefined;

  const document = await HTMLtoDOCX(browserPayload.html, null, options);
  if (document instanceof Uint8Array) {
    return document;
  }
  if (document instanceof ArrayBuffer) {
    return new Uint8Array(document);
  }
  if (document instanceof Blob) {
    return new Uint8Array(await document.arrayBuffer());
  }
  return new Uint8Array(document);
}
```

- [ ] **Step 2: Verify TypeScript compilation**

Run: `cd apps/docgen && npx tsc --noEmit`
Expected: success (no type errors)

- [ ] **Step 3: Commit**

```bash
git add apps/docgen/src/generateBrowser.ts
git commit -m "feat(docgen): parse margins from payload and pass to HTMLtoDOCX options"
```

---

## Task 7: CSS — section header bars, two-column layout, shell adjustment

**Files:**
- Modify: `frontend/apps/web/src/styles/document-content.css`

- [ ] **Step 1: Replace the `.md-section > h2` rule with `.md-section-header`**

In `frontend/apps/web/src/styles/document-content.css`, replace lines 33-42 (the `.ck-content .md-section > h2` block):

Old:
```css
.ck-content .md-section > h2 {
  font-size: 1rem;
  font-weight: 700;
  color: #3e1018;
  padding-left: 0.8rem;
  border-left: 3px solid #6b1f2a;
  margin-top: 0;
  margin-bottom: 1.25rem;
  line-height: 1.4;
}
```

New:
```css
/* ── Section Header Bar ───────────────────────────────────────── */
.ck-content .md-section-header {
  width: 100%;
  border-collapse: collapse;
  margin-bottom: 0.75rem;
}

.ck-content .md-section-header td {
  background-color: #6b1f2a;
  color: #ffffff;
  padding: 8px 14px;
  font-size: 13px;
  font-weight: 700;
  letter-spacing: 0.5px;
}
```

- [ ] **Step 2: Add `.md-two-col` rules**

Add after the new `.md-section-header` block:

```css
/* ── Two-Column Layout ────────────────────────────────────────── */
.ck-content .md-two-col {
  width: 100%;
  border-collapse: collapse;
  margin-bottom: 1rem;
}

.ck-content .md-two-col > tbody > tr > td,
.ck-content .md-two-col > tr > td {
  vertical-align: top;
  width: 50%;
}

.ck-content .md-two-col > tbody > tr > td:first-child,
.ck-content .md-two-col > tr > td:first-child {
  padding-right: 0.5rem;
}

.ck-content .md-two-col > tbody > tr > td:last-child,
.ck-content .md-two-col > tr > td:last-child {
  padding-left: 0.5rem;
}
```

- [ ] **Step 3: Adjust shell padding for narrower visual margins**

In the `.ck-content .md-doc-shell` rule (line 9), change `padding: 2rem;` to `padding: 1.5rem;`.

- [ ] **Step 4: Commit**

```bash
git add frontend/apps/web/src/styles/document-content.css
git commit -m "feat(css): add md-section-header and md-two-col classes, adjust shell padding"
```

---

## Task 8: Update template body HTML + migration + parity tests

This is the largest task. It changes the PO template body (section headers + Section 3 side-by-side), creates a new migration with the matching body, and updates parity tests to validate the latest migration.

**Files:**
- Modify: `internal/modules/documents/domain/template.go` (Body field of po-default-browser)
- Create: `migrations/0060_update_po_browser_template_visual_polish.sql`
- Modify: `internal/modules/documents/domain/template_test.go` (parity tests)

- [ ] **Step 1: Update the Body in template.go**

Replace the entire `Body` string of the `po-default-browser` template in `DefaultDocumentTemplateVersions()` (currently at line 91). The new body replaces every `<h2>` with a `<table class="md-section-header">` bar and restructures Section 3 as side-by-side:

```go
Body: `<section class="md-doc-shell">
  <section class="md-section">
    <table class="md-section-header" style="width:100%;border-collapse:collapse;margin-bottom:0.75rem;">
      <tr><td style="background-color:#6b1f2a;color:#ffffff;padding:8px 14px;font-size:13px;font-weight:700;letter-spacing:0.5px;">2. Identificação do Processo</td></tr>
    </table>
    <table class="md-field-table" style="width:100%;border-collapse:collapse;margin-bottom:1rem;">
      <tbody>
        <tr>
          <td class="md-field-label" style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:30%;font-weight:600;font-size:0.84rem;color:#3e1018;background:#f9f3f3;">Objetivo</td>
          <td style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:70%;"><p class="restricted-editing-exception">Descreva o objetivo deste procedimento, incluindo o resultado esperado ao final da execução.</p></td>
        </tr>
        <tr>
          <td class="md-field-label" style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:30%;font-weight:600;font-size:0.84rem;color:#3e1018;background:#f9f3f3;">Escopo</td>
          <td style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:70%;"><p class="restricted-editing-exception">Defina os limites de aplicação deste procedimento: onde começa, onde termina e o que está fora do escopo.</p></td>
        </tr>
        <tr>
          <td class="md-field-label" style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:30%;font-weight:600;font-size:0.84rem;color:#3e1018;background:#f9f3f3;">Cargo responsável</td>
          <td style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:70%;"><p class="restricted-editing-exception">Informe o cargo ou função responsável pela execução deste procedimento.</p></td>
        </tr>
        <tr>
          <td class="md-field-label" style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:30%;font-weight:600;font-size:0.84rem;color:#3e1018;background:#f9f3f3;">Canal / Contexto</td>
          <td style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:70%;"><p class="restricted-editing-exception">Descreva o canal ou contexto em que este procedimento se aplica.</p></td>
        </tr>
        <tr>
          <td class="md-field-label" style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:30%;font-weight:600;font-size:0.84rem;color:#3e1018;background:#f9f3f3;">Participantes</td>
          <td style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:70%;"><p class="restricted-editing-exception">Liste os cargos, funções ou áreas que participam da execução deste procedimento.</p></td>
        </tr>
      </tbody>
    </table>
  </section>

  <section class="md-section">
    <table class="md-section-header" style="width:100%;border-collapse:collapse;margin-bottom:0.75rem;">
      <tr><td style="background-color:#6b1f2a;color:#ffffff;padding:8px 14px;font-size:13px;font-weight:700;letter-spacing:0.5px;">3. Entradas e Saídas</td></tr>
    </table>
    <table class="md-two-col" style="width:100%;border-collapse:collapse;margin-bottom:1rem;">
      <tr>
        <td style="width:50%;vertical-align:top;padding-right:0.5rem;">
          <table class="md-field-table" style="width:100%;border-collapse:collapse;">
            <tbody>
              <tr>
                <td class="md-field-label" style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:30%;font-weight:600;font-size:0.84rem;color:#3e1018;background:#f9f3f3;">Entradas</td>
                <td style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:70%;"><p class="restricted-editing-exception">Liste os insumos, informações ou materiais necessários para iniciar o processo.</p></td>
              </tr>
              <tr>
                <td class="md-field-label" style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:30%;font-weight:600;font-size:0.84rem;color:#3e1018;background:#f9f3f3;">Documentos relacionados</td>
                <td style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:70%;"><p class="restricted-editing-exception">Liste documentos, formulários ou registros utilizados ou gerados durante o processo.</p></td>
              </tr>
            </tbody>
          </table>
        </td>
        <td style="width:50%;vertical-align:top;padding-left:0.5rem;">
          <table class="md-field-table" style="width:100%;border-collapse:collapse;">
            <tbody>
              <tr>
                <td class="md-field-label" style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:30%;font-weight:600;font-size:0.84rem;color:#3e1018;background:#f9f3f3;">Saídas</td>
                <td style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:70%;"><p class="restricted-editing-exception">Liste os produtos, resultados ou entregas gerados ao final do processo.</p></td>
              </tr>
              <tr>
                <td class="md-field-label" style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:30%;font-weight:600;font-size:0.84rem;color:#3e1018;background:#f9f3f3;">Sistemas utilizados</td>
                <td style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:70%;"><p class="restricted-editing-exception">Liste os sistemas, ferramentas ou plataformas utilizadas na execução do processo.</p></td>
              </tr>
            </tbody>
          </table>
        </td>
      </tr>
    </table>
  </section>

  <section class="md-section">
    <table class="md-section-header" style="width:100%;border-collapse:collapse;margin-bottom:0.75rem;">
      <tr><td style="background-color:#6b1f2a;color:#ffffff;padding:8px 14px;font-size:13px;font-weight:700;letter-spacing:0.5px;">4. Visão Geral do Processo</td></tr>
    </table>
    <div class="md-field">
      <p class="md-field-label"><strong>Descrição do processo</strong></p>
      <div class="restricted-editing-exception"><p>Descreva o processo de forma detalhada, incluindo o contexto, fluxo geral e principais decisões envolvidas.</p></div>
    </div>
    <div class="md-field">
      <p class="md-field-label"><strong>Ferramenta do fluxograma</strong></p>
      <p><span class="restricted-editing-exception">Informe a ferramenta utilizada para criar o fluxograma (ex: Bizagi, Visio, Miro).</span></p>
    </div>
    <div class="md-field">
      <p class="md-field-label"><strong>Link do fluxograma</strong></p>
      <p><span class="restricted-editing-exception">Cole o link de acesso ao fluxograma do processo.</span></p>
    </div>
    <div class="md-field">
      <p class="md-field-label"><strong>Diagrama</strong></p>
      <div class="restricted-editing-exception"><p>Insira ou descreva o diagrama do processo. Pode utilizar imagens ou representações textuais.</p></div>
    </div>
  </section>

  <section class="md-section">
    <table class="md-section-header" style="width:100%;border-collapse:collapse;margin-bottom:0.75rem;">
      <tr><td style="background-color:#6b1f2a;color:#ffffff;padding:8px 14px;font-size:13px;font-weight:700;letter-spacing:0.5px;">5. Detalhamento das Etapas</td></tr>
    </table>
    <p class="md-section-hint">Descreva cada etapa como uma seção livre. Duplique o bloco abaixo para adicionar mais etapas.</p>
    <div class="md-free-block restricted-editing-exception">
      <h3>Etapa 1 — [Nome da etapa]</h3>
      <p>Descreva esta etapa livremente. Adicione parágrafos, listas, referências a outros documentos ou qualquer informação relevante para descrever o que acontece nesta etapa do processo.</p>
    </div>
  </section>

  <section class="md-section">
    <table class="md-section-header" style="width:100%;border-collapse:collapse;margin-bottom:0.75rem;">
      <tr><td style="background-color:#6b1f2a;color:#ffffff;padding:8px 14px;font-size:13px;font-weight:700;letter-spacing:0.5px;">6. Controle e Exceções</td></tr>
    </table>
    <table class="md-field-table" style="width:100%;border-collapse:collapse;margin-bottom:1rem;">
      <tbody>
        <tr>
          <td class="md-field-label" style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:30%;font-weight:600;font-size:0.84rem;color:#3e1018;background:#f9f3f3;">Pontos de controle</td>
          <td style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:70%;"><p class="restricted-editing-exception">Descreva os pontos de verificação, aprovação ou controle existentes no processo.</p></td>
        </tr>
        <tr>
          <td class="md-field-label" style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:30%;font-weight:600;font-size:0.84rem;color:#3e1018;background:#f9f3f3;">Exceções e desvios</td>
          <td style="border:1px solid #dfc8c8;padding:0.5rem 0.75rem;vertical-align:top;width:70%;"><p class="restricted-editing-exception">Descreva situações excepcionais e como devem ser tratadas.</p></td>
        </tr>
      </tbody>
    </table>
  </section>

  <section class="md-section">
    <table class="md-section-header" style="width:100%;border-collapse:collapse;margin-bottom:0.75rem;">
      <tr><td style="background-color:#6b1f2a;color:#ffffff;padding:8px 14px;font-size:13px;font-weight:700;letter-spacing:0.5px;">7. Indicadores de Desempenho</td></tr>
    </table>
    <div class="md-field">
      <p class="md-field-label"><strong>KPIs</strong></p>
      <figure class="table md-table restricted-editing-exception">
        <table>
          <thead>
            <tr><th>Indicador / KPI</th><th>Meta</th><th>Frequência</th></tr>
          </thead>
          <tbody>
            <tr><td>Ex: Taxa de retrabalho</td><td>Ex: &lt; 5%</td><td>Ex: Mensal</td></tr>
          </tbody>
        </table>
      </figure>
    </div>
  </section>

  <section class="md-section">
    <table class="md-section-header" style="width:100%;border-collapse:collapse;margin-bottom:0.75rem;">
      <tr><td style="background-color:#6b1f2a;color:#ffffff;padding:8px 14px;font-size:13px;font-weight:700;letter-spacing:0.5px;">8. Documentos e Referências</td></tr>
    </table>
    <figure class="table md-table restricted-editing-exception">
      <table>
        <thead>
          <tr><th>Código</th><th>Título / Descrição</th><th>Link / Localização</th></tr>
        </thead>
        <tbody>
          <tr><td>Ex: PO-001</td><td>Ex: Procedimento de compras</td><td>Ex: /docs/po-001</td></tr>
        </tbody>
      </table>
    </figure>
  </section>

  <section class="md-section">
    <table class="md-section-header" style="width:100%;border-collapse:collapse;margin-bottom:0.75rem;">
      <tr><td style="background-color:#6b1f2a;color:#ffffff;padding:8px 14px;font-size:13px;font-weight:700;letter-spacing:0.5px;">9. Glossário</td></tr>
    </table>
    <figure class="table md-table restricted-editing-exception">
      <table>
        <thead>
          <tr><th>Termo</th><th>Definição</th></tr>
        </thead>
        <tbody>
          <tr><td>Ex: SLA</td><td>Ex: Acordo de nível de serviço</td></tr>
        </tbody>
      </table>
    </figure>
  </section>

  <section class="md-section">
    <table class="md-section-header" style="width:100%;border-collapse:collapse;margin-bottom:0.75rem;">
      <tr><td style="background-color:#6b1f2a;color:#ffffff;padding:8px 14px;font-size:13px;font-weight:700;letter-spacing:0.5px;">10. Histórico de Revisões</td></tr>
    </table>
    <figure class="table md-table">
      <table>
        <thead>
          <tr><th>Versão</th><th>Data</th><th>O que foi alterado</th><th>Por</th></tr>
        </thead>
        <tbody>
          <tr>
            <td><p class="restricted-editing-exception">{{versao}}</p></td>
            <td><p class="restricted-editing-exception">{{data_criacao}}</p></td>
            <td><p class="restricted-editing-exception"></p></td>
            <td><p class="restricted-editing-exception">{{elaborador}}</p></td>
          </tr>
        </tbody>
      </table>
    </figure>
  </section>
</section>`,
```

- [ ] **Step 2: Create migration 0060**

Create `migrations/0060_update_po_browser_template_visual_polish.sql`. The body between `$$` delimiters must be **byte-for-byte identical** to the Go `Body` string above. Copy the exact HTML from the Go template body:

Create the file `migrations/0060_update_po_browser_template_visual_polish.sql` with the following structure. The `body_html` between `$$` delimiters must be the **exact same string** as the Go `Body` field from Step 1 — copy it character-for-character. If there is any whitespace, newline, or character mismatch, `TestPOBrowserTemplateGoSQLParity` will fail.

```sql
-- 0060_update_po_browser_template_visual_polish.sql
-- Visual polish: section header bars (md-section-header tables), Section 3 side-by-side
-- (md-two-col), and export_config with 0.625" margins.
-- IMPORTANT: body_html must match the Go seed in domain/template.go.
-- Validated by TestPOBrowserTemplateGoSQLParity.

UPDATE metaldocs.document_template_versions
SET body_html = $$<COPY THE EXACT BODY FROM template.go HERE — the string between the backtick delimiters of the Body field>$$,
    export_config = '{"marginTop":0.625,"marginRight":0.625,"marginBottom":0.625,"marginLeft":0.625}'::jsonb
WHERE template_key = 'po-default-browser' AND version = 1;
```

**How to copy:** Open `internal/modules/documents/domain/template.go`, find the `po-default-browser` entry's `Body` field. Copy the string content (everything between the backtick delimiters, not including the backticks themselves) and paste it between the `$$` delimiters in the SQL file. The parity test `TestPOBrowserTemplateGoSQLParity` will catch any mismatch.

- [ ] **Step 3: Update parity tests**

In `internal/modules/documents/domain/template_test.go`:

**Replace** `TestPOBrowserTemplateGoSQLParity` to check against migration 0060 instead of 0057:

```go
func TestPOBrowserTemplateGoSQLParity(t *testing.T) {
	var goTemplate *DocumentTemplateVersion
	for _, tmpl := range DefaultDocumentTemplateVersions() {
		if tmpl.TemplateKey == "po-default-browser" {
			found := tmpl
			goTemplate = &found
			break
		}
	}
	if goTemplate == nil {
		t.Fatal("po-default-browser template not found in Go seed")
	}

	migrationPath := "../../../../migrations/0060_update_po_browser_template_visual_polish.sql"
	sqlBytes, err := os.ReadFile(migrationPath)
	if err != nil {
		t.Fatalf("read migration file: %v", err)
	}
	sqlContent := string(sqlBytes)

	parts := strings.SplitN(sqlContent, "$$", 3)
	if len(parts) < 3 {
		t.Fatal("migration file does not contain $$ delimited body_html")
	}
	sqlBody := parts[1]

	goNormalized := strings.TrimSpace(strings.ReplaceAll(goTemplate.Body, "\r\n", "\n"))
	sqlNormalized := strings.TrimSpace(strings.ReplaceAll(sqlBody, "\r\n", "\n"))

	if goNormalized != sqlNormalized {
		t.Fatalf("Go seed body and 0060 migration body differ.\nGo length=%d, SQL length=%d\nFirst difference at character %d",
			len(goNormalized), len(sqlNormalized), firstDiffIndex(goNormalized, sqlNormalized))
	}
}
```

**Replace** `TestPOBrowserTemplate0058Parity` with `TestPOBrowserTemplate0060Parity` that validates the same thing (or simply remove it since `TestPOBrowserTemplateGoSQLParity` now checks 0060):

```go
func TestPOBrowserTemplate0060ExportConfig(t *testing.T) {
	migrationPath := "../../../../migrations/0060_update_po_browser_template_visual_polish.sql"
	sqlBytes, err := os.ReadFile(migrationPath)
	if err != nil {
		t.Fatalf("read 0060 migration file: %v", err)
	}
	sqlContent := string(sqlBytes)

	if !strings.Contains(sqlContent, `"marginTop":0.625`) {
		t.Error("0060 migration missing marginTop in export_config")
	}
	if !strings.Contains(sqlContent, `"marginRight":0.625`) {
		t.Error("0060 migration missing marginRight in export_config")
	}
	if !strings.Contains(sqlContent, `"marginBottom":0.625`) {
		t.Error("0060 migration missing marginBottom in export_config")
	}
	if !strings.Contains(sqlContent, `"marginLeft":0.625`) {
		t.Error("0060 migration missing marginLeft in export_config")
	}
}
```

- [ ] **Step 4: Verify template content assertions still pass**

The existing `TestPOBrowserTemplateCoversPOv3Schema` checks that section titles and field labels exist in the body. The new body contains all the same text — only the HTML structure changed. Verify:

Run: `cd internal/modules/documents/domain && go test -v -count=1`
Expected: All tests pass including `TestPOBrowserTemplateCoversPOv3Schema`, `TestPOBrowserTemplateGoSQLParity`, `TestPOBrowserTemplate0060ExportConfig`, and `TestPOBrowserTemplateHasExportConfig`.

- [ ] **Step 5: Commit**

```bash
git add internal/modules/documents/domain/template.go internal/modules/documents/domain/template_test.go migrations/0060_update_po_browser_template_visual_polish.sql
git commit -m "feat(template): section header bars, Section 3 side-by-side, export_config migration"
```

---

## Task 9: Final verification

- [ ] **Step 1: Run all Go tests**

Run: `cd internal/modules/documents && go test ./... -v -count=1`
Expected: All tests pass.

- [ ] **Step 2: Run TypeScript typecheck**

Run: `cd apps/docgen && npx tsc --noEmit`
Expected: No type errors.

- [ ] **Step 3: Verify template body has section headers**

Run: `cd internal/modules/documents/domain && go test -run TestPOBrowserTemplateCoversPOv3Schema -v`
Expected: PASS — all section titles and field labels present.

- [ ] **Step 4: Verify no `<h2>` remains in template body**

The template body should contain zero `<h2>` tags (all replaced with `md-section-header` tables). Verify in Go test output or by searching the body string.
