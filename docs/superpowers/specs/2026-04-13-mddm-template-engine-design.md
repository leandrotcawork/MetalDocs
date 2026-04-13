# MDDM Template Engine Design

## Goal

Replace the hardcoded Go template system with a declarative template engine that defines document structure as MDDM blocks with capabilities, style overrides, and typed codecs. The engine uses shared Layout Interpreters to produce ViewModels consumed by both React and DOCX emitters — guaranteeing visual parity from a single source of truth.

### Success criteria

- Templates are pure declarative JSON — no code, no Go functions, no logic in the template itself
- Every MDDM block type has: typed codec + Layout Interpreter + ViewModel + React emitter + DOCX emitter
- React `render()` and DOCX emitter call the SAME `interpret()` function — ViewModel is the parity contract
- Template capabilities (locked/editable/fixed/dynamic) enforced at the UI level and validated on save
- DataTable supports two modes: fixed (fill-only form) and dynamic (add/remove rows)
- Phase 1 templates defined in TypeScript, Phase 2 admin UI produces the SAME JSON format
- Document creation from template = `structuredClone(template.blocks)` — no transformation layer
- All 4 test layers pass: interpreter unit tests, ViewModel conformance, golden fixtures, template validation

### Non-goals

- Dark mode support
- Landscape / custom page sizes
- Real-time collaboration
- Server-side DOCX generation
- Batch/bulk export
- ODT/RTF export formats
- Drag-to-reorder sections (sections are template-locked)
- Template inheritance (one template extending another)
- Conditional blocks (show/hide based on document data)
- Custom fonts per template
- Offline PDF generation
- Undo/redo beyond BlockNote native
- Template marketplace or sharing between organizations

### Relationship to existing specs

This spec builds on and supersedes the rendering architecture from:

- **2026-04-10 Unified Document Engine** — Layout IR tokens, DOCX emitters, version pinning, compatibility contract. All retained. This spec adds the template layer and ViewModel-driven rendering on top.
- **2026-04-12 React Parity Layer** — Layout Interpreters, ViewModels, three-layer rendering stack. This spec implements that architecture with the addition of typed codecs, the capability model, and the template schema.

## Architecture

### Template = MDDM Blocks + Metadata

Following the industry pattern (Word, Notion, Google Docs), a template IS a document in the same format as the content it produces, with annotations for what's locked, editable, and configurable.

```
Template Definition (JSON)
    │
    ├── metadata (key, version, status, profile)
    ├── theme (accent colors)
    └── blocks[] — standard MDDM blocks, each with:
            ├── props (what the block IS)
            ├── style (how it LOOKS — overrides Layout IR defaults)
            └── capabilities (what the document author CAN DO)
```

### Rendering Flow

```
Layout IR tokens (tokens.ts) — SINGLE SOURCE OF TRUTH
    │
    ├──→ tokensToCssVars(tokens) → CSS custom properties on editor root
    │       Used by: global BlockNote chrome (side menu, drag handles,
    │       spacing, font) — editor-wide, not block-specific
    │
    ├──→ interpret*(block, tokens) → ViewModel
    │       Used by: React render() and DOCX emitter
    │       Merges block.style overrides with token defaults
    │       Resolves theme references ("theme.accent" → "#6b1f2a")
    │       Computes derived values (section numbers, column widths)
    │
    └──→ Direct token import
            Used by: toExternalHTML (PDF pipeline, no CSS vars available)
```

### Two CSS Layers

| Layer | What it styles | How it gets values | File |
|---|---|---|---|
| **Global chrome** | BlockNote UI: side menu, drag handles, nesting lines, block spacing, font | CSS custom properties from `tokensToCssVars()` on `.mddm-editor-root` | `mddm-editor-global.css` |
| **Component rendering** | Each MDDM block: section headers, field labels, table cells, repeatable borders | ViewModel values applied as inline styles or CSS module classes | `Section.module.css`, `Field.module.css`, etc. |

