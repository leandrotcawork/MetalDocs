---
title: Template instantiation
status: draft
area: templates
priority: HIGH
---

# 29 — Template instantiation

How a saved template becomes a filled document. Covers the data flow, the
Fill-mode plugin set, caret landing, server-side pre-fill, live field
binding, validation, version drift, autosave, conflicts, and handoff.

Cross-refs: [10 — Restricted editing](./10-restricted-editing.md),
[11 — Widgets](./11-widgets.md),
[19 — Sections mapping](./19-sections-mapping.md),
[20 — Repeatables mapping](./20-repeatables-mapping.md),
[21 — DataTable mapping](./21-datatable-mapping.md),
[22 — FieldGroup mapping](./22-fieldgroup-mapping.md),
[23 — Rich block mapping](./23-rich-block-mapping.md),
[24 — Data format](./24-data-format.md),
[25 — MDDM IR bridge](./25-mddm-ir-bridge.md),
[28 — Template authoring](./28-template-authoring.md).

---

## Recommended design

**Instantiation is a copy.** There is no IR expansion, no render step, no
field-graph reification. A saved template is already CK5 HTML (see
[25 — §3](./25-mddm-ir-bridge.md)); a document instantiated from it is that
same HTML, copied into a new document record, optionally pre-filled with
defaults, and opened in a Fill-mode editor.

### Data flow

```
 ┌──────────────────┐
 │ template record  │   content_html (canonical, authored via
 │  (content_html)  │   StandardEditingMode; see page 28)
 └────────┬─────────┘
          │ POST /documents {templateId}
          ▼
 ┌──────────────────────────────────────────────┐
 │ server: instantiate                          │
 │  1. load template.content_html               │
 │  2. pre-fill defaults (linkedom rewrite)     │
 │  3. INSERT documents(content_html, tplId,    │
 │                      tplVersion, status=draft)│
 └────────┬─────────────────────────────────────┘
          │ 201 { documentId, content_html }
          ▼
 ┌──────────────────────────────────────────────┐
 │ client: Fill editor                          │
 │  • ClassicEditor.create(el, fillConfig)      │
 │  • editor.setData(content_html)              │
 │  • editor.execute(                           │
 │      'goToNextRestrictedEditingException')   │
 │  • MddmFieldPlugin ↔ React sidebar           │
 │  • Autosave → PATCH /documents/:id           │
 └────────┬─────────────────────────────────────┘
          │ user clicks "Submit"
          ▼
 ┌──────────────────────────────────────────────┐
 │ validate required fields → lock read-only    │
 │ (editor.enableReadOnlyMode('submitted'))     │
 └──────────────────────────────────────────────┘
```

No step in this chain reads or writes an IR. The HTML is the document.

---

## 1. Instantiation = copy + open

A document is a **frozen copy** of the template's HTML at the moment of
instantiation. Subsequent template edits do not retroactively flow into
existing documents (see §7).

Server-side flow (Node / linkedom):

```ts
// apps/api/src/documents/instantiate.ts  (illustrative)
import { parseHTML } from 'linkedom';

export async function instantiateDocument(
  templateId: string,
  actor: { userId: string; orgId: string },
): Promise<{ id: string; contentHtml: string }> {
  const tpl = await db.templates.findByIdOrThrow(templateId);

  // Pre-fill well-known defaults into the template HTML.
  const hydrated = preFillDefaults(tpl.contentHtml, {
    'field.today':        new Date().toISOString().slice(0, 10),
    'field.current.user': await db.users.fullName(actor.userId),
    'field.current.org':  await db.orgs.name(actor.orgId),
  });

  return db.documents.create({
    templateId:      tpl.id,
    templateVersion: tpl.version,     // frozen at instantiation
    contentHtml:     hydrated,
    status:          'draft',
    createdBy:       actor.userId,
  });
}

function preFillDefaults(
  html: string,
  defaults: Record<string, string>,
): string {
  const { document } = parseHTML(html);
  for (const span of document.querySelectorAll('span.mddm-field')) {
    const fieldId = span.getAttribute('data-field-id');
    if (fieldId && defaults[fieldId] != null) {
      span.textContent = defaults[fieldId];
      span.setAttribute('data-field-value', defaults[fieldId]);
    }
  }
  return document.toString();
}
```

