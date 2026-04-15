---
title: Data format
status: draft
area: storage
priority: HIGH
---

# 24 — Data format

CKEditor 5's data format on the wire is an **HTML string**, produced and
consumed through `editor.getData()` and `editor.setData()`. There is no
native JSON document format exposed as the persistence contract. Internally
CK5 keeps a tree-shaped model, but the model is not the wire format — the
data pipeline converts between the model and an HTML string on every
get/set.

This is the foundation for the MDDM bridge (page 25): **CK5 HTML is
transport; MDDM IR remains Source of Truth.**

## 1. Wire format is HTML

From the getting/setting data guide:

> "In some scenarios you may wish to replace the editor content on demand
> with new data. For this operation use the `setData()` method."
>
> "You can use the `getData()` method, if you need to get the editor
> content on demand, such as for sending it to the server using JavaScript
> APIs."

Example usage from the official docs:

```js
editor.setData( '<p>Some text.</p>' );
const data = editor.getData();
```

There is no official "serialize to JSON" counterpart on the public API.
The model tree is accessible (see §7) but is not the contract the rest of
the framework is built around.

## 2. `getData()` / `setData()` contract

- `editor.setData(html)` — parses an HTML string, runs it through the
  **upcast** converters, and replaces the current model document.
- `editor.getData()` — runs the model through the **downcast** converters
  into the data view, then serializes the data view into an HTML string.

Both operate on the `main` root by default. A `rootName` option exists for
multi-root editors (not documented on the page we fetched, but exposed on
`DataController`). `getData()` also accepts options that can alter output
(e.g. a `trim` mode); exact option surface is not covered on the primary
guide page and should be verified against the `DataController` API
reference before relying on it (see Open questions).

Key transform on read/write: **HTML ↔ model via converters**. Anything not
registered with a converter is discarded.

## 3. Two pipelines: data vs editing

CK5 maintains one model document and **two views**:

> "The architecture has three layers: model, controller, and view. There
> is one model document that gets converted into two separate views."
>
> "The editing view shows the content users see in the browser and
> interact with. The data view handles the editor's input and output data
> in a format the data processor understands."

- **Editing pipeline**: model → editing view → real DOM in the editable.
  `EditingController` keeps a persistent `ViewDocument` instance across
  the editor's lifetime, because it has to be incrementally rendered.
- **Data pipeline**: model → data view → HTML string (via the data
  processor). Stateless per call.

> "The data pipeline is much simpler than the editing pipeline."

### Why two pipelines

The editing view needs things the data view must not contain:

- Widget chrome (selection handles, resize grips, fake selection wrappers).
- Placeholder text for empty blocks.
- Drag handles, hover rings, UI affordances.
- `contenteditable="false"` / `contenteditable="true"` flags that make
  widgets behave as atomic units in the browser.

If the editor serialized the editing view directly, the saved HTML would
be full of UI scaffolding. The data view is the "clean" projection
intended for persistence.

## 4. Conversion: upcast and downcast

> "Data upcasting is a process that starts in the view layer, passes
> through a converter in the controller layer to the model document."
>
> "Data downcasting is the opposite process."

Three converter directions:

| Direction         | Source → Target            | Trigger                                  |
|-------------------|---------------------------|------------------------------------------|
| Upcast            | data view → model          | `setData()`, paste, initial data         |
| Data downcast     | model → data view          | `getData()`                              |
| Editing downcast  | model → editing view       | every model change (live rendering)      |

Widgets and custom blocks therefore define **two** downcast converters:
one for editing (with UI) and one for data (plain).

## 5. Schema filters unknown content

> "HTML elements and attributes not converted by registered converters
> are filtered out before becoming model nodes."

Implication: `setData()` is **not** a raw HTML setter. Anything that has
no upcast converter — or is rejected by the schema — is silently dropped.
This is the single biggest round-trip gotcha.

## 6. Lossless round-trip?

`editor.setData(x); editor.getData()` is **not guaranteed** to return
`x` byte-for-byte. Realistic deltas:

- **Stripped tags/attrs** — anything not covered by a converter or the
  schema disappears (§5). General HTML Support (GHS) widens the net but
  does not remove the rule.
- **Normalization** — attribute quoting, attribute order, self-closing
  vs paired tags, boolean attribute form, and entity encoding are
  re-emitted by the data processor, not preserved verbatim.
