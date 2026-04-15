---
title: Sections mapping
status: draft
area: mapping
---

# 19 — Sections mapping

How MetalDocs maps the concept of a **Section** (a Word-style titled block
of content that can be fully locked, fully editable, or mixed) onto CKEditor
5 v48 primitives.

> **Legacy deviation.** The old BlockNote `Section` block shipped as a
> custom React component with its own JSON schema, variant enum
> (`locked | editable | mixed`), and bespoke serialisation. That model is
> **reference only** for the MetalDocs CK5 port. We are not porting the
> BlockNote block shape, React renderer, or toolbar affordances; we keep
> only the authoring/UX intent.

---

## Recommended design

**A bespoke block widget `mddmSection` with two nested editables** — a
limit-locked header and a container body — **plus automatic restricted-
editing exception markers** planted at insertion time. A single `variant`
attribute (`locked | editable | mixed`) drives reconversion via
`triggerBy`.

Rationale:

1. A section **is** a structural primitive (header + body, appears in
   the outline, exports as a heading + paragraphs in DOCX). That is what
   an `$blockObject` widget is for. Using GHS + bare `<section>` would
   leave us without selection semantics, drag-and-drop, widget toolbar,
   or a stable upcast target — every round-trip becomes fragile (see
   page 24 on schema filtering unknown content).
2. Restricted-editing markers are **orthogonal to structure** (page 04
   §9 and page 10). Building the lock state as a separate widget type
   or as schema `disallowChildren` breaks programmatic insertion and
   fights the engine. A single `variant` attribute reconverts the
   widget chrome without touching the model shape.
3. The v48 restricted-editing feature now exposes both
   `restrictedEditingException` (inline) and
   `restrictedEditingExceptionBlock` (block). Our insertion command
   plants the correct flavour per variant so Fill-mode users land on a
   usable caret with zero extra clicks.

Deviation from legacy: the BlockNote `mixed` variant allowed arbitrary
inline editable runs inside a locked header **and** a locked prefix
inside an editable body. In CK5 we narrow `mixed` to: **locked header
with inline exception spans + fully-editable body**. Purely custom
locked/editable interleaving inside the body is not supported in the
first cut. ⚠ uncertain whether we will ever need the general case.

---

## 1. What is a "Section" for us?

| Field       | Type                                                    |
|-------------|---------------------------------------------------------|
| Header      | One block of inline-only content (title, headings).     |
| Body        | Zero or more block children (paragraphs, lists, tables).|
| `variant`   | `locked` \| `editable` \| `mixed`                       |
| `sectionId` | Stable string ID for template binding / IR round-trip.  |

Semantics by variant (Fill / Restricted mode):

- `locked` — header and body are both read-only; no caret lands inside.
- `editable` — header is locked (template-controlled title); body is a
  block restricted-editing exception (full editor commands).
- `mixed` — header carries **inline** restricted-editing exceptions
  (`<span class="restricted-editing-exception">`) around fill-in-blank
  placeholders; body is a block exception as in `editable`.

In Author / Standard mode all three variants are fully editable — the
exception markers are authoring metadata, not a runtime gate.

## 2. Widget or GHS wrapper?

**Widget. Rejected: GHS + bare `<section>`.**

| Concern                           | Widget (chosen)             | GHS wrapper                        |
|-----------------------------------|-----------------------------|------------------------------------|
| Atomic selection / drag           | Free (`toWidget`)           | Manual; fights engine selection    |
| Round-trip stability              | Schema-owned; deterministic | Depends on GHS config per tag      |
| Header as limit editable          | `toWidgetEditable` + limit  | No nested-editable contract        |
| Reconversion on variant change    | `triggerBy: [ 'variant' ]`  | N/A — re-style via CSS only        |
| DOCX mapping (page 31)            | One branch per model elt    | Must pattern-match HTML classes    |
| Accessibility (role + label)      | `toWidget` `label:` option  | Author-time ARIA only              |

