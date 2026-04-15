---
title: Widgets
status: draft
area: editing
priority: HIGH
---

# 11 — Widgets

Widgets are the foundation for every structured MDDM primitive: `mddmSection`,
`mddmRepeatable`, `mddmDataTable`, `mddmFieldGroup`, and `mddmRichBlock` all map
to either a block or inline widget, usually with one or more nested editables.
Understanding the widget contract is therefore a prerequisite for pages 19–23.

## 1. What a widget is

A widget is a view element (backed by a model element) that the editor treats
as an **atomic unit** for selection, copy, paste and deletion. The user can
click it to select the whole thing, drag it, and delete it in one keystroke,
but cannot place the caret *inside* the widget chrome. Editable regions, if
any, live in explicitly marked sub-elements called *nested editables*.

> "You can now click the widget to select it. Once the widget is selected, it
> is easier to copy and paste." — *Implementing a block widget*

Schema-wise, a widget is any element whose schema item has `isObject: true`.
The engine docs put it bluntly:

> "A `image` is of the 'object' type and a `paragraph` is of the 'block' type"
> — enabling the editor to treat object elements as atomic units where
> selection cannot be placed inside them. — *Editing engine architecture*

Two helpers from `@ckeditor/ckeditor5-widget/src/utils` turn ordinary view
elements into widget-aware ones:

- `toWidget( viewElement, writer, options? )` — marks a container view element
  as a widget: non-editable chrome, selectable as a unit, receives the widget
  outline, drag handle and selection handler.
- `toWidgetEditable( viewElement, writer )` — marks a view element *inside* a
  widget as editable. Its contents are editable; the surrounding widget chrome
  stays locked.

Both helpers apply **only in the editing downcast pipeline**. The data
downcast must emit plain, semantic HTML so that stored content is not polluted
with editing-only attributes like `contenteditable` or `data-ck-*`.

## 2. Block widget — the canonical pattern

The official *Simple Box* tutorial shows the full block-widget skeleton. Our
`mddmSection` follows the same shape.

### 2.1 Schema

```js
schema.register( 'simpleBox', {
    inheritAllFrom: '$blockObject'        // isObject + isBlock + allowed where blocks are
} );

schema.register( 'simpleBoxTitle', {
    isLimit: true,                        // caret cannot escape by normal means
    allowIn: 'simpleBox',
    allowContentOf: '$block'              // inline text + attributes, no nested blocks
} );

schema.register( 'simpleBoxDescription', {
    isLimit: true,
    allowIn: 'simpleBox',
    allowContentOf: '$root'               // full blocks allowed
} );
```

`$blockObject` is the generic "block + object" base. `isLimit: true` on the
nested editables is critical: it prevents selection from crossing out into the
widget chrome and stops destructive merges when the user presses Backspace at
the start of the region.

### 2.2 Converters

Three pipelines, three converters. The data view is a plain `<section>`; the
editing view is the same element wrapped by `toWidget` with nested editables.

```js
// Upcast (HTML -> model)
conversion.for( 'upcast' ).elementToElement( {
    model: 'simpleBox',
    view: { name: 'section', classes: 'simple-box' }
} );

// Data downcast (model -> saved HTML) — plain, no widget wrapping
conversion.for( 'dataDowncast' ).elementToElement( {
    model: 'simpleBox',
    view: { name: 'section', classes: 'simple-box' }
} );

// Editing downcast (model -> editing view) — wrapped as widget
conversion.for( 'editingDowncast' ).elementToElement( {
    model: 'simpleBox',
    view: ( modelElement, { writer } ) => {
        const section = writer.createContainerElement( 'section', {
            class: 'simple-box'
        } );
        return toWidget( section, writer, { label: 'simple box widget' } );
    }
} );

// Nested editable — title
conversion.for( 'editingDowncast' ).elementToElement( {
    model: 'simpleBoxTitle',
    view: ( modelElement, { writer } ) => {
        const h1 = writer.createEditableElement( 'h1', {
            class: 'simple-box-title'
        } );
        return toWidgetEditable( h1, writer );
    }
} );
```

The `label` option passed to `toWidget` is the accessible name announced by
screen readers and shown on the widget selection handle.

## 3. Inline widget

Use when the widget flows inline with text — fields, placeholders, chips.
`mddmFieldGroup` inline fields map here.