- **Whitespace** — the HTML data processor collapses insignificant
  whitespace between block elements and trims around block boundaries.
  Significant whitespace inside text nodes is preserved. The official
  getting/setting guide does not spell this out; behaviour is documented
  in the HTML data processor and engine internals (Open questions).
- **Empty content** — an empty editor returns `''` by default (there is
  a `trim` concept in `getData` options).

Treat the round-trip as **semantically** lossless for content covered
by your schema and converters, not **textually** identical.

## 7. Is there a JSON / model format?

Yes, but it is not the persistence contract:

- `editor.model.document.getRoot()` exposes the live model tree.
- Model nodes can be walked programmatically (`Element`, `Text`,
  attributes as `Map`-like).
- Operations and deltas are serializable (used by real-time
  collaboration), but this is an internal/collab-oriented surface, not
  a stable "save format".

The canonical, stable, round-trippable representation for storage and
transport is still the HTML string from `getData()`.

## 8. How widget attributes serialize

A model element `mddmSection` with a `variant` attribute typically
downcasts to something like:

```html
<mddm-section data-variant="mixed">
  <h2>…</h2>
  <p>…</p>
</mddm-section>
```

Conventions:

- Custom elements use a namespaced tag (`mddm-section`) to survive
  HTML parsing and not collide with semantic tags.
- Model attributes map to `data-*` HTML attributes in the data view
  (editing view may add classes/handles on top of this).
- Upcast converters recognise the same `data-*` attributes and rebuild
  the model element on `setData()`.

The editing downcast for the same element will additionally emit
widget chrome (`contenteditable`, class names, UI nodes) — **those never
appear in `getData()` output**.

## 9. Persisting markers

Markers are ranges attached to the model:

> "Markers are a special type of range … Can be synchronized over the
> network with collaborating clients. Automatically update when the
> document structure changes."

By default markers are ephemeral. To make them survive `getData()` /
`setData()`, register a marker downcast/upcast with:

- `usingOperation: true` — marker changes flow through operations (so
  collaboration and undo see them).
- `affectsData: true` — the marker is emitted by the data pipeline
  (e.g. as a boundary element or `data-*` attribute wrapper).

Without `affectsData: true`, the marker exists only in the editing
view and is lost on save.

## 10. Implication for MDDM

Our layout-IR is structured JSON, tuned for docx/pdf export. CK5's wire
format is HTML. The bridge is therefore a **two-way HTML ↔ IR
converter**, not a direct model-tree mapper:

- **IR → HTML** (load): emit the HTML CK5 expects; CK5 upcasts it into
  its model. MDDM stays authoritative; CK5 sees only what the schema
  covers.
- **HTML → IR** (save): parse `getData()` output and reconstruct the IR.
  Every MDDM block type needs a stable HTML shape (custom tag + `data-*`
  attributes) that is cheap to parse and cannot be accidentally produced
  by user typing.

Corollaries:

- CK5 schema and converters must cover **every** IR block type, or
  `setData()` will strip IR content (§5).
- Anything the IR needs but CK5 would not render (page metadata,
  pagination hints, template bindings) should ride as attributes on the
  outermost wrapper element — not as free-floating comments, which
  CK5 will drop.
- Widget chrome is invisible to the bridge; we only ever parse the
  data view output.

Detailed bridge design lives in page **25 — MDDM bridge**.

## Open questions

- Exact option surface of `editor.getData({ … })`: `trim`, `rootName`,
  and whether there is an option to disable data-processor HTML
  normalization. Needs verification against the `DataController` API
  reference rather than the high-level guide.
- Precise whitespace rules of the default HTML data processor
  (between blocks, inside `<pre>`, around inline widgets). The intro
  guide does not cover this; the engine source / data-processor page
  should.
- Whether GHS (General HTML Support) is sufficient to keep arbitrary
  MDDM-adjacent attributes alive, or whether we must register explicit
  converters for every `data-mddm-*` we emit.
- Markers strategy for MDDM cross-references: operation-backed with
  `affectsData` vs plain element wrappers. Resolved in page 25.

## Sources

- https://ckeditor.com/docs/ckeditor5/latest/getting-started/setup/getting-and-setting-data.html
- https://ckeditor.com/docs/ckeditor5/latest/framework/architecture/intro.html
- https://ckeditor.com/docs/ckeditor5/latest/framework/architecture/editing-engine.html
  (followed from the architecture intro for pipeline and conversion detail)