The only argument for GHS would be "let authors drop in arbitrary
`<section>` markup." We are explicitly not that product — MetalDocs
emits templates from a controlled primitive set.

## 3. Schema

```js
// Inside the plugin's init() — once, after StandardEditingMode is loaded
// so the shared marker group is registered first. (Page 10 §1.)

const schema = editor.model.schema;

schema.register( 'mddmSection', {
    inheritAllFrom: '$blockObject',
    allowChildren: [ 'mddmSectionHeader', 'mddmSectionBody' ],
    allowAttributes: [ 'sectionId', 'variant' ]
} );

schema.register( 'mddmSectionHeader', {
    isLimit: true,
    allowIn: 'mddmSection',
    allowContentOf: '$block'    // inline text + attributes only
} );

schema.register( 'mddmSectionBody', {
    isLimit: true,
    allowIn: 'mddmSection',
    allowContentOf: '$root'     // paragraphs, lists, tables, images…
} );

// "Exactly one header + one body" — post-fixer rather than addChildCheck
// so paste normalisation is forgiving. (Page 04 §8.1.)
editor.model.document.registerPostFixer( writer => {
    let changed = false;
    for ( const section of iterateSections( editor.model.document.getRoot() ) ) {
        const headers = [ ...section.getChildren() ]
            .filter( c => c.is( 'element', 'mddmSectionHeader' ) );
        const bodies = [ ...section.getChildren() ]
            .filter( c => c.is( 'element', 'mddmSectionBody' ) );
        // drop extras, synthesise missing
        for ( const extra of headers.slice( 1 ) ) { writer.remove( extra ); changed = true; }
        for ( const extra of bodies.slice( 1 ) )  { writer.remove( extra ); changed = true; }
        if ( headers.length === 0 ) {
            writer.insertElement( 'mddmSectionHeader', section, 0 );
            changed = true;
        }
        if ( bodies.length === 0 ) {
            const body = writer.createElement( 'mddmSectionBody' );
            writer.append( body, section );
            writer.appendElement( 'paragraph', body ); // mandatory block
            changed = true;
        }
    }
    return changed;
} );
```

## 4. Converters

Three pipelines (page 05 §1). Data downcast must be widget-chrome-free
so `getData()` emits the HTML the backend stores (page 24 §3). Editing
downcast wraps the same element in `toWidget` / `toWidgetEditable`.

