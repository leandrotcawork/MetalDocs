# PO Browser Template — Visual Polish

**Date:** 2026-04-06
**Status:** Approved
**Depends on:** 2026-04-06-po-browser-template-final-form (merged)

## Context

The "final-form" implementation addressed three structural gaps: table-based header, bordered field tables, and Section 10 revision history. A side-by-side comparison of the generated DOCX against the reference template (`template-po-v2.docx`) reveals three remaining visual gaps:

1. **No colored section header bars** — sections use plain `<h2>` text with a CSS left-border accent. In DOCX export, these render as unstyled bold text with no background.
2. **Section 3 stacks vertically** — Entradas/Saidas and related fields are stacked rows. The reference shows a two-column side-by-side layout.
3. **Page margins too wide** — `HTMLtoDOCX` defaults to ~1.25" margins. The reference uses ~0.625".

## Design Principles

- **Scalable for multiple templates.** Every change must work as a reusable primitive, not a PO-specific hack. Future templates use the same CSS classes and set their own `ExportConfig`.
- **Both browser and DOCX.** CSS classes drive browser styling; inline styles on `<table>` elements drive DOCX rendering (since `HTMLtoDOCX` only maps `background-color` on table cells to DOCX shading).
- **Brand palette.** All colors use the existing vinho palette (`#6b1f2a`, `#3e1018`, `#f9f3f3`, `#dfc8c8`).

## Architecture

Three independent layers:

1. **CSS design language** — generic reusable classes in `document-content.css` for the browser view (`md-section-header`, `md-two-col`)
2. **Template HTML** — section header rows and side-by-side columns in `template.go` with inline styles for DOCX compatibility
3. **Export config pipeline** — `TemplateExportConfig` on `DocumentTemplateVersion` -> `BrowserRenderPayload.Margins` -> `generateBrowser.ts` -> `HTMLtoDOCX` options

Each layer is independent. Future templates use the same CSS classes and set their own `ExportConfig` — no code changes required.

---

## 1. ExportConfig Data Model

### Domain types

Add to `domain/template.go`:

```go
type TemplateExportConfig struct {
    MarginTop    float64 // inches
    MarginRight  float64
    MarginBottom float64
    MarginLeft   float64
}
```

Add `ExportConfig *TemplateExportConfig` field to both `DocumentTemplateVersion` and `DocumentTemplateSnapshot`.

When `nil`, the docgen service uses its own defaults (backward compatible).

### PO template values

```go
ExportConfig: &TemplateExportConfig{
    MarginTop: 0.625, MarginRight: 0.625,
    MarginBottom: 0.625, MarginLeft: 0.625,
},
```

### Database

Store as JSONB column `export_config` on `document_template_versions`. New migration adds the column; another migration UPDATEs the PO template row.

---

## 2. Docgen Margin Pipeline

Values flow through four layers:

### Go service (`service_content_docx.go`)

`generateBrowserDocxBytes` reads `templateVersion.ExportConfig` and sets margins on the payload.

### Go payload (`docgen/types.go`)

```go
type BrowserRenderMargins struct {
    Top    float64 `json:"top"`
    Right  float64 `json:"right"`
    Bottom float64 `json:"bottom"`
    Left   float64 `json:"left"`
}
```

Field on `BrowserRenderPayload`: `Margins *BrowserRenderMargins `json:"margins,omitempty"``

### Node.js docgen (`generateBrowser.ts`)

Parse `margins` from the payload. Convert inches to twips (1 inch = 1440 twips). Pass as the options argument:

```ts
await HTMLtoDOCX(html, null, {
    margins: { top: 900, right: 900, bottom: 900, left: 900 }
});
```

When `margins` is absent from the payload, omit the options argument entirely — preserving backward compatibility.

### Browser CSS (`document-content.css`)

Reduce `md-doc-shell` max-width or padding to visually approximate the narrower margins.

---

## 3. Colored Section Header Bars

### Current pattern

```html
<div class="md-section">
  <h2>3. Entradas e Saidas</h2>
  <!-- content -->
</div>
```

The `<h2>` gets a CSS left-border accent in the browser. In DOCX it renders as plain bold text.

### New pattern

Replace every `<h2>` with a single-row `<table>`:

```html
<div class="md-section">
  <table class="md-section-header" style="width:100%;border-collapse:collapse;margin-bottom:0.75rem;">
    <tr>
      <td style="background-color:#6b1f2a;color:#ffffff;padding:8px 14px;font-size:13px;font-weight:700;letter-spacing:0.5px;">
        3. Entradas e Saidas
      </td>
    </tr>
  </table>
  <!-- content -->
</div>
```