```js
schema.register( 'placeholder', {
    inheritAllFrom: '$inlineObject',      // allowed wherever $text is allowed
    allowAttributes: [ 'name' ]
} );
```

Inline widgets still use `toWidget` on the view element (typically a `<span>`),
but selection and typing behaviour differ: the caret can sit *next to* the
widget but not inside it, and arrow keys cross it as a single step.

Because the model element has no children but the view renders real content,
an inline widget needs a **position mapper** so the engine can translate
between "position after the placeholder" in the view and the equivalent model
position:

```js
this.editor.editing.mapper.on(
    'viewToModelPosition',
    viewToModelPositionOutsideModelElement(
        this.editor.model,
        viewElement => viewElement.hasClass( 'placeholder' )
    )
);
```

## 4. Nested editables

A widget can expose **multiple** editable regions. This is the pattern we need
for `mddmSection` (locked header + editable body), `mddmRepeatable` (N editable
items), and any field group with multiple slots.

The tutorial emphasises that each editable must contain at least one block for
text input to work:

> "There must be at least one paragraph for the description to be editable."
> — *Implementing a block widget*

When the structure is more complex than one-model-element-to-one-view-element,
prefer `elementToStructure` in the editing downcast so the whole view tree is
built in one pass and children are slotted in via `writer.createSlot()`. A
sketch:

```js
conversion.for( 'editingDowncast' ).elementToStructure( {
    model: { name: 'mddmSection', children: true },
    view: ( modelElement, { writer } ) => {
        const section = writer.createContainerElement( 'section', {
            class: 'mddm-section'
        } );
        const header  = writer.createContainerElement( 'header', {
            class: 'mddm-section__header'
        } );
        const body    = writer.createEditableElement( 'div', {
            class: 'mddm-section__body'
        } );

        writer.insert( writer.createPositionAt( section, 0 ), header );
        writer.insert( writer.createPositionAt( section, 'end' ),
            toWidgetEditable( body, writer ) );

        // Slot nested children of the section model element into `body`.
        writer.insert( writer.createPositionAt( body, 0 ), writer.createSlot() );

        return toWidget( section, writer, { label: 'section widget' } );
    }
} );
```

`elementToStructure` also re-runs automatically when any attribute listed in
its `triggerBy.attributes` config changes — this is the mechanism we'll use in
section 7 to keep widget chrome in sync with model attributes.

## 5. Commands for widget insertion

The idiomatic insertion command uses `model.insertObject`, which handles
splitting surrounding blocks and placing selection correctly for an object:

```js
export default class InsertSectionCommand extends Command {
    execute() {
        this.editor.model.change( writer => {
            const section = writer.createElement( 'mddmSection' );
            const header  = writer.createElement( 'mddmSectionHeader' );
            const body    = writer.createElement( 'mddmSectionBody' );
            writer.append( header, section );
            writer.append( body, section );
            writer.appendElement( 'paragraph', body );  // mandatory block
            this.editor.model.insertObject( section );
        } );
    }

    refresh() {
        const sel = this.editor.model.document.selection;
        this.isEnabled = this.editor.model.schema
            .findAllowedParent( sel.getFirstPosition(), 'mddmSection' ) !== null;
    }
}
```

## 6. Selection and keyboard mechanics

Widgets mounted with `toWidget` get three baked-in behaviours from the
`Widget` plugin:

1. **Widget selection handler** — click anywhere on the chrome selects the
   widget as a whole ("fake selection" on the model side).
2. **Arrow-key navigation** — arrows move the caret to the edge adjacent to
   the widget, then select it, then cross it. Enter/typing past a selected
   block widget inserts a paragraph after it.
3. **Delete/Backspace** removes the widget as a unit when it is selected.

Esc is used by the widget toolbar to return focus to the main editable. Inside
a nested editable, Esc does **not** automatically jump out of the widget —
that is handled by the standard selection/keystroke logic.

## 7. Widget attributes and reconversion

Widget chrome often depends on model attributes (e.g. a section variant
`data-variant="callout"`). Two strategies:

- **Attribute-to-attribute converters** for simple DOM attribute mirroring.
- **Element reconversion via `triggerBy`** when the attribute changes the
  structure of the view (different tag, different nested editables, added
  toolbar affordance). Configure `triggerBy: { attributes: [ 'variant' ] }`
  on the `elementToStructure` call and CK5 will re-run the converter whenever
  the listed attribute changes.