Global chrome is token-driven (CSS vars). Component rendering is ViewModel-driven (interpreter output). They never overlap.

## Components

### 1. Template Schema

```json
{
  "templateKey": "po-standard",
  "version": 1,
  "profileCode": "po",
  "status": "published",
  "meta": {
    "name": "Procedimento Operacional Padrão",
    "description": "Template padrão para procedimentos operacionais",
    "createdAt": "2026-04-13T00:00:00Z",
    "updatedAt": "2026-04-13T00:00:00Z"
  },
  "theme": {
    "accent": "#6b1f2a",
    "accentLight": "#f9f3f3",
    "accentDark": "#3e1018",
    "accentBorder": "#dfc8c8"
  },
  "blocks": [
    {
      "type": "section",
      "props": { "title": "IDENTIFICAÇÃO" },
      "style": {
        "headerHeight": "10mm",
        "headerBackground": "#6b1f2a",
        "headerColor": "#ffffff",
        "headerFontSize": "13pt",
        "headerFontWeight": "bold"
      },
      "capabilities": {
        "locked": true,
        "removable": false,
        "reorderable": false
      },
      "children": []
    }
  ]
}
```

If `style` is omitted, Layout IR token defaults apply. Templates can override per-block when needed. Theme colors can be referenced by name: `"headerBackground": "theme.accent"`.

Template lifecycle: `draft` → `published` → `deprecated`. Only published templates appear in document creation.

### 2. Block Capability Model

Each block has three configuration layers from the template:

- **`props`** — What the block IS (label, title, content)
- **`style`** — How the block LOOKS (sizes, colors, backgrounds) — overrides Layout IR defaults
- **`capabilities`** — What the document author CAN DO with it

**Universal capabilities (all block types):**

| Capability | Type | Default | Description |
|---|---|---|---|
| `locked` | `boolean` | `true` | Block structure can't be modified |
| `removable` | `boolean` | `false` | Author can delete this block |
| `reorderable` | `boolean` | `false` | Author can drag-reorder |

**Per-block-type capabilities:**

**Section:**
No extra capabilities. Sections are structural, always template-locked.

**Field:**
- `editableZones: ("value")[]` — Label locked, value editable.

**FieldGroup:**
- `columns: 1 | 2` — Layout configuration, locked by template.

**DataTable — two modes:**

Fixed table (fill-only form):
```json
{
  "capabilities": {
    "locked": true,
    "mode": "fixed",
    "editableZones": ["cells"],
    "addRows": false,
    "removeRows": false,
    "addColumns": false,
    "removeColumns": false,
    "resizeColumns": false,
    "headerLocked": true
  },
  "columns": [
    { "key": "item", "label": "Item", "width": "40%", "locked": true },
    { "key": "resp", "label": "Responsável", "width": "30%", "locked": true },
    { "key": "prazo", "label": "Prazo", "width": "30%", "locked": true }
  ]
}
```

Dynamic table (expandable):
```json
{
  "capabilities": {
    "locked": false,
    "mode": "dynamic",
    "editableZones": ["cells"],
    "addRows": true,
    "removeRows": true,
    "addColumns": false,
    "removeColumns": false,
    "resizeColumns": false,
    "headerLocked": true,
    "maxRows": 100
  }
}
```

| | Fixed Table | Dynamic Table |
|---|---|---|
| Add row button | Hidden | Shown |
| Remove row button | Hidden | Shown per row |
| Add column | No | Per capability |
| Column resize | No | Per capability |
| Header row | Locked, styled | Locked, styled |
| Cell editing | Yes (fill the form) | Yes (add data) |

**Repeatable:**
- `addItems: boolean` — Author can add items
- `removeItems: boolean` — Author can remove items
- `maxItems: number` — Maximum items allowed
- `minItems: number` — Minimum items required
- `itemTemplate: { children: Block[] }` — Blocks each new item gets

