# MDDM Unified Document Engine Design

## Goal

Replace the current docgen-based DOCX export pipeline with a client-side rendering engine that produces corporate-grade DOCX, PDF, and editor views from a single source of truth (MDDM blocks). The engine establishes a formal compatibility contract across three rendering targets (React editor, Chromium-rendered PDF, docx.js-rendered DOCX) so that visual drift is bounded, predictable, and auditable.

This is not an export feature. It is a document engine — a formalized layer of tokens, contracts, emitters, and compatibility rules that makes MetalDocs's document output competitive with industry-grade document management systems.

### Success criteria

- Docgen service is fully decommissioned. Zero backend involvement in DOCX generation.
- DOCX generated client-side via docx.js (MIT license), typical latency ~1s, file size ~60-100KB.
- PDF generated via Gotenberg Chromium route from full-fidelity HTML, typical latency ~2-3s.
- Editor-to-PDF pixel diff < 2%. Editor-to-DOCX (rasterized) pixel diff < 5%.
- Open-source, cross-platform fonts only (Carlito default — metric-compatible with Calibri). No embedded fonts in DOCX.
- Read-only view is instant — same React components as editor, no external rendering pipeline.
- Formal Render Compatibility Contract specifies what must be identical, what may differ, what is forbidden.
- Released documents render identically forever (version-pinned renderer + IR hash).

### Non-goals

- Migrating `native` or `docx_upload` content sources to MDDM (separate project).
- Batch or server-side export. Exports are always interactive (user clicks a button).
- Real-time collaboration on documents.
- Offline PDF generation (Gotenberg is required).
- Custom font embedding in the initial release (template-level opt-in deferred to future work).

## Architecture

### System Layers

```
┌─────────────────────────────────────────────────────┐
│                    User Interface                    │
│  ┌──────────────┐  ┌──────────────┐  ┌───────────┐  │
│  │ BlockNote     │  │ Export DOCX  │  │ Export PDF │  │
│  │ MDDM Editor  │  │ (client)     │  │ (server)  │  │
│  └──────┬───────┘  └──────┬───────┘  └─────┬─────┘  │
├─────────┼─────────────────┼────────────────┼────────┤
│         │    Render Layer  │                │        │
│         ▼                 ▼                │        │
│  ┌──────────────┐  ┌──────────────┐        │        │
│  │ React        │  │ DOCX Emitter │        │        │
│  │ Renderer     │  │ (docx.js)    │        │        │
│  └──────┬───────┘  └──────┬───────┘        │        │
├─────────┼─────────────────┼────────────────┼────────┤
│         │   Style Contract │                │        │
│         ▼                 ▼                │        │
│  ┌─────────────────────────────────┐       │        │
│  │      Layout IR / Design Tokens  │       │        │
│  │  (dimensions, colors, fonts,    │       │        │
│  │   spacing, component rules,     │       │        │
│  │   compatibility contract)       │       │        │
│  └──────────────┬──────────────────┘       │        │
├─────────────────┼──────────────────────────┼────────┤
│                 │   Content Model          │        │
│                 ▼                          │        │
│  ┌─────────────────────────────────┐       │        │
│  │     MDDM Envelope (blocks)     │       │        │
│  │  + Template Definition          │       │        │
│  └─────────────────────────────────┘       │        │
├────────────────────────────────────────────┼────────┤
│                                    Backend │        │
│                              ┌─────────────▼──────┐ │
│                              │  Gotenberg          │ │
│                              │  (HTML → PDF)       │ │
│                              │  (Chromium route)   │ │
│                              └─────────────────────┘ │
└─────────────────────────────────────────────────────┘
```

### What changes from today

| Component | Today | After |
|-----------|-------|-------|
| **DOCX generation** | Go backend → external docgen service → `POST /render/mddm-docx` | Client-side via docx.js (browser) |
| **PDF generation** | docgen → DOCX → Gotenberg LibreOffice route → PDF | `blocksToFullHTML()` → backend → Gotenberg Chromium route → PDF |
| **Style consistency** | CSS vars in editor, separate rendering logic in docgen | Shared Layout IR + formal compatibility contract |
| **docgen service** | Required, problematic (restarts, ContentSource bugs) | **Eliminated entirely** |
| **Gotenberg** | Kept, serves DOCX→PDF conversion | Kept, switches to HTML→PDF (Chromium route) |
| **Template definition** | Loose `map[string]any` with theme | Formalized Layout IR with typed tokens |
| **Export endpoints** | `POST /documents/{id}/export/docx` (backend) | `POST /documents/{id}/render/pdf` only. DOCX is client-side. |
| **Read-only view** | Not formally defined | BlockNote editor in `readOnly` mode — same React components |

### Key principle

**DOCX is generated where the content lives — in the browser.** The frontend has the BlockNote blocks, the template definition, and the Layout IR. It produces the DOCX directly. The backend's only remaining role in export is converting HTML to PDF via Gotenberg.

### Leveraging BlockNote's built-in infrastructure

BlockNote's MIT-licensed core already provides:

| Feature | Use |
|---------|-----|
| `blocksToFullHTML()` | Full-fidelity HTML export — base for PDF pipeline |
| `toExternalHTML` on custom blocks | Define how each MDDM block serializes to HTML |
| `ServerBlockNoteEditor` | Headless processing (available for future server-side needs) |
| Custom schema mappings | Block props, types, content handling — already in use |

The AGPL wall only blocks `@blocknote/xl-docx-exporter` and `@blocknote/xl-pdf-exporter`. Everything in core BlockNote is MIT and fully available.

## Components

### 1. Layout IR (Layout Intermediate Representation)

The Layout IR is the single source of truth for all visual properties. Both React and docx.js read from it. It lives in a TypeScript module consumed by all renderers.

#### Structure

```typescript
// layout-ir/tokens.ts

export type LayoutTokens = {
  page: {
    widthMm: 210;          // A4
    heightMm: 297;
    marginTop: 25;         // mm
    marginRight: 20;
    marginBottom: 25;
    marginLeft: 25;
    contentWidthMm: 165;   // 210 - 25 - 20
  };

  typography: {
    editorFont: "Inter";          // editor-only
    exportFont: "Carlito";        // DOCX + PDF (metric-compatible with Calibri, open-source)
    baseSizePt: 11;
    headingSizePt: { h1: 18; h2: 15; h3: 13 };
    lineHeightPt: 15;             // absolute pt value, not unitless ratio
    labelSizePt: 9;
  };

  spacing: {
    sectionGapMm: 6;
    fieldGapMm: 3;
    blockGapMm: 2;
    cellPaddingMm: 2;
  };

  theme: {                    // from template definition
    accent: string;           // e.g., "#6b1f2a"
    accentLight: string;
    accentDark: string;
    accentBorder: string;
  };
};
```

#### Component Layout Rules

Each MDDM block gets a structured layout rule, not just colors:

```typescript
// layout-ir/components.ts

export type SectionLayout = {
  headerHeightMm: 8;
  headerBackground: "theme.accent";
  headerFontColor: "#ffffff";
  headerFontSizePt: 13;
  headerFontWeight: "bold";
  fullWidth: true;
};

export type FieldLayout = {
  labelWidthPercent: 35;
  valueWidthPercent: 65;
  labelBackground: "theme.accentLight";
  labelFontSizePt: 9;
  borderColor: "theme.accentBorder";
  borderWidthPt: 0.5;
  minHeightMm: 7;
};

export type FieldGroupLayout = {
  columns: 1 | 2;
  gapMm: 0;
  fullWidth: true;
};

export type DataTableLayout = {
  headerBackground: "theme.accentLight";
  headerFontWeight: "bold";
  cellBorderColor: "theme.accentBorder";
  cellPaddingMm: 2;
  density: "normal" | "compact";
};

// Plus rules for Repeatable, RepeatableItem, RichBlock
```

#### How renderers consume it

- **React**: tokens converted to CSS variables + component props. `labelWidthPercent: 35` → `grid-template-columns: 35% 65%`.
- **docx.js**: tokens converted to OOXML values. `labelWidthPercent: 35` → `TableCell` width = 35% of page content width. `headerHeightMm: 8` → `TableRow` height in twips.

### 2. DOCX Emitters (docx.js)

One emitter per block type. Each emitter reads the Layout IR and produces docx.js elements.

```typescript
// docx-emitter/emitter.ts

export interface BlockEmitter {
  emit(block: MDDMBlock, tokens: LayoutTokens, context: EmitContext): DocxElement[];
}

const emitters: Record<string, BlockEmitter> = {
  // MDDM custom blocks
  section:        new SectionEmitter(),
  field:          new FieldEmitter(),
  fieldGroup:     new FieldGroupEmitter(),
  repeatable:     new RepeatableEmitter(),
  repeatableItem: new RepeatableItemEmitter(),
  richBlock:      new RichBlockEmitter(),
  dataTable:      new DataTableEmitter(),
  dataTableRow:   new DataTableRowEmitter(),
  dataTableCell:  new DataTableCellEmitter(),

  // Standard BlockNote blocks
  paragraph:        new ParagraphEmitter(),
  heading:          new HeadingEmitter(),
  bulletListItem:   new BulletListEmitter(),
  numberedListItem: new NumberedListEmitter(),
  image:            new ImageEmitter(),
  quote:            new QuoteEmitter(),
  divider:          new DividerEmitter(),
};

// Main export entry point
export async function mddmToDocx(
  envelope: MDDMEnvelope,
  tokens: LayoutTokens
): Promise<Blob> {
  const doc = new Document({
    sections: [{
      properties: {
        page: {
          size: {
            width: mmToTwip(tokens.page.widthMm),
            height: mmToTwip(tokens.page.heightMm),
          },
          margin: {
            top:    mmToTwip(tokens.page.marginTop),
            right:  mmToTwip(tokens.page.marginRight),
            bottom: mmToTwip(tokens.page.marginBottom),
            left:   mmToTwip(tokens.page.marginLeft),
          },
        },
      },
      children: envelope.blocks.flatMap(block =>
        emitters[block.type].emit(block, tokens, { depth: 0 })
      ),
    }],
  });

  return Packer.toBlob(doc);
}
```

**Total emitters**: 16 (9 MDDM + 7 standard BlockNote).

**Everything is a table in DOCX.** OOXML has no Grid or Flexbox — tables are the only layout primitive for side-by-side content, colored backgrounds, and borders. This is how Word itself works internally. See Section 8 (Render Compatibility Contract) for forbidden constructs and degradation rules.

#### Block-to-DOCX mapping summary

| MDDM Block | React (editor) | DOCX (docx.js) |
|---|---|---|
| **Section** | Colored header `<div>` | `Table` — 1 row, 1 cell, accent background, white bold text |
| **Field** | CSS Grid 35%/65% | `Table` — 1 row, 2 cells (label shaded, value white) |
| **FieldGroup** | CSS Grid container | `Table` — rows of nested Field tables |
| **Repeatable** | Container with gradient header | `Table` — header row + child rows |
| **RepeatableItem** | Bordered container | `Table` — bordered cell group with left accent border |
| **RichBlock** | Labeled container | `Table` — optional label row + content rows |
| **DataTable** | CSS Grid with columns | `Table` — header row + data rows, explicit column widths |
| **DataTableRow** | Grid row | `TableRow` |
| **DataTableCell** | Grid cell with inline content | `TableCell` with `Paragraph` + text runs |

### 3. `toExternalHTML` hooks on custom blocks

Each MDDM custom block implements `toExternalHTML` to control its full-fidelity HTML serialization. This is consumed by `blocksToFullHTML()` and fed to Gotenberg's Chromium route for PDF.

```typescript
// Example: Section block

export const Section = createReactBlockSpec(
  {
    type: "section",
    propSchema: { title: { default: "" }, color: { default: "red" } },
    content: "none",
  },
  {
    render: ({ block }) => (
      <div className={styles.sectionHeader} style={{ background: accent }}>
        <span className={styles.sectionTitle}>{block.props.title}</span>
      </div>
    ),
    toExternalHTML: ({ block }) => (
      // Semantic HTML with inline styles for print.
      // Uses table layout (allowed by contract) instead of flexbox (forbidden).
      <table
        className="mddm-section-header"
        data-mddm-block="section"
        style={{
          width: "100%",
          borderCollapse: "collapse",
        }}
      >
        <tbody>
          <tr>
            <td
              style={{
                background: "var(--mddm-accent)",
                height: "8mm",
                color: "#ffffff",
                fontWeight: "bold",
                fontSize: "13pt",
                padding: "0 4mm",
                verticalAlign: "middle",
              }}
            >
              {block.props.title}
            </td>
          </tr>
        </tbody>
      </table>
    ),
  }
);
```

All 9 MDDM blocks get `toExternalHTML` implementations. Standard BlockNote blocks already have built-in HTML serialization.

### 4. Print stylesheet for Chromium

A dedicated CSS file injected into the HTML sent to Gotenberg:

```css
/* mddm-print.css */

@page {
  size: A4;
  margin: 25mm 20mm 25mm 25mm;
}

body {
  /* Carlito: open-source, metric-compatible with Calibri, available on
     Linux/Gotenberg/Chromium. Fallback to Liberation Sans / system sans. */
  font-family: "Carlito", "Liberation Sans", sans-serif;
  font-size: 11pt;
  line-height: 15pt;
  -webkit-print-color-adjust: exact;
  print-color-adjust: exact;
  font-kerning: normal;
  font-feature-settings: "liga" 1, "kern" 1;
  font-synthesis: none;
}

/* Page break rules */
.mddm-section-header,
.mddm-field {
  page-break-inside: avoid;
}

.mddm-repeatable-item {
  page-break-inside: avoid;
}

.mddm-data-table-row {
  page-break-inside: avoid;
}

/* Hide editor-only chrome */
.bn-side-menu,
.bn-formatting-toolbar,
.bn-slash-menu {
  display: none !important;
}
```

