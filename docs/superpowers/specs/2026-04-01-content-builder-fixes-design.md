# Content Builder Fixes Design

Date: 2026-04-01
Status: Approved
Scope: Fix three broken areas in the content-builder: .docx export for new documents, DynamicPreview visual fidelity, and TipTap rich field verification.

## Background

The schema-runtime platform is in place. The editor, preview, and docgen all share one schema contract. Three issues remain unfixed since the carbone to docgen migration:

1. "Exportar .docx" silently does nothing for new (unsaved) documents.
2. The "Ao vivo" preview renders as a plain form list instead of a document-like layout.
3. The TipTap rich field inside the `etapas` repeat has not been verified end-to-end.

## Goals

- Make "Exportar .docx" work for new documents via auto-save-then-export.
- Make "Ao vivo" preview look like a simplified HTML approximation of the final .docx output.
- Verify (and fix if needed) the TipTap rich field in the Processo/etapas/descricao path.

## Non-Goals

- Changing the export architecture (it stays Go to docgen).
- Making the preview pixel-perfect to the .docx.
- Changing `DynamicEditor` (the editing panel).
- Changing the schema runtime types or backend.
- Building a client-side docx generator in the browser.

---

## Fix 1: Export Auto-Save Flow

### Problem

`ContentBuilderView.handleExportDocx` exits immediately if `documentId` is empty:

```ts
if (!documentId || isExporting) return;
```

For new documents created via the native editor path, `documentId` is `""` until the user explicitly clicks "Gerar PDF". Clicking "Exportar .docx" silently does nothing.

### Solution

Add a pre-flight step at the top of `handleExportDocx`:

```
User clicks "Exportar .docx"
       ↓
Is documentId empty?
  YES + onCreateFromDraft exists
      → Call onCreateFromDraft(contentDraft)
      → Receive { documentId, pdfUrl, version }
      → Use the returned documentId for the export call
  YES + no onCreateFromDraft
      → Show error: "Salve o rascunho antes de exportar"
  NO
      → Continue with existing flow
             ↓
       Is status dirty?
          YES → save first (already handled)
          NO  → proceed
             ↓
       api.exportDocumentDocx(documentId)
             ↓
       Download .docx file
```

The button shows a loading/spinner state for the entire duration (saving + generating + downloading). The user sees one seamless action.

### Files Changed

- `frontend/apps/web/src/components/content-builder/ContentBuilderView.tsx` — update `handleExportDocx` only.

---

## Fix 2: DynamicPreview Visual Redesign

### Problem

The current preview renders a flat list of section cards, each with field labels and dash placeholders. It looks like a form, not a document. The vision from the schema-runtime spec is that editor, preview, and docgen all execute from the same schema — preview should be a simplified HTML approximation of the .docx output.

### Solution

Two components change: `PreviewDocumentPage` and the field rendering inside `DynamicPreview`.

#### PreviewDocumentPage — New Header

Replace the current minimal header with a two-row table matching the docgen `buildHeader()` structure:

**Row 1:**
- Left cell (purple background): document type label + document title
- Middle cell: Código value
- Right cell: Versão value

**Row 2:**
- Left cell (teal background): profile/document type name
- Right cell (spans 2 columns): Status value

Implemented as an HTML table or CSS Grid. Colors use the existing brand variables or dedicated preview tokens. Not pixel-perfect to the .docx — readable and recognizable as the same document.

#### DynamicPreview — Section Rendering

Replace the rounded card + "PREVIEW" badge with:
- A colored left-border bar for the section header (e.g. `border-left: 3px solid var(--vinho)`)
- Section number + title in a slightly emphasized style
- Section description in muted text below

#### DynamicPreview — Field Rendering in Preview Mode

Scalar fields (`text`, `textarea`) currently show a bordered box around the value. Replace with:
- Label in small-caps, muted color, above
- Value below, normal weight, no border box
- For empty values: an em-dash in muted color

Table fields: already render as HTML tables — style to match document aesthetic (header row with brand background, cell borders).

Repeat fields: each item renders as a block with a subtle separator between items, fields inside in the same label/value style.

Rich fields: already render sanitized HTML content via DOMPurify — no functional change needed, just ensure surrounding context looks document-like.

#### Files Changed

- `frontend/apps/web/src/components/content-builder/preview/PreviewDocumentPage.tsx`
- `frontend/apps/web/src/styles.css` — preview-document global styles live here; header and field styles added here
- `frontend/apps/web/src/features/documents/runtime/DynamicPreview.tsx`
- `frontend/apps/web/src/features/documents/runtime/DynamicEditor.module.css`

`DynamicEditor.tsx` (the editing panel) is **not changed**.

---

## Fix 3: TipTap Rich Field Verification

### Location

`rich` field: `descricao` (Descrição) inside the `etapas` repeat, inside section 3 "Processo".

To reach it: open content-builder → section Processo → scroll to ETAPAS → click "Adicionar item" → expand the new etapa card → find the Descrição field.

### Expected Behavior

- A TipTap editor renders with a full toolbar: B, I, S, Paragrafo, H1/H2/H3, Lista, Numerada, Citar, Tabela, Imagem, Undo, Redo, color picker.
- Typing into Descrição updates the "Ao vivo" preview in real time.
- The onChange chain flows: `RichField → RepeatField → DynamicEditor → ContentBuilderView`.

### Verify-First Policy

No preemptive code changes. Test in the browser first. Only fix what is actually broken.

### Known Risk Areas

1. If TipTap mounts but the toolbar is invisible: CSS class mismatch between `DynamicEditor.module.css` and `RichField.module.css`.
2. If typing does not update the preview: the `onChange` chain through `RepeatField` is broken (likely a missing `onChange` prop pass-through).
3. If the editor area is blank: `useEditor` returned null and the component rendered nothing — would require a null guard.

---

## Testing

| Area | How to verify |
|---|---|
| Export new doc | Create new document via "Novo documento" → native mode → go to content-builder → click "Exportar .docx" → should auto-save and download |
| Export existing doc | Open existing saved document → make an edit → click "Exportar .docx" → should save then download |
| Preview header | Open content-builder → preview panel shows branded two-row header with document type, code, version, status |
| Preview fields | Type in OBJETIVO field → preview shows value in document-like layout, no border box |
| Preview repeat | Add an etapa → preview shows the item in a block with label/value fields |
| TipTap mount | Add etapa → Descrição field shows TipTap toolbar and editable area |
| TipTap to preview | Type in Descrição → preview panel updates in real time |

## Acceptance Criteria

- Clicking "Exportar .docx" on a new unsaved document auto-saves and downloads the file without manual intervention.
- The "Ao vivo" preview panel looks recognizably like the final document — branded header, section structure, content-not-form field rendering.
- The TipTap rich field in etapas works end-to-end: mounts, accepts input, updates the preview.