**RepeatableItem:**
- `editableZones: ("content")[]` — Structure locked, content editable.

**RichBlock:**
- `editableZones: ("content")[]` — Label locked, content editable.

### 3. Typed Codecs

Every block type has a codec that defines parse, validate, normalize, and serialize rules for its style + capabilities. All read/write paths go through the codec — no raw `JSON.parse()` anywhere.

```typescript
// codecs/section-codec.ts

export type SectionStyle = {
  headerHeight?: string;
  headerBackground?: string;
  headerColor?: string;
  headerFontSize?: string;
  headerFontWeight?: string;
};

export type SectionCapabilities = {
  locked: boolean;
  removable: boolean;
  reorderable: boolean;
};

export const SectionCodec = {
  parseStyle(json: string): SectionStyle {
    const raw = safeParse(json, {});
    return {
      headerHeight: expectString(raw.headerHeight),
      headerBackground: expectString(raw.headerBackground),
      headerColor: expectString(raw.headerColor),
      headerFontSize: expectString(raw.headerFontSize),
      headerFontWeight: expectString(raw.headerFontWeight),
      // Unknown fields stripped — never persisted or forwarded
    };
  },

  parseCaps(json: string): SectionCapabilities {
    const raw = safeParse(json, {});
    return {
      locked: expectBoolean(raw.locked, true),
      removable: expectBoolean(raw.removable, false),
      reorderable: expectBoolean(raw.reorderable, false),
    };
  },

  defaultStyle(): SectionStyle { return {}; },
  defaultCaps(): SectionCapabilities {
    return { locked: true, removable: false, reorderable: false };
  },

  serializeStyle(style: SectionStyle): string {
    return JSON.stringify(stripUndefined(style));
  },
  serializeCaps(caps: SectionCapabilities): string {
    return JSON.stringify(caps);
  },
};
```

Each block type has its own codec: `SectionCodec`, `FieldCodec`, `DataTableCodec`, `RepeatableCodec`, `RichBlockCodec`. The interpreter, React render, DOCX emitter, template instantiation, and save/load all go through the same codec. Unknown fields are stripped, invalid combinations are rejected, defaults are explicit.

### 4. Layout Interpreters + ViewModels

One interpreter per block type. Takes a block + Layout IR tokens → ViewModel. The ViewModel is a plain object describing WHAT to render without any rendering technology.

```typescript
// interpreters/section.ts
export function interpretSection(
  block: Block,
  tokens: LayoutTokens,
  context: { sectionIndex: number }
): SectionViewModel {
  const style = SectionCodec.parseStyle(block.props.styleJson as string);
  const caps = SectionCodec.parseCaps(block.props.capabilitiesJson as string);

  return {
    number: String(context.sectionIndex + 1),
    title: block.props.title as string,
    headerHeight: style.headerHeight ?? `${tokens.components.section.headerHeightMm}mm`,
    headerBg: resolveThemeRef(style.headerBackground, tokens) ?? tokens.theme.accent,
    headerColor: style.headerColor ?? "#ffffff",
    headerFontSize: style.headerFontSize ?? `${tokens.components.section.headerFontSizePt}pt`,
    headerFontWeight: style.headerFontWeight ?? "bold",
    locked: caps.locked,
    removable: caps.removable,
    children: block.children,
  };
}
```

**React component consumes the ViewModel:**

