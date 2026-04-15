---
title: Repeatables mapping
status: draft
area: mapping
---

# 20 — Repeatables mapping

`Repeatable` + `RepeatableItem` (the old BlockNote blocks) → a CK5-native list
container with N structured items, add/remove commands, min/max enforcement,
auto-numbering, and Author/Fill-mode awareness.

> Legacy BlockNote blocks are **reference only, not binding**. The migration
> target is idiomatic CK5 v48. See also `03-engine-model.md`, `04-schema.md`,
> `05-conversion.md`, `07-markers.md`, `10-restricted-editing.md`,
> `11-widgets.md`, `24-data-format.md`.

---

## Recommended design

**Hybrid, leaning B (bespoke widget) for the container, with a deliberately
thin item structure that reads like a list.**

Register a `mddmRepeatable` block widget that owns an ordered sequence of
`mddmRepeatableItem` nested editables. Each item body is wrapped in a
`restrictedEditingExceptionBlock` so Fill-mode users can edit it freely while
the widget chrome (the "N items" boundary) stays Author-only. Numbering is
**purely presentational** — CSS counters, no model attributes. Add/remove are
custom commands that respect configured `min`/`max`.

### Why not pure native lists (Option A)

Option A — reuse `bulletedList` / `numberedList` + `schema.addChildCheck` to
force a fixed per-item structure — is tempting (free numbering, free list UX,
free keyboard model) but loses on four fronts MetalDocs cares about:

1. **Structural guarantees.** A template defines "each item has exactly
   *label* + *paragraph* + optional *amount*." Native list items are
   `$container`-like; constraining them via `addChildCheck` fights the built-in
   list commands (`Enter`, `indent`, `outdent`) which assume free-form content.