### Rationale

`HTMLtoDOCX` only maps `background-color` on table cells to DOCX cell shading. A `<h2>` with `background-color` becomes unstyled text.

### Brand color

Single color `#6b1f2a` (vinho) for all section headers (2-10). No per-section color variation.

### CSS

`.md-section-header` class provides browser polish. Inline styles serve the DOCX path. The existing `.md-section h2` left-border rule is removed.

---

## 4. Section 3 Side-by-Side Layout

### Current pattern

Single `md-field-table` with 4 stacked rows: Entradas, Saidas, Documentos relacionados, Sistemas utilizados.

### New pattern

Two-column outer table, each column containing its own `md-field-table`:

```html
<table class="md-two-col" style="width:100%;border-collapse:collapse;margin-bottom:1rem;">
  <tr>
    <td style="width:50%;vertical-align:top;padding-right:0.5rem;">
      <table class="md-field-table" style="width:100%;...">
        <tr><!-- Entradas --></tr>
        <tr><!-- Documentos relacionados --></tr>
      </table>
    </td>
    <td style="width:50%;vertical-align:top;padding-left:0.5rem;">
      <table class="md-field-table" style="width:100%;...">
        <tr><!-- Saidas --></tr>
        <tr><!-- Sistemas utilizados --></tr>
      </table>
    </td>
  </tr>
</table>
```

### Field pairing

- Left column: Entradas + Documentos relacionados (inputs)
- Right column: Saidas + Sistemas utilizados (outputs)

Matches the reference template's logical input/output grouping.

### Inner field tables

Same 30%/70% label-value split with identical inline styles (borders, colors). Label width is relative to the column, not the page.

### CSS

`.md-two-col` gets minimal browser styling. The `md-field-table` rules work within nested contexts. The class is generic — any future template can use it.

---

## 5. Files Affected

### Go backend
- `internal/modules/documents/domain/template.go` — `TemplateExportConfig` struct, field on `DocumentTemplateVersion` and `DocumentTemplateSnapshot`, updated PO template body HTML + ExportConfig
- `internal/modules/documents/application/service_content_docx.go` — read ExportConfig, populate Margins on payload. Must load the template version (not just the document version) to access ExportConfig.
- `internal/modules/documents/application/service_browser_editor.go` — `documentTemplateSnapshotFromVersion()` copies ExportConfig to the snapshot so the frontend receives it (for future browser-side margin rendering)
- `internal/platform/render/docgen/types.go` — `BrowserRenderMargins` struct, field on `BrowserRenderPayload`
- `internal/modules/documents/infrastructure/postgres/repository.go` — scan `export_config` JSONB column into `*TemplateExportConfig`
- `internal/modules/documents/delivery/http/handler.go` — no changes needed; ExportConfig is not exposed in the HTTP response (margins are a backend concern for DOCX export). The snapshot response struct remains unchanged.

### Node.js docgen
- `apps/docgen/src/generateBrowser.ts` — parse margins, convert to twips, pass to HTMLtoDOCX options

### Frontend
- `frontend/apps/web/src/styles/document-content.css` — add `md-section-header`, `md-two-col` classes; remove `md-section h2` left-border; adjust `md-doc-shell` for narrower margins

### Migrations
- New migration: add `export_config JSONB` column to `document_template_versions`
- New migration: UPDATE PO template with export config and updated body HTML

### Tests
- `domain/template_test.go` — ExportConfig assertion, updated parity tests
- `service_content_docx_test.go` — verify margins populated on payload
- `apps/docgen/src/generateBrowser.test.ts` — margins passed to HTMLtoDOCX

---

## 6. Testing Strategy

### Unit tests
- `TestTemplateExportConfig` — PO template has non-nil ExportConfig with expected margin values
- `TestGenerateBrowserDocxPassesMargins` — payload.Margins populated from template ExportConfig
- Docgen: margins present -> HTMLtoDOCX receives options with correct twip values; margins absent -> no options (backward compat)

### Parity tests
- Update `TestPOBrowserTemplateGoSQLParity` (0057) and `TestPOBrowserTemplate0058Parity` (0058) for body changes
- New parity check for export_config migration

### Integration
- Export PO document to DOCX and verify:
  - Section headers render as colored bars
  - Section 3 renders two columns side by side
  - Page margins are 0.625" (measurable in Word)

### CSS
- Visual verification — manual check that browser view matches DOCX output for all three changes