```typescript
export const Section = createReactBlockSpec(
  {
    type: "section",
    propSchema: {
      title: { default: "" },
      locked: { default: true },
      styleJson: { default: "{}" },
      capabilitiesJson: { default: "{}" },
    },
    content: "none",
  },
  {
    render: (props) => {
      const tokens = useLayoutTokens();
      const sectionIndex = useSectionIndex(props.block);
      const vm = interpretSection(props.block, tokens, { sectionIndex });

      return (
        <div
          className={styles.sectionHeader}
          data-mddm-block="section"
          data-locked={vm.locked}
          style={{
            height: vm.headerHeight,
            background: vm.headerBg,
            color: vm.headerColor,
            fontSize: vm.headerFontSize,
            fontWeight: vm.headerFontWeight,
          }}
        >
          <span>{vm.number}.</span>
          <span>{vm.title}</span>
        </div>
      );
    },

    toExternalHTML: (props) => {
      const tokens = getLayoutTokens();
      const vm = interpretSection(props.block, tokens, { sectionIndex: 0 });
      return (
        <table data-mddm-block="section" style={{ width: "100%", borderCollapse: "collapse" }}>
          <tbody><tr>
            <td style={{
              background: vm.headerBg, height: vm.headerHeight,
              color: vm.headerColor, fontSize: vm.headerFontSize,
              fontWeight: vm.headerFontWeight, padding: "0 4mm", verticalAlign: "middle",
            }}>
              {vm.number}. {vm.title}
            </td>
          </tr></tbody>
        </table>
      );
    },
  }
);
```

**DOCX emitter consumes the same ViewModel:**

```typescript
export function emitSection(block: Block, tokens: LayoutTokens, context: EmitContext): DocxElement[] {
  const vm = interpretSection(block, tokens, { sectionIndex: context.sectionIndex });
  return [new Table({
    rows: [new TableRow({
      height: { value: mmToTwip(parseFloat(vm.headerHeight)), rule: HeightRule.EXACT },
      children: [new TableCell({
        shading: { fill: vm.headerBg.replace("#", "") },
        children: [new Paragraph({
          children: [new TextRun({
            text: `${vm.number}. ${vm.title}`,
            color: vm.headerColor.replace("#", ""),
            size: ptToHalfPt(parseFloat(vm.headerFontSize)),
            bold: vm.headerFontWeight === "bold",
          })],
        })],
      })],
    })],
  })];
}
```

The parity guarantee: all three emitters (React, toExternalHTML, DOCX) call the same `interpretSection()` with the same block + tokens. Same ViewModel → same visual output.

### 5. Template Instantiation + Document Lifecycle

**Creating a document from a template:**

```typescript
function instantiateTemplate(template: TemplateDefinition): MDDMEnvelope {
  return {
    mddm_version: CURRENT_MDDM_VERSION,
    template_ref: {
      templateKey: template.templateKey,
      templateVersion: template.version,
      instantiatedAt: new Date().toISOString(),
    },
    blocks: structuredClone(template.blocks),
  };
}
```

Deep clone. No transformation. Blocks retain props, capabilities, style.

**Template provenance:**

Documents retain a `template_ref` linking back to the template key, version, and instantiation time. This enables audit trails and future template upgrade features.

**Template versioning:**

When a published template is modified, a new version is created. Existing documents keep their `template_ref` pointing to the version they were created from. No automatic migration of existing documents.

**Capability enforcement at render time:**

- `locked: true` → no drag handle, no delete button, `data-locked="true"` hides side menu
- `editableZones: ["value"]` → only the value portion is `contenteditable`, label is read-only
- `mode: "fixed"` → hide add/remove row buttons on DataTable
- `mode: "dynamic"` → show add/remove row buttons, enforce `maxRows`
- `addItems: false` → hide "+ Adicionar" button on Repeatable

**Capability enforcement at save time:**

Soft validation checks that locked blocks weren't structurally modified. Warns on save but doesn't block it. Hard enforcement is at the UI level.

### 6. Template Validation

Standalone function called at template publish time and optionally at document load:

```typescript
export function validateTemplate(template: TemplateDefinition): ValidationError[] {
  // - Every block.type exists in the MDDM block registry
  // - Required props present (section needs title, field needs label)
  // - Capabilities valid for the block type (addRows only on dataTable)
  // - Children valid for parent type (fieldGroup only contains field children)
  // - Column widths sum to ~100% on DataTable
  // - maxItems >= minItems on Repeatable
  // - Codec parse succeeds for all styleJson and capabilitiesJson
  // - No unknown fields in style or capabilities (codec strips them)
  // - Theme references point to existing theme keys
}
```