### 5. Backend PDF endpoint

A single new endpoint on the Go backend. Its only job is to forward HTML to Gotenberg and stream the PDF back.

```go
// internal/modules/documents/delivery/http/handler_render.go

func (h *Handler) handleDocumentRenderPDF(w http.ResponseWriter, r *http.Request, documentID string) {
    // Read multipart form: index.html + fonts/styles
    // Forward to Gotenberg /forms/chromium/convert/html
    // Stream PDF response back to client
}
```

```
POST /documents/{documentId}/render/pdf
Content-Type: multipart/form-data

Parts:
  - index.html   (main HTML document)
  - style.css    (optional additional styles)

Response:
  Content-Type: application/pdf
  (binary PDF body)
```

The old `POST /export/docx` endpoint is removed. `generateBrowserDocxBytesWithTemplate`, the docgen client, and all `mddm-docx` rendering code are deleted from the backend.

### 6. Frontend export functions

```typescript
// frontend/apps/web/src/features/documents/mddm-editor/export.ts

export async function exportDocx(envelope: MDDMEnvelope, tokens: LayoutTokens): Promise<Blob> {
  return mddmToDocx(envelope, tokens);
}

export async function exportPdf(
  blocks: BlockNoteBlock[],
  tokens: LayoutTokens,
  documentId: string
): Promise<Blob> {
  const html = await editor.blocksToFullHTML(blocks);
  const fullHtml = wrapInPrintDocument(html, tokens);

  const formData = new FormData();
  formData.append("index.html", new Blob([fullHtml], { type: "text/html" }));
  formData.append("style.css", new Blob([printStylesheet], { type: "text/css" }));

  const response = await fetch(`/documents/${documentId}/render/pdf`, {
    method: "POST",
    body: formData,
  });

  if (!response.ok) throw new Error("PDF render failed");
  return await response.blob();
}
```

### 7. MDDM Viewer component

A new React component that renders MDDM content read-only, using the same BlockNote editor underneath:

```typescript
// frontend/apps/web/src/features/documents/mddm-editor/MDDMViewer.tsx

export function MDDMViewer({ blocks, theme }: MDDMViewerProps) {
  return (
    <MDDMEditor
      initialContent={blocks}
      theme={theme}
      readOnly={true}
      // No onChange — read-only
    />
  );
}
```

The viewer is the instant, primary "see the document" experience for non-editors. PDF is available as an explicit action, not the default view.

### 8. Feature flag

```typescript
// frontend/apps/web/src/features/featureFlags.ts

export const featureFlags = {
  MDDM_NATIVE_EXPORT: boolean,  // Controls rollout of the new engine
};
```

Read once per page load from a config endpoint. When `false`, exports go through the old docgen backend path. When `true`, exports go through the new client-side engine.

### 9. Asset Resolution Contract

Images, and any other external assets referenced from MDDM content, need different handling per renderer because authentication and sandbox contexts differ.

#### The problem

MDDM documents reference images via URLs like `/api/images/{uuid}`. These URLs:
- Work in the **editor** because the browser sends session cookies automatically
- Work in the **viewer** for the same reason
- **Fail silently in PDF export** because Gotenberg's Chromium container has no auth session
- **Fail in DOCX export** because docx.js needs binary image bytes, not URL references

#### The contract

Every asset referenced by an MDDM block must be resolvable to **inline bytes** before entering any export pipeline. The `AssetResolver` is a new frontend module:

```typescript
// mddm-editor/asset-resolver.ts

export interface AssetResolver {
  // Fetches an asset using the authenticated session and returns its bytes
  // plus metadata needed for embedding.
  resolveAsset(url: string): Promise<ResolvedAsset>;
}

export type ResolvedAsset = {
  bytes: Uint8Array;
  mimeType: "image/png" | "image/jpeg" | "image/gif" | "image/webp";
  widthPx: number;
  heightPx: number;
  sizeBytes: number;
};
```

#### Behavior per renderer

| Renderer | Asset handling |
|----------|---------------|
| **Editor** | URLs are used directly. Browser session handles auth. |
| **Viewer** | Same as editor. |
| **DOCX export** | `mddmToDocx` walks the block tree, calls `AssetResolver.resolveAsset()` for every image URL, and embeds the bytes directly into the `docx.Document` via `ImageRun`. No URLs in the final DOCX. |
| **PDF export** | `exportPdf` walks the block tree before calling `blocksToFullHTML()`, resolves every asset, and rewrites each `src` attribute to a `data:` URI with inline base64. No external URLs in the HTML sent to Gotenberg. |

#### Allowed asset types

- **Images**: `image/png`, `image/jpeg`, `image/gif`, `image/webp`
- **Prohibited**: `image/svg+xml` (security risk from embedded scripts), video, audio, iframes, external scripts/styles

#### Resource ceilings (per asset)

| Limit | Value | Behavior on violation |
|-------|-------|----------------------|
| Max individual image size | 5 MB | Hard failure: abort export, show error toast |
| Max total asset payload per document | 50 MB | Hard failure: abort export, show error toast with suggestion to reduce image count/sizes |
| Max image dimensions | 10000 × 10000 px | Hard failure: abort export |
| Max images per document | 200 | Hard failure: abort export |

#### Security

- The asset resolver is allowlisted: it only fetches URLs matching `^/api/images/[a-f0-9-]+$` (the MetalDocs image endpoint pattern). Arbitrary external URLs are rejected.
- The resolver strips EXIF metadata from JPEGs before embedding (privacy).
- Fetched bytes are validated against the declared MIME type via magic-number inspection (not just Content-Type header).

### 10. Version Pinning for Released Documents

Released documents — documents that have passed approval and entered distribution — must render identically forever, even as the Layout IR and emitters evolve.

#### The problem

The Layout IR and emitters are code. When we update them (bug fix, new feature, refinement), every future export changes. A released PO from 6 months ago would re-render with new token values, different emitter logic, and potentially different page break positions. This is unacceptable for corporate/compliance documents that serve as "what was approved."

#### The contract

Every document version stores a **renderer pin**: a tuple of `(renderer_version, layout_ir_hash, template_ref_locked)` captured at the moment the version transitions from DRAFT to RELEASED.

```typescript
// New fields on Version domain model
type Version = {
  // ... existing fields ...
  rendererPin?: {
    rendererVersion: string;        // e.g., "1.0.0", bumped on any emitter change
    layoutIRHash: string;           // sha256 of the serialized layout-ir module at pin time
    templateRefLocked: {
      templateKey: string;
      templateVersion: number;      // explicit pinned version, not "latest"
    };
    pinnedAt: string;               // ISO timestamp
  };
};
```

#### Pinning rules