Notes:

- Pre-fill is a **string rewrite**, not a CK5 round-trip. Cheap — tens of
  ms even on large templates. ⚠ uncertain on exact cost; benchmark when
  templates exceed ~500 KB HTML.
- We write both the span's visible text and a `data-field-value` attribute
  for later marker-value extraction on save (see §5).
- The marker wrapper class is `mddm-field`, not
  `restricted-editing-exception` — see [28 — Template authoring](./28-template-authoring.md)
  for the decision to nest a `span.mddm-field[data-field-id]` *inside* an
  inline restricted-editing exception rather than reusing the exception
  span as the field carrier. This keeps field identity stable across edits
  that split/merge exceptions.

---

## 2. Fill-mode plugin set

Different from the Author (template) config. Fill mode is intentionally
smaller: no primitive-insertion, no structural commands, no exception
toggling.

```ts
// apps/ck5-studio/src/lib/fillEditorConfig.ts
import { ClassicEditor } from '@ckeditor/ckeditor5-editor-classic';
import { Essentials } from '@ckeditor/ckeditor5-essentials';
import { Paragraph } from '@ckeditor/ckeditor5-paragraph';
import { Bold, Italic } from '@ckeditor/ckeditor5-basic-styles';
import { Link } from '@ckeditor/ckeditor5-link';
import { List } from '@ckeditor/ckeditor5-list';
import { Table, TableToolbar } from '@ckeditor/ckeditor5-table';
import { Image, ImageInsert, ImageStyle } from '@ckeditor/ckeditor5-image';
import { Alignment } from '@ckeditor/ckeditor5-alignment';
import { Font } from '@ckeditor/ckeditor5-font';
import { RestrictedEditingMode } from '@ckeditor/ckeditor5-restricted-editing';
import { Autosave } from '@ckeditor/ckeditor5-autosave';

import { MddmTableLockPlugin } from './plugins/mddmTableLock';
import { MddmFieldPlugin }     from './plugins/mddmField';

export const fillEditorConfig = {
  plugins: [
    Essentials, Paragraph,
    Bold, Italic, Link, List,
    Table, TableToolbar,
    Image, ImageInsert, ImageStyle,
    Alignment, Font,
    RestrictedEditingMode,            // NOT StandardEditingMode
    Autosave,
    MddmTableLockPlugin,              // active in Fill mode (see page 13)
    MddmFieldPlugin,                  // field chip ↔ sidebar binding
  ],
  restrictedEditing: {
    // inline-exception allow-list (page 10 §Configuration)
    allowedCommands: ['bold', 'italic', 'link', 'alignment', 'fontColor'],
  },
  toolbar: {
    // No insertTable, no insertImage at top level — structural commands
    // fire only from inside block exceptions where authors opted in.
    items: [
      'undo', 'redo', '|',
      'bold', 'italic', 'link', '|',
      'bulletedList', 'numberedList', '|',
      'alignment', 'fontColor', '|',
      'goToPreviousRestrictedEditingException',
      'goToNextRestrictedEditingException',
    ],
  },
  autosave: {
    save(editor) { return saveToServer(editor.getData()); },
    waitingTime: 1500,
  },
};
```

Not loaded in Fill mode:

- `StandardEditingMode` — mutually exclusive with `RestrictedEditingMode`
  (page 10).
- MetalDocs primitive-insertion commands (`insertMddmField`,
  `insertMddmRepeatable`, `insertMddmSection`). Authors add primitives;
  fillers fill.
- Block-structural commands outside exceptions. `MddmTableLockPlugin`
  (page 13) keeps fixed tables structurally locked (no add/remove rows
  or columns); dynamic tables unlock the structural subset inside their
  block exception.

---

## 3. Caret landing

On editor ready, the caret lands in the first fillable region:

```ts
const editor = await ClassicEditor.create(host, fillEditorConfig);
editor.setData(documentHtml);

// defer one microtask so setData's post-fixers settle before we execute
// a navigation command that walks markers.
await Promise.resolve();

editor.execute('goToNextRestrictedEditingException');
editor.editing.view.focus();
```

Command names verified against [page 10 §Commands](./10-restricted-editing.md):

- `goToNextRestrictedEditingException`
- `goToPreviousRestrictedEditingException`