For data fetched asynchronously (external sources), the pattern is to call
`editor.editing.reconvertItem( element )` after the fetch completes:

> "Iterate over whole editor content, search for external data widget
> instances and trigger `reconvertItem` function" — *External data widget*

## 8. Drag and drop, copy and paste

A correctly-marked widget gets drag, drop, copy and paste for free. The drag
handle is rendered automatically by `toWidget` and the clipboard pipeline
serialises the model sub-tree rooted at the widget element.

## 9. React inside a widget

The official *Using React in a widget* tutorial mounts React via a renderer
function injected through editor config. The widget's editing view exposes a
raw DOM node and the app owns the React root:

```js
// editor config
products: {
    productRenderer: ( id, domElement ) => {
        const root = createRoot( domElement );
        root.render( <ProductPreview id={ id } /> );
    }
}

// inside the editing downcast
const reactWrapper = writer.createRawElement(
    'div',
    { class: 'product__react-wrapper' },
    function ( domElement ) {
        editor.config.get( 'products.productRenderer' )( id, domElement );
    }
);
```

Data view stays a plain `<section class="product" data-id="...">` so that
stored content is not tied to the rendering strategy:

> "The data view ... ensures data doesn't become outdated when styling
> changes." — *Using React in a widget*

Custom React event handlers inside a widget should be placed on elements
tagged with `data-cke-ignore-events` so CK5's own listeners do not interfere.

For **editable** React content inside a widget the portal pattern is not
officially documented; rendering arbitrary editable React inside a nested
editable would fight the engine's view diff. ⚠ uncertain — if we ever need
custom UI *inside* a `mddmRichBlock`, keep React to non-editable chrome
(rawElement) and use plain CK5 editables for text.

## 10. Focus tracking

Nested editables each get their own `contenteditable="true"` in the DOM. As a
consequence, moving the caret between them fires DOM focus/blur events:

> "Every nested editable in the content has the `contenteditable` attribute,
> too, and for the web browser moving your caret inside it means the main
> editable element is blurred and the nested one is focused." —
> *Focus tracking deep-dive*

The `FocusTracker` API (`add`, `remove`, observable `isFocused`,
`focusedElement`) is how widget UI components (e.g. a section toolbar) decide
when to show themselves without flickering off while the user moves the caret
between the section header and the section body.

## 11. Widgets and restricted editing

This is the interaction that matters most for MDDM and is the least
documented.

- The official restricted-editing docs do **not** describe how exceptions
  compose with widgets. ⚠ uncertain on exact interaction rules.
- **Correction (2026-04-15):** CK5 v48 supports **both** inline and block
  restricted-editing exceptions via two separate commands —
  `restrictedEditingException` (inline `<span>`) and
  `restrictedEditingExceptionBlock` (block wrapper, likely `<div>`). The
  older "span-only" memory was obsolete and has been superseded. See page
  [10 — Restricted editing](./10-restricted-editing.md) for the resolved
  finding and the three command variants.
- A widget's nested editable is **not** automatically a restricted-editing
  exception. In restricted mode the engine locks everything except explicit
  exception markers; without a marker, even an editable nested region reads
  as non-editable to the feature.
- **Working plan (revised):** at widget insertion time
  (in `InsertSectionCommand` and friends), wrap each nested editable that
  should stay writable in Fill mode with the appropriate exception:
  - **Block exception** for regions that should accept full editing (lists,
    tables, images, paragraphs) — e.g. `mddmSectionBody`, `mddmRichBlock`,
    each `mddmRepeatableItem`. Execute
    `editor.execute( 'restrictedEditingExceptionBlock' )` over the nested
    editable range, or insert the marker programmatically via
    `writer.addMarker()` at creation time.
  - **Inline exception** for short fill-in-the-blank values inside a
    locked paragraph or heading — e.g. a single field value inside
    `mddmSectionHeader`. Use `restrictedEditingException`.
- `restrictedEditing.allowedCommands` config only applies **inside inline
  exceptions**. Block exceptions automatically unlock all editor commands
  loaded in the editor — so restrict the plugin list itself if a block
  exception must not support, say, table insertion.
- Modes are mutually exclusive (`StandardEditingMode` vs
  `RestrictedEditingMode`). Toggling requires destroying and re-initialising
  the editor instance; the marker data survives via serialised HTML.

Open confirmations (still warrant a spike):