- **DRAFT versions**: no pin. Exports always use the current (`HEAD`) renderer, IR, and template version. Authors see their evolving edits rendered with the latest engine.
- **Pending approval**: no pin yet.
- **Released / Approved**: **pin immediately** on transition. From this moment, any export of this version uses the pinned renderer, IR, and template.
- **Archived**: still uses the pin captured at release time.

#### Runtime behavior

```typescript
export async function exportDocx(envelope: MDDMEnvelope, version: Version): Promise<Blob> {
  const { tokens, emitters } = version.rendererPin
    ? loadPinnedRenderer(version.rendererPin)  // reads historical bundle from /renderers/{rendererVersion}/
    : loadCurrentRenderer();                    // reads current src/layout-ir

  return mddmToDocx(envelope, tokens, emitters);
}
```

#### Implementation approach

The frontend ships with a **renderer bundle registry**: historical Layout IR snapshots + emitter code are preserved under `frontend/apps/web/src/features/documents/mddm-editor/renderers/{version}/`. When a released document needs export, the appropriate bundle is loaded.

**Practical limit**: keep the most recent 10 renderer versions. Older pins fall back to the oldest retained version with a warning in telemetry. For truly archival documents, the stored DOCX (from the last export at release time) can be retained separately — but this is deferred to a later project (not in scope here).

#### Renderer version bump rules

A `rendererVersion` bump is required when any of the following changes:
- Layout IR token values (dimensions, colors, font sizes)
- Emitter output structure (new OOXML elements, changed attributes)
- Print stylesheet structure affecting layout
- Asset resolution behavior
- Forbidden constructs list

A version bump is **not** required for:
- Pure bug fixes that restore intended behavior (the old behavior was a bug, not a spec)
- Performance improvements that don't change output
- Tooling/test changes

### 11. Export State Contract

**Rule**: Export operations (DOCX and PDF) always run against the **last persisted state**, not the current in-memory editor state.

#### Flow

```
User clicks [Exportar DOCX] or [Exportar PDF]
    ↓
if (isDirty) {
    showSaveBeforeExportDialog()
    // Options:
    //   - "Salvar e exportar" (save then export)
    //   - "Exportar versão salva" (export last saved state, discarding unsaved changes for export)
    //   - "Cancelar"
} else {
    proceedToExport(lastSavedEnvelope)
}
```

#### Rationale

- **Traceability**: the exported file always corresponds to a persisted version on the backend. Audit trails and document signatures reference the stored version hash.
- **Compliance**: regulated documents must match exactly what was saved and approved. Exporting unsaved state would create "ghost" documents that don't exist in the database.
- **Released documents**: if a document is released and the user somehow has unsaved edits (e.g., in a review session), the export must use the released version, not the uncommitted edits.

#### Default dialog behavior

- **Draft documents with unsaved edits**: default to "Salvar e exportar" (user clicked export, so they want the current state out)
- **Released documents with unsaved edits**: default to "Exportar versão salva" (released state is immutable; any local edits are unauthorized by definition)

### 12. Canonicalize + Migrate Pipeline

Every MDDM envelope passes through a canonicalization and migration step before any rendering (editor, viewer, DOCX, PDF). This ensures older documents with outdated `mddm_version` values render correctly with the current engine.

```typescript
// mddm-editor/pipeline.ts

export async function canonicalizeAndMigrate(envelope: MDDMEnvelope): Promise<MDDMEnvelope> {
  const canonical = canonicalize(envelope);        // stable key order, normalized whitespace
  const migrated = await migrate(canonical);       // applies version-upgrade transforms
  validate(migrated);                               // schema validation; throws on corruption
  return migrated;
}
```

#### Migration chain

```
mddm_version: 1 → 2: adds `layout` prop to Field blocks
mddm_version: 2 → 3: renames `legacyTitle` to `title` on Section blocks
mddm_version: 3 → current: ...
```

Each migration is a pure function `(envelope) => envelope` registered in a migration registry. The pipeline applies them in sequence until the envelope matches the current `mddm_version`.

#### Failure behavior