```js
import { toWidget, toWidgetEditable } from 'ckeditor5/src/widget.js';

const conv = editor.conversion;

// ── Upcast ───────────────────────────────────────────────────────────
conv.for( 'upcast' ).elementToElement( {
    view: { name: 'section', classes: 'mddm-section' },
    model: ( viewElement, { writer } ) => writer.createElement( 'mddmSection', {
        sectionId: viewElement.getAttribute( 'data-section-id' ) ?? undefined,
        variant:   viewElement.getAttribute( 'data-variant' )    ?? 'editable'
    } )
} );

conv.for( 'upcast' ).elementToElement( {
    view: { name: 'header', classes: 'mddm-section__header' },
    model: 'mddmSectionHeader'
} );

conv.for( 'upcast' ).elementToElement( {
    view: { name: 'div', classes: 'mddm-section__body' },
    model: 'mddmSectionBody'
} );

// ── Data downcast (clean HTML → backend) ─────────────────────────────
conv.for( 'dataDowncast' ).elementToStructure( {
    model: { name: 'mddmSection', attributes: [ 'sectionId', 'variant' ] },
    view: ( modelElement, { writer } ) => {
        const section = writer.createContainerElement( 'section', {
            class: 'mddm-section',
            'data-section-id': modelElement.getAttribute( 'sectionId' ),
            'data-variant':    modelElement.getAttribute( 'variant' )
        }, [
            writer.createContainerElement( 'header', {
                class: 'mddm-section__header'
            }, [ writer.createSlot( n => n.is( 'element', 'mddmSectionHeader' ) ) ] ),
            writer.createContainerElement( 'div', {
                class: 'mddm-section__body'
            }, [ writer.createSlot( n => n.is( 'element', 'mddmSectionBody' ) ) ] )
        ] );
        return section;
    }
} );

// ── Editing downcast (widgetised) ────────────────────────────────────
conv.for( 'editingDowncast' ).elementToStructure( {
    model: { name: 'mddmSection', attributes: [ 'variant' ] },
    view: ( modelElement, { writer } ) => {
        const variant = modelElement.getAttribute( 'variant' ) ?? 'editable';

        const header = writer.createEditableElement( 'header', {
            class: 'mddm-section__header',
            'data-variant': variant
        } );
        const body = writer.createEditableElement( 'div', {
            class: 'mddm-section__body',
            'data-variant': variant
        } );

        const section = writer.createContainerElement( 'section', {
            class: `mddm-section mddm-section--${ variant }`,
            'data-variant': variant,
            role: 'region'
        }, [
            header,
            body
        ] );

        // Slot routing — route model children into the correct editable.
        writer.insert( writer.createPositionAt( header, 0 ),
            writer.createSlot( n => n.is( 'element', 'mddmSectionHeader' ) ) );
        writer.insert( writer.createPositionAt( body, 0 ),
            writer.createSlot( n => n.is( 'element', 'mddmSectionBody' ) ) );

        toWidgetEditable( header, writer );
        toWidgetEditable( body, writer );

        return toWidget( section, writer, {
            label: () => sectionLabel( modelElement ), // see §10 accessibility
            hasSelectionHandle: true
        } );
    },
    triggerBy: {
        attributes: [ 'variant' ]   // reconvert on variant change (page 11 §7)
    }
} );
```

Note the two downcast pipelines emit **different** header/body tags only
in the editing view (both use `<header>` / `<div>` — same tags, but the
editing variant is widget-ised). We intentionally do **not** emit
`contenteditable` in the data pipeline (page 11 §1).

## 5. Variants — one attribute, not three elements

Considered and rejected: three separate model elements
(`mddmSectionLocked`, `mddmSectionEditable`, `mddmSectionMixed`). That
would triple the converter surface, duplicate schema, and make author
toggling require a full "delete + insert" dance losing child content.

Chosen: single `mddmSection` + `variant` attribute. Toggling variant is
a single `writer.setAttribute()` that triggers `elementToStructure`
reconversion (page 11 §7). Exception markers are **recomputed** by the
`SetSectionVariantCommand` so the lock state stays consistent with the
variant — see §6.

## 6. Insertion command