- Exact block wrapper tag emitted by `restrictedEditingExceptionBlock` on
  v48 (docs describe behaviour but do not quote the tag literally).
- Whether a block exception wrapper may surround a widget's nested
  editable directly, or must sit inside it around block children.
- Whether nested exceptions (inline inside block, or block inside block)
  are supported.

See page 10 (restricted editing) and page 19 (sections mapping) for the
full confirmation plan.

## 12. Widgets and native tables

CK5's native `table` feature is itself implemented as a widget whose cells are
nested editables. That is the pattern we will study when deciding the
`mddmDataTable` strategy on page 21: either configure the native table with a
constrained schema (no cell merging, fixed columns) or build a custom widget
following the same N-row, per-cell-editable shape.

## 13. Debugging — CKEditor Inspector

`@ckeditor/ckeditor5-inspector` attaches a panel that shows, live:

- the model tree (with attributes) — essential to confirm `isObject`/`isBlock`
  flags and nested editable hierarchy;
- the view tree — shows `toWidget`/`toWidgetEditable` markers and
  `contenteditable` attribution;
- commands state, markers, schema definitions.

Install in dev builds only. Call `CKEditorInspector.attach( editor )` after
editor creation. Inspector is invaluable when a widget "looks right" but the
caret behaves oddly — nine times out of ten the model schema is wrong, not
the CSS.

## 14. Mapping to MDDM primitives (pointers)

Concrete schemas live on pages 19–23; this is only a routing table.

| MDDM primitive     | Widget kind          | Notes                                                                 |
|--------------------|----------------------|-----------------------------------------------------------------------|
| `mddmSection`      | block widget         | Locked header + body wrapped in a **block** restricted-editing exception. Inline `<span>` exceptions used only for fill-in-blank values inside the locked header. See page 19. |
| `mddmRepeatable`   | block widget         | N nested editables, each wrapped in a block exception; add/remove commands via `insertObject`/`remove`. See page 20. |
| `mddmDataTable`    | native CK5 table or custom widget | Decision pending — page 21. Avoid nested CKEditor tables (memory: nested tables break DOCX export). |
| `mddmFieldGroup`   | inline widget *or* marker anchor  | Inline widget where the field renders as a chip; marker-based where we only need a semantic anchor. Decision on page 22. |
| `mddmRichBlock`    | nested editable inside a block widget, wrapped in a block exception | Full toolbar allowed inside. React content only in non-editable chrome. Page 23. |

## Open questions

- Restricted-editing exception composition with widget nested editables —
  needs a concrete spike. ⚠
- Whether `elementToStructure` + `triggerBy` is sufficient for
  `mddmRepeatable` row add/remove without manual `reconvertItem` calls.
- Whether the native table widget can be constrained enough for
  `mddmDataTable` or if a bespoke widget is cheaper long-term.

## Sources

- https://ckeditor.com/docs/ckeditor5/latest/features/custom-components.html
  — overview page; routes to the block / inline / external-data / React widget
  tutorials already listed below. No new widget-construction content beyond
  those tutorials; UI-component construction (buttons, dropdowns, balloons) is
  deferred to `framework/architecture/ui-library.html`, which belongs on a
  future UI-focused page rather than this widgets page.
- https://ckeditor.com/docs/ckeditor5/latest/framework/tutorials/widgets/implementing-a-block-widget.html
- https://ckeditor.com/docs/ckeditor5/latest/framework/tutorials/widgets/implementing-an-inline-widget.html
- https://ckeditor.com/docs/ckeditor5/latest/framework/tutorials/widgets/data-from-external-source.html
- https://ckeditor.com/docs/ckeditor5/latest/framework/tutorials/widgets/using-react-in-a-widget.html
- https://ckeditor.com/docs/ckeditor5/latest/framework/deep-dive/ui/focus-tracking.html
- https://ckeditor.com/docs/ckeditor5/latest/framework/architecture/editing-engine.html
- https://ckeditor.com/docs/ckeditor5/latest/features/restricted-editing.html
- https://ckeditor.com/docs/ckeditor5/latest/framework/development-tools/inspector.html

Fetched but 404 (not available at the documented paths, to revisit):

- `framework/deep-dive/ui/keyboard-support.html`
- `framework/development-tools/inspector.html` (landing redirected)
- `features/editor-restricted-editing.html` (moved to `features/restricted-editing.html`)
