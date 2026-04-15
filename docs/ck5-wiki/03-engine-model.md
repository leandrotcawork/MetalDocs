---
title: Engine model
status: draft
area: core
---

# 03 — Engine model (model / view / DOM)

CKEditor 5's editing engine (`@ckeditor/ckeditor5-engine`) is the largest
subsystem in the stack. The official intro calls it "the biggest and by far the
most complex piece" and notes it "features a custom, editing-oriented virtual
DOM implementation that aims to hide browser quirks from your sight."

Everything MetalDocs does — schema, conversion, widgets, restricted editing,
DOCX export — sits on top of the three trees this engine maintains. Getting
this page right makes the rest of the wiki make sense.

---

## 1. The three trees

CK5 does **not** edit the DOM directly. It maintains three tree structures and
converts between them.

| Tree            | Owner                         | Purpose                                             |
| --------------- | ----------------------------- | --------------------------------------------------- |
| **Model**       | `editor.model.document`       | Abstract, semantic source of truth                  |
| **Editing view**| `editor.editing.view.document`| Intermediate tree whose render target is the DOM    |
| **Data view**   | produced by `editor.data`     | Intermediate tree used to serialize HTML in/out     |
| **DOM**         | browser                       | Only the editing view is rendered into it           |

> "The editing view shows the content users see in the browser and interact
> with. The data view handles the editor's input and output data in a format
> the data processor understands."
> — *Editing engine* docs.

The DOM is attached **only** to the editing side. The data view never becomes
DOM; it is converted to a string via the data processor (`HtmlDataProcessor`
by default) when the host app calls `editor.getData()`.

```
            ┌─────────────┐   upcast     ┌──────────────┐
  HTML ───► │  Data View  │ ───────────► │              │
            └─────────────┘              │     MODEL    │
            ┌─────────────┐   downcast   │   (source    │
  HTML ◄─── │  Data View  │ ◄─────────── │   of truth)  │
            └─────────────┘              │              │
                                         │              │
            ┌─────────────┐   downcast   │              │
  DOM ◄───► │Editing View │ ◄──────────► │              │
            └─────────────┘   upcast     └──────────────┘
            (rendered by
             engine Renderer)
```

---

## 2. Why three layers, not one

The editing view and data view **diverge on purpose**. Things that help the
user edit must not leak into saved HTML:

- **Widget chrome** — handles, resize dots, "click-to-select" outlines live
  only in the editing view (as `uiElement`s / `attributeElement`s).
- **`contenteditable` flags** — the editing view marks roots/widgets with
  `contenteditable=true/false`; the data view has none of that.
- **Placeholder nodes** — empty-state text ("Type here…") is rendered in
  editing view only.
- **Selection, markers, fake selections** — live in editing view.

The data view is the clean representation: exactly what goes to disk, to the
server, or to `HTMLtoDOCX`. Our DOCX pipeline consumes `editor.getData()`
output, so anything we need persisted must round-trip through the **model**
and appear in the **data-view downcast** — not just the editing downcast.

---

## 3. Model nodes

> "[The model is] a DOM-like tree structure consisting of elements and text
> nodes. … both elements and text nodes in the model can have attributes."
> — *Editing engine* docs.

Node types (from `@ckeditor/ckeditor5-engine/src/model/*`):

- **`Element`** — generic container, has `name` and attributes.
- **`Text`** — a run of characters with attributes. Bold/italic/etc. are
  **attributes on text nodes**, not wrapper elements. The docs spell this out:
  `<strong>foo</strong>` becomes a text node with `bold=true`, which
  "significantly [reduces] algorithmic complexity."
- **`RootElement`** — a named top-level element owned by the document
  (`main`, or custom roots in multi-root / decoupled setups).
- **`DocumentFragment`** — a detached batch of nodes not yet inserted into a
  root; used by clipboard/paste and programmatic insertions.

Attributes are Map-like (`getAttribute`, `setAttribute`, `hasAttribute`) and
live on both `Element` and `Text`. Schema (see `04-schema.md`) governs which
attributes are legal where.

---

## 4. View nodes

View elements come in **variants** that don't exist in the model. The variant
controls how the renderer treats the node and how conversion handles
selection.

| Variant            | Class                  | Typical use                                   |
| ------------------ | ---------------------- | --------------------------------------------- |
| `containerElement` | `ContainerElement`     | Block-like wrappers (`<p>`, `<section>`)      |
| `attributeElement` | `AttributeElement`     | Inline formatting (`<strong>`, `<em>`)        |
| `emptyElement`     | `EmptyElement`         | Self-closing (`<img>`, `<hr>`)                |
| `uiElement`        | `UIElement`            | Editor chrome; **not serialized by data view**|
| `rawElement`       | `RawElement`           | Escape hatch whose inner DOM we control       |

Key difference from the model: view elements have **no attributes-on-text**
concept. Inline styling in the view is done by wrapping text in
`attributeElement`s, which the conversion pipeline merges/splits as needed.

---

## 5. Operations and batches

Every mutation of the model goes through **operations**, grouped into
**batches**.

> "All document changes [occur] through operations … [the system organizes]
> operations into batches, which act as single undo steps."
> — *Editing engine* docs.

- Operations are atomic (`InsertOperation`, `AttributeOperation`,
  `MoveOperation`, `MergeOperation`, `SplitOperation`, `RenameOperation`,
  `MarkerOperation`, …).
