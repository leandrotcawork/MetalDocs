---
title: Markers
status: draft
area: core
priority: HIGH
---

# 07 — Markers

Markers are CKEditor 5's mechanism for attaching **named metadata to a range** of the model without altering the structural tree. They are the right tool for things that span content but are not content themselves: restricted-editing exceptions, comment anchors, track-changes suggestions, search highlights, field IDs, spellcheck squiggles.

This page explains what a marker is, how it differs from attributes and elements, how persistence is controlled, and how the MetalDocs IR bridge should think about them.

> Related reading — these two pages discuss marker *persistence* in concrete contexts and should not be duplicated here:
> - [10 — Restricted editing](./10-restricted-editing.md) (how `restrictedEditingException:*` markers are created and downcast)
> - [24 — Data format](./24-data-format.md) (how markers are serialised into the HTML the backend stores)

---

## 1. What a marker *is*

A marker is a **named live range on the model document**. From the API reference for the `Marker` class:

> "Range of the marker is updated automatically when document changes, using live range mechanism."
> — [`module_engine_model_markercollection-Marker`](https://ckeditor.com/docs/ckeditor5/latest/api/module_engine_model_markercollection-Marker.html)

And from `MarkerCollection`:

> "The collection of all markers attached to the document. … To create, change or remove markers use model writers' methods: `addMarker` or `removeMarker`. Since the writer is the only proper way to change the data model it is not possible to change markers directly using this collection."
> — [`module_engine_model_markercollection-MarkerCollection`](https://ckeditor.com/docs/ckeditor5/latest/api/module_engine_model_markercollection-MarkerCollection.html)

Key consequences:

- A marker **tracks its range automatically**: insertions before it shift it, deletions inside it shrink it, removal of its container dissolves it.
- A marker is **not** in the model tree. Elements don't "have" markers as children. You access them via `editor.model.markers`.
- A marker has a **unique name** (string). Names are the identity — two markers with the same name cannot coexist.

---

## 2. Marker vs attribute vs element

Decide what kind of artefact a piece of metadata should be by asking *what it semantically is*:

| If the thing is …                                                                     | Use        |
| ------------------------------------------------------------------------------------- | ---------- |
| Part of the document's structure (section, table cell, image)                         | Element    |
| A per-character formatting trait that should follow the text when copied (bold, link) | Attribute  |
| A named region of arbitrary extent, possibly spanning blocks, that edits should move  | **Marker** |

Heuristics:

- If the metadata must survive copy/paste *as a formatting property of the text*, it's an attribute.
- If the metadata is "a thing someone added to point at a region" (a comment, a suggestion, an exception window, a found-search hit), it's a marker.
- If the metadata has its own children and can be navigated into, it's an element.

Markers are specifically designed for **range metadata that can span multiple blocks**, which attributes cannot represent cleanly.

---

## 3. Marker API surface

All marker mutations go through the writer inside a `model.change()` block:

```js
editor.model.change( writer => {
    writer.addMarker( 'mddmField:customerName', {
        range,
        usingOperation: true,
        affectsData: true,
    } );

    writer.updateMarker( 'mddmField:customerName', { range: newRange } );

    writer.removeMarker( 'mddmField:customerName' );
} );
```

Read access:

```js
const marker = editor.model.markers.get( 'mddmField:customerName' );
for ( const m of editor.model.markers.getMarkersGroup( 'mddmField' ) ) { /* … */ }
for ( const m of editor.model.markers ) { /* all */ }
```

Events: `editor.model.markers` fires `update:<name>` whenever a marker is added, moved, or removed — useful for syncing external UI state (e.g. the sidebar list of fields).

---

## 4. Persistence flags

Two booleans on `addMarker` determine the marker's lifecycle. Defaults are `false` for both.

### `usingOperation` (a.k.a. `managedUsingOperations`)

> "This property indicates whether the marker uses operations for management. … those managed directly without operations (useful as bookmarks), and those managed through operations (handled in undo stack and synchronized between clients)."
> — `Marker` API ref

- `usingOperation: true` → marker changes are **operations**. They are undoable, redoable, and travel over collaboration sync.
- `usingOperation: false` → marker is a purely local bookmark. Undo/redo will **not** restore it, collaborators will not see it.

### `affectsData`

> "Specifies whether the marker affects the data produced by the data pipeline (is persisted in the editor's data)."
> — `MarkerCollection._set` API ref

- `affectsData: true` → the marker is considered part of the document's data. Combined with a downcast-to-data converter (see §5), it round-trips through `editor.getData()` / `editor.setData()`.
- `affectsData: false` → the marker exists only in the live editor session; it does not appear in saved HTML.

### Combinations

| `usingOperation` | `affectsData` | Behaviour                                                                                           |
| ---------------- | ------------- | --------------------------------------------------------------------------------------------------- |
| `true`           | `true`        | **Persisted, collaborative, undoable.** Use for restricted-editing exceptions, comments, field IDs. |
| `true`           | `false`       | Undoable/collab but not saved. Rarely useful.                                                       |
| `false`          | `false`       | Ephemeral UI highlight (search result, hover target). Dies on reload and on undo.                   |
| `false`          | `true`        | ⚠ uncertain — not a combination we've seen used in practice.                                        |

---

## 5. Downcast helpers for markers

Markers are invisible until you convert them. CKEditor provides three dedicated downcast helpers; each targets a different pipeline.

> Note — the CKEditor public downcast-helpers guide currently lists only element/attribute helpers; marker helpers are documented via the API reference and feature examples rather than the main deep-dive page. Verified by fetching [`framework/deep-dive/conversion/helpers/downcast.html`](https://ckeditor.com/docs/ckeditor5/latest/framework/deep-dive/conversion/helpers/downcast.html) on 2026-04-15.

### `markerToHighlight`

Wraps the marker's range in a highlight span (or applies classes to block elements fully contained in the range) in the **editing view**. Used for visual-only effects: find-and-replace highlights, comment range styling.

```js
editor.conversion.for( 'editingDowncast' ).markerToHighlight( {
    model: 'findResult',
    view: { classes: 'ck-find-result' },
} );
```

### `markerToElement`

Emits a **UI element** (usually empty, often a widget) in the editing view at the marker's boundaries. Useful for pin icons, comment indicators, or field start/end sentinels that the user can click.

```js
editor.conversion.for( 'editingDowncast' ).markerToElement( {
    model: 'commentPin',
    view: ( data, { writer } ) => writer.createUIElement( 'span', { class: 'cmt-pin' } ),
} );
```

### `markerToData`

Persists marker boundaries as **`data-*` attributes** on existing elements, or as dedicated boundary tags, in the `dataDowncast` pipeline. This is the helper that makes `affectsData: true` actually survive a round-trip.

```js
editor.conversion.for( 'dataDowncast' ).markerToData( { model: 'mddmField' } );
```

By default `markerToData` emits attributes of the form `data-<group>-start-before="<rest-of-name>"` / `data-<group>-end-after="<rest-of-name>"` on the nearest suitable view element. Name format matters: the part **before the first colon** is the group, the part after is the ID.

Corresponding **upcast** — when data is loaded, the matching upcast converter (registered automatically by `markerToData` for the dataUpcast direction, or explicitly via `dataToMarker`) recognises those attributes and re-creates `addMarker( 'group:id', … )` calls.

See [24 — Data format](./24-data-format.md) for the concrete HTML shape MetalDocs emits.

---

## 6. Naming convention

Marker names follow the pattern **`group:id`**. The prefix is not cosmetic:

- `markerToData` / `dataToMarker` **match on the group prefix**, so all `comment:*` markers share one conversion definition.
- Features use `MarkerCollection#getMarkersGroup( 'comment' )` to enumerate their markers.
- Undo/history serialisation groups by prefix.

Groups we expect to use in MetalDocs:

- `restrictedEditingException:<uid>` — managed by the restricted-editing feature itself (we don't register this group; see [10 — Restricted editing](./10-restricted-editing.md)).
- `mddmField:<fieldId>` — proposed anchor for template field IDs. ⚠ uncertain — decision pending, see §8.
- `comment:<threadId>` — if/when we adopt the commercial comments plugin.

---

## 7. Data round-trip (upcast)

A persisted marker survives a save/load cycle like this:

1. **Downcast** (`editor.getData()`): `markerToData` walks the model, finds markers whose group has a registered data converter and `affectsData: true`, and emits boundary attributes in the HTML output.
2. **Storage**: backend stores the HTML verbatim.
3. **Upcast** (`editor.setData( html )`): the matching upcast converter recognises the boundary attributes, builds a model range from the positions, and calls `writer.addMarker( name, { range, usingOperation: true, affectsData: true } )` during document load.

If the upcast converter is missing, markers are silently dropped even though `affectsData: true`. Always register both directions.

---

## 8. Practical patterns for MetalDocs

### Restricted-editing exceptions

The `RestrictedEditingMode` / `StandardEditingMode` plugins create `restrictedEditingException:<uid>` markers automatically with both flags set to `true` and ship their own `markerToData` / `dataToMarker` converters. **We do not register this group ourselves.** Treat these markers as read-only infrastructure. See [10 — Restricted editing](./10-restricted-editing.md).

### Template field anchors — open design question

Two candidate representations for "this span of prose is field `customerName`":

1. **Marker approach** — `mddmField:customerName` with `usingOperation: true, affectsData: true`, downcast via `markerToData`.
    - Pro: survives text edits around the field, can span multiple blocks, moves naturally with author edits, clean separation of data from structure.
    - Con: HTML output uses boundary attributes, which are less obvious to human readers and to non-CK tools.
2. **Named-element wrapper approach** — a `<fieldRegion name="customerName">…</fieldRegion>` model element.
    - Pro: obvious in HTML, direct to downcast, easy for server-side string replacement.
    - Con: structural; splits paragraphs at boundaries; block/inline duality is awkward.

The IR bridge ([25 — MDDM IR bridge](./25-mddm-ir-bridge.md)) will decide. Leading candidate right now is the element wrapper for simplicity, falling back to markers only if we need multi-block field regions. ⚠ uncertain — not yet locked.

### Ephemeral UI (search, hover, validation)

Use `usingOperation: false, affectsData: false` with `markerToHighlight`. Cheap, disposable, does not pollute undo or saved HTML.

---

## 9. Gotchas

- **No `usingOperation: true` → no undo.** The marker will vanish when the user hits Ctrl+Z and will not reappear on redo. Collaboration clients also won't see it.
- **No `affectsData: true` → lost on save.** Even if the marker exists in memory, `getData()` will not emit its boundaries.
- **No data converter registered → still lost on save.** `affectsData: true` is necessary but not sufficient; `markerToData` (or a custom equivalent) must exist for the group.
- **Content removal removes the marker.** If the user deletes all model content that the marker's range covered, the marker is auto-removed — no explicit cleanup needed, but listeners must handle the `update:*` event going to `newRange == null`.
- **Schema changes can collapse markers.** Splitting a block, merging blocks, or running `writer.remove()` on a parent may shrink a marker to zero length; CK then removes it.
- **Names are global.** `addMarker( 'foo', … )` when `foo` exists updates the existing marker. To avoid collisions across features, always use group prefixes.
- **Markers are not attributes.** You cannot query "does this text node have marker X"; you must test whether a position lies inside a marker's range.

---

## Sources

Fetched 2026-04-15:

- https://ckeditor.com/docs/ckeditor5/latest/framework/architecture/intro.html — architecture overview (marker section not present in current revision; retained for context).
- https://ckeditor.com/docs/ckeditor5/latest/framework/deep-dive/conversion/downcast.html — downcast overview.
- https://ckeditor.com/docs/ckeditor5/latest/framework/deep-dive/conversion/helpers/downcast.html — downcast helpers listing (marker helpers not included in the public listing; API ref used instead).
- https://ckeditor.com/docs/ckeditor5/latest/api/module_engine_model_markercollection-MarkerCollection.html — `MarkerCollection` API (verbatim quotes above).
- https://ckeditor.com/docs/ckeditor5/latest/api/module_engine_model_markercollection-Marker.html — `Marker` class API (verbatim quotes above).
- https://ckeditor.com/docs/ckeditor5/latest/framework/deep-dive/custom-data.html — ⚠ 404 at fetch time; the canonical "custom data" deep-dive may have moved. Follow-up needed.

Cross-references inside this wiki:

- [10 — Restricted editing](./10-restricted-editing.md)
- [24 — Data format](./24-data-format.md)
- [25 — MDDM IR bridge](./25-mddm-ir-bridge.md)
