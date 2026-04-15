---
title: Conversion (upcast / downcast)
status: draft
area: core
---

# 05 — Conversion

CKEditor 5's engine is split into a **model** (abstract document tree) and a **view** (renderable DOM-like tree). The **conversion** layer is the bridge. From the official deep-dive:

> "The process of transforming one into the other is called conversion."
> — [Conversion intro](https://ckeditor.com/docs/ckeditor5/latest/framework/deep-dive/conversion/intro.html)

Everything that appears on screen, gets serialized to HTML, or gets parsed from HTML goes through converters. If your model element has no converter, it is invisible and unsavable.

## 1. Three pipelines, two directions

There are **two** downcast pipelines and **one** upcast pipeline:

| Pipeline          | Direction      | When it runs                                    |
|-------------------|---------------|-------------------------------------------------|
| `upcast`          | view → model  | `editor.setData()`, paste, initial data load    |
| `dataDowncast`    | model → view  | `editor.getData()`, autosave serialization      |
| `editingDowncast` | model → view  | every model change, rendered to the live DOM    |

There is no "data upcast" vs "editing upcast" distinction — both the data view and editing view originate from the same upcasted model.

From the downcast deep-dive:

> "Sometimes you may want to alter the converter logic for a specific pipeline. For example, in the editing view you may want to add some additional class to the view element."
> — [Downcast conversion](https://ckeditor.com/docs/ckeditor5/latest/framework/deep-dive/conversion/downcast.html)

The **data pipeline** must emit clean, round-trippable HTML (this is what lands in the backend and what `upcast` has to read back). The **editing pipeline** emits the same structure wrapped with widget chrome (drag handles, selection outlines, editable regions).

## 2. API shape

All converters are registered through `editor.conversion.for(group)`:

```js
editor.conversion.for( 'upcast' )          // view → model
editor.conversion.for( 'downcast' )        // shorthand: both data + editing
editor.conversion.for( 'dataDowncast' )    // data pipeline only
editor.conversion.for( 'editingDowncast' ) // editing pipeline only
```

The typical pattern:

```js
editor.conversion.for( 'downcast' ).elementToElement( {
    model: 'paragraph',
    view: 'p'
} );
```

The shorthand `'downcast'` registers the same converter for both data and editing pipelines — fine for plain content, not enough for widgets (see §5).

## 3. Helpers cheat-sheet

### `elementToElement({ model, view })`

One model element ↔ one view element. The workhorse.

```js
editor.conversion.for( 'downcast' ).elementToElement( {
    model: 'heading',
    view: ( modelElement, { writer } ) =>
        writer.createContainerElement( 'h' + modelElement.getAttribute( 'level' ) )
} );
```

Symmetric on upcast:

```js
editor.conversion.for( 'upcast' ).elementToElement( {
    view: 'p',
    model: 'paragraph'
} );
```

### `elementToStructure({ model, view })` — downcast only

Use when one model element must render as **multiple** view elements or contain **nested editables**. Required for our Section widget (header + body slot) and for datatable row structure.

```js
editor.conversion.for( 'downcast' ).elementToStructure( {
    model: 'mddmSection',
    view: ( modelElement, { writer } ) => {
        return writer.createContainerElement( 'section', { class: 'mddm-section' }, [
            writer.createContainerElement( 'header', {}, [ writer.createSlot( 'header' ) ] ),
            writer.createContainerElement( 'div', { class: 'body' }, [ writer.createSlot() ] )
        ] );
    }
} );
```

Why not `elementToElement`? Because it assumes **one** container wrapping **all** children. A Section with a header editable and a body editable needs `createSlot(...)` to route children.

### `attributeToElement` — inline formatting

Used for text-level attributes that should become view elements.

```js
editor.conversion.for( 'downcast' ).attributeToElement( {
    model: 'bold',
    view: 'strong'
} );
```

### `attributeToAttribute`

Copies an attribute between model and view, e.g. `src`, `href`, `data-*`.

```js
editor.conversion.for( 'upcast' ).attributeToAttribute( {
    view: 'src',
    model: 'source'
} );
```

### `elementToAttribute` — upcast only

A view element collapses into a model text attribute:

```js
editor.conversion.for( 'upcast' ).elementToAttribute( {
    view: 'strong',
    model: 'bold'
} );
```

The asymmetric naming (`elementToAttribute` upcast ↔ `attributeToElement` downcast) trips people up — they are inverses of each other.

### `markerToElement`, `markerToData`, `markerToHighlight`

Markers are ranges that annotate the model without being elements. Downcast options:

- `markerToElement` — insert a view element (e.g. a bookmark pin) at each marker boundary.
- `markerToData` — serialize marker as `data-*` attributes on surrounding elements. Round-trippable through the **data** pipeline; this is what lets markers survive `getData()` → `setData()`.
- `markerToHighlight` — wrap the marked range in a CSS-class highlight (visual only, typically `editingDowncast`).

⚠ uncertain — exact signatures not fetched from the helpers deep-dive in this session. Confirm against [helpers/downcast.html](https://ckeditor.com/docs/ckeditor5/latest/framework/deep-dive/conversion/helpers/downcast.html) before implementing.

See page **07 — Markers** and page **24 — Data format** for the MetalDocs marker strategy (exception regions persisted via `markerToData`).

## 4. Priority

When multiple converters match the same model/view node, the one with the highest **priority** wins. Values: `'highest'`, `'high'`, `'normal'` (default), `'low'`, `'lowest'`.

> "The second one overrides it by setting the priority to `high`."
> — [helpers/downcast.html](https://ckeditor.com/docs/ckeditor5/latest/framework/deep-dive/conversion/helpers/downcast.html)

Rules of thumb:

- Plugin authors registering fallbacks → `low`.
- Overriding a third-party plugin's default converter → `high`.
- Avoid `highest` / `lowest` without a very specific reason; they fight each other across plugins.

## 5. Data vs editing downcast in practice

The same model element usually needs **two** downcast converters:

```js
// Data pipeline: clean HTML, what the backend stores.
editor.conversion.for( 'dataDowncast' ).elementToElement( {
    model: 'mddmSection',
    view: ( modelElement, { writer } ) =>
        writer.createContainerElement( 'mddm-section', {
            'data-variant': modelElement.getAttribute( 'variant' )
        } )
} );

// Editing pipeline: same structure, but wrapped as a widget.
editor.conversion.for( 'editingDowncast' ).elementToStructure( {
    model: 'mddmSection',
    view: ( modelElement, { writer } ) => {
        const section = writer.createContainerElement( 'mddm-section', {
            'data-variant': modelElement.getAttribute( 'variant' )
        }, [ writer.createSlot() ] );
        return toWidget( section, writer, { label: 'Section' } );
    }
} );
```

`toWidget` + `toWidgetEditable` (from `@ckeditor/ckeditor5-widget/src/utils`) add focus outlines, selection handling, and editable-region contracts. They must **only** appear in `editingDowncast` — if they leak into `dataDowncast`, the serialized HTML will contain `contenteditable`, `ck-widget` classes, etc.

## 6. Custom (callback) converters

When the helpers can't express the mapping (conditional logic, consuming sibling attributes, event-order dependencies), drop to the dispatcher API:

```js
editor.conversion.for( 'upcast' ).add( dispatcher => {
    dispatcher.on( 'element:div', ( evt, data, conversionApi ) => {
        if ( !data.viewItem.hasClass( 'mddm-field' ) ) return;
        if ( !conversionApi.consumable.test( data.viewItem, { name: true } ) ) return;
        // ...build model element, consume, safeInsert...
    }, { priority: 'high' } );
} );
```

Three primitives to know: `consumable.test()`, `consumable.consume()`, `safeInsert()`. Prefer helpers first; only fall back to callbacks when a helper genuinely can't express the rule.

## 7. Reconversion on attribute change

`elementToStructure` reconverts the **whole** view structure whenever a listed attribute on the model element changes. This matters for attributes that change structural shape (e.g. `mddmSection`'s `variant` changing the header layout).

⚠ uncertain — the exact option name (`triggerBy` vs implicit via structure reads) was not confirmed in the pages fetched. Verify against the current downcast helpers reference.

Attributes that only toggle a single `data-*` on the wrapper should use `attributeToAttribute` instead; it's cheaper than a full reconversion.

## 8. MetalDocs patterns

| Model                                | Data view                                            | Editing view                          |
|--------------------------------------|------------------------------------------------------|---------------------------------------|
| `mddmSection` (element + `variant`)  | `<mddm-section data-variant="...">…</mddm-section>`  | widget wrapper + editable body slot   |
| `mddmField` (text attribute)         | `data-mddm-field-id="..."` on inline span            | styled pill (editing class only)      |
| Exception region (marker)            | `data-exception-start`/`-end` via `markerToData`     | `markerToHighlight` — blue underline  |
| `mddmRepeatable`                     | `<mddm-repeatable>` with children                    | widget with per-row controls          |

Marker-based exception persistence is covered in **24 — Data format**; not duplicated here.

## 9. Pitfalls

- **Register after the schema rule, before first `setData()`.** If a converter fires for a model element whose `schema.register(...)` hasn't run, the element is dropped silently. If `setData()` runs before the converter is registered, you will see raw source or empty content on first paint.
- **`'downcast'` shorthand + widgets = broken HTML.** Always split into `dataDowncast` (clean) and `editingDowncast` (widgetized) once widget chrome is involved.
- **Upcast priority collisions with GHS.** If General HTML Support is enabled, its permissive upcasters may claim your element first — register your upcaster at `high` priority.
- **Attribute-only changes trigger reconversion with `elementToStructure`.** Don't park presentational attributes on a Structure parent; put them on a child element via `attributeToAttribute`.
- **Markers don't round-trip by default.** Only `markerToData` (or a custom persistence scheme) survives `getData()` → `setData()`.

## Sources

- [Architecture intro (conversion section)](https://ckeditor.com/docs/ckeditor5/latest/framework/architecture/intro.html)
- [Conversion deep-dive — intro](https://ckeditor.com/docs/ckeditor5/latest/framework/deep-dive/conversion/intro.html)
- [Downcast conversion](https://ckeditor.com/docs/ckeditor5/latest/framework/deep-dive/conversion/downcast.html)
- [Upcast conversion](https://ckeditor.com/docs/ckeditor5/latest/framework/deep-dive/conversion/upcast.html)
- [Downcast helpers reference](https://ckeditor.com/docs/ckeditor5/latest/framework/deep-dive/conversion/helpers/downcast.html)
- [Implementing a block widget (tutorial)](https://ckeditor.com/docs/ckeditor5/latest/framework/tutorials/widgets/implementing-a-block-widget.html)

## Cross-refs

- **04 — Schema** — converters only work for registered model elements.
- **07 — Markers** — marker lifecycle and API.
- **11 — Widgets** — `toWidget`, `toWidgetEditable`, selection behavior.
- **19 — Sections mapping**, **20 — Repeatables mapping**, **21 — Datatable mapping** — concrete converter sets.
- **24 — Data format** — marker persistence via `markerToData`.