Both are registered by `RestrictedEditingModeEditing` and only exist when
`RestrictedEditingMode` is loaded. Bind them to Tab / Shift+Tab in the UI
layer (see page 10).

⚠ uncertain — whether `setData` is fully synchronous w.r.t. marker restoration
on v48. A single-tick defer is the safe default; tighten if benchmarks show
it is unnecessary.

---

## 4. Server-side pre-fill

Pre-fill happens once, at instantiation. Well-known defaults:

| Field ID              | Source                                |
|-----------------------|---------------------------------------|
| `field.today`         | server date (`YYYY-MM-DD`)            |
| `field.current.user`  | authenticated user's full name        |
| `field.current.org`   | user's active org                     |
| `field.doc.number`    | sequence allocation (template-specific) |

The rewrite runs on the raw template HTML using linkedom (see code in §1).
Spans that have no matching default are left untouched — the filler types
them in Fill mode.

Why not client-side? Two reasons:

- Some defaults (doc number, org display name) are server-authoritative.
- Keeps the client's `setData` input consistent with what's persisted — no
  "unsaved pre-fill" race on first autosave.

---

## 5. Live field binding (MddmFieldPlugin)

The plugin is thin. It does two things:

1. **Editor → sidebar**: on `model.document.on('change:data')`, walk the
   change delta for `mddmField` model elements, read their text content
   (the filled value), and push a `{ fieldId, value }` update into a Zustand
   store the React sidebar subscribes to.
2. **Sidebar → editor**: when the sidebar commits a change, call into the
   plugin's `setFieldValue(fieldId, value)`, which runs
   `editor.model.change(writer => { … })` to replace the child text of the
   matching `mddmField` element and set `writer.setAttribute('fieldValue',
   value, element)` for indexed querying on save.

Sketch:

```ts
// plugins/mddmField.ts  (illustrative)
export class MddmFieldPlugin extends Plugin {
  static get pluginName() { return 'MddmFieldPlugin'; }

  init() {
    const editor = this.editor;

    editor.model.document.on('change:data', () => {
      const snapshot = collectFieldValues(editor.model);
      fieldStore.getState().replaceAll(snapshot);
    });

    editor.commands.add('setMddmFieldValue',
      new SetMddmFieldValueCommand(editor));
  }
}
```

The React sidebar component lives outside CK5 (see
[09 — Decoupled editor](./09-decoupled-editor.md) for the separation
pattern). It reads `fieldStore` and, on commit, dispatches:

```ts
editor.execute('setMddmFieldValue', { fieldId, value });
```

⚠ uncertain — whether we want per-keystroke sync or debounced (250 ms). Per
keystroke is simpler; debounced cuts re-renders on a 200-field document.
Start per keystroke; revisit if profiler flags it.

---

## 6. Validation on submit

Before transitioning a document from `draft` → `submitted`:

```ts
function validateRequired(editor: Editor): { ok: true } | { ok: false; missing: string[] } {
  const missing: string[] = [];
  for (const { element } of editor.model.document.getRoot().getChildren()) {
    for (const node of walkFields(element)) {
      if (node.getAttribute('required') && isEmpty(node)) {
        missing.push(node.getAttribute('fieldId') as string);
      }
    }
  }
  return missing.length ? { ok: false, missing } : { ok: true };
}
```

On failure:

- Block the submit action (keep status `draft`).
- Surface a toast: "3 required fields are missing".
- Highlight unfilled chips — add a view class via
  `editor.editing.view.change(writer => writer.addClass('mddm-field--missing', viewEl))`
  for each missing field. Clear on next `change:data`.

Required-ness is declared at authoring time (page 28) and persisted as
`data-required="true"` on the field span.

---

## 7. Template version drift

Policy: **documents are frozen copies**. A document records
`templateId` + `templateVersion` at instantiation; it does not auto-migrate
when the template is re-published.

