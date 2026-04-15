---
title: RichBlock mapping
status: draft
area: mapping
---

# 23 — RichBlock mapping

`RichBlock` is MetalDocs' name for a **template-declared region where a
Fill-mode user may type fully-rich content** — paragraphs, lists, flat tables,
images, links, headings. It is the "miniature full editor inside a locked
template" primitive. The legacy BlockNote `RichBlock` (free-form region with
its own toolbar) is reference only; CK5 has a native, cheaper construct.

---

## Recommended design

**Do not build a widget.** A `RichBlock` is just a **block restricted-editing
exception** around a `<div class="mddm-rich-block">` in the template. The
block exception automatically enables every editor command inside its range
(see [10 — Restricted editing](./10-restricted-editing.md) §allowedCommands),
so Fill users already get the full toolbar when the caret sits inside. No
custom schema, no widget chrome, no converters beyond a small GHS/element
rule for the class.

Legacy deviation: BlockNote's `RichBlock` carried its own inline toolbar and
bespoke block model. CK5's restricted-editing block exception subsumes both
responsibilities. Preserving a 1:1 port would be redundant.

Build the widget flavour (see §Widget alternative, below) **only** if product
needs one of:

1. An Author-mode "Insert rich block" button with a visible empty placeholder
   ("Type here…") before the user starts filling.
2. A labelled region chrome (accessible name shown in the Author editor).
3. Structural guarantees not expressible via the exception alone (e.g. "this
   rich block cannot be deleted", "must always exist in position N").

If none of those apply, ship the exception-only path.

---

## 1. What a RichBlock is (recommended, exception-only)

- A template author, in Author mode, selects one or more blocks inside a
  `<div class="mddm-rich-block">` they inserted and runs
  `restrictedEditingExceptionBlock`.
- The resulting marker (`restrictedEditingException:<uid>`) survives
  `getData()` / `setData()` via the feature's own `markerToData` converters.
- In Fill mode (`RestrictedEditingMode`), the caret can enter that range and
  **every loaded editor command is enabled** while it is inside.
- Outside the range, the document is read-only.

The `<div class="mddm-rich-block">` is pure styling / semantic classification.
The exception marker is what grants write access.

### Schema / converters needed

None beyond what the template root already permits. Two small choices:

- **Via GHS** — add `{ name: 'div', classes: [ 'mddm-rich-block' ] }` to
  `htmlSupport.allow`. Cheapest; uses the generic `htmlDivParagraph`
  converter. Recommended for v1.
- **Via a minimal element rule** — register a `mddmRichBlockContainer`
  container element inheriting from `$container` with
  `allowContentOf: '$root'`, plus symmetric element-to-element converters.
  Slightly more control (you can hang a `post-fixer` off it to forbid nested
  rich blocks — see §8). Use this if GHS proves lossy on round-trip.

### Insertion command (Author mode)

```ts
export class InsertRichBlockCommand extends Command {
  execute() {
    const { model } = this.editor;
    model.change( writer => {
      // 1. Insert the container div + an initial paragraph.
      const div = writer.createElement( 'htmlDivParagraph', {
        htmlDivAttributes: { classes: [ 'mddm-rich-block' ] }
      } );
      writer.appendElement( 'paragraph', div );
      model.insertObject( div, null, null, { setSelection: 'in' } );

      // 2. Wrap it in a block exception so Fill mode unlocks it.
      this.editor.execute( 'restrictedEditingExceptionBlock' );
    } );
  }

  refresh() {
    const sel = this.editor.model.document.selection;
    this.isEnabled = this.editor.model.schema
      .findAllowedParent( sel.getFirstPosition()!, 'htmlDivParagraph' ) !== null;
  }
}
```

`⚠ uncertain` — exact GHS model element name (`htmlDivParagraph` vs
`htmlDiv`) depends on the v48 GHS build. Confirm in the Inspector before
relying on it (see [03 — Engine model](./03-engine-model.md) §Inspector). If
you switch to the bespoke `mddmRichBlockContainer`, replace the element name
and drop the `htmlDivAttributes` payload.

---

## 2. Widget alternative (only if product asks for chrome)

Use this shape if you need an empty-state placeholder, an Author button that
creates discoverable scaffolding, or guaranteed non-deletability.

### 2.1 Schema

```ts
schema.register( 'mddmRichBlock', {
  inheritAllFrom: '$blockObject',
  allowChildren: [ 'mddmRichBlockContent' ],
  allowAttributes: [ 'richBlockId', 'label' ]
} );

schema.register( 'mddmRichBlockContent', {
  isLimit: true,
  allowIn: 'mddmRichBlock',
  allowContentOf: '$root'        // paragraphs, lists, tables, images, headings
} );
```

`isLimit` on the nested editable is mandatory (see
[11 — Widgets](./11-widgets.md) §nested-editables) — it prevents Backspace
from merging content into the widget chrome.

### 2.2 Converters

```ts
// Data downcast — plain HTML the backend stores.
conversion.for( 'dataDowncast' ).elementToStructure( {
  model: 'mddmRichBlock',
  view: ( modelEl, { writer } ) => {
    const wrapper = writer.createContainerElement( 'div', {
      class: 'mddm-rich-block',
      'data-rich-block-id': modelEl.getAttribute( 'richBlockId' )
    } );
    writer.insert( writer.createPositionAt( wrapper, 0 ), writer.createSlot() );
    return wrapper;
  }
} );

// Editing downcast — same shape + widget chrome + widget-editable slot.
conversion.for( 'editingDowncast' ).elementToStructure( {
  model: 'mddmRichBlock',
  view: ( modelEl, { writer } ) => {
    const wrapper = writer.createContainerElement( 'div', {
      class: 'mddm-rich-block',
      role: 'region',
      'aria-label': modelEl.getAttribute( 'label' ) ?? 'Rich block'
    } );
    const body = writer.createEditableElement( 'div', {
      class: 'mddm-rich-block__body'
    } );
    writer.insert( writer.createPositionAt( body, 0 ), writer.createSlot() );
    writer.insert( writer.createPositionAt( wrapper, 0 ),
      toWidgetEditable( body, writer ) );
    return toWidget( wrapper, writer, { label: 'Rich block' } );
  }
} );

// Upcast — data → model.
conversion.for( 'upcast' ).elementToElement( {
  view: { name: 'div', classes: 'mddm-rich-block' },
  model: ( viewEl, { writer } ) => writer.createElement( 'mddmRichBlock', {
    richBlockId: viewEl.getAttribute( 'data-rich-block-id' )
  } )
} );
```

### 2.3 Insertion command (widget flavour)

```ts
export class InsertRichBlockCommand extends Command {
  execute( { label }: { label?: string } = {} ) {
    const { model } = this.editor;
    model.change( writer => {
      const block = writer.createElement( 'mddmRichBlock', {
        richBlockId: crypto.randomUUID(),
        label: label ?? 'Rich content'
      } );
      const body = writer.createElement( 'mddmRichBlockContent' );
      writer.appendElement( 'paragraph', body );   // mandatory seed block
      writer.append( body, block );
      model.insertObject( block, null, null, { setSelection: 'in' } );

      // Grant Fill-mode write access over the nested editable's range.
      const range = model.createRangeIn( body );
      writer.addMarker( `restrictedEditingException:rb-${ crypto.randomUUID() }`, {
        range, usingOperation: true, affectsData: true
      } );
    } );
  }
}
```

Rationale for `addMarker` instead of executing
`restrictedEditingExceptionBlock`: at insertion time the selection is inside
the widget's nested editable, and it is simpler to drop the marker
programmatically than to move the selection around. `⚠ uncertain` — whether a
marker created this way is recognised by `RestrictedEditingModeEditing` as a
block exception on re-init; verify by round-tripping through `getData()` +
`setData()` with the Fill-mode plugin list.

---

## 3. Empty-state UX

Two options; pick one consistently across Author and Fill views.

- **CK5 placeholder (preferred)** — pass
  `placeholder: 'Type content here…'` on the editable config. For a widget
  nested editable, set it per-editable via the
  `Placeholder` plugin's `enablePlaceholder({ view, element, text })`
  API after editor init.
- **CSS `:empty::before`** — style
  `.mddm-rich-block:empty::before { content: attr(data-placeholder); … }`.
  Simpler, but loses the behaviour of CK5's placeholder (hides on focus,
  etc.).