2. **Template authoring affordance.** Authors need to pin a schema ("this is
   the bullets section of the PO"), not a generic `<ul>`. A bespoke widget
   gives us a selectable object with its own toolbar (add item, set min/max,
   label template).
3. **Min/max enforcement.** Command `refresh()` over a typed widget is clean;
   the same logic bolted onto the shared list feature would fight every list
   command in the editor.
4. **DOCX round-trip.** A bespoke widget owns its downcast; we can emit an
   ordered numbered paragraph style or a native `<w:numPr>` without colliding
   with the List feature's exporter.

### Why not pure bespoke (heavier B)

We do **not** reinvent list numbering, keyboard navigation, or bullet glyphs
in the model. Items are sequential children; numbering is CSS. The only
bespoke mechanics are: the parent widget, the item widget, min/max refresh,
and the add/remove commands.

### Shape at a glance

```
mddmRepeatable  (block widget, $blockObject)
├── attributes: repeatableId, label, min, max, numberingStyle
└── mddmRepeatableItem  ($container, isLimit, ×N)
    └── restrictedEditingExceptionBlock wrapper (Fill-mode editable)
        └── rich body (paragraph, inline formatting, inline fields…)
```

Data-view HTML (what `getData()` emits):

```html
<ol class="mddm-repeatable" data-id="bullets" data-min="1" data-max="10">
  <li class="mddm-repeatable__item">
    <div class="restricted-editing-exception">
      <p>First bullet.</p>
    </div>
  </li>
  <li class="mddm-repeatable__item">
    <div class="restricted-editing-exception">
      <p>Second bullet.</p>
    </div>
  </li>
</ol>
```

Using `<ol>`/`<li>` at the wire is deliberate: it is the semantically correct
hook for print CSS counters, ARIA list semantics, and DOCX numbering mapping
(see §10–§12). The *model* never uses `listItem` — those tags are only the
data-downcast shape.

---

## 1. Schema

```ts
// plugin: MddmRepeatableEditing
import { Plugin } from 'ckeditor5';

editor.model.schema.register( 'mddmRepeatable', {
    inheritAllFrom: '$blockObject',
    allowChildren: [ 'mddmRepeatableItem' ],
    allowAttributes: [
        'repeatableId',     // string, stable across renders
        'label',             // author-facing name
        'min',               // number, default 0
        'max',               // number, default Infinity
        'numberingStyle',    // 'decimal' | 'lower-alpha' | 'bullet' | 'none'
    ],
} );

editor.model.schema.register( 'mddmRepeatableItem', {
    inheritAllFrom: '$container',
    allowIn: 'mddmRepeatable',
    isLimit: true,           // caret cannot leak out of the item body
} );

// Forbid nested repeatables (they would break numbering + DOCX mapping).
editor.model.schema.addChildCheck( ( context, childDef ) => {
    if ( context.endsWith( 'mddmRepeatableItem' )
         && childDef.name === 'mddmRepeatable' ) {
        return false;
    }
} );
```

Cross-links: `$blockObject`, `$container`, `isLimit` → `04-schema.md` §2–§4.
`addChildCheck` semantics → `04-schema.md` §7.

⚠ uncertain: whether `isLimit` on the item is sufficient to keep the default
`Enter` behaviour (splitting a paragraph) from escaping when the item body
ends on an empty paragraph. If not, add a keystroke handler bound to the
widget that intercepts `Enter` at the last position and calls `addItem`
instead. Verify empirically.

---

## 2. Converters (upcast + data downcast + editing downcast)

Use `elementToStructure` for downcast so we can nest `<li>` inside `<ol>` in a
single declarative mapping. `toWidget` + `toWidgetEditable` apply **only** to
editing downcast (`11-widgets.md` §1).

```ts
import { toWidget, toWidgetEditable } from 'ckeditor5/src/widget';

// Upcast: <ol class="mddm-repeatable"> → model parent;
//         <li class="mddm-repeatable__item"> → model item.
editor.conversion.for( 'upcast' ).elementToElement( {
    view: { name: 'ol', classes: 'mddm-repeatable' },
    model: ( viewElement, { writer } ) => writer.createElement( 'mddmRepeatable', {
        repeatableId: viewElement.getAttribute( 'data-id' ),
        label: viewElement.getAttribute( 'data-label' ) ?? '',
        min: Number( viewElement.getAttribute( 'data-min' ) ) || 0,
        max: Number( viewElement.getAttribute( 'data-max' ) ) || Infinity,
        numberingStyle:
            viewElement.getAttribute( 'data-numbering' ) ?? 'decimal',
    } ),
} );

editor.conversion.for( 'upcast' ).elementToElement( {
    view: { name: 'li', classes: 'mddm-repeatable__item' },
    model: 'mddmRepeatableItem',
} );

// Data downcast: clean HTML for persistence / DOCX / PDF.
editor.conversion.for( 'dataDowncast' ).elementToStructure( {
    model: {
        name: 'mddmRepeatable',
        attributes: [ 'repeatableId', 'label', 'min', 'max', 'numberingStyle' ],
    },
    view: ( modelElement, { writer } ) => writer.createContainerElement( 'ol', {
        class: 'mddm-repeatable',
        'data-id': modelElement.getAttribute( 'repeatableId' ),
        'data-label': modelElement.getAttribute( 'label' ) ?? '',
        'data-min': String( modelElement.getAttribute( 'min' ) ?? 0 ),
        'data-max': String( modelElement.getAttribute( 'max' ) ?? '' ),
        'data-numbering': modelElement.getAttribute( 'numberingStyle' ) ?? 'decimal',
    } ),
} );

editor.conversion.for( 'dataDowncast' ).elementToElement( {
    model: 'mddmRepeatableItem',
    view: ( _m, { writer } ) =>
        writer.createContainerElement( 'li', { class: 'mddm-repeatable__item' } ),
} );

// Editing downcast: same shape wrapped as a widget with editable items.
editor.conversion.for( 'editingDowncast' ).elementToElement( {
    model: 'mddmRepeatable',
    view: ( modelElement, { writer } ) => {
        const ol = writer.createContainerElement( 'ol', {
            class: 'mddm-repeatable',
            'data-numbering': modelElement.getAttribute( 'numberingStyle' ) ?? 'decimal',
        } );
        return toWidget( ol, writer, { label: 'repeatable list' } );
    },
} );

editor.conversion.for( 'editingDowncast' ).elementToElement( {
    model: 'mddmRepeatableItem',
    view: ( _m, { writer } ) => {
        const li = writer.createEditableElement( 'li', {
            class: 'mddm-repeatable__item',
        } );
        return toWidgetEditable( li, writer );
    },
} );
```

The `restrictedEditingExceptionBlock` marker is **added on top** of the item
body by the insertion command (§3), not by the converter — markers are not
downcast through structural converters (`07-markers.md` §1, `10-restricted-editing.md`).

⚠ uncertain: whether `elementToStructure` is strictly required here or whether
two separate `elementToElement` calls suffice. Two calls are simpler and let
us keep item-level reconversion cheap. Recommendation: two calls (as above).

---

## 3. Commands

### 3.1 `insertMddmRepeatable`

```ts
import { Command } from 'ckeditor5';

export class InsertMddmRepeatableCommand extends Command {
    execute( { id, label = '', min = 1, max = Infinity, initialCount = min } = {} ) {
        const model = this.editor.model;
        model.change( writer => {
            const parent = writer.createElement( 'mddmRepeatable', {
                repeatableId: id,
                label,
                min,
                max,
                numberingStyle: 'decimal',
            } );
            const count = Math.max( min, initialCount );
            for ( let i = 0; i < count; i++ ) {
                const item = writer.createElement( 'mddmRepeatableItem' );
                const p = writer.createElement( 'paragraph' );
                writer.append( p, item );
                writer.append( item, parent );
                // Mark the item body as a block exception for Fill-mode edits.
                writer.addMarker( `restrictedEditingException:${ id }:${ i }`, {
                    usingOperation: true,
                    affectsData: true,
                    range: writer.createRangeIn( item ),
                } );
            }
            model.insertContent( parent );
        } );
    }

    refresh() {
        const schema = this.editor.model.schema;
        const selection = this.editor.model.document.selection;
        this.isEnabled = schema.checkChild(
            selection.getFirstPosition()!, 'mddmRepeatable',
        );
    }
}
```

### 3.2 `addMddmRepeatableItem` / `removeMddmRepeatableItem`

```ts
export class AddMddmRepeatableItemCommand extends Command {
    execute() {
        const model = this.editor.model;
        const parent = this._getParent();
        if ( !parent ) return;
        model.change( writer => {
            const item = writer.createElement( 'mddmRepeatableItem' );
            writer.append( writer.createElement( 'paragraph' ), item );
            writer.append( item, parent );
            writer.addMarker(
                `restrictedEditingException:${ parent.getAttribute( 'repeatableId' ) }:${ parent.childCount - 1 }`,
                { usingOperation: true, affectsData: true, range: writer.createRangeIn( item ) },
            );
        } );
    }

    refresh() {
        const parent = this._getParent();
        const max = Number( parent?.getAttribute( 'max' ) ?? Infinity );
        this.isEnabled = !!parent && parent.childCount < max;
    }

    private _getParent() {
        // Nearest ancestor mddmRepeatable of the selection.
        const sel = this.editor.model.document.selection;
        return sel.getFirstPosition()?.findAncestor( 'mddmRepeatable' ) ?? null;
    }
}

export class RemoveMddmRepeatableItemCommand extends Command {
    execute() {
        const model = this.editor.model;
        const sel = model.document.selection;
        const item = sel.getFirstPosition()?.findAncestor( 'mddmRepeatableItem' );
        if ( !item ) return;
        const parent = item.parent;
        const min = Number( parent?.getAttribute( 'min' ) ?? 0 );
        if ( !parent || parent.childCount <= min ) return;
        model.change( writer => writer.remove( item ) );
    }

    refresh() {
        const sel = this.editor.model.document.selection;
        const item = sel.getFirstPosition()?.findAncestor( 'mddmRepeatableItem' );
        const parent = item?.parent;
        const min = Number( parent?.getAttribute( 'min' ) ?? 0 );
        this.isEnabled =
            !!item && !!parent && parent.childCount > min;
    }
}
```

Both commands must be **Author-mode only** — they mutate structure, which is
out of scope for Fill mode (`10-restricted-editing.md`). Gate the toolbar
registration accordingly.

Cross-link: command `refresh()` semantics → `06-commands.md`.

---

## 4. Min / max enforcement (post-fixer)

`refresh()` handles the "may I click this button?" side. A **post-fixer**
handles the "paste dropped me below min" side.

```ts
editor.model.document.registerPostFixer( writer => {
    let changed = false;
    const root = editor.model.document.getRoot( 'main' )!;
    for ( const node of root.getChildren() ) {
        if ( !node.is( 'element', 'mddmRepeatable' ) ) continue;
        const min = Number( node.getAttribute( 'min' ) ?? 0 );
        while ( node.childCount < min ) {
            const item = writer.createElement( 'mddmRepeatableItem' );
            writer.append( writer.createElement( 'paragraph' ), item );
            writer.append( item, node );
            changed = true;
        }
    }
    return changed;
} );
```

We deliberately do **not** auto-trim when above max on paste — we show a
warning instead, because silently dropping pasted content is user-hostile.

⚠ uncertain: interaction with undo. Post-fixer insertions land in the same
batch as the triggering operation, which is correct for undo. Verify with a
paste-then-undo golden test (`34-golden-tests.md`).

---

## 5. Numbering — CSS counters, not model attributes

Prefer CSS counters over a `data-index` post-fixer. Reasons:

- Numbering is purely presentational; it should not leak into the model,
  should not produce undo steps, and should not appear in stored HTML.
- Adding/removing items does not invalidate any attributes — renumbering is
  free.
- Native `<ol>` behaviour already gives us correct numbering in most printing
  contexts; CSS counters let us override with lower-alpha / roman / bullet.

```css
.mddm-repeatable {
  counter-reset: mddm-item;
  list-style: none;
  padding-left: 0;
}
.mddm-repeatable__item {
  counter-increment: mddm-item;
  display: grid;
  grid-template-columns: 2.25em 1fr;
}
.mddm-repeatable__item::before {
  content: counter(mddm-item, decimal) ".";
  /* decimal | lower-alpha | lower-roman | disc | none — driven by data-numbering */
}
.mddm-repeatable[data-numbering="lower-alpha"] .mddm-repeatable__item::before {
  content: counter(mddm-item, lower-alpha) ")";
}
.mddm-repeatable[data-numbering="bullet"] .mddm-repeatable__item::before {
  content: "•";
}
.mddm-repeatable[data-numbering="none"] .mddm-repeatable__item::before {
  content: "";
}
```

Rejected alternative: a post-fixer writing `indexOneBased` onto every item.
It would work but bloats the model, noisily mutates on every add/remove, and
provides no value DOCX/PDF cannot already derive from position.

---

## 6. Restricted-editing integration

Each `mddmRepeatableItem` body is wrapped in a
`restrictedEditingException:<id>:<n>` marker — the block variant (see
`10-restricted-editing.md` "Critical finding — exception element type", which
supersedes the legacy memory that claimed only inline `<span>` works).

- **Author mode** (`StandardEditingMode`): widget chrome visible, `addItem` /
  `removeItem` commands available in the widget toolbar.
- **Fill mode** (`RestrictedEditingMode`): chrome is read-only, but each
  item's inner body is a block exception, so typing, formatting, and inline
  widgets (inline fields) work normally. `addItem` / `removeItem` are not
  registered as toolbar items in Fill mode. ⚠ uncertain whether to expose a
  *per-widget* "Add item" button in Fill mode for variable-length lists; this
  is a product decision — default **off**.

Marker naming convention: `restrictedEditingException:<repeatableId>:<itemIndex>`.
Item indexes in the marker name are cosmetic (for debugging); the marker
itself is positioned by live range, so reordering/remove operations keep it
anchored correctly (`07-markers.md`).

---

## 7. Widget toolbar (Author mode)

```ts
editor.plugins.get( 'WidgetToolbarRepository' ).register( 'mddmRepeatable', {
    items: [ 'addMddmRepeatableItem', 'removeMddmRepeatableItem', '|', 'mddmRepeatableSettings' ],
    getRelatedElement: ( selection ) =>
        selection.getFirstPosition()?.findAncestor( 'mddmRepeatable' ) ?? null,
} );
```

`mddmRepeatableSettings` is a dialog for label, min, max, and numbering style.
⚠ uncertain: whether changing `min` after the fact should auto-extend via the
post-fixer or require explicit user confirmation. Recommend *explicit* — cheap
to implement, avoids surprise insertions.

---

## 8. DOCX export mapping (see `31-docx-export.md`)

Because we emit `<ol class="mddm-repeatable"><li>...` in the data view, the
existing HTMLtoDOCX numbered-list mapping handles the common case for free.
Two adjustments:

- `data-numbering` on the `<ol>` must drive the chosen `<w:numFmt>` (decimal,
  lowerLetter, bullet, none).
- The inner `<div class="restricted-editing-exception">` wrapper should be
  **unwrapped** in the DOCX exporter pre-pass — it has no DOCX analogue and
  nesting a `<div>` inside `<li>` breaks some DOCX viewers. Keep it on the web
  viewer though (PDF and HTML).

Remember the known constraint (MetalDocs memory): CKEditor wraps tables in
`<figure>`, which HTMLtoDOCX mis-nests — so **forbid tables inside a
repeatable item** in the schema until the DOCX pipeline is upgraded:

```ts
editor.model.schema.addChildCheck( ( context, childDef ) => {
    if ( context.endsWith( 'mddmRepeatableItem' ) && childDef.name === 'table' ) {
        return false;
    }
} );
```

---

## 9. PDF / print mapping (see `32-pdf-export.md`)

The print stylesheet reuses §5's CSS counters verbatim. No additional work
unless the product decides to show the label (`mddmRepeatable[label]`) as a
print-only heading:

```css
@media print {
  .mddm-repeatable::before {
    content: attr(data-label);
    display: block;
    font-weight: 600;
  }
}
```

---

## 10. Accessibility

- Data view is a real `<ol>` / `<li>` → screen readers announce "list, N
  items" natively.
- Editing view adds `role="group"` on the widget and `aria-label` from
  `toWidget( …, { label } )`. Items keep their native `<li>` semantics because
  `toWidgetEditable` does not override `role`.
- Widget toolbar buttons need `aria-label`s ("Add item", "Remove item") and
  must be reachable via `Alt+F10` (CK5's widget-toolbar shortcut).
- Numbering is presentational (CSS `::before`) — assistive tech already reads
  `<ol>` enumeration, so the generated label is skipped, which is the
  desired behaviour.

⚠ uncertain: whether CK5's widget selection handle announces itself
correctly with NVDA on the `<ol>` wrapper (non-standard pairing of list +
widget). Needs an empirical a11y pass.

---

## 11. Legacy deviations (BlockNote → CK5)

| BlockNote                                         | CK5 equivalent                                                    |
| ------------------------------------------------- | ----------------------------------------------------------------- |
| `Repeatable` + `RepeatableItem` custom blocks     | `mddmRepeatable` + `mddmRepeatableItem`                           |
| `props.min` / `props.max` on the block            | Model attributes `min` / `max`, enforced by command + post-fixer  |
| Auto-numbering as a model prop `indexOneBased`    | CSS counters (model-free)                                         |
| Custom add/remove UI inside the block body        | Widget toolbar + keyboard (⚠ `Enter`-at-end handler, §1)          |
| Fill-mode permission via BlockNote `editable` prop| `restrictedEditingExceptionBlock` markers per item                |

Explicitly not carried over: BlockNote's per-item "subfield schema". For the
MDDM v1, items are rich-text with inline fields (`22-fieldgroup-mapping.md`).
A stricter per-item template schema can be layered later via `addChildCheck`
once we have a concrete template that needs it.

---

## 12. Open questions

- ⚠ uncertain: `Enter` at end-of-last-item → should it create a new item or
  exit the widget? Default list behaviour exits; MetalDocs UX probably wants
  "create item until max, then exit". Decide before Author-mode ships.
- ⚠ uncertain: whether to reuse the built-in `List` feature's keyboard model
  via programmatic command delegation, or wire our own minimal keystroke
  handler. Leaning *our own*, narrowly scoped to the widget.
- ⚠ uncertain: DOCX numbering style mapping for `numberingStyle: 'none'` —
  our exporter currently has no "unnumbered ordered list" affordance. May
  need a `<w:numPr>` with a blank format definition. Verify against
  `31-docx-export.md`.
- ⚠ uncertain: whether post-fixer-inserted items during paste should carry a
  fresh `restrictedEditingException` marker automatically. Current plan: yes,
  scoped by repeatableId + index; validate via golden test.

---

## Sources

- `03-engine-model.md` — batches, differ, `change()` block semantics
- `04-schema.md` — `$blockObject`, `$container`, `addChildCheck`, `isLimit`
- `05-conversion.md` — `elementToElement`, `elementToStructure`, data vs
  editing downcast
- `07-markers.md` — marker naming, live-range behaviour
- `10-restricted-editing.md` — `restrictedEditingExceptionBlock` is supported
  (supersedes legacy "inline-span-only" note)
- `11-widgets.md` — `toWidget`, `toWidgetEditable`, nested editables
- `24-data-format.md` — HTML as the wire format; getData/setData contract
- CK5 v48 API: `WidgetToolbarRepository`, `Command#refresh`
