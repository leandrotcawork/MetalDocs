# Codex Implementation Prompt — MDDM Block Styling

## What You Are Building

MetalDocs uses a custom BlockNote-based editor (called MDDM) with 9 custom block types.
These blocks currently render as raw unstyled HTML. You are implementing a token-driven
styling system so the editor looks like a production-grade document tool (Notion/Coda quality).

**After this task:**
- All 9 block components have CSS Module styling
- A global CSS file carries design tokens (CSS custom properties)
- A theme object from the template definition injects brand colors at runtime
- The DOCX exporter sends MDDM JSON to an existing `/render/mddm-docx` endpoint instead
  of converting to HTML and using Chromium

**Plan file:** `docs/superpowers/plans/2026-04-10-mddm-block-styling.md`
**Architecture spec:** `docs/superpowers/specs/2026-04-10-mddm-block-styling-architecture.md`

Read the plan file task-by-task and implement each one. Commit after each task.
Do not batch commits. Use `rtk` prefix on all shell commands (e.g. `rtk git add ...`, `rtk tsc`).

---

## Critical Context (Read Before Implementing)

### BlockNote DOM Architecture
BlockNote renders a block's component inside `.bn-block-content[data-content-type="X"]`.
Child blocks render in `.bn-block-group` which is a **sibling** to `.bn-block-content`,
both inside `.bn-block-outer`.

Structure:
```
.bn-block-outer
  .bn-block-content[data-content-type="fieldGroup"]
    <div data-columns="2" />   ← FieldGroup component renders here
  .bn-block-group              ← Field children render here (sibling!)
    .bn-block-outer (Field 1)
    .bn-block-outer (Field 2)
```

This means: **CSS grid on the FieldGroup component div does NOT contain child Field blocks.**
Grid must be applied to `.bn-block-group` using bridge CSS in the global CSS file.
Use CSS `:has()` to read `data-columns` from the sibling:
```css
.bn-container [data-content-type="fieldGroup"] > .bn-block-group {
  display: grid;
  grid-template-columns: 1fr;
}
.bn-container [data-content-type="fieldGroup"]:has([data-columns="2"]) > .bn-block-group {
  grid-template-columns: 1fr 1fr;
}
```
Do NOT put `display: grid` in `FieldGroup.module.css` — it will have no effect there.

### Blocks With `content: "inline"` (need `contentRef`)
Only these blocks have editable inline content: `field`, `dataTableCell`.
All other custom blocks use `content: "none"` — no `contentRef` prop.

### Adapter — Variant Props Are Dropped on Save
File: `frontend/apps/web/src/features/documents/mddm-editor/adapter.ts`

The `toMDDMProps` function whitelists props for each block type. These new variant props
are NOT in the whitelist and will be silently dropped when the document saves.
You MUST add them:

```ts
case "section":
  // ADD: variant
  return {
    title: asString(next.title),
    color: asString(next.color),
    locked: Boolean(next.locked),
    ...(next.optional === true ? { optional: true } : {}),
    ...(next.variant ? { variant: asString(next.variant) } : {}),  // ADD
  };

case "field":
  // ADD: layout
  const props: UnknownRecord = {
    label: asString(next.label),
    valueMode: "inline",
    locked: Boolean(next.locked),
  };
  const hint = asOptionalString(next.hint);
  if (hint) props.hint = hint;
  if (next.layout) props.layout = asString(next.layout);  // ADD
  return props;

case "repeatableItem":
  // ADD: style
  return {
    title: asString(next.title),
    ...(next.style ? { style: asString(next.style) } : {}),  // ADD
  };

case "richBlock":
  // ADD: chrome
  return {
    label: asString(next.label),
    locked: Boolean(next.locked),
    ...(next.chrome ? { chrome: asString(next.chrome) } : {}),  // ADD
  };

case "dataTable":
  // ADD: density
  return {
    label: asString(next.label),
    columns: parseColumns(next.columnsJson ?? next.columns),
    locked: Boolean(next.locked),
    minRows: normalizeInt(next.minRows, 0),
    maxRows: normalizeInt(next.maxRows, 500),
    ...(next.density ? { density: asString(next.density) } : {}),  // ADD
  };
```

`toBlockNoteProps` uses `cloneRecord(props)` for most types (passes everything through),
so the LOAD direction is fine. Only SAVE (toMDDMProps) needs the changes above.

### Go DOCX Export — Correct File and Approach
The current browser DOCX export path:
```
ExportDocumentDocxAuthorized (service_document_runtime.go:230)
  → fetches tmpl = GetDocumentTemplateVersion(...)
  → generateBrowserDocxBytes(ctx, doc, version, exportConfig, traceID)
      (in service_content_docx.go:167)
      → converts MDDM JSON to raw HTML via mddmBlocksToHTML()
      → calls GenerateBrowser → /generate-browser (Chromium)
```

The docgen server ALREADY has `/render/mddm-docx` → `exportMDDMToDocx`.
The Go client does NOT have a method for it. You need to:
1. Add types to `internal/platform/render/docgen/types.go`
2. Add `GenerateMDDM` method to `internal/platform/render/docgen/client.go`
3. Update `generateBrowserDocxBytes` in `service_content_docx.go` to use it
4. Also update the call site in `service_document_runtime.go` to pass `tmpl` (not just `exportConfig`) so the theme can be extracted from `tmpl.Definition`