---

## 4. Toolbar scope

The editor's **balloon toolbar** should expose all inline formatting
(bold, italic, underline, link, inline code, font, color) — it pops when
the caret is inside the rich block. The **block toolbar** (gutter handle
on empty paragraphs) should expose `insertTable`, `insertImage`, `bulletedList`,
`numberedList`, `heading`.

In Fill mode the restricted-editing feature automatically disables commands
outside exceptions, so there is no need to conditionally hide toolbar
buttons — clicking a disabled button does nothing. Toolbar composition is
identical for Author and Fill; only the enable/disable layer differs.

---

## 5. Contents allowed

Inside a RichBlock we want: `$block` (paragraphs), `heading1..3`, `listItem`
via the List plugin, `imageBlock`, `imageInline`, `table` (flat), `link`
attributes on text.

Must forbid, via `schema.addChildCheck`:

- `mddmSection` inside any rich block (sections are template-level).
- Nested `mddmRichBlock` inside `mddmRichBlock` (flatten or reject).
- Nested `table` inside a `table` cell — see the Brain memory
  `feedback_ckeditor5_nested_tables_docx` (nested tables break DOCX export).

```ts
schema.addChildCheck( ( ctx, childDef ) => {
  if ( ctx.endsWith( 'mddmRichBlockContent' ) ) {
    if ( childDef.name === 'mddmSection' ) return false;
    if ( childDef.name === 'mddmRichBlock' ) return false;
  }
  if ( ctx.endsWith( 'tableCell' ) && childDef.name === 'table' ) return false;
} );
```