```js
import { Command } from 'ckeditor5/src/core.js';
import { uid } from 'ckeditor5/src/utils.js';

export default class InsertSectionCommand extends Command {
    /**
     * @param {object} opts
     * @param {'locked'|'editable'|'mixed'} opts.variant
     * @param {string} [opts.sectionId]
     * @param {string} [opts.headerText]
     */
    execute( opts = {} ) {
        const variant = opts.variant ?? 'editable';
        const sectionId = opts.sectionId ?? `section_${ uid() }`;
        const model = this.editor.model;

        model.change( writer => {
            // 1. Build the tree.
            const section = writer.createElement( 'mddmSection', { sectionId, variant } );
            const header  = writer.createElement( 'mddmSectionHeader' );
            const body    = writer.createElement( 'mddmSectionBody' );
            const bodyPara = writer.createElement( 'paragraph' );

            if ( opts.headerText ) {
                writer.insertText( opts.headerText, header );
            }
            writer.append( header, section );
            writer.append( body, section );
            writer.append( bodyPara, body );

            // 2. Insert as an object (splits surrounding block correctly).
            model.insertObject( section, null, null, { setSelection: 'on' } );

            // 3. Plant restricted-editing exception markers so Fill mode
            //    has a usable caret without extra user clicks. The
            //    `restrictedEditingException` marker group is owned by
            //    the restricted-editing feature; we only add ranges.
            //    (Page 07 §8, page 10 "Commands".)
            if ( variant === 'editable' || variant === 'mixed' ) {
                const bodyRange = writer.createRangeIn( body );
                writer.addMarker( `restrictedEditingException:${ sectionId }_body`, {
                    range: bodyRange,
                    usingOperation: true,
                    affectsData: true
                } );
            }
            if ( variant === 'mixed' && opts.headerExceptionRanges ) {
                // Optional: caller may request inline exception spans over
                // specific header ranges (fill-in-blank values).
                for ( const { offset, length, id } of opts.headerExceptionRanges ) {
                    const start = writer.createPositionAt( header, offset );
                    const end   = writer.createPositionAt( header, offset + length );
                    writer.addMarker(
                        `restrictedEditingException:${ sectionId }_h_${ id }`,
                        {
                            range: writer.createRange( start, end ),
                            usingOperation: true,
                            affectsData: true
                        }
                    );
                }
            }

            // 4. Move selection to the body's first paragraph so the user
            //    can type immediately.
            writer.setSelection( bodyPara, 0 );
        } );
    }

    refresh() {
        const selection = this.editor.model.document.selection;
        const allowedParent = this.editor.model.schema.findAllowedParent(
            selection.getFirstPosition(),
            'mddmSection'
        );
        this.isEnabled = allowedParent !== null;
    }
}
```

Why a single `model.change()` batch: the whole insertion — element
tree, object placement, exception markers, selection move — must be
one undo step (page 03 §5). Splitting into multiple batches would
leave Ctrl+Z restoring a half-constructed section.

⚠ uncertain: the docs do not guarantee that `addMarker` with group
`restrictedEditingException` will interop with the feature's own
internal counter. If the feature rejects externally-minted IDs we
will switch to executing
`editor.execute( 'restrictedEditingExceptionBlock' )` over the body
range after selecting it — functionally equivalent, slightly less
declarative. Needs an empirical check on v48.

## 7. Toolbar & UX

- **Block toolbar button** `insertMddmSection` (icon + dropdown for
  variant) wired to `InsertSectionCommand`. Lives next to the
  "insert paragraph / table / image" cluster.
- **Widget toolbar** (v48 `WidgetToolbarRepository`) attached to
  `mddm-section`: variant toggle, delete, duplicate, "edit ID".
- **Balloon toolbar** inside `mddmSectionBody`: the standard rich-text
  toolbar (bold/italic/lists/etc.). Author mode: always on.
  Fill mode: gated by `RestrictedEditingModeEditing` (it only enables
  inside exceptions — page 10).
- **Header toolbar**: deliberately minimal. In Author mode it shows
  "wrap selection as exception" (inline); in Fill mode nothing —
  caret can only land inside inline exception spans.

Keyboard: Tab / Shift+Tab in Fill mode hops between exceptions via
`goToNextRestrictedEditingException` / `goToPreviousRestrictedEditingException`
(page 10 §Commands). This means a `mixed` header's fill-in spans are
reachable purely by keyboard without touching the mouse.

## 8. DOCX export mapping (cross-ref page 31)

| Model                         | DOCX                                                                     |
|-------------------------------|--------------------------------------------------------------------------|
| `mddmSection` (wrapper)       | No node of its own — emits a section break context only if template asks.|
| `mddmSectionHeader`           | `Heading 1` or `Heading 2` paragraph style (template-level setting).     |
| `mddmSectionBody` / paragraph | `Normal` paragraph style.                                                |
| `mddmSectionBody` / list      | `ListParagraph` with numbering/bullets preserved.                        |
| `mddmSectionBody` / table     | Flat table (no nesting — see memory on nested-tables break).             |
| Inline exception span         | Preserved as run; no DOCX-specific marker (fills are resolved pre-export).|

