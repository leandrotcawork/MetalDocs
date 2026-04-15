---
title: Template authoring
status: draft
area: templates
---

# 28 — Template authoring

How a MetalDocs **template** is authored inside CK5 "Standard" mode: the
editor build, the plugin set, the toolbar, the step-by-step authoring flow,
how exception markers get planted, and how the template is saved. The
authored artefact is the same kind of CK5 HTML that Fill mode later
consumes (see [29 — Template instantiation](./29-template-instantiation.md))
and exporters walk (see [31 — DOCX export](./31-docx-export.md),
[32 — PDF export](./32-pdf-export.md)).

> Scope: this page covers the Author-side editor. Fill mode (end-user
> template filling) is page 29; the library / versioning UI is page 30.

---

## Recommended design

- **Editor type: `DecoupledEditor`.** Document-chrome layout (detached
  toolbar above, editable in the centre, side panels for outline + fields)
  matches the Word-like authoring UX that template builders expect, and is
  the build the CK5 team themselves showcase as "Document editor"
  ([Document editor example](https://ckeditor.com/docs/ckeditor5/latest/examples/builds/document-editor.html)).
  Classic/Balloon builds bundle the toolbar with the editable and leave no
  room for the outline/fields panels we need.
- **Mode plugin: `StandardEditingMode`** — not `RestrictedEditingMode`.
  The author needs full editing power *plus* the ability to mark
  exception regions. The two modes are mutually exclusive
  ([10 — Restricted editing](./10-restricted-editing.md) §"You cannot run
  both modes at once"); Fill mode is a separate editor instance loaded
  from the same HTML.
- **Exception markers are planted by primitive-insertion commands**, not
  by the author running `restrictedEditingException{,Block}` manually.
  Every `mddmSection` / `mddmRepeatable` / `mddmField` insertion sets up
  its own markers during the same `model.change()` batch
  (see [19 §6](./19-sections-mapping.md), [20 §3](./20-repeatables-mapping.md)).
- **Save flow: HTML + metadata manifest.** `editor.getData()` is the
  template body; a sidecar JSON carries template name, version, and field
  definitions (id/label/type/group/required/default). No IR, no content
  hash ([25 §Recommended architecture](./25-mddm-ir-bridge.md)).
- **Validation: `setData` → `getData` round-trip** to assert
  canonicalization, plus a structural pass rejecting malformed
  primitives (section missing a header, repeatable below `min`, etc.).
- **Preview: a second CK5 instance** (read-only, no Standard mode plugin)
  loading the same HTML, so authors see what Fill users will see.

---

## 1. Editor build

### Why Decoupled

Document Editor example shows three relevant affordances:

- Toolbar rendered into a standalone container (we mount it into the top
  chrome of the Author shell).
- Editable rendered into a paged white "canvas" — matches the template
  preview we want.
- No bundled UI that forces toolbar adjacency; we get to place the
  outline panel and field panel on the sides.

Classic / Balloon builds would force us to re-skin the toolbar slot and
wouldn't give us clean DOM separation. `DecoupledEditor.create()` returns
toolbar + editable as separate DOM nodes that the Author shell composes.

```ts
import { DecoupledEditor } from 'ckeditor5';

const editor = await DecoupledEditor.create( sourceEl, {
    plugins: AUTHOR_PLUGINS,
    toolbar: AUTHOR_TOOLBAR,
    // no `restrictedEditing.allowedCommands` in Author; that config is
    // Fill-mode only (page 10 §Configuration).
} );

topbarEl.appendChild( editor.ui.view.toolbar.element! );
canvasEl.appendChild( editor.ui.getEditableElement()! );
```

### Plugin list — Author mode

Built-in:

- `Essentials`, `Paragraph`, `Heading`
- `Bold`, `Italic`, `Underline`
- `Link`, `List` (bulleted + numbered)
- `Table`, `TableToolbar`, `TableProperties`, `TableCellProperties`,
  `TableColumnResize`, `TableCaption`
- `Image`, `ImageToolbar`, `ImageStyle`, `ImageResize`, `ImageCaption`,
  `ImageUpload` (+ the MetalDocs upload adapter)
- `Alignment`, `Font`, `RemoveFormat`
- `Autosave` (see [26 — Autosave & persistence](./26-autosave-persistence.md))
- `SourceEditing` — **dev builds only**. Gives template authors a raw
  HTML escape hatch; disabled in production so they can't emit
  unvalidated markup.
- `GeneralHtmlSupport` — scoped narrowly (allow a small set of `data-*`
  attributes on the primitives; do NOT open it up to arbitrary tags).
  See [18 — HTML support](./18-html-support.md).

Author-mode glue:

- `StandardEditingMode` — the exception feature in "author who can mark"
  mode. Registers `restrictedEditingException`,
  `restrictedEditingExceptionBlock`, `restrictedEditingExceptionAuto`
  commands, plus the shared `restrictedEditingException` marker group.

MetalDocs custom plugins:

- `MddmSectionPlugin` — `mddmSection` + header/body ([19](./19-sections-mapping.md)).
- `MddmRepeatablePlugin` — `mddmRepeatable` + items ([20](./20-repeatables-mapping.md)).
- `MddmFieldPlugin` — inline field chips ([22](./22-fieldgroup-mapping.md)).
- `MddmTableVariantPlugin` — fixed vs dynamic table attribute on the
  native `table` element ([21](./21-datatable-mapping.md)).
- `MddmTableLockPlugin` — inert in Author mode (no gating), but registered
  so schema and converters are identical across Author/Fill editors.
- `MddmRichBlockPlugin` — optional, see [23](./23-rich-block-mapping.md).

> ⚠ uncertain: whether `GeneralHtmlSupport` is needed at all for Author
> once our primitives cover sections/repeatables/tables/fields. Leaning
> "no" — keep the schema closed and rely on primitives. Revisit if a
> template needs a one-off inline tag we don't want to make a primitive.

### Plugins we deliberately DO NOT load in Author mode

- `RestrictedEditingMode` — conflicts with `StandardEditingMode`
  ([10 §"You cannot run both modes at once"](./10-restricted-editing.md)).
- `RealTimeCollaboration`, `TrackChanges`, `Comments` — out of scope for
  v1 templates. Revisit later.

---

## 2. Toolbar configuration

Document-editor-style layout: structural primitives on the left, rich-text
formatting in the middle, exception/source tools on the right.

```ts
const AUTHOR_TOOLBAR = [
    // --- primitive insertion ---
    'insertMddmSection',
    'insertMddmRepeatable',
    'insertMddmField',
    'insertMddmRichBlock',
    'insertTable',              // native; MddmTableVariantPlugin hooks variant toggle
    'imageUpload',
    '|',
    // --- block structure ---
    'heading',
    'bulletedList',
    'numberedList',
    'alignment',
    '|',
    // --- inline formatting ---
    'bold', 'italic', 'underline',
    'fontSize', 'fontColor',
    'link',
    'removeFormat',
    '|',
    // --- exception tools (author-only) ---
    'restrictedEditingException',      // inline span over selection
    'restrictedEditingExceptionBlock', // block wrapper over selected blocks
    '|',
    // --- dev-only ---
    'sourceEditing',
    '|',
    'undo', 'redo',
];
```

Widget toolbars (balloon, appear on widget selection) are registered per
primitive — see [12 — Toolbar UI](./12-toolbar-ui.md). Relevant for
authors: the section widget toolbar exposes the variant toggle, the
repeatable widget toolbar exposes add/remove-item + min/max settings,
and the data table widget toolbar exposes fixed ↔ dynamic toggle.

---

## 3. Authoring flow (step-by-step)

### 3.1 Insert a Section

1. Click "Insert section" → dropdown with `locked | editable | mixed`.
2. `InsertSectionCommand` runs ([19 §6](./19-sections-mapping.md)). In the
   same `model.change()` batch:
   - Creates `mddmSection` + `mddmSectionHeader` + `mddmSectionBody` +
     one `paragraph`.
   - Sets `variant` + `sectionId`.
   - For `editable` / `mixed`, plants a block exception marker
     (`restrictedEditingException:<sectionId>_body`) over the body range.
   - Moves selection into the body paragraph.
3. Author types the header title, then tabs/clicks into the body and
   fills regular prose / tables / lists / images.

### 3.2 Inside the body — free content

Body is a `$container` ([19 §3](./19-sections-mapping.md)), so all loaded
editor features work: headings, lists, tables, images, links, fonts.
These are the "static template content" that Fill users will see as
read-only chrome (Author) or as fill-enabled content inside the body's
block exception (Fill).

### 3.3 Header fill-in-blanks (`mixed` variant)

For a `mixed` section header — e.g. "Subject: ______ — Amendment ___":

1. Author selects the blank run.
2. Clicks widget-toolbar "Insert inline field" (or runs
   `restrictedEditingException` directly, though the UI button is
   preferred because it also registers the field ID in the sidebar).
3. `InsertInlineFieldCommand` does, in one batch:
   - Inserts a `mddmField` chip (or leaves the text as-is, depending on
     field kind — [22](./22-fieldgroup-mapping.md)).
   - Wraps the span in a `restrictedEditingException:<sectionId>_h_<id>`
     marker so Fill mode can type inside that span only.
4. Result in data view:
   ```html
   <header class="mddm-section__header">
     Subject:
     <span class="restricted-editing-exception" data-mddm-field="subject"></span>
     — Amendment
     <span class="restricted-editing-exception" data-mddm-field="amendmentNo"></span>
   </header>
   ```

### 3.4 Repeatable

1. Click "Insert repeatable" → dialog for label, `min`, `max`,
   `numberingStyle`, `initialCount`.
2. `InsertMddmRepeatableCommand` ([20 §3.1](./20-repeatables-mapping.md))
   builds the parent + N items; each item body is wrapped in a block
   exception marker keyed `restrictedEditingException:<repeatableId>:<n>`.
3. Author fills the body of each initial item with the template prose.
4. Widget toolbar exposes "Add item" / "Remove item" (Author-mode only;
   Fill mode gating is handled by [20 §6](./20-repeatables-mapping.md)).

### 3.5 DataTable variant

1. Click "Insert table" — native CK5 insert, same as everywhere.
2. With caret inside the table, widget toolbar shows a variant toggle:
   `fixed` (cells are individual inline exceptions) or `dynamic` (whole
   table body is a block exception, rows addable at Fill time).
3. `MddmTableVariantPlugin` writes `data-mddm-table-variant` on the
   `<table>` and plants the correct marker shape
   ([21](./21-datatable-mapping.md)).
4. Constraint: **no nested tables** (CKEditor wraps tables in `<figure>`
   and HTMLtoDOCX mis-nests; hard constraint carried over from the old
   stack). Schema rejects table-in-table via `addChildCheck`.

### 3.6 RichBlock region

Optional primitive for "anywhere this author wants a whole free-form
subdocument inside a locked page" — [23](./23-rich-block-mapping.md).
Same command shape as Section (insert + mark body as block exception).

---

## 4. Exception markers — automatic, not manual

Author toolbar exposes the raw `restrictedEditingException` and
`restrictedEditingExceptionBlock` buttons because they are occasionally
useful (ad-hoc free-form region inside a locked paragraph). **But the
authoring contract is that authors do NOT hand-plant markers for
primitives.** Every structural primitive's insertion command plants its
own markers in the same batch as the insertion.

Reasons:

- Marker names follow a **naming convention** tied to the primitive's
  stable ID (`restrictedEditingException:<sectionId>_body`,
  `restrictedEditingException:<repeatableId>:<index>`). Hand-planted
  markers would break the convention and the sidebar UI that enumerates
  them.
- Undo must take the primitive and its marker back in one step
  ([03 §batches](./03-engine-model.md)). Two user actions = two undo
  steps = half-restored document on `Ctrl+Z`.
- Fill-mode navigation (`goToNextRestrictedEditingException`) relies on
  exactly one marker per "fillable slot". Hand-planted duplicates
  cause double-tabs.
- ⚠ uncertain: whether externally-minted marker IDs in the
  `restrictedEditingException` group fully interop with the feature's
  internal counter. See [19 §6](./19-sections-mapping.md) open questions.
  If the empirical check fails, insertion commands will `execute()` the
  built-in `restrictedEditingExceptionBlock` command instead of calling
  `writer.addMarker` directly — functionally equivalent.

---

## 5. Template metadata

HTML alone is not enough: Fill mode needs to know what fields exist, how
to label them, what types they have, and which group they belong to. We
store a sidecar manifest alongside the HTML.

### 5.1 Shape

```ts
interface TemplateManifest {
    id: string;                     // stable template ID
    name: string;                   // author-facing name
    description?: string;
    version: string;                // semver; bumped per [30 — Template library]
    createdAt: string;              // ISO
    updatedAt: string;              // ISO
    fields: FieldDefinition[];
    groups?: FieldGroupDefinition[]; // see page 22
}

interface FieldDefinition {
    id: string;                     // stable; referenced by data-mddm-field=
    label: string;
    type: 'text' | 'number' | 'date' | 'select' | 'currency' | 'boolean';
    group?: string;                 // optional FieldGroup id
    required?: boolean;
    default?: string | number | boolean | null;
    options?: string[];             // for 'select'
    validation?: { pattern?: string; min?: number; max?: number };
}

interface FieldGroupDefinition {
    id: string;
    label: string;
    order?: number;
}
```

### 5.2 Storage

Backend stores two columns per template row:

- `content_html TEXT` — the authored HTML from `editor.getData()`.
- `manifest_json JSONB` — the `TemplateManifest` above.

No IR, no `ir-hash`, no computed fingerprints
([25 §Recommended architecture](./25-mddm-ir-bridge.md)).

### 5.3 Keeping HTML and manifest in sync

Author-side state: the sidebar's Field panel is the **editable view** of
`manifest.fields`. Every time the author inserts a field, renames it, or
changes its type, the plugin mutates the in-memory manifest *and* updates
the inline chip's `data-mddm-field-label` / `data-mddm-field-type`
attributes so the HTML remains self-describing for degraded read paths.

On save, we run a consistency check:

1. Collect every `data-mddm-field` ID from the HTML.
2. Assert the manifest has one `FieldDefinition` per ID and no extras.
3. Reject the save with a targeted error if mismatched.

---

## 6. Save flow

```ts
async function saveTemplate() {
    // 1. Flush autosave debouncer so pending edits land.
    await editor.plugins.get( 'Autosave' ).save();

    // 2. Get the canonical HTML.
    const html = editor.getData();

    // 3. Validate (see §7).
    const problems = await validateTemplate( editor, html, manifest );
    if ( problems.length ) throw new ValidationError( problems );

    // 4. POST to backend: HTML + manifest in one transaction.
    await api.saveTemplate( {
        id: manifest.id,
        html,
        manifest,
    } );
}
```

No hash, no IR emit, no golden comparison. The backend treats the HTML
as opaque bytes; it does not parse or re-render during save
([25 §1](./25-mddm-ir-bridge.md)).

---

## 7. Validation before save

Two passes:

### 7.1 Round-trip canonicalization

```ts
const html1 = editor.getData();
editor.setData( html1 );
const html2 = editor.getData();
if ( html1 !== html2 ) {
    // Non-idempotent → schema / converter bug. Block save; log both.
    throw new ValidationError( [ 'Non-canonical HTML on round-trip' ] );
}
```

Per [25 §10](./25-mddm-ir-bridge.md), CK5's `setData`/`getData` is the
canonicalization contract. If `html1 !== html2`, something in our
upcast/downcast is asymmetric and the backend would store a version the
editor can't re-emit.

### 7.2 Structural checks

Walk the data view (we already have it in memory post-round-trip) and
assert:

- Every `mddmSection` has exactly one header and one body ([19 §3](./19-sections-mapping.md) post-fixer should prevent this, but belt-and-braces).
- Every `mddmRepeatable` has `min ≤ childCount ≤ max`
  ([20 §4](./20-repeatables-mapping.md) post-fixer).
- Every `data-mddm-field` ID maps to a `FieldDefinition` and vice versa.
- No nested tables (hard constraint; carried from memory about HTMLtoDOCX).
- No exception markers with zero-length ranges (they were collapsed by
  content removal and should be cleaned up before save).

Problems are collected into a list and surfaced in the save dialog; we do
not partially save.

---

## 8. Author-mode ergonomics

### Outline panel (left)

Live tree view of the current document:

- Sections (with variant badges: `locked` / `editable` / `mixed`).
- Repeatables (with item count + min/max).
- RichBlocks.
- Tables (with variant badge: `fixed` / `dynamic`).

Click → scroll + select the widget. Drives discoverability in long
templates. Data source: a `Document#change` listener that walks the root
and produces a lightweight tree snapshot.

### Field panel (right)

Two-level list grouped by `FieldGroup`:

- Add / rename / reorder fields.
- Set type, required, default, options.
- Clicking a field scrolls the editor to the first chip with that ID and
  flashes it.
- "Unused fields" warning — manifest definition without a matching chip.
- "Orphan chips" warning — chip whose ID has no manifest definition.

### Keyboard affordances

- `Alt+F10` cycles through widget toolbars ([11 §Widget toolbar](./11-widgets.md)).
- `Tab` / `Shift+Tab` in Author mode retains native block behaviour; in
  Fill mode it hops exceptions via
  `goToNextRestrictedEditingException`
  ([10 §Commands](./10-restricted-editing.md)).

---

## 9. Preview toggle

A "Preview" button in the Author shell swaps the editor for a read-only
`ClassicEditor`-or-`DecoupledEditor` instance loaded with
`RestrictedEditingMode` (or simply `isReadOnly = true` if we want a
pure view) using the same HTML via `setData()`. Because the restricted-
editing markers are already in the HTML, the preview shows the exact
chrome a Fill user will see — fillable spans, block exceptions, locked
regions.

Implementation:

1. Author clicks Preview.
2. Current Author editor is **destroyed** (not hidden — `StandardEditingMode`
   and `RestrictedEditingMode` cannot coexist;
   [10 §mutual exclusion](./10-restricted-editing.md)).
3. Preview editor is created on the same DOM container with the Fill-mode
   plugin set and the freshly-captured HTML.
4. Clicking "Exit preview" destroys the preview editor and reinitialises
   the Author editor with the (unchanged) HTML.

> ⚠ uncertain: destroying/recreating the editor on every toggle is
> O(100ms+) for large templates. If authors preview frequently we may
> want to cache the most recent HTML and skip the round-trip. Measure
> before optimising.

---

## 10. Cross-references

- [01 — License (GPL)](./01-license-gpl.md) — constraints on which plugins are free to ship.
- [03 — Engine model](./03-engine-model.md) — batch / `model.change()` semantics used by every insertion command.
- [04 — Schema](./04-schema.md) — `$blockObject`, `isLimit`, `addChildCheck` patterns used by primitives.
- [05 — Conversion](./05-conversion.md) — two pipelines; why Author data downcast must stay widget-chrome-free.
- [07 — Markers](./07-markers.md) — naming conventions, persistence flags, `markerToData` round-trip.
- [09 — Decoupled editor](./09-decoupled-editor.md) — build details, DOM wiring.
- [10 — Restricted editing](./10-restricted-editing.md) — Standard vs Restricted modes, commands.
- [11 — Widgets](./11-widgets.md) — widget toolbars, nested editables, reconversion.
- [12 — Toolbar UI](./12-toolbar-ui.md) — registering toolbar items / widget toolbars.
- [19 — Sections mapping](./19-sections-mapping.md) — `InsertSectionCommand` plants body markers.
- [20 — Repeatables mapping](./20-repeatables-mapping.md) — `InsertMddmRepeatableCommand` plants per-item markers.
- [21 — DataTable mapping](./21-datatable-mapping.md) — fixed vs dynamic variant.
- [22 — FieldGroup mapping](./22-fieldgroup-mapping.md) — inline field chips.
- [23 — RichBlock mapping](./23-rich-block-mapping.md) — free-form subdocument primitive.
- [24 — Data format](./24-data-format.md) — `getData`/`setData` contract.
- [25 — MDDM IR bridge](./25-mddm-ir-bridge.md) — HTML is canonical, no IR, no hash, round-trip canonicalization.
- [26 — Autosave & persistence](./26-autosave-persistence.md) — how Autosave interacts with explicit save.
- [29 — Template instantiation](./29-template-instantiation.md) — Fill-mode counterpart to this page.
- [30 — Template library](./30-template-library.md) — versioning, listing, rollback.

---

## 11. Open questions

- ⚠ uncertain: whether `GeneralHtmlSupport` is needed in Author mode once
  primitives are complete.
- ⚠ uncertain: whether the Author flow should forbid the raw
  `restrictedEditingException{,Block}` toolbar buttons entirely (force all
  exceptions to come from primitive commands) or keep them as an
  escape-hatch.
- ⚠ uncertain: block-exception wrapper tag in data view — assumed
  `<div class="restricted-editing-exception">` per [10 §Critical finding](./10-restricted-editing.md),
  but not literally quoted in CK docs. Empirical check needed before the
  DOCX/PDF exporters rely on the exact tag.
- ⚠ uncertain: whether preview should be an in-place editor swap or a
  separate route/modal. Swap is simpler; route survives refresh.
- ⚠ uncertain: whether externally-minted `restrictedEditingException:<id>`
  marker names interop with the feature's internal counter (see
  [19 §6](./19-sections-mapping.md) and [20 §3](./20-repeatables-mapping.md)
  open questions — same empirical test resolves both).

---

## Sources

Fetched 2026-04-15:

- https://ckeditor.com/docs/ckeditor5/latest/features/ — features index; confirmed plugin names used in §1.
- https://ckeditor.com/docs/ckeditor5/latest/examples/index.html — examples index.
- https://ckeditor.com/docs/ckeditor5/latest/examples/builds/document-editor.html — Document editor example, basis for the DecoupledEditor recommendation.
- https://ckeditor.com/docs/ckeditor5/latest/examples/builds-custom/full-featured-editor.html — full-featured editor example; cross-referenced for the toolbar plugin set.
- https://ckeditor.com/docs/ckeditor5/latest/features/restricted-editing.html — Standard vs Restricted mode contract (used via [10](./10-restricted-editing.md)).
