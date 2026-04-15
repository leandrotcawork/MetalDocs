---
title: FieldGroup mapping
status: draft
area: mapping
priority: HIGH
---

# 22 — FieldGroup / Field mapping

How MetalDocs implements typed, named, fillable fields in CKEditor 5 v48
under GPL (no access to the premium `Merge Fields` plugin — see
[01 — License / GPL](./01-license-gpl.md)).

> **Legacy deviation.** BlockNote shipped `FieldGroup` + `FieldSlot` /
> `RichSlot` / `slotBindings` as a structural construct with an
> out-of-band values map. That shape is **reference only** on the CK5
> rewrite. CK5's marker + widget + restricted-editing primitives subsume
> the old "slot binding" layer; we do not carry a parallel value map into
> the runtime (see §5 and [25 — MDDM IR bridge](./25-mddm-ir-bridge.md)).

---

## Recommended design (TL;DR)

- **Option C — Hybrid inline widget + marker.**
  - In the **Author** editor (`StandardEditingMode`), a field is an
    **inline widget** (`mddmField`, `$inlineObject`) rendering as a chip
    with the label, the type icon, and a default/placeholder value. This
    gives authors discoverability and a single click to configure the
    field.
  - On `dataDowncast` the widget collapses to a plain
    `<span class="mddm-field" data-field-id data-field-type data-field-label>…value…</span>`
    **wrapped by an inline `restrictedEditingException` marker** so the
    same HTML opens in Fill mode (`RestrictedEditingMode`) as an editable
    slot inside an otherwise read-only document.
  - The **value lives as text inside the span** (single source of truth).
    There is no parallel `{ fieldId: value }` map at runtime.
  - **FieldGroup is metadata only.** It is a property of the field
    definition (template-level), not a model element. The Author-side
    side panel groups fields by `group` for UX; the document tree never
    carries a `mddmFieldGroup` element.

Rationale in one line: inline widgets give the best authoring UX; a
plain span plus an inline restricted-editing exception gives the
cleanest DOCX/PDF export (the value is literally inline text) and the
smallest custom surface to maintain.

---

## 1. What is a "Field" for MetalDocs

A **Field** is a *typed, named, filled-in value embedded in document
content*. Concretely:

- Identity: `fieldId` (stable string, unique per template).
- Label: human-readable, shown on the chip and in accessibility tree.
- Type: one of `text | date | number | currency:<ISO> | select:[...] | boolean`.
- Default / placeholder value.
- Required / optional flag.
- Optional `group` (metadata string — "Customer info", "Order details").

Examples: customer name, order date, total amount, payment terms.

A **FieldGroup** is *not* a container in the document. It is a
grouping key on field definitions used by the Author-side side panel
and by validation summaries. It has no widget and no model element.

---

## 2. Candidates considered

| Option | Author view | Data shape | Fill UX | Export cleanliness |
|---|---|---|---|---|
| A — Inline widget only | Chip widget | `<span class="mddm-field" …>` with widget chrome stripped | Needs widget-aware typing; fights `RestrictedEditingMode` |  Cleanish, but chrome can leak |
| B — Marker-only anchor | Subtle highlight, no chip | `<span data-mddm-field-start-before=… data-mddm-field-end-after=…>` | Author cannot see/select the field as an atomic thing; hard to configure type | Excellent (just text) |
| **C — Hybrid (recommended)** | Chip widget in Author; plain span + inline restricted-editing exception on save | `<span class="mddm-field" data-*>…value…</span>` inside a `<span class="restricted-editing-exception">` | Native Fill-mode typing inside the exception | Excellent |

Why not A alone: a widget in the *data* view means Fill mode has to
make a widget editable, which is the scenario
[11 — Widgets §11](./11-widgets.md) flags as uncertain and
[10 — Restricted editing §7](./10-restricted-editing.md) warns about
(exception boundaries crossing widget boundaries are not supported).

Why not B alone: no author affordance. Configuring type / label /
required on a bare marker is a worse UX than clicking a chip.