- **Canonicalization failure**: hard error, show toast "Documento corrompido. Contate o suporte."
- **Migration failure** (version gap the registry can't bridge): hard error, log to telemetry with document ID and version, show toast "Este documento usa um formato não suportado por esta versão."
- **Validation failure** (post-migration schema mismatch): hard error, log with full diff, show toast

The pipeline runs **before** the renderer pin is applied: we always migrate to the pinned `mddm_version` (stored in the pin), not to the current engine's version, so released documents remain frozen.

### 13. Renderer Completeness Gate

Before any block type can be enabled for user editing, it must pass a **renderer completeness gate** — a CI check that verifies the block has working implementations for all three render targets.

```typescript
// tests/contract/renderer-completeness.test.ts

const REQUIRED_RENDERERS = ["render", "toExternalHTML", "docxEmitter"] as const;

for (const [blockType, spec] of Object.entries(customBlockSpecs)) {
  describe(`Block "${blockType}" completeness`, () => {
    it("has React render() implementation", () => {
      expect(spec.render).toBeDefined();
    });

    it("has toExternalHTML() implementation", () => {
      expect(spec.toExternalHTML).toBeDefined();
    });

    it("has docxEmitter registration", () => {
      expect(docxEmitters[blockType]).toBeDefined();
    });

    it("has at least one golden fixture exercising this block", () => {
      const fixtures = findGoldenFixturesUsingBlock(blockType);
      expect(fixtures.length).toBeGreaterThan(0);
    });

    it("passes roundtrip: render → toExternalHTML → re-parse produces equivalent structure", async () => {
      const block = makeFixture(blockType);
      const html = await renderToExternalHTML(block);
      const parsed = await blockNoteParseHTML(html);
      expect(normalizeBlock(parsed)).toEqual(normalizeBlock(block));
    });
  });
}
```

This test runs in CI on every PR. A new block type **cannot be merged** to main unless it has all three renderers, a golden fixture, and passes the roundtrip test. This prevents the "looks great in the editor, disappears in PDF" class of bug.

### 14. Global Resource Ceilings

Hard limits enforced at the pipeline level. Exceeding any limit aborts the export with a user-visible error.

| Limit | Value | Applies to |
|-------|-------|-----------|
| Max MDDM block count per document | 5000 | All exports |
| Max nesting depth | 10 levels | All exports |
| Max single text run length | 100,000 chars | All exports |
| Max DOCX generation time (client-side) | 30s | DOCX export (timeout, runs in Web Worker) |
| Max HTML payload size sent to Gotenberg | 10 MB | PDF export |
| Max Gotenberg conversion time | 60s | PDF export (server-side timeout) |
| Max concurrent exports per user | 3 | Frontend rate limit |
| Max image bytes per document (total) | 50 MB | DOCX + PDF (see Asset Resolution Contract) |
| Max single image size | 5 MB | DOCX + PDF |

When any limit is exceeded, the export fails fast with a specific error message telling the user what to do ("Your document has too many images. Please reduce to under 200.").

## Data Flow

### Editor → Save

```
User edits in MDDM editor
    ↓
onChange fires with new BlockNote blocks
    ↓
blockNoteToMDDM(blocks) → MDDMEnvelope JSON
    ↓
POST /documents/{id}/content/browser { body: JSON, draftToken }
    ↓
Backend stores MDDM JSON in version.Content
    ↓
Returns new draftToken
```

**What's stored**: MDDM envelope JSON only. No DOCX, no PDF, no pre-rendering. The MDDM JSON in the database is the sole source of truth.

### Viewer → Read-only display

```
User navigates to /documents/{id}/view
    ↓
GET /documents/{id}/browser-editor-bundle
    ↓
Frontend receives bundle.body (MDDM JSON) + bundle.templateSnapshot
    ↓
canonicalizeAndMigrate(envelope) → envelope at current mddm_version
    ↓
mddmToBlockNote(envelope) → BlockNoteBlocks
    ↓
<MDDMViewer blocks={blocks} theme={bundle.templateSnapshot.definition.theme} />
    ↓
Instant render (no external rendering pipeline)
```

Latency: **~instant** — one HTTP round-trip, then a pure in-memory JS transform (`canonicalizeAndMigrate` + `mddmToBlockNote`) that takes single-digit milliseconds even for large documents. No PDF/DOCX rendering, no server-side conversion, no network I/O beyond the initial bundle fetch. The MDDM→BlockNote transform is a trivial shape mapping (not a rendering operation).

### Export DOCX

```
User clicks [Exportar DOCX]
    ↓
Export State Contract check (Section 11)
    ├── isDirty? → showSaveBeforeExportDialog()
    └── proceed with lastSavedEnvelope
    ↓
canonicalizeAndMigrate(envelope) → envelope at pinned or current mddm_version
    ↓
loadRenderer(version.rendererPin ?? currentRenderer) → { tokens, emitters }
    ↓
Asset Resolution: walk blocks, resolve every image URL to bytes
    ├── assetResolver.resolveAsset(url) for each image
    ├── check resource ceilings (max per image, max total)
    └── build assetMap: { url → ResolvedAsset }
    ↓
mddmToDocx(envelope, tokens, emitters, assetMap)
    ├── emitters for each block produce docx.js elements
    ├── ImageEmitter embeds assetMap bytes inline (no URLs in output)
    ├── Document(sections: [{ children: [...] }])
    └── Packer.toBlob(doc)    [runs in Web Worker]
    ↓
Blob (~60-100KB for text-only, larger with images)
    ↓
Trigger download (<a href="blob:..." download="PO-182.docx">)
```

Latency: **~1s** on typical hardware for text-only documents; asset resolution adds ~200-500ms per image. Large documents (>50 pages) run in a Web Worker to avoid blocking the UI thread. Hard timeout at 30s — see Resource Ceilings (Section 14).

### Export PDF

```
User clicks [Exportar PDF]
    ↓
Export State Contract check (Section 11)
    ├── isDirty? → showSaveBeforeExportDialog()
    └── proceed with lastSavedEnvelope
    ↓
canonicalizeAndMigrate(envelope) → envelope at pinned or current mddm_version
    ↓
loadRenderer(version.rendererPin ?? currentRenderer) → { tokens, printStylesheet }
    ↓
Asset Resolution: walk blocks, resolve every image URL to bytes + base64
    ├── assetResolver.resolveAsset(url) for each image
    ├── check resource ceilings
    └── build assetMap with data: URIs
    ↓
Apply canonicalized envelope to a headless BlockNote instance
    ↓
html = blocksToFullHTML(blocks) with src attrs rewritten to assetMap data: URIs
    ↓
fullHtml = wrapInPrintDocument(html, tokens, printStylesheet)
    (injects @page rules, font-face for Carlito, layout vars)
    ↓
check fullHtml size ≤ 10MB limit
    ↓
POST /documents/{id}/render/pdf (multipart form: index.html + style.css)
    ↓
Backend receives multipart form, applies request size limit
    ↓
Backend forwards to Gotenberg /forms/chromium/convert/html
    (60s server-side timeout)
    ↓
Gotenberg launches Chromium, renders HTML (all assets inline — no network fetch),
prints to PDF
    ↓
Gotenberg returns PDF bytes
    ↓
Backend streams PDF back to frontend
    ↓
Frontend receives Blob
    ├── Download intent: trigger file download
    └── View intent: window.open(blobUrl) → inline browser PDF viewer
```

Latency: **~2-3s** (mostly Gotenberg conversion time). Asset resolution adds ~100-200ms per image before the request. Spinner shown during the full flow.

**Critical**: the HTML sent to Gotenberg has **zero external URL references**. All images are inline `data:` URIs, fonts are served from Carlito installed in the Gotenberg container, and the stylesheet is passed as a separate multipart part. This eliminates the auth/network failure class entirely.

## Error Handling

### DOCX generation errors (client-side)

| Scenario | Handling |
|----------|----------|
| Malformed MDDM JSON | Hard failure: show error toast, log to telemetry, do not export |
| Missing block emitter (unknown block type) | Soft failure: skip block, log warning, include placeholder text `[unsupported block: <type>]` |
| docx.js internal error | Hard failure: show error toast "Falha ao gerar DOCX. Tente novamente.", log stack trace |
| Timeout (>30s) | Hard failure: show error toast, abort generation |

### PDF render errors (server-side)

| Scenario | Handling |
|----------|----------|
| Backend PDF endpoint returns 5xx | Show toast "Serviço de PDF indisponível.", offer DOCX as fallback |
| Gotenberg timeout | Retry once; if still failing, show error toast |
| Network error | Show retry dialog |

### Degraded content (contract violations)

When a block contains a construct that violates the compatibility contract (see Section 8.7):

1. The emitter falls back to a simplified rendering
2. A warning is logged to the console and telemetry
3. A toast notification appears: "Seu documento contém elementos que foram simplificados no DOCX. Veja o console para detalhes."
4. Export completes successfully with the degraded content

**Content is never silently dropped.** Text is always preserved; only layout is simplified.

## Testing Approach

Three layers of tests, each catching a different class of regression:

### 1. Unit tests (Vitest)

One test file per emitter (16 total). Each test verifies the emitter produces correct OOXML for a given block input + Layout IR.

```typescript
describe("SectionEmitter", () => {
  it("produces a full-width table with accent background", () => {
    const block = makeSection({ title: "1. Procedimento", color: "red" });
    const tokens = makeTokens({ accent: "#6b1f2a" });

    const [docxElement] = sectionEmitter.emit(block, tokens, { depth: 0 });
    const xml = docxElement.toXml();

    expect(xml).toContain('<w:shd w:fill="6B1F2A"');
    expect(xml).toContain('<w:tblW w:w="5000" w:type="pct"');
    expect(xml).toContain('1. Procedimento');
    expect(xml).toContain('<w:color w:val="FFFFFF"/>');
  });

  it("respects multi-line titles", () => { /* ... */ });
  it("falls back to default color when theme missing", () => { /* ... */ });
});
```

### 2. Golden file tests

Reference documents covering every block type and feature combination. Each fixture has approved artifacts committed to the repo:

```
frontend/apps/web/src/features/documents/mddm-editor/__golden__/
  ├── 01-simple-po/
  │   ├── input.mddm.json
  │   ├── expected.full.html
  │   ├── expected.docx.xml
  │   └── expected.snapshot.png
  ├── 02-complex-table/
  ├── 03-repeatable-sections/
  ├── 04-all-inline-styles/
  ├── 05-multi-page-long-doc/
  ├── 06-theme-override/
  └── 07-legacy-migration/
```

Test run compares actual output to expected output byte-exactly (after XML normalization). Any drift fails the test. Regenerating the golden files requires explicit developer approval (`npm run test:golden:update`) and manual diff review before committing.

### 3. Visual parity tests (Playwright)

End-to-end tests that screenshot the editor and diff it against the rasterized PDF:

```typescript
test("simple-po: editor and PDF screenshots match within 2%", async ({ page }) => {
  await page.goto(`/test-harness/mddm?doc=01-simple-po`);
  const editorScreenshot = await page.locator(".mddm-editor").screenshot();

  await page.click("text=Exportar PDF");
  const pdfBlob = await page.waitForEvent("download");
  const pdfPng = await rasterizePdfFirstPage(pdfBlob);

  const diffPercent = pixelmatch(editorScreenshot, pdfPng, { threshold: 0.1 });
  expect(diffPercent).toBeLessThan(0.02);
});
```

**Acceptance thresholds** (enforced in CI):
- Editor vs PDF: **< 2%** pixel diff
- Editor vs DOCX (via LibreOffice → PNG): **< 5%** pixel diff
- Golden files: **byte-exact**

### Test harness page

A dedicated route `/test-harness/mddm?doc=<fixture>` loads golden fixtures in a clean MDDM editor environment with programmatic export APIs (`window.__mddmExportDocx()`, `window.__mddmExportPdf()`). Only enabled in non-production builds.

### Contract validator

A test-suite-time validator checks that:
- Emitters don't use forbidden OOXML constructs (see Section 8.7)
- Golden tests conform to the Tier 1 byte-exact guarantee
- Visual parity thresholds match the compatibility contract

## Font Strategy

### Decision: Open-source cross-platform fonts, no embedding

**Primary font**: **Carlito** — an open-source font specifically designed to be metric-compatible with Calibri (same character widths, same line break positions, same appearance at common sizes).

**Why not Calibri directly?** Calibri is a Microsoft proprietary font and is **NOT installed on Linux by default**. Gotenberg typically runs in a Linux container — if we reference Calibri in CSS, Chromium silently substitutes it with a fallback font, breaking wrap positions and fidelity. Carlito solves this: it is pre-installed in most Linux distributions (Debian/Ubuntu `fonts-crosextra-carlito` package), Gotenberg's official Docker images include it, and it renders identically to Calibri on Windows/Mac where Word opens the DOCX.

**Font fallback stack (consistent across all renderers)**:
```
"Carlito", "Liberation Sans", "Arial", sans-serif
```

All three fonts are metric-compatible with Calibri/Arial at the corporate-document level. Any recipient opening the DOCX sees visually identical output regardless of their OS.

**Rationale**:
- DOCX files stay small (~60-100KB vs ~600KB+ with embedding)
- Zero licensing concerns — Carlito, Liberation Sans, and Arial are all free to embed/distribute
- Zero font substitution surprises — every target platform has at least one font in the stack
- Universal compatibility — works on Windows (Arial), Mac (Arial), Linux editor machines (Liberation), Linux Gotenberg containers (Carlito)

### Gotenberg container requirement

The Gotenberg Docker image **must** include Carlito. Verification step in Phase 0:

```bash
docker exec <gotenberg-container> fc-list | grep -i carlito
# Expected: Carlito-Regular.ttf, Carlito-Bold.ttf, Carlito-Italic.ttf, Carlito-BoldItalic.ttf
```

If Carlito is missing, install it in the image via a Dockerfile layer:

```dockerfile
RUN apt-get update && apt-get install -y fonts-crosextra-carlito && rm -rf /var/lib/apt/lists/*
```

This verification is a gating check before Phase 1 rollout.

### Editor vs export font separation

The Layout IR allows different fonts for the editor display vs the exported output:

```typescript
typography: {
  editorFont: "Inter",                       // nice editing experience on screen
  exportFont: "Carlito",                     // universal cross-platform compatibility
  exportFontFallbacks: ["Liberation Sans", "Arial", "sans-serif"],
  embedInDocx: false,
}
```

Users see Inter while editing (crisp, modern on screen), and the exported DOCX/PDF uses Carlito with a metric-compatible fallback stack (universal, small files, Calibri-equivalent metrics).

### Font metric consistency

All three renderers use consistent font handling to minimize wrap-position drift:

- `font-kerning: normal`
- `font-feature-settings: "liga" 1, "kern" 1`
- `font-synthesis: none`

These settings go into both the editor CSS and the print stylesheet for PDF.

### Future: Optional per-template embedding

For a later phase, templates may opt in to custom font embedding:

```typescript
// Template definition
{
  theme: { accent: "#6b1f2a", ... },
  typography: {
    exportFont: "Inter",
    embedInDocx: true,    // opt-in embedding
  }
}
```

When `embedInDocx: true`, the docx.js emitter fetches the font file and embeds it in the DOCX archive. File size jumps to ~600KB but brand fonts are guaranteed across recipients.

## Render Compatibility Contract

This section formalizes the rules of engagement between the three renderers. It is the "constitution" of the document engine — what is guaranteed identical, what is tolerated, what is forbidden.

### Core principles

1. **Semantics before appearance** — the meaning of a document (which blocks, in what order, with what content) must be identical across all outputs. Visual rendering is secondary.
2. **Explicit over implicit** — every dimension, color, and spacing must be explicit in the Layout IR. No "browser default" is allowed.
3. **Absolute over relative** — prefer mm/pt over percentages/em.
4. **Fail loud** — unsupported constructs emit warnings + telemetry + user notifications, never silent drops.

### Three compatibility tiers

#### Tier 1 — MUST be identical (enforced)

Any divergence is a bug. Enforced by golden file tests, which compare **normalized** outputs (not raw file bytes). Normalization strips engine-specific metadata that isn't part of the contract: DOCX creation timestamps, PDF producer strings, HTML data attributes for editor-only use, OOXML revision IDs, and relative ordering of equivalent attributes.

| Property | Source | Comparison method |
|----------|--------|-------------------|
| Block structure (which blocks, in what order, parent/child relationships) | MDDM JSON | Byte-exact on canonicalized MDDM JSON |
| Block types | MDDM JSON | Byte-exact |
| Block props (label, columns, color, locked, variant, etc.) | MDDM JSON | Byte-exact |
| Text content | MDDM JSON | Byte-exact |
| Inline styles (bold, italic, underline, strike, code) | MDDM JSON | Byte-exact |
| Colors (accent, accentLight, accentDark, accentBorder) | Layout IR | Normalized XML/CSS equality |
| Font family (Carlito default or template override) | Layout IR | Normalized XML/CSS equality |
| Font size (pt) | Layout IR | Normalized XML/CSS equality |
| Column proportions (Field 35/65, FieldGroup, DataTable) | Layout IR | Normalized XML/CSS equality |
| Section header heights (8mm fixed) | Layout IR | Normalized XML/CSS equality |
| Page margins (mm) | Layout IR | Normalized XML/CSS equality |
| Border colors and widths | Layout IR | Normalized XML/CSS equality |

**Clarification on "byte-exact"**:
- **Content (MDDM JSON)** is compared byte-exactly after canonicalization (stable key order, normalized whitespace).
- **Output files (DOCX .xml, HTML)** are compared byte-exactly **after normalization** that strips Tier 3 metadata (timestamps, producer strings, etc).
- **Raw DOCX zip archive bytes** are NOT compared — they vary between docx.js runs even with identical input due to timestamps and compression metadata. The normalized `document.xml` inside the archive is the canonical comparison target.

#### Tier 2 — MAY differ within tolerance

Engine-specific behavior expected. Explicit tolerances defined and tested.

| Property | Tolerance |
|----------|-----------|
| Line wrap position | Text must be complete; wrap position may differ |
| Sub-pixel text positioning | ≤ 1 pixel horizontal drift per character |
| Kerning | Acceptable if font-kerning: normal is set consistently |
| Exact cell height on wrapped content | ≤ 3px vertical drift per cell |
| Page break position | Content complete; break position may differ |
| Table row split across pages | Rows not dropped or duplicated |
| Image scaling | Image appears; ≤ 2% size difference |

**Visual parity thresholds**:
- Editor vs PDF: **< 2%** overall pixel diff
- Editor vs DOCX (rasterized): **< 5%** overall pixel diff

#### Tier 3 — MAY differ freely

Engine-specific. Not part of the contract. Not tested.

- Editor chrome (cursor, selection, block handles, slash menu)
- Interaction states (hover, focus, active)
- Animation and transitions
- Scroll position (editor scrolls, PDF has pages, DOCX has flow)
- Internal metadata (DOCX creation timestamp, PDF producer string, HTML data attributes)

### Pagination ownership

| Output | Owner | Behavior |
|--------|-------|----------|
| Editor | N/A | Continuous scroll, no pages |
| PDF | **Chromium** | Chromium's HTML print engine decides page breaks based on CSS `page-break-*` hints + content height |
| DOCX | **Word/LibreOffice at open time** | docx.js writes page break hints; actual break decided by the reader |

**Contract rule**: Authors cannot manually place a page break in the editor that is honored by both PDF and DOCX. If hard page breaks become needed, they must be an explicit `pageBreak` block type with explicit `page-break-before: always` CSS + OOXML `w:br w:type="page"`.

### Forbidden constructs

| Forbidden | Why | Use instead |
|-----------|-----|-------------|
| Auto-fit table columns (`width: auto`) | CSS auto-layout and OOXML column resolution differ wildly | Explicit column widths in Layout IR |
| Unitless or `em`-based line-heights (`line-height: 1.4`, `line-height: 1.5em`) | Computed against different base sizes per engine | Absolute line-height in pt (e.g., `line-height: 15pt`) |
| Negative margins | Not representable in OOXML | Positive padding on adjacent elements |
| CSS Flexbox (`display: flex`, `display: inline-flex`) | No OOXML equivalent | CSS Grid or HTML tables (both map cleanly to OOXML tables) |
| CSS Grid with `fr` or `auto` tracks | Fractional units don't translate to OOXML | Explicit `mm` or `%` column widths |
| Nested DataTables deeper than 2 levels | Word's nested table rendering is unreliable | Flatten; use RepeatableItem grouping |
| Percentage-based font sizes (`font-size: 1.2em`) | Compounds differently per engine | Explicit pt values |
| Transforms (`transform: rotate(45deg)`) | No OOXML equivalent | Not supported |
| Filters (`filter: blur(...)`) | No OOXML equivalent | Not supported |
| Fixed/sticky positioning | Not representable in page-flow documents | Use normal flow |
| Viewport units (`100vh`, `50vw`) | Not a concept in paged layouts | mm or pt |
| External URLs in `<img src>` / `<link href>` during PDF export | Gotenberg has no auth; protected URLs fail silently | Inline as base64 via asset resolver (see Asset Resolution Contract) |

Enforcement:
1. **Lint rules** on MDDM editor CSS modules (stylelint)
2. **TypeScript types** reject forbidden values in the Layout IR
3. **Emitter assertions** log warnings and use safe fallbacks when forbidden input is encountered

### Degradation rules

When an emitter encounters a forbidden construct at runtime:

```typescript
function dataTableEmitter(block: DataTableBlock, tokens: LayoutTokens, depth = 0) {
  if (depth >= 2) {
    logWarning({
      code: "MDDM_NESTED_TABLE_TOO_DEEP",
      blockId: block.id,
      depth,
      message: "Nested DataTable flattened to preserve compatibility",
    });
    return renderAsFlatTable(block, tokens);
  }
  return renderAsNestedTable(block, tokens, depth + 1);
}
```

Rules:
1. **Never drop content silently** — degraded blocks still contain the author's text
2. **Log every degradation** — console warning + telemetry
3. **User notification** — toast: "Seu documento contém elementos que foram simplificados no DOCX. Veja o console para detalhes."
4. **Fail only on corruption** — malformed MDDM JSON is a hard failure; unsupported constructs are a soft degradation

### Contract enforcement module

```typescript
// layout-ir/compatibility-contract.ts

export const COMPATIBILITY_CONTRACT = {
  tier1: {
    blockStructure: "byte-exact",
    blockProps: "byte-exact",
    colors: "byte-exact",
    fontFamily: "byte-exact",
    columnProportions: "byte-exact",
  },
  tier2: {
    pixelDiffEditorToPdf: 0.02,
    pixelDiffEditorToDocx: 0.05,
    verticalCellDriftPx: 3,
    horizontalCharDriftPx: 1,
  },
  forbidden: {
    autoFitColumns: "error",
    percentageLineHeight: "error",
    negativeMargins: "error",
    nestedDataTableMaxDepth: 2,
    viewportUnits: "error",
  },
  degradation: {
    logLevel: "warn",
    telemetry: true,
    userNotification: "toast",
  },
} as const;
```

A validator runs as part of the test suite:

```typescript
validateContract(allEmitters, COMPATIBILITY_CONTRACT);
validateGoldenTestsConformContract(goldenFixtures, COMPATIBILITY_CONTRACT);
validateVisualParityThresholds(playwrightConfig, COMPATIBILITY_CONTRACT);
```

### Living contract

The compatibility contract is a living document. When we discover a new engine difference or add a new block type, we:

1. Document the new difference in the appropriate tier
2. Update golden fixtures to cover the edge case
3. Update the forbidden constructs list if needed
4. Review the contract in a design doc before shipping

This is the difference between "we tested some examples" and "we have a spec the engine conforms to."

## Migration & Rollout

### Constraints

1. **No data migration** — existing MDDM content stays as-is. The new engine reads the same MDDM envelope format.
2. **No disruption** — users continue editing and exporting throughout the migration.
3. **Reversible** — feature flag flips the new engine on/off in seconds.
4. **Parity-validated** — no flip to production until golden-file tests pass on real documents.

### Phased rollout

#### Phase 0 — Foundation (no user impact)

- Layout IR tokens + component rules module
- docx.js emitters for all 9 MDDM blocks + 7 standard blocks
- `toExternalHTML()` hooks on custom blocks
- Print stylesheet for Chromium PDF
- Asset resolver module + security allowlist
- Canonicalize + migrate pipeline
- Renderer version + pin storage (backend domain + schema)
- Gotenberg container Carlito verification (gating check)
- Renderer completeness gate CI test
- Resource ceiling enforcement
- Test harness page
- Unit tests + initial golden files

#### Phase 1 — Shadow testing (no user impact)

- Feature flag `MDDM_NATIVE_EXPORT = false` (docgen authoritative for the user-visible result)
- **Frontend**, on export click, runs both paths in parallel:
  1. Calls backend docgen via the existing `POST /export/docx` endpoint → gets the docgen DOCX (this is what the user receives and downloads)
  2. Runs the new client-side `mddmToDocx(envelope, tokens)` in the background → gets a "shadow" DOCX
- Frontend computes a structural diff between the two (normalized XML compare using the same normalization pipeline as golden tests)
- Frontend POSTs the diff summary (not the full DOCX bytes, only a hash + diff metrics + block coverage) to a new telemetry endpoint `POST /telemetry/mddm-shadow-diff`
- Shadow DOCX is discarded; user only sees the docgen result
- Backend aggregates diffs and exposes them on an internal dashboard
- Engineers use the dashboard to build the golden corpus from real production documents and iterate on emitters until diff rate is below threshold

**Why frontend-side**: the new DOCX emitter is client-side by design (see architecture). Running it on the backend would contradict the entire architecture and require a Node.js-based sidecar we've committed to not building. The browser is the only place both paths can run simultaneously.

**Performance safety**: shadow DOCX generation runs in a Web Worker with a 30-second timeout and is skipped entirely on low-memory devices (< 4GB RAM detected via `navigator.deviceMemory`). Shadow failures never affect the user-visible export.

#### Phase 2 — Canary (5% of users)

- Feature flag enabled for 5% of users
- Those users get the new client-side DOCX + Chromium PDF path
- Monitor error rates, user complaints, export success rates
- Instant rollback if issues emerge

#### Phase 3 — Full rollout

- Feature flag enabled for all users
- Keep docgen running as fallback for 2 weeks
- Monitor closely
- Confirm no regressions

#### Phase 4 — Decommission

- Remove docgen from infrastructure
- Delete docgen service code + Docker image
- Remove `POST /documents/{id}/export/docx` backend endpoint
- Delete `generateBrowserDocxBytesWithTemplate`, `docgen.Client`, related backend code
- Remove feature flag
- Update documentation

### Handling legacy content sources

The new engine only activates for `ContentSource = "browser_editor"` documents. `native` and `docx_upload` content sources continue using the backend path indefinitely (migrating them is out of scope).

```typescript
if (version.contentSource === "browser_editor" && featureFlags.MDDM_NATIVE_EXPORT) {
  // New client-side path
} else {
  // Old backend path (for native, docx_upload, or flag disabled)
}
```

### Rollback plan

1. **Immediate** (< 1 min): Flip `MDDM_NATIVE_EXPORT = false` in config. All users revert to docgen on next page load.
2. **Short-term**: Fix on main branch, redeploy.
3. **If docgen has been removed** (Phase 4+): git revert the deletion commits, redeploy docgen. This is why docgen runs for 2 weeks after Phase 3 as a safety net.

### Risks & mitigations

| Risk | Mitigation |
|------|-----------|
| Golden tests miss a real-world edge case | Shadow testing in Phase 1 catches production-only edge cases |
| Client-side docx.js slow on large documents | Web Worker, progress indicator, 30s timeout |
| Chromium PDF differs from expected on specific font | System fonts only (Calibri) — universal |
| Gotenberg outage | Already in production; no new operational risk |
| User hits export during feature flag flip | Flag read once per page load; changes apply on refresh |

### Success metrics

- **Error rate** on DOCX export: ≤ current docgen baseline (~2-3%)
- **Export latency**: p95 DOCX ≤ 3s, p95 PDF ≤ 5s
- **User-reported issues**: zero regressions in first month post-rollout
- **Golden file drift**: zero (any drift requires explicit approval)

## Source of Truth Clarification

The phrase "MDDM JSON in the database is the single source of truth" applies specifically to **document content** — the author-created text, blocks, and structure. The complete rendering of a document is determined by **content + renderer context**:

```
Rendered Document = MDDM JSON (content)
                  + Template Definition (structure, theme override)
                  + Layout IR tokens (dimensions, fonts, colors)
                  + Emitter code (block-to-output logic)
                  + Renderer Pin (for released docs, freezes the above)
```

For **DRAFT** documents: content is persisted, everything else is sourced from the current engine (latest tokens, latest emitters, latest template version).

For **RELEASED** documents: content is persisted, and the other inputs are pinned via the Renderer Pin (Section 10). The content itself never changes after release.

This means "single source of truth for content" is accurate; "single source of truth for the rendered output" is only true when combined with the pinned renderer context. The spec uses "source of truth" in the content sense throughout.

## Out of Scope

- Migrating `native` or `docx_upload` content sources to MDDM (separate project)
- Batch or server-side export
- Real-time collaboration
- Offline PDF generation (Gotenberg required)
- Custom font embedding at launch (deferred to template-level opt-in in a later phase)
- Replacing Gotenberg with a WASM-based PDF renderer (experimental, not suitable for corporate-grade output)
- Rewriting the MDDM editor itself (existing BlockNote-based editor is kept)
- Changes to the MDDM envelope format or schema (unchanged — this project reads the existing format)
- Persistent archival storage of exported DOCX/PDF files for released documents (the Renderer Pin + re-generation covers most cases; true cold-archival is a separate concern)
- Supporting SVG images in exports (security review needed; PNG/JPEG only at launch)
- Migration of more than 10 historical renderer versions (the retention cap is explicit)
