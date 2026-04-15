---
title: DataTable mapping
status: draft
area: mapping
priority: HIGH
---

# 21 — DataTable mapping

How MetalDocs implements the legacy BlockNote `DataTable` primitive on top of CKEditor 5 v48. Covers Author template design, Fill-mode editing (fixed vs dynamic), DOCX/PDF export, and accessibility.

> **Legacy deviation.** The BlockNote `DataTable` custom block with hard-coded `fixed` and `dynamic` variants is **reference only, not binding**. This design intentionally abandons that custom-widget shape in favour of the native CK5 `table` plugin plus a thin command-gating layer. Re-implementing BlockNote's widget on CK5 would duplicate mature, battle-tested functionality (cell/row/column commands, headers, merge/split, captions, column resize, cell properties) for no gain.

---

## Recommended design

1. **Use the native CK5 `Table` plugin as the primitive.** Do **not** introduce a custom `mddmDataTable` widget. Every table in a MetalDocs document — author templates and filled documents alike — is a native CK5 table (`<figure class="table"><table>…</table></figure>`). Table schema §8.3 in page 04 is superseded for this primitive.
2. **Encode the variant as an attribute on the outer `table` element**, not as two distinct widget types. Model attribute `mddmTableVariant ∈ { 'fixed', 'dynamic' }`, downcast to `data-mddm-variant="fixed|dynamic"` on `<table>`. Default is `dynamic` (matches CK5's native affordances).
3. **Enforce the variant in Fill mode by gating commands**, not schema. A small `MddmTableLockPlugin` watches the selection; when inside a `data-mddm-variant="fixed"` table and the editor is in Restricted (Fill) mode, it disables the structural commands (`insertTableRowAbove/Below`, `insertTableColumnLeft/Right`, `removeTableRow`, `removeTableColumn`, `mergeTableCells`, `splitTableCell*`, `setTableColumnHeader`, `setTableRowHeader`). Cell content editing is still allowed. In Author mode the plugin is inert.
4. **Make cells fillable in Restricted mode via per-cell block exceptions.** At template save time (Author mode), the Author editor walks every `tableCell` inside a fillable table and wraps its content in a `restrictedEditingExceptionBlock`. This is the one template-authoring side effect unique to DataTables — see §5 and the snippet under "Per-cell exception insertion".
5. **Forbid nested tables via `schema.addChildCheck`.** Hard rule — HTMLtoDOCX cannot render CK5's `<figure>`-wrapped tables when nested inside a cell (recorded blocker). Reject `table` inside `tableCell` regardless of mode.
6. **Typed cells are a separate concern.** If a cell must hold a date/number/enum, the author drops an inline field widget (page 22) into the cell body — the cell itself stays a normal rich-content cell. Do not attempt typed-cell schemas.

This keeps the primitive "native first, variant is presentation, locking is runtime," which is the cheapest design that satisfies the four goals (author, fill-fixed, fill-dynamic, export).

---

## 1. Why native table over a custom widget

The native `@ckeditor/ckeditor5-table` plugin is GPL-tier, maintained, and ships the full structural surface we need:

- Core: `Table`, `TableToolbar`, `TableSelection`, `TableCaption`, `TableColumnResize`
- Presentation: `TableProperties` (border, background, alignment), `TableCellProperties` (padding, vertical align, cell border/background)
- Commands: `insertTable`, `insertTableRowAbove/Below`, `insertTableColumnLeft/Right`, `removeTableRow`, `removeTableColumn`, `mergeTableCells`, `splitTableCellVertically/Horizontally`, `mergeTableCell{Right,Down,Left,Up}`, `selectTableRow`, `selectTableColumn`, `setTableColumnHeader`, `setTableRowHeader`.
- Data shape is stable and docx-friendly: `<figure class="table"><table><tbody><tr><td|th>…</td></tr></tbody></table></figure>` with `<caption>` and `<thead>` when used.

A custom widget would have to re-expose all of this. The only reason to build one is if we needed non-HTML cell semantics (e.g. computed cells), which MDDM does not.

---

## 2. Fixed vs dynamic — why option A (one table + flag)

Two designs were considered:

| | A. One native table + `variant` flag | B. Two widgets (`mddmFixedTable`, `mddmDynamicTable`) |
|---|---|---|
| Schema surface | Native `table` + one attribute | Two custom `$blockObject` widgets + nested editables |
| Code | ~60 LOC plugin (command gating) | ~400 LOC (two widgets, converters, commands, toolbars) |
| Export | Native `<table>`, no mapping work | Custom element → `<table>` downcast for both variants |
| Toolbar differences | Filter built-in toolbar per mode | Separate toolbar configs per widget |
| Author UX | Toggle an attribute on an existing table | Must pick the right widget up-front |
| Variant change | Flip `data-mddm-variant` | Replace widget — loses selection/history |

**Chosen: A.** The variant is presentational (it describes who is allowed to change shape, not what the thing is). Modelling it as two widgets double-books every converter, every toolbar, and every export path for no semantic gain.

### Author UX

- Insert table via native `insertTable`.
- Dropdown / button on the table balloon toolbar sets `mddmTableVariant` on the selected table.
- Switching between `fixed` and `dynamic` is a pure attribute change — no structural edit.

### Attribute wiring (schema + converters)

```js
// MddmTableVariantPlugin (loaded in both Author and Fill)
editor.model.schema.extend( 'table', {
    allowAttributes: [ 'mddmTableVariant' ]
} );

editor.conversion.attributeToAttribute( {
    model: {
        name: 'table',
        key: 'mddmTableVariant'
    },
    view: 'data-mddm-variant'
} );
```

Default value is `'dynamic'`; absent attribute in upcast is read as `'dynamic'` by the lock plugin (§3).

---

## 3. Fill-mode lock — command gating plugin

The lock plugin owns the fixed-variant contract. It lives only in the Fill editor's plugin list.

```js
// mddm-table-lock.ts  — loaded ONLY alongside RestrictedEditingMode
import { Plugin } from 'ckeditor5';

const STRUCTURAL_COMMANDS = [
    'insertTableRowAbove',
    'insertTableRowBelow',
    'insertTableColumnLeft',
    'insertTableColumnRight',
    'removeTableRow',
    'removeTableColumn',
    'mergeTableCells',
    'splitTableCellVertically',
    'splitTableCellHorizontally',
    'mergeTableCellRight',
    'mergeTableCellDown',
    'mergeTableCellLeft',
    'mergeTableCellUp',
    'setTableColumnHeader',
    'setTableRowHeader'
];

export class MddmTableLockPlugin extends Plugin {
    static get pluginName() { return 'MddmTableLockPlugin'; }

    init() {
        const editor = this.editor;
        const model = editor.model;

        // Re-evaluate on every selection change.
        model.document.selection.on( 'change:range', () => this._sync() );
        model.document.on( 'change:data', () => this._sync() );
        this._sync();
    }

    _sync() {
        const selection = this.editor.model.document.selection;
        const table = findAncestor( selection.getFirstPosition(), 'table' );
        const locked = !!table && table.getAttribute( 'mddmTableVariant' ) === 'fixed';

        for ( const name of STRUCTURAL_COMMANDS ) {
            const cmd = this.editor.commands.get( name );
            if ( !cmd ) continue;
            if ( locked ) {
                // forceDisabled takes a unique key; the key unlocks in the matching off() call.
                cmd.forceDisabled( 'mddmTableLock' );
            } else {
                cmd.clearForceDisabled( 'mddmTableLock' );
            }
        }
    }
}

function findAncestor( position, name ) {
    let node = position?.parent;
    while ( node ) {
        if ( node.is && node.is( 'element', name ) ) return node;
        node = node.parent;
    }
    return null;
}
```

Why command gating and not schema:

- Schema is mode-agnostic (page 04 §9). Expressing "cannot add a row" in schema would also break programmatic template hydration and `setData()` round-trips.
- `forceDisabled` is the idiomatic CK5 hook (keyed, reversible, stacks with other disablers — e.g. RestrictedEditingMode's own gating).

⚠ uncertain — interaction with `RestrictedEditingMode`: that plugin also force-disables commands when the caret is outside an exception. In practice the two disablers compose (both must clear before a command enables), so a fixed cell exception will still leave structural commands off. Verify empirically that the keys do not collide.

---

## 4. Schema constraints

Only **one** MDDM-specific schema rule is needed — ban nested tables:

```js
editor.model.schema.addChildCheck( ( context, childDef ) => {
    if ( context.endsWith( 'tableCell' ) && childDef.name === 'table' ) {
        return false;
    }
} );
```

This closes the HTMLtoDOCX blocker for good (CK5 wraps every table in a `<figure>`; a nested figure breaks the DOCX table renderer).

Everything else (row/column count limits, header config, cell content) is handled by the native plugin's schema and by command gating. Do **not** set `isLimit` on `table` — it already behaves correctly and flipping limits can break caret navigation between cells.

---

## 5. Restricted-editing integration — per-cell exceptions

Each cell's **content** (`tableCell` → its child `paragraph`s) must sit inside a `restrictedEditingException` block marker so that the Fill-mode user can type into it. This is done **once**, at template save time in the Author editor — not at Fill load time.

### Per-cell exception insertion (Author, template save)

```js
// Run in Author editor, once per fillable table.
editor.model.change( writer => {
    const root = editor.model.document.getRoot();

    for ( const value of editor.model.createRangeIn( root ).getWalker() ) {
        if ( value.type !== 'elementStart' ) continue;
        if ( !value.item.is( 'element', 'tableCell' ) ) continue;

        const cell = value.item;

        // Skip cells that already contain an exception marker.
        if ( hasExceptionMarker( editor.model, cell ) ) continue;

        // Select the full cell body and toggle a block exception.
        writer.setSelection( writer.createRangeIn( cell ) );
        editor.execute( 'restrictedEditingExceptionBlock' );
    }
} );

function hasExceptionMarker( model, element ) {
    const range = model.createRangeIn( element );
    for ( const marker of model.markers ) {
        if ( !marker.name.startsWith( 'restrictedEditingException:' ) ) continue;
        if ( marker.getRange().containsRange( range, true ) ) return true;
    }
    return false;
}
```

Why block and not inline: block exceptions enable the full command set inside (lists, hard-return, font — see page 10 §"allowedCommands"), which matches what users expect in a cell. Inline exceptions would lock cells to a single-paragraph plain-text shape.

### Alternative considered: cell as nested editable

Register `tableCell` itself as a restricted-editing-aware element via `allowedContentIn` — **not supported**. The restricted-editing plugin only knows about the marker. There is no "this element is implicitly an exception" API. `⚠ uncertain` — the docs are silent on registering custom editables with the plugin; if a future CK5 release exposes this, revisit.

### What gets saved to HTML

The data view emits, per cell:

```html
<td>
  <div class="restricted-editing-exception">
    <p>Cell content here.</p>
  </div>
</td>
```

⚠ uncertain — whether `<div>` nested inside `<td>` survives the DOCX export cleanly. If HTMLtoDOCX chokes on the div wrapper, options are (a) strip the div at export time and re-wrap the `<p>`, or (b) register a custom data-downcast for exception markers inside table cells that emits a class on the `<td>` rather than a wrapper. Decision deferred until export golden tests run (page 34).

---

## 6. Toolbar configuration

Two separate configs — Author gets everything, Fill gets only cell-content tools (for the dynamic variant we add structural buttons; the lock plugin still disables them inside fixed tables).

```ts
// Author
table: {
    contentToolbar: [
        'tableColumn', 'tableRow', 'mergeTableCells',
        'tableProperties', 'tableCellProperties',
        'toggleTableCaption'
    ],
    tableToolbar: [ 'bold', 'italic', 'link' ]
}

// Fill
table: {
    contentToolbar: [
        // Keep structural buttons visible — lock plugin disables them inside fixed tables.
        'tableColumn', 'tableRow', 'mergeTableCells'
    ],
    tableToolbar: [ 'bold', 'italic', 'link' ]
}
```

`tableColumn` and `tableRow` are dropdown groups provided by `TableUI` that already include the insert/remove/header variants. We do not need to enumerate each command.

For the **fixed** variant in Fill mode, the buttons remain in the toolbar but render disabled (CK5's standard `forceDisabled` visual). We considered filtering the toolbar config at runtime per selected table variant but rejected it — dynamic toolbar reshuffling is jarring and CK5 does not directly support it. Greyed-out buttons give the right "structure is locked" signal.

---

## 7. DOCX export

HTMLtoDOCX maps the native CK5 table HTML directly:

| CK5 data HTML | DOCX |
|---|---|
| `<figure class="table">` wrapper | dropped (HTMLtoDOCX understands the inner `<table>`) |
| `<table>` | `<w:tbl>` |
| `<thead>` / `<tbody>` | row grouping |
| `<tr>` / `<td>` / `<th>` | `<w:tr>` / `<w:tc>` |
| `<caption>` | table caption paragraph above table |
| Native column resize (inline styles on `<col>`) | `<w:tblGrid>` widths |
| `TableProperties` / `TableCellProperties` inline styles | mapped per existing HTMLtoDOCX rules |

Flat only. The `addChildCheck` in §4 guarantees no nested tables reach the exporter.

⚠ uncertain — figure wrapping. Empirically HTMLtoDOCX accepts `<figure class="table"><table>…</table></figure>` by ignoring the figure; verify on the pinned HTMLtoDOCX version and if necessary enable `PlainTableOutput` (GPL) which emits the `<table>` without the figure wrapper. `PlainTableOutput` also changes caption downcast; run a round-trip test before enabling.

---

## 8. PDF / print

Print CSS lives in `apps/ck5-studio/src/styles/app.css` (or the PDF-specific stylesheet — page 32):

```css
.ck-content figure.table { break-inside: avoid; }
.ck-content table { border-collapse: collapse; width: 100%; }
.ck-content table th,
.ck-content table td { border: 1px solid #111; padding: 4px 6px; }

/* Repeat header rows across page breaks. */
.ck-content table thead { display: table-header-group; }
.ck-content table tfoot { display: table-footer-group; }

/* Keep captions with the table. */
.ck-content figure.table > figcaption,
.ck-content table > caption { caption-side: top; break-after: avoid; }
```

Row-level `break-inside: avoid` is not recommended — long cells need to split or the page explodes.

---

## 9. Accessibility

Rules:

- **Always prefer a header row.** Author toolbar defaults the first row to `<th scope="col">` via `setTableColumnHeader` on insert.
- **Preserve `scope` on headers.** CK5 emits `scope="col"` / `scope="row"` — do not strip in export.
- **Caption, not paragraph-above.** Use `TableCaption` (`toggleTableCaption` command). Screen readers announce `<caption>`; a preceding `<p>` is invisible as an announcement.
- **Do not use `summary`.** Deprecated in HTML5. Put descriptive context in the `<caption>` or a visually-hidden paragraph before the table.
- **Keep ARIA off the table.** CK5's editing view uses `role="presentation"` on the DOM `<table>` while editing; the data view is plain HTML with native semantics. No custom ARIA is needed and adding it risks double-announcement.

---

## 10. Open questions (⚠ uncertain)

- Whether `restrictedEditingExceptionBlock`'s data-view wrapper inside `<td>` round-trips through HTMLtoDOCX cleanly (page 10 §"Confidence").
- Whether the lock plugin's `forceDisabled` key collides with `RestrictedEditingModeEditing`'s internal disabler keys. Source dive needed.
- Whether `PlainTableOutput` changes caption/header downcast in a way that breaks our exporter. Golden test required.
- Cell-level typed fields: page 22 owns this, but we need to confirm inline field widgets nest cleanly inside a block exception inside a `<td>`.
- Paste behaviour — if a user pastes a non-MDDM table into an Author template, should we auto-set `mddmTableVariant = 'dynamic'` in an upcast converter? Currently absent attribute is read as dynamic, so no runtime bug; open as a UX question.
- Column resize + DOCX width export precision (CK5 emits percentage widths; `<w:tblGrid>` wants twips).

---

## Sources

- https://ckeditor.com/docs/ckeditor5/latest/features/tables/tables.html — Tables feature overview (plugins, output shape, toolbar, caption, headers, nesting).
- https://ckeditor.com/docs/ckeditor5/latest/api/table.html — Table package API (commands registered by `Table`, `TableCaption`, `TableSelection`, `PlainTableOutput`).
- [10 — Restricted editing](./10-restricted-editing.md) — `restrictedEditingExceptionBlock` command and marker behaviour.
- [04 — Schema](./04-schema.md) — `addChildCheck`, schema vs restricted-editing separation.
- [05 — Conversion](./05-conversion.md) — attribute-to-attribute downcast used for `mddmTableVariant`.
- [07 — Markers](./07-markers.md) — `restrictedEditingException` marker persistence.
- [11 — Widgets](./11-widgets.md) — why we are *not* wrapping the table in a widget.
- [24 — Data format](./24-data-format.md) — HTML wire format, `<figure>` wrapping, attribute round-trip.
- MetalDocs memory: HTMLtoDOCX blocker on nested tables wrapped in `<figure>` (use flat tables).