Option C keeps widget ergonomics where they matter (Author) and uses
plain HTML + restricted-editing markers where they matter (Fill,
export, server rendering).

---

## 3. Schema

```js
// editor.model.schema
schema.register( 'mddmField', {
    inheritAllFrom: '$inlineObject',     // inline + object + selectable
    allowAttributes: [
        'fieldId',
        'fieldType',       // 'text' | 'date' | 'number' | 'currency:BRL' | 'select:[a,b]' | 'boolean'
        'fieldLabel',
        'fieldRequired',   // boolean
        'fieldValue',      // string — current value; empty = unfilled
    ],
} );
```

No `mddmFieldGroup` element. `group` is only a field-definition
property held in the template manifest (see
[28 — Template authoring](./28-template-authoring.md)).

---

## 4. Converters

### 4.1 Upcast (HTML → model)

```js
conversion.for( 'upcast' ).elementToElement( {
    view: { name: 'span', classes: 'mddm-field' },
    model: ( viewEl, { writer } ) => writer.createElement( 'mddmField', {
        fieldId:       viewEl.getAttribute( 'data-field-id' ),
        fieldType:     viewEl.getAttribute( 'data-field-type' ) || 'text',
        fieldLabel:    viewEl.getAttribute( 'data-field-label' ) || '',
        fieldRequired: viewEl.getAttribute( 'data-field-required' ) === 'true',
        fieldValue:    viewEl.getChild( 0 )?.data || '',
    } ),
} );
```

The inline restricted-editing-exception `<span>` around the field is
upcast independently by the `RestrictedEditingMode` / `StandardEditingMode`
plugin — we do **not** register that group ourselves (see
[07 — Markers §8](./07-markers.md),
[10 — Restricted editing](./10-restricted-editing.md)).

### 4.2 Data downcast (model → saved HTML)

```js
conversion.for( 'dataDowncast' ).elementToElement( {
    model: 'mddmField',
    view: ( modelEl, { writer } ) => {
        const span = writer.createContainerElement( 'span', {
            class: 'mddm-field',
            'data-field-id':       modelEl.getAttribute( 'fieldId' ),
            'data-field-type':     modelEl.getAttribute( 'fieldType' ),
            'data-field-label':    modelEl.getAttribute( 'fieldLabel' ),
            'data-field-required': String( !!modelEl.getAttribute( 'fieldRequired' ) ),
        } );
        const value = modelEl.getAttribute( 'fieldValue' ) || '';
        writer.insert( writer.createPositionAt( span, 0 ),
            writer.createText( value ) );
        return span;
    },
} );
```

### 4.3 Editing downcast (model → editing view — the chip)

```js
import { toWidget } from '@ckeditor/ckeditor5-widget/src/utils';

conversion.for( 'editingDowncast' ).elementToElement( {
    model: 'mddmField',
    view: ( modelEl, { writer } ) => {
        const id    = modelEl.getAttribute( 'fieldId' );
        const type  = modelEl.getAttribute( 'fieldType' ) || 'text';
        const label = modelEl.getAttribute( 'fieldLabel' ) || id;
        const value = modelEl.getAttribute( 'fieldValue' ) || '';

        const chip = writer.createContainerElement( 'span', {
            class: `mddm-field mddm-field--${ type.split( ':' )[ 0 ] }`,
            'data-field-id': id,
            'aria-label': `${ label } (${ type })`,
            role: 'textbox',
        } );
        writer.insert( writer.createPositionAt( chip, 0 ),
            writer.createText( value || `{{${ label }}}` ) );

        return toWidget( chip, writer, { label: `${ label } field` } );
    },
    triggerBy: { attributes: [ 'fieldValue', 'fieldLabel', 'fieldType', 'fieldRequired' ] },
} );
```

`triggerBy` re-runs the converter whenever any field attribute
changes, so the chip label updates live (see
[11 — Widgets §7](./11-widgets.md)).

### 4.4 Position mapper

Because `mddmField` is an inline widget with no children in the model
but renders text in the view, we need the standard outside-mapper (see
[11 — Widgets §3](./11-widgets.md)):