## Data Flow

### Document Creation

```
User clicks "Novo documento" → selects template
    → instantiateTemplate(template) → MDDMEnvelope with template_ref
    → POST /documents (save initial content)
    → Editor opens with template blocks
```

### Editor Rendering

```
Editor mounts → blocks loaded
    → For each block: interpret(block, tokens) → ViewModel
    → React component renders from ViewModel
    → Capabilities enforced: locked = no chrome, editableZones = selective contenteditable
```

### Export

```
DOCX: For each block → interpret(block, tokens) → ViewModel → DOCX emitter → docx.js → Blob
PDF:  For each block → interpret(block, tokens) → ViewModel → toExternalHTML → Gotenberg → PDF
Both use the SAME interpret() functions. Same ViewModel. Same visual output.
```

## Error Handling

- **Invalid template JSON**: `validateTemplate()` returns errors, template cannot be published
- **Corrupt styleJson/capabilitiesJson**: codec returns defaults, logs warning, never crashes
- **Unknown block type in template**: validation rejects, block skipped at render time with warning
- **Missing Layout IR tokens**: interpreter uses hardcoded fallbacks from `defaultLayoutTokens`
- **Capability violation on save**: soft warning shown to user, save proceeds

## Testing Approach

### Layer 1: Interpreter unit tests (Vitest)

Pure function tests for each interpreter — no React, no DOM, no DOCX. Tests cover: default values from Layout IR, style overrides, theme reference resolution, section numbering, DataTable mode switching.

### Layer 2: ViewModel conformance tests (Vitest)

Verify that React and DOCX emitters both consume all ViewModel fields. Given the same ViewModel, React render produces HTML with the correct values, and DOCX emitter produces elements with matching values.

### Layer 3: Golden fixture tests (Vitest)

Full template → interpret → emit → compare cycle. DOCX XML output and toExternalHTML output compared against golden fixture files. Structural regression detection.

### Layer 4: Template validation tests (Vitest)

Tests for `validateTemplate()` — rejects unknown block types, invalid capabilities, bad column widths, and accepts valid templates.

### Visual validation

Claude in Chrome browser automation + manual review. Load documents, screenshot sections, compare against DOCX export visually. Iterate until satisfied.

## Phase Plan

### Phase 1: Engine + Code-Defined Templates

| # | Deliverable |
|---|---|
| 1 | Template JSON schema types + `validateTemplate()` function |
| 2 | Typed codecs for all 7 MDDM block types |
| 3 | Layout Interpreters + ViewModels (Section, Field, FieldGroup first) |
| 4 | React emitters refactored to use interpreters |
| 5 | DOCX emitters refactored to use same interpreters |
| 6 | DataTable interpreter (fixed + dynamic modes) |
| 7 | Repeatable + RepeatableItem interpreters |
| 8 | RichBlock interpreter |
| 9 | PO template defined as JSON, wired into document creation flow |
| 10 | Global chrome CSS cleanup (token-driven) |
| 11 | Golden fixtures updated |
| 12 | IR hash update for any Layout IR changes |

### Phase 2: Admin UI + Database Storage (future, out of scope)

- Template admin page under "Tipos documentais"
- Templates stored as JSON in database, served via API
- Template CRUD + versioning + preview
- Produces the SAME JSON format as Phase 1 — engine doesn't change

## Out of Scope

- Dark mode support
- Landscape / custom page sizes
- Real-time collaboration
- Server-side DOCX generation
- Batch/bulk export
- ODT/RTF export formats
- Drag-to-reorder sections
- Template inheritance
- Conditional blocks
- Custom fonts per template
- Offline PDF generation
- Undo/redo beyond BlockNote native
- Template marketplace or sharing