Rule: export walks the data view HTML and maps `<header class="mddm-section__header">`
→ heading style, `<div class="mddm-section__body"> > <p>` → normal
paragraphs. The `mddm-section` wrapper itself does not produce a DOCX
element — it's a logical grouping. **No nested tables under any
circumstance** (hard constraint from prior experience).

## 9. PDF / print

Template-level CSS, applied to the `mddm-section` data-view class:

```css
.mddm-section { page-break-inside: auto; }               /* default */
.mddm-section[data-break-before="page"] { page-break-before: always; }
.mddm-section[data-keep-together="true"] { break-inside: avoid; }
.mddm-section__header { page-break-after: avoid; }        /* widow-avoid */
```

Template author picks `data-break-before` / `data-keep-together` via a
section-level properties dialog (not a per-instance user control). These
attributes round-trip through the data view as `data-*` on the outer
`<section>` tag.

⚠ uncertain: Chrome's Paged Media support for `break-inside: avoid`
inside flex/grid containers is patchy. Our PDF pipeline uses headless
Chrome (page 32); verify against the real stack before locking this in.

## 10. Accessibility

- The outer `<section>` in the editing view gets `role="region"` plus an
  `aria-label` computed from the header's plain text. `toWidget` accepts
  a `label:` option that can be a function, so we recompute on change:

  ```js
  function sectionLabel( sectionElement ) {
      const header = [ ...sectionElement.getChildren() ]
          .find( c => c.is( 'element', 'mddmSectionHeader' ) );
      const text = header ? [ ...header.getChildren() ]
          .filter( n => n.is( '$text' ) )
          .map( n => n.data ).join( '' ).trim() : '';
      return text ? `Section: ${ text }` : 'Section (untitled)';
  }
  ```

- Data view keeps the same `role="region"` + `aria-label` so the
  rendered PDF / HTML viewer retain the label (page 33).
- Variant state is conveyed via `data-variant` + matching CSS; we do
  **not** expose it through ARIA (`aria-readonly` is not appropriate
  for a region where only parts are locked).
- Fill-mode exception regions inherit the feature's own class; we add
  `aria-label="Fillable field"` on inline spans via
  `markerToHighlight` in editing downcast so screen-reader users hear
  what they just entered.

⚠ uncertain: whether CK5's `toWidget` forwards our custom `role` or
overwrites it with its own widget semantics. If overwritten, we'll
apply the role on the `<header>` / `<div>` children instead.

---

## Open questions

- ⚠ Whether externally-minted `restrictedEditingException:<id>` marker
  names are accepted by the feature's internal management. Fallback:
  `editor.execute( 'restrictedEditingExceptionBlock' )` post-insert.
- ⚠ Block-exception wrapper tag in data view (docs don't quote it;
  page 10 flags empirical test needed).
- ⚠ Whether `elementToStructure` `triggerBy.attributes` reconverts
  cleanly when the attribute change is part of the same batch as a
  selection move — might cause selection to be re-positioned to the
  widget boundary. Needs a test.
- ⚠ General `mixed` support (locked islands inside a body) deferred;
  current plan covers only inline exceptions in header + full body
  exception.

## Sources

- [03 — Engine model](./03-engine-model.md)
- [04 — Schema](./04-schema.md) §8.1 — `mddmSection` schema pattern
- [05 — Conversion](./05-conversion.md) §3 `elementToStructure`, §5 data vs editing downcast
- [07 — Markers](./07-markers.md) §4 persistence flags, §8 restricted-editing group
- [10 — Restricted editing](./10-restricted-editing.md) — `restrictedEditingException{,Block}` commands
- [11 — Widgets](./11-widgets.md) §2 block widget, §4 nested editables, §7 reconversion
- [24 — Data format](./24-data-format.md) §5 schema filtering, §8 widget attribute serialisation
- [31 — DOCX export](./31-docx-export.md) (forward ref)
- [32 — PDF export](./32-pdf-export.md) (forward ref)
- [41 — Accessibility](./41-accessibility.md) (forward ref)