```js
import { viewToModelPositionOutsideModelElement }
    from '@ckeditor/ckeditor5-widget/src/utils';

editor.editing.mapper.on(
    'viewToModelPosition',
    viewToModelPositionOutsideModelElement(
        editor.model,
        view => view.hasClass( 'mddm-field' ),
    ),
);
```

---

## 5. Field values storage — decision

**Store the value as text inside the field element** (single source of
truth). Reject the parallel `{ fieldId: value }` map.

Reasons:

- No drift risk. Marker-moved, block-deleted, undo/redo all stay
  consistent because the value follows the element.
- Export is trivial — the value is already inline text in the DOCX/PDF
  pipeline (see §8, §9).
- Querying all field values is still O(n) but cheap: walk
  `editor.model.document.getRoot()` for `mddmField` elements, or
  iterate `markers.getMarkersGroup('restrictedEditingException')` and
  pick elements inside.
- Cross-ref: [25 — MDDM IR bridge](./25-mddm-ir-bridge.md) treats CK5
  HTML as transport only. Any persisted per-field value map on the
  server is derived from the HTML at save time, not maintained
  alongside it.

⚠ uncertain — whether a small server-side `fieldValues` cache is worth
maintaining for analytics / template reporting. Not required for
correctness; defer.

---

## 6. Insertion command

```js
// InsertFieldCommand.js
import { Command } from '@ckeditor/ckeditor5-core';

export default class InsertFieldCommand extends Command {
    /**
     * @param {{fieldId:string, fieldType:string, fieldLabel:string,
     *         defaultValue?:string, required?:boolean}} def
     */
    execute( def ) {
        const model = this.editor.model;
        model.change( writer => {
            const field = writer.createElement( 'mddmField', {
                fieldId:       def.fieldId,
                fieldType:     def.fieldType || 'text',
                fieldLabel:    def.fieldLabel || def.fieldId,
                fieldRequired: !!def.required,
                fieldValue:    def.defaultValue || '',
            } );
            // Insert as inline object — splits text nodes as needed.
            model.insertObject( field, null, null, { setSelection: 'after' } );

            // Wrap with an inline restricted-editing exception so Fill
            // mode lets the user edit the value. The feature manages
            // the marker group itself (see page 10).
            this.editor.execute( 'restrictedEditingException' );
        } );
    }

    refresh() {
        const sel = this.editor.model.document.selection;
        this.isEnabled = this.editor.model.schema
            .checkChild( sel.getFirstPosition(), 'mddmField' );
    }
}
```

Registration is standard (`editor.commands.add( 'insertField', … )`).
The Author-side palette (see [28](./28-template-authoring.md)) reads
the template's field definitions, groups them by `group`, and invokes
this command on drop.

---

## 7. Restricted-editing integration

- **Author mode**: on insert we immediately call
  `restrictedEditingException` (inline) so the field sits inside an
  exception marker. Without this wrap, Fill mode would not allow the
  value to be edited.
- **Fill mode**: the user Tab-navigates between fields using the built-in
  `goToNextRestrictedEditingException` / `goToPreviousRestrictedEditingException`
  commands (see [10 §commands](./10-restricted-editing.md)). The caret
  lands inside the exception span and typing replaces the content.
- Because the exception is inline, `restrictedEditing.allowedCommands`
  applies. Keep it to the typing defaults plus what the type validator
  needs (none for v1).

Open check: the field widget's own element (`<span class="mddm-field">`)
is non-editable widget chrome in the editing view, but in the **data
view** the `dataDowncast` emits a plain `<span>` — so the Fill-mode
editor sees a plain span inside the exception, not a widget. This is
the whole point of the hybrid design. ⚠ uncertain — verify
empirically that loading a saved document into `RestrictedEditingMode`
(no Author plugin set) produces a typable slot where the chip used to
be. Record result.

---

## 8. Typed fields & validation