- Publishing template v2 does not touch existing v1-derived documents.
- Listing UIs may surface a badge ("Template has been updated since this
  document was created") but no content changes.
- Explicit opt-in migration would be a separate action:
  `POST /documents/:id/upgrade-template` → re-instantiate from v2, attempt
  to copy field values forward by `fieldId`, return a diff for user review.
  Not implemented; shape documented so we don't paint ourselves into a
  corner.

⚠ uncertain — migration conflict resolution when a field is removed or
renamed between v1 and v2. Defer until a concrete user need exists; the
re-instantiate-with-value-copy shape above is a sketch, not a commitment.

---

## 8. Autosave

`Autosave` plugin (CK5 built-in) with a 1.5 s debounce:

```ts
autosave: {
  save(editor) {
    return fetch(`/api/documents/${documentId}`, {
      method: 'PATCH',
      headers: {
        'Content-Type':   'application/json',
        'If-Match':       etag,                  // optimistic concurrency
      },
      body: JSON.stringify({ contentHtml: editor.getData() }),
    }).then(res => {
      if (res.status === 409) throw new ConflictError();
      etag = res.headers.get('ETag')!;
    });
  },
  waitingTime: 1500,
}
```

- Optimistic UI: we render the new value immediately; the PATCH runs in
  background.
- On network error: CK5's Autosave retries with its internal backoff.
- See [26 — Autosave & persistence](./26-autosave-persistence.md) for the
  broader retry / offline policy.

---

## 9. Conflict handling

No real-time collaboration — CKEditor's RTC is a commercial add-on that
conflicts with our GPL posture (see
[01 — license](./01-license-gpl.md)). Two-tab conflict resolution is
coarse:

- PATCH returns 409 on ETag mismatch.
- Client surfaces a modal: "This document was changed in another tab.
  Reload?"
- On reload: fetch latest `contentHtml`, destroy editor, recreate with
  new data, restore caret via `goToNextRestrictedEditingException` (loses
  exact caret position — acceptable trade-off vs building an operational
  merge).

No operational transform. No three-way merge.

---

## 10. Handoff / submit

On submit (after validation passes in §6):

```ts
async function submitDocument(editor: Editor, documentId: string) {
  const result = validateRequired(editor);
  if (!result.ok) { highlightMissing(editor, result.missing); return; }

  // Flush pending autosave first.
  await editor.plugins.get('Autosave').save(editor);

  await fetch(`/api/documents/${documentId}/submit`, { method: 'POST' });

  // Lock the editor client-side. Server now rejects further PATCHes.
  editor.enableReadOnlyMode('mddm-submitted');
}
```

Read-only in CK5 is a lock-id API (`enableReadOnlyMode(id)` /
`disableReadOnlyMode(id)`), not a boolean — multiple sources can hold a
lock; the editor is read-only while any lock is held.

Alternative: load submitted documents in a plain HTML viewer instead of
CK5 — see [33 — HTML viewer](./33-html-viewer.md) for the lighter-weight
path (no editor bundle, no CK5 startup cost).

Recommendation: use `enableReadOnlyMode` when the user submits in the same
session (already in the editor); use the HTML viewer when opening a
submitted document cold.

---

## Open questions

- ⚠ uncertain — whether the defer before `goToNextRestrictedEditingException`
  (§3) can be removed on v48. Empirical check needed.
- ⚠ uncertain — per-keystroke vs debounced sidebar sync (§5).
- ⚠ uncertain — pre-fill cost at template sizes > 500 KB (§4). Benchmark.
- ⚠ uncertain — template-version migration mechanics (§7). Shape only.
- ⚠ uncertain — whether autosave should flush on window `beforeunload`.
  CK5's Autosave claims it does; verify on pinned v48.

---

## Sources & cross-refs

- [09 — Decoupled editor](./09-decoupled-editor.md)
- [10 — Restricted editing](./10-restricted-editing.md)
- [11 — Widgets](./11-widgets.md)
- [13 — Tables](./13-tables.md)
- [19 — Sections mapping](./19-sections-mapping.md)
- [20 — Repeatables mapping](./20-repeatables-mapping.md)
- [21 — DataTable mapping](./21-datatable-mapping.md)
- [22 — FieldGroup mapping](./22-fieldgroup-mapping.md)
- [23 — Rich block mapping](./23-rich-block-mapping.md)
- [24 — Data format](./24-data-format.md)
- [25 — MDDM IR bridge](./25-mddm-ir-bridge.md)
- [26 — Autosave & persistence](./26-autosave-persistence.md)
- [28 — Template authoring](./28-template-authoring.md)
- [33 — HTML viewer](./33-html-viewer.md)
- [37 — Validation](./37-validation.md)