The same checks apply to the exception-only path — the check's context test
can match on the `htmlDivParagraph` ancestor with class `mddm-rich-block`
(requires reading `htmlDivAttributes` — `⚠ uncertain` whether the standard
schema context exposes those attributes; if not, fall back to the bespoke
`mddmRichBlockContainer` element).

---

## 6. DOCX export

Rich content inside a RichBlock exports as ordinary Word content.

- The `<div class="mddm-rich-block">` wrapper has no Word equivalent and
  can be dropped by the HTML→DOCX converter, or preserved as a named
  paragraph style (e.g. `MddmRichBlock`) that the downstream template
  stylesheet leaves unstyled.
- Headings, lists, tables, images, links flow normally through HTMLtoDOCX.
- Restricted-editing marker boundary attributes (`data-restricted-editing-…`)
  are stripped by the exporter; they have no Word semantic. This mirrors
  the approach used for `mddmSection`. `⚠ uncertain` — confirm the marker
  boundary tag names once empirically captured (see
  [10 — Restricted editing](./10-restricted-editing.md) §recommended
  empirical check).

---

## 7. PDF / print

Same as DOCX — content flows as ordinary block content into whichever
print pipeline we use (Chromium print, or a server-side HTML→PDF).
Page-break hints, widow/orphan rules, etc. apply to the inner paragraphs,
not the wrapper. No special handling.

---

## 8. Accessibility

- Emit `role="region"` on the wrapper in the **editing view only**
  (stripping it from the data view is fine since it is derived from
  `label`).
- Provide `aria-label` from a sibling label element when the template
  author gave the block a human name. In the widget flavour this comes
  from the `label` model attribute; in the exception-only flavour, the
  template author can add `data-aria-label` on the `<div>` and a tiny
  editing-downcast attribute converter copies it to `aria-label`.
- `⚠ uncertain` — whether screen readers announce entering a
  restricted-editing exception. The feature sets `contenteditable="false"`
  on locked content and `contenteditable="true"` on exception ranges; this
  is the standard ARIA contract, but the announcement wording depends on
  the AT. Empirically test with NVDA + Chromium before shipping.

---

## 9. Decision summary

| Concern                        | Recommended (exception-only)           | Alternative (widget)              |
| ------------------------------ | -------------------------------------- | --------------------------------- |
| Schema work                    | None / GHS rule                        | 2 elements + converters           |
| Fill-mode write access         | Block exception over inner blocks      | Block exception over nested editable |
| Toolbar scope                  | All loaded commands                    | All loaded commands               |
| Placeholder                    | CSS `:empty::before` or `Placeholder`  | `Placeholder` per nested editable |
| Delete-resistant               | No (user can `Backspace` the block)    | Yes (`$blockObject`)              |
| "Insert rich block" UX button  | Thin — inserts div + exception         | First-class command               |
| DOCX round-trip                | Wrapper optional, content flows        | Wrapper optional, content flows   |

Ship the exception-only path first. Promote to the widget flavour only
when a concrete product requirement forces the extra machinery.

---

## Open questions

- `⚠ uncertain` — exact GHS model element name + attribute shape for
  `<div class="mddm-rich-block">` on the v48 build (`htmlDivParagraph` vs
  `htmlDiv`). Confirm via Inspector.
- `⚠ uncertain` — whether
  `writer.addMarker( 'restrictedEditingException:…', … )` from an insertion
  command is honoured by `RestrictedEditingModeEditing` on re-init, or
  whether only markers created via the feature's own commands are
  recognised.
- `⚠ uncertain` — how nested-table prevention should surface to the user
  (silent schema rejection vs toolbar disable). Product-level decision.
- `⚠ uncertain` — DOCX exporter handling of the `mddm-rich-block` div:
  drop vs preserve as paragraph style. Coordinate with
  [31 — DOCX export](./31-docx-export.md).

---

## Sources

- [CKEditor 5 — Restricted editing](https://ckeditor.com/docs/ckeditor5/latest/features/restricted-editing.html)
- [`RestrictedEditingExceptionBlockCommand` API](https://ckeditor.com/docs/ckeditor5/latest/api/module_restricted-editing_restrictededitingexceptionblockcommand-RestrictedEditingExceptionBlockCommand.html)
- [Implementing a block widget](https://ckeditor.com/docs/ckeditor5/latest/framework/tutorials/widgets/implementing-a-block-widget.html)
- [General HTML Support](https://ckeditor.com/docs/ckeditor5/latest/features/html/general-html-support.html)

## Cross-refs

- [04 — Schema](./04-schema.md) — element registration, `$container`, checks.
- [05 — Conversion](./05-conversion.md) — data vs editing downcast split.
- [07 — Markers](./07-markers.md) — marker persistence flags.
- [10 — Restricted editing](./10-restricted-editing.md) — block vs inline
  exception, allowedCommands semantics.
- [11 — Widgets](./11-widgets.md) — when chrome is worth the cost.
- [24 — Data format](./24-data-format.md) — round-trip contract.