Type is stored as `fieldType` on the model. The Fill-mode wrapper
attaches a small popup (positioned via
`editor.editing.view.domConverter` → DOM rect of the exception span)
when the caret enters a field:

| Type | Fill UI | Validator (on blur) |
|---|---|---|
| `text` | Plain typing | Length check if `max` given |
| `date` | Popup calendar; typed fallback `YYYY-MM-DD` | `Date.parse()` round-trip |
| `number` | Numeric keypad on mobile; `inputmode="decimal"` | `Number.isFinite()` |
| `currency:BRL` | Mask `R$ 0.000,00` | Parse to cents ≥ 0 |
| `select:[a,b,c]` | Dropdown; typing filters | Value ∈ options |
| `boolean` | Toggle chip (Yes/No) | `true`/`false` literal |

Validation surfaces:

- Inline: red border on the exception span when invalid.
- Side panel: list of required-but-empty + invalid fields, Tab to jump.
- Submit gate: `canExport()` returns false while any `required` field
  is empty or invalid.

See [37 — Validation](./37-validation.md) for the validator module
contract.

---

## 9. DOCX export

The field's value is a plain inline `<span class="mddm-field">value</span>`
in the saved HTML; the DOCX exporter (HTMLtoDOCX-based — see
[31 — DOCX export](./31-docx-export.md)) emits it as a run of inline
text. **Recommended**: no special treatment for v1; the value is
already the right thing.

Optional richer mapping:

- Wrap the value in a Word **Structured Document Tag** (SDT / Content
  Control) so downstream Word users can see "Customer name" as a
  labelled control.
- The `docx` npm library exposes `StructuredDocumentTag` /
  `ContentControl` primitives. ⚠ uncertain — whether the version
  pinned in MetalDocs supports inline SDT with our labelling needs;
  check before committing. If not, ship plain inline text and revisit.

---

## 10. PDF export

No special treatment. The field value is plain inline text in the
HTML; the PDF pipeline ([32 — PDF export](./32-pdf-export.md)) renders
it as-is. If we want "unfilled" fields to appear as `________` in the
PDF, substitute empty `fieldValue` with an underscore run at the HTML
pre-processing step — do **not** bake that into the model.

---

## 11. Accessibility

- Editing view chip carries `role="textbox"` and
  `aria-label="<label> (<type>)"`.
- Required fields add `aria-required="true"` on both editing and data
  spans.
- Invalid state adds `aria-invalid="true"` while the validator reports
  an error.
- Fill-mode navigation (Tab / Shift+Tab via the navigation commands) is
  the canonical keyboard interaction; announce each field's label on
  entry via the live-region that the side panel owns.

See [41 — Accessibility](./41-accessibility.md) for the global contract.

---

## 12. Open questions / uncertainties

- ⚠ Does `StandardEditingMode`'s `restrictedEditingException` inline
  command wrap cleanly around a freshly-inserted inline widget on the
  same tick? Verify; if not, defer the wrap to a `model.enqueueChange`
  block after the insert.
- ⚠ Data-view round-trip: loading `<span class="mddm-field">` **outside**
  an exception should still upcast to a `mddmField` element in Author
  mode. Confirm upcast order (ours runs before the restricted-editing
  plugin's).
- ⚠ DOCX SDT support in the pinned `docx` library version.
- ⚠ Whether server-side templating needs any value-query helper beyond
  walking the HTML — defer until a consumer asks.

---

## Sources

- [03 — Engine model](./03-engine-model.md)
- [04 — Schema](./04-schema.md)
- [05 — Conversion](./05-conversion.md)
- [07 — Markers](./07-markers.md)
- [10 — Restricted editing](./10-restricted-editing.md)
- [11 — Widgets](./11-widgets.md)
- [24 — Data format](./24-data-format.md)
- [25 — MDDM IR bridge](./25-mddm-ir-bridge.md)
- https://ckeditor.com/docs/ckeditor5/latest/framework/tutorials/widgets/implementing-an-inline-widget.html
- https://ckeditor.com/docs/ckeditor5/latest/features/restricted-editing.html