**The docgen `MDDMExportRequest` type** (in `apps/docgen/src/mddm/types.ts`) uses:
```ts
type MDDMExportRequest = {
  envelope: MDDMEnvelope;
  metadata: { document_code, title, revision_label, mode };
};
```
So the Go payload must send the envelope as a parsed JSON object. You'll need to add
`templateTheme` to this existing type.

**Theme location in Go:** `tmpl.Definition["theme"]` where `tmpl` is `*domain.DocumentTemplateVersion`.
`Definition` is `map[string]any`. `TemplateExportConfig` only has margin fields — theme is NOT there.
The function signature for `generateBrowserDocxBytes` needs to be updated to receive the full
template definition (or `tmpl.Definition`) so it can extract the theme.

### `lib.types.ts` Theme Type (Missing from Original Plan)
`frontend/apps/web/src/lib.types.ts` has `DocumentBrowserTemplateSnapshotItem.definition?: Record<string, unknown>`.
Add a typed theme interface and update the field:

```ts
export interface MDDMTemplateTheme {
  accent?: string;
  accentLight?: string;
  accentDark?: string;
  accentBorder?: string;
}
```

Update `definition` field type to include `theme?: MDDMTemplateTheme`.

### React.CSSProperties Import
In `MDDMEditor.tsx`, use:
```ts
import { useMemo, type CSSProperties } from "react";
```
NOT `React.CSSProperties` (no React namespace import).
Cast as `vars as CSSProperties` not `vars as React.CSSProperties`.

---

## File Map

```
CREATE:
  frontend/apps/web/src/features/documents/mddm-editor/mddm-editor-global.css
  frontend/apps/web/src/features/documents/mddm-editor/blocks/Section.module.css
  frontend/apps/web/src/features/documents/mddm-editor/blocks/FieldGroup.module.css
  frontend/apps/web/src/features/documents/mddm-editor/blocks/Field.module.css
  frontend/apps/web/src/features/documents/mddm-editor/blocks/Repeatable.module.css
  frontend/apps/web/src/features/documents/mddm-editor/blocks/RepeatableItem.module.css
  frontend/apps/web/src/features/documents/mddm-editor/blocks/RichBlock.module.css
  frontend/apps/web/src/features/documents/mddm-editor/blocks/DataTable.module.css
  frontend/apps/web/src/features/documents/mddm-editor/blocks/DataTableRow.module.css
  frontend/apps/web/src/features/documents/mddm-editor/blocks/DataTableCell.module.css
  migrations/0068_add_theme_to_po_mddm_template.sql

MODIFY:
  frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.module.css
  frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.tsx
  frontend/apps/web/src/features/documents/mddm-editor/adapter.ts
  frontend/apps/web/src/features/documents/mddm-editor/blocks/Section.tsx
  frontend/apps/web/src/features/documents/mddm-editor/blocks/FieldGroup.tsx
  frontend/apps/web/src/features/documents/mddm-editor/blocks/Field.tsx
  frontend/apps/web/src/features/documents/mddm-editor/blocks/Repeatable.tsx
  frontend/apps/web/src/features/documents/mddm-editor/blocks/RepeatableItem.tsx
  frontend/apps/web/src/features/documents/mddm-editor/blocks/RichBlock.tsx
  frontend/apps/web/src/features/documents/mddm-editor/blocks/DataTable.tsx
  frontend/apps/web/src/features/documents/mddm-editor/blocks/DataTableRow.tsx
  frontend/apps/web/src/features/documents/mddm-editor/blocks/DataTableCell.tsx
  frontend/apps/web/src/features/documents/browser-editor/BrowserDocumentEditorView.tsx
  frontend/apps/web/src/lib.types.ts
  internal/platform/render/docgen/types.go
  internal/platform/render/docgen/client.go
  internal/modules/documents/application/service_content_docx.go
  internal/modules/documents/application/service_document_runtime.go
  apps/docgen/src/mddm/types.ts
  apps/docgen/src/mddm/exporter.ts
  apps/docgen/src/mddm/render-tables.ts
  apps/docgen/src/mddm/render-data-table.ts
```

---

## Implementation Order

Follow the plan file exactly. Implement task by task, commit after each.

After each TypeScript task run:
```bash
rtk tsc --noEmit -p frontend/apps/web/tsconfig.json
```
Expected: zero errors.

After Go tasks run:
```bash
rtk go build ./...
```

Do not skip verification steps.

---

## Definition of Done

- [ ] `rtk tsc --noEmit` passes with zero errors
- [ ] `rtk go build ./...` passes
- [ ] All 9 custom block `.module.css` files created
- [ ] `mddm-editor-global.css` created with design tokens and `:has()` FieldGroup bridge
- [ ] `MDDMEditor.tsx` exports `MDDMTheme` type and injects CSS vars
- [ ] adapter.ts `toMDDMProps` preserves variant/layout/style/chrome/density
- [ ] migration `0068_add_theme_to_po_mddm_template.sql` created
- [ ] Go docgen client has `GenerateMDDM` method calling `/render/mddm-docx`
- [ ] `generateBrowserDocxBytes` calls `GenerateMDDM` (not HTML conversion)
- [ ] `exportMDDMToDocx` reads and applies `templateTheme` from request