- Batches are what **undo/redo** operates on — one batch = one Ctrl-Z.
- The same operation model is what **real-time collaboration** transforms
  against (operational transformation). We do not use CK Cloud collab, but
  the guarantee that all changes are expressible as operations is what makes
  `editor.model.change(writer => { … })` safe and atomic.

---

## 6. The differ

When a batch finishes, the engine doesn't re-render everything. The
**Differ** (`model.document.differ`) walks the operations in the batch and
emits a minimal list of changes (`insert`, `remove`, `attribute`, …). The
downcast converters consume that diff to patch the editing view, and the
**Renderer** then diffs the view against the DOM.

Two-stage diffing (model→view, view→DOM) is why CK5 stays responsive on
large documents, and why our `change:data` handlers must stay cheap — they
fire on every batch.

⚠ uncertain: the public API surface of `Differ` beyond `getChanges()` is
sparsely documented; treat it as read-only from plugin code.

---

## 7. EditingController vs DataController

Two controllers, two jobs.

**`editor.editing` — `EditingController`**

> "Maintains a single instance of the ViewDocument for its entire life.
> Every change in the model is converted to changes in that view."

Runs continuously while the editor is alive. Owns the rendered DOM and
the editing-downcast converters.

**`editor.data` — `DataController`**

> "Controls how data is retrieved from the document and set inside it."

Runs on demand: `getData()`, `setData()`, `toModel()`, `toView()`. Builds a
throwaway data-view tree, serializes via the data processor, then discards
it. Has its own downcast pipeline (data-downcast) registered separately from
editing-downcast.

**Practical rule**: if a feature should appear in the saved HTML, register a
converter on `downcastDispatcher` for **both** `editor.data` and
`editor.editing` (usually via `editor.conversion.for('downcast')` which
covers both; use `for('editingDowncast')` or `for('dataDowncast')` for the
split cases). See `05-conversion.md`.

---

## 8. Document vs Model (and where to reach)

Common confusion: `model` vs `model.document`.

- `editor.model` — the **Model** facade. Owns `change()`, schema,
  markers-collection, the writer API.
- `editor.model.document` — the **Document** inside the model. Owns roots,
  selection, history, and fires `change:data`.
- `editor.editing.view` — the editing **View** facade. Owns its own
  `change()`, `document`, and `domConverter`.
- `editor.editing.view.document` — the editing **ViewDocument**. Fires
  `render`, owns view selection, roots.
- `editor.data` — the `DataController`. Has no persistent document.

Quick cheat sheet:

```ts
editor.model.document.getRoot('main')        // RootElement (model)
editor.editing.view.document.getRoot('main') // RootEditableElement (view)
editor.data.get({ rootName: 'main' })        // serialized HTML string
```

---

## 9. Inspector (debugging tool)

`@ckeditor/ckeditor5-inspector` is the official dev tool. It renders a panel
that shows, live:

- Model tree (with attributes, selection, markers)
- Editing view tree
- Commands (state, value, `isEnabled`, `execute` button)
- Schema (per-element allowed children / attributes)

**Install + attach:**

```ts
// dev-only
import CKEditorInspector from '@ckeditor/ckeditor5-inspector';
CKEditorInspector.attach(editor);
// or: CKEditorInspector.attachToAll();
```

**Licensing**: the Inspector repo ships under a **dual license — GPL v2+ or
commercial**, same family as the core editor. It's a development tool, safe
to include in dev builds. We should **not** ship it in production bundles
(both for bundle size and to keep the commercial/GPL boundary tidy — see
`01-license-gpl.md`). ⚠ uncertain whether npm installs trigger any license
notice beyond the standard GPL terms; verify before bundling into any
customer-facing build.

---

## 10. Hooks MetalDocs actually uses

Three hooks cover 95% of what our plugins do:

**Mutate the model (always inside a `change` block):**

```ts
editor.model.change(writer => {
  const p = writer.createElement('paragraph');
  writer.insertText('hello', p);
  writer.append(p, editor.model.document.getRoot('main'));
});
```

`change()` opens a batch. Nested `change()` calls join the outer batch; use
`enqueueChange()` for a fresh batch after the current one.

**Observe model data changes** (autosave, dirty flags, validation):

```ts
editor.model.document.on('change:data', () => {
  scheduleAutosave(editor.getData());
});
```

Fires only for batches that changed document data (not for pure
selection/marker changes — use `change` for those).

**Observe rendering** (measure after the DOM is up-to-date):

```ts
editor.editing.view.document.on('render', () => {
  measurePaginationOverflow();
});
```

Fires after the view has been flushed to the DOM. Safe place to read layout.

Do **not** mutate the DOM directly from these hooks — go through
`view.change(writer => …)` so the engine's renderer stays in charge.

---

## Sources

- https://ckeditor.com/docs/ckeditor5/latest/framework/architecture/intro.html
- https://ckeditor.com/docs/ckeditor5/latest/framework/architecture/editing-engine.html
- https://ckeditor.com/docs/ckeditor5/latest/framework/architecture/core-editor-architecture.html
- https://github.com/ckeditor/ckeditor5-inspector (README — install, license)
- ⚠ `framework/development-tools/inspector.html` — returned 404 on fetch
  (2026-04-15); inspector info taken from the GitHub README instead.

## Related wiki pages

- `04-schema.md` — what attributes/elements are legal where
- `05-conversion.md` — upcast/downcast, editing vs data pipelines
- `06-commands.md` — command state, `refresh()`, `affectsData`
- `27-undo-history.md` — batches as undo steps
