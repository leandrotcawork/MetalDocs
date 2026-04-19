# Content Builder Fixes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix three broken areas in the content-builder: silent .docx export failure for new documents, flat-form live preview that doesn't look like the final document, and unverified TipTap rich text field in the etapas repeat.

**Architecture:** Frontend-only changes across five files. The export fix reuses the existing `onCreateFromDraft` save path before calling the Go API. The preview redesign replaces the simple MetalDocs header with a two-row branded table and removes bordered boxes from field values. TipTap is verified in the browser and fixed only if broken.

**Tech Stack:** React 18, TypeScript, CSS Modules (`DynamicEditor.module.css`), global CSS (`styles.css`), Vite, TipTap v2.

---

## File Map

| File | Change |
|------|--------|
| `frontend/apps/web/src/components/content-builder/ContentBuilderView.tsx` | Fix `handleExportDocx` — add pre-flight save for new documents |
| `frontend/apps/web/src/components/content-builder/preview/PreviewPanel.tsx` | Add `documentStatus: string` prop, pass to DocumentPreviewRenderer |
| `frontend/apps/web/src/components/content-builder/preview/DocumentPreviewRenderer.tsx` | Add `documentStatus: string` prop, pass to DynamicPreview |
| `frontend/apps/web/src/features/documents/runtime/DynamicPreview.tsx` | Add `documentStatus: string` prop, pass to PreviewDocumentPage |
| `frontend/apps/web/src/components/content-builder/preview/PreviewDocumentPage.tsx` | Replace current header with two-row branded table; remove title block |
| `frontend/apps/web/src/styles.css` | Replace old `.preview-document-header-*` classes with new header table classes; update `.preview-document-page` |
| `frontend/apps/web/src/features/documents/runtime/DynamicEditor.module.css` | Pull `.previewValue` out of shared border rule; add `.previewSectionMode` override class |
| `frontend/apps/web/src/features/documents/runtime/DynamicPreview.tsx` | Apply `.previewSectionMode` to section elements |

---

## Task 1: Fix export auto-save flow

**Files:**
- Modify: `frontend/apps/web/src/components/content-builder/ContentBuilderView.tsx:288-315`

- [ ] **Step 1: Replace `handleExportDocx` with the auto-save pre-flight version**

  Open `ContentBuilderView.tsx`. Replace the entire `handleExportDocx` function (lines 288–315) with:

  ```typescript
  async function handleExportDocx() {
    if (isExporting) return;
    setIsExporting(true);
    dispatch({ type: "set_error", payload: { message: "" } });
    try {
      let exportId = documentId;

      if (!exportId) {
        if (!props.onCreateFromDraft) {
          dispatch({ type: "set_error", payload: { message: "Salve o rascunho antes de exportar." } });
          return;
        }
        dispatch({ type: "set_status", payload: { status: "saving" } });
        const created = await props.onCreateFromDraft(contentDraft ?? {});
        autoSave.acknowledgeSave(contentDraft ?? {}, created.pdfUrl);
        dispatch({
          type: "load_success",
          payload: {
            contentDraft: contentDraft ?? {},
            schema,
            version: created.version ?? null,
            pdfUrl: created.pdfUrl,
          },
        });
        exportId = created.documentId;
      } else if (status === "dirty") {
        const saved = await handleSave();
        if (!saved) return;
      }

      const blob = await api.exportDocumentDocx(exportId);
      const url = window.URL.createObjectURL(blob);
      const link = document.createElement("a");
      const downloadName =
        `${(documentCode || exportId || "documento")
          .toLowerCase()
          .replace(/[^a-z0-9._-]+/gi, "-")
          .replace(/^-+|-+$/g, "") || "documento"}.docx`;
      link.href = url;
      link.download = downloadName;
      document.body.appendChild(link);
      link.click();
      link.remove();
      window.URL.revokeObjectURL(url);
    } catch {
      dispatch({ type: "set_error", payload: { message: "Nao foi possivel exportar o DOCX." } });
    } finally {
      setIsExporting(false);
    }
  }
  ```

- [ ] **Step 2: Typecheck**

  ```bash
  cd frontend/apps/web && npx tsc --noEmit 2>&1 | head -40
  ```

  Expected: zero errors.

- [ ] **Step 3: Commit**

  ```bash
  git add frontend/apps/web/src/components/content-builder/ContentBuilderView.tsx
  git commit -m "fix(content-builder): auto-save new document before docx export"
  ```

---

## Task 2: Thread `documentStatus` through the preview prop chain

**Files:**
- Modify: `frontend/apps/web/src/components/content-builder/ContentBuilderView.tsx` (PreviewPanel call site)
- Modify: `frontend/apps/web/src/components/content-builder/preview/PreviewPanel.tsx`
- Modify: `frontend/apps/web/src/components/content-builder/preview/DocumentPreviewRenderer.tsx`
- Modify: `frontend/apps/web/src/features/documents/runtime/DynamicPreview.tsx`
- Modify: `frontend/apps/web/src/components/content-builder/preview/PreviewDocumentPage.tsx`

The preview header needs the document's workflow status (e.g. "DRAFT"). It is available in `ContentBuilderView` as `props.document?.status`. This task threads it down to `PreviewDocumentPage` without changing anything else.

- [ ] **Step 1: Add `documentStatus` to `PreviewPanel` props and pass it to `DocumentPreviewRenderer`**

  In `PreviewPanel.tsx`, add `documentStatus: string` to `PreviewPanelProps`:

  ```typescript
  type PreviewPanelProps = {
    schema: DocumentProfileSchemaItem | null;
    contentDraft: Record<string, unknown>;
    pdfUrl: string;
    profileCode: string;
    documentCode: string;
    documentTitle: string;
    version: number | null;
    activeSectionKey?: string | null;
    isDirty: boolean;
    collapsed: boolean;
    documentStatus: string;
    onToggleCollapse: (collapsed: boolean) => void;
  };
  ```

  Destructure `documentStatus` in the function body:

  ```typescript
  export function PreviewPanel(props: PreviewPanelProps) {
    const {
      schema,
      contentDraft,
      pdfUrl,
      profileCode,
      documentCode,
      documentTitle,
      version,
      activeSectionKey,
      isDirty,
      collapsed,
      documentStatus,
      onToggleCollapse,
    } = props;
  ```

  Pass it to `DocumentPreviewRenderer` in the JSX:

  ```tsx
  <DocumentPreviewRenderer
    sections={sections}
    content={contentDraft}
    profileCode={profileCode}
    documentCode={documentCode}
    title={documentTitle}
    version={version}
    activeSectionKey={activeSectionKey}
    documentStatus={documentStatus}
  />
  ```

- [ ] **Step 2: Add `documentStatus` to `DocumentPreviewRenderer` and pass it to `DynamicPreview`**

  In `DocumentPreviewRenderer.tsx`, update the props type and forward the value:

  ```typescript
  type DocumentPreviewRendererProps = {
    sections: SchemaSection[];
    content: Record<string, unknown>;
    profileCode: string;
    documentCode: string;
    title: string;
    version: number | null;
    activeSectionKey?: string | null;
    documentStatus: string;
  };

  export function DocumentPreviewRenderer({
    sections,
    content,
    profileCode,
    documentCode,
    title,
    version,
    activeSectionKey,
    documentStatus,
  }: DocumentPreviewRendererProps) {
    return (
      <DynamicPreview
        schema={{ profileCode, version: version ?? 0, isActive: true, metadataRules: [], contentSchema: { sections } }}
        content={content}
        profileCode={profileCode}
        documentCode={documentCode}
        title={title}
        version={version}
        activeSectionKey={activeSectionKey}
        documentStatus={documentStatus}
      />
    );
  }
  ```

- [ ] **Step 3: Add `documentStatus` to `DynamicPreview` and pass it to `PreviewDocumentPage`**

  In `DynamicPreview.tsx`, update the props type:

  ```typescript
  type DynamicPreviewProps = {
    schema: DocumentProfileSchemaItem | null;
    content: Record<string, unknown>;
    profileCode: string;
    documentCode: string;
    title: string;
    version: number | null;
    activeSectionKey?: string | null;
    documentStatus: string;
  };
  ```

  Destructure `documentStatus` and pass it to `PreviewDocumentPage`:

  ```tsx
  export function DynamicPreview({
    schema,
    content,
    profileCode,
    documentCode,
    title,
    version,
    activeSectionKey,
    documentStatus,
  }: DynamicPreviewProps) {
    const runtimeSchema = toRuntimeDocumentSchema(schema?.contentSchema);

    return (
      <PreviewDocumentPage
        profileCode={profileCode}
        documentCode={documentCode}
        title={title}
        version={version}
        documentStatus={documentStatus}
      >
        {/* existing section rendering — unchanged */}
      </PreviewDocumentPage>
    );
  }
  ```

  Keep all existing section-rendering JSX inside the children — only the `PreviewDocumentPage` opening tag changes.

- [ ] **Step 4: Add `documentStatus` to `PreviewDocumentPage` props**

  In `PreviewDocumentPage.tsx`, add the new prop to the type (actual JSX changes come in Task 3):

  ```typescript
  type PreviewDocumentPageProps = {
    profileCode: string;
    documentCode: string;
    title: string;
    version: number | null;
    documentStatus: string;
    children: ReactNode;
  };
  ```

  Destructure it in the function signature:

  ```typescript
  export function PreviewDocumentPage({ profileCode, documentCode, title, version, documentStatus, children }: PreviewDocumentPageProps) {
  ```

  Leave the JSX body unchanged for now — Task 3 rewrites it.

- [ ] **Step 5: Pass `documentStatus` from `ContentBuilderView` to `PreviewPanel`**

  In `ContentBuilderView.tsx`, find the `<PreviewPanel ...>` JSX block (around line 417) and add:

  ```tsx
  <PreviewPanel
    schema={schema}
    contentDraft={contentDraft}
    pdfUrl={pdfUrl}
    profileCode={profileCode}
    documentCode={documentCode}
    documentTitle={documentTitle}
    version={version}
    activeSectionKey={activeSectionKey}
    isDirty={status === "dirty"}
    collapsed={previewCollapsed}
    documentStatus={props.document?.status ?? "DRAFT"}
    onToggleCollapse={(collapsed) => dispatch({ type: "set_preview", payload: { collapsed } })}
  />
  ```

- [ ] **Step 6: Typecheck**

  ```bash
  cd frontend/apps/web && npx tsc --noEmit 2>&1 | head -40
  ```

  Expected: zero errors.

- [ ] **Step 7: Commit**

  ```bash
  git add \
    frontend/apps/web/src/components/content-builder/ContentBuilderView.tsx \
    frontend/apps/web/src/components/content-builder/preview/PreviewPanel.tsx \
    frontend/apps/web/src/components/content-builder/preview/DocumentPreviewRenderer.tsx \
    frontend/apps/web/src/features/documents/runtime/DynamicPreview.tsx \
    frontend/apps/web/src/components/content-builder/preview/PreviewDocumentPage.tsx
  git commit -m "refactor(preview): thread documentStatus through preview prop chain"
  ```

---

## Task 3: Redesign PreviewDocumentPage header

**Files:**
- Modify: `frontend/apps/web/src/components/content-builder/preview/PreviewDocumentPage.tsx`
- Modify: `frontend/apps/web/src/styles.css`

- [ ] **Step 1: Replace the JSX in `PreviewDocumentPage`**

  Replace the entire return body of `PreviewDocumentPage` with:

  ```tsx
  return (
    <div className="preview-document">
      <div className="preview-document-page">
        <div className="preview-doc-header-table">
          <div className="preview-doc-header-row preview-doc-header-row--purple">
            <div className="preview-doc-header-cell preview-doc-header-cell--title">
              <span className="preview-doc-header-type">{profileCode.toUpperCase()}</span>
              <span className="preview-doc-header-doc-title">{title || "Sem titulo"}</span>
            </div>
            <div className="preview-doc-header-cell preview-doc-header-cell--meta">
              <span className="preview-doc-header-meta-label">Código</span>
              <span className="preview-doc-header-meta-value">{documentCode || "—"}</span>
            </div>
            <div className="preview-doc-header-cell preview-doc-header-cell--meta">
              <span className="preview-doc-header-meta-label">Versão</span>
              <span className="preview-doc-header-meta-value">v{version ?? "–"}</span>
            </div>
          </div>
          <div className="preview-doc-header-row preview-doc-header-row--teal">
            <div className="preview-doc-header-cell preview-doc-header-cell--profile">
              <span className="preview-doc-header-profile-label">{profileCode.toUpperCase()}</span>
            </div>
            <div className="preview-doc-header-cell preview-doc-header-cell--status">
              <span className="preview-doc-header-meta-label">Status</span>
              <span className="preview-doc-header-meta-value">{documentStatus}</span>
            </div>
          </div>
        </div>

        <div className="preview-document-content">{children}</div>

        <footer className="preview-document-footer">
          <span>{profileCode.toUpperCase()} — {documentCode || "—"}</span>
          <span>Versao {version ?? "–"}</span>
        </footer>
      </div>
    </div>
  );
  ```

- [ ] **Step 2: Replace old header CSS in `styles.css` with new header table classes**

  Find the block starting at `.preview-document-header {` and ending before `/* LIVE EDITOR */` comment (approximately lines 3379–3466). Replace those lines (the header/title-block rules only — keep `.preview-document {`, `.preview-document-page {`, `.preview-document-content {`, `.preview-document-footer {`) with:

  ```css
  .preview-doc-header-table {
    border-radius: 6px;
    overflow: hidden;
    margin-bottom: 16px;
    border: 1px solid rgba(74, 61, 181, 0.2);
  }

  .preview-doc-header-row {
    display: grid;
  }

  .preview-doc-header-row--purple {
    grid-template-columns: 1fr 60px 60px;
    background: #4a3db5;
  }

  .preview-doc-header-row--teal {
    grid-template-columns: 1fr 80px;
    background: #0f6e56;
  }

  .preview-doc-header-cell {
    padding: 6px 8px;
    border-right: 1px solid rgba(255, 255, 255, 0.15);
  }

  .preview-doc-header-cell:last-child {
    border-right: none;
  }

  .preview-doc-header-cell--title {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .preview-doc-header-cell--meta {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 1px;
  }

  .preview-doc-header-cell--profile {
    display: flex;
    align-items: center;
  }

  .preview-doc-header-cell--status {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 1px;
  }

  .preview-doc-header-type {
    font-size: 8px;
    font-weight: 700;
    letter-spacing: 0.08em;
    color: rgba(255, 255, 255, 0.7);
    text-transform: uppercase;
  }

  .preview-doc-header-doc-title {
    font-size: 10px;
    font-weight: 600;
    color: #fff;
    line-height: 1.3;
  }

  .preview-doc-header-meta-label {
    font-size: 7px;
    color: rgba(255, 255, 255, 0.65);
    text-transform: uppercase;
    letter-spacing: 0.06em;
  }

  .preview-doc-header-meta-value {
    font-size: 9px;
    font-weight: 700;
    color: #fff;
  }

  .preview-doc-header-profile-label {
    font-size: 9px;
    font-weight: 700;
    color: #fff;
    text-transform: uppercase;
    letter-spacing: 0.06em;
  }
  ```

  The old classes to delete are (these are no longer referenced):
  - `.preview-document-header`
  - `.preview-document-header-left`
  - `.preview-document-header-right`
  - `.preview-document-logo-area`
  - `.preview-document-header-info`
  - `.preview-document-header-brand`
  - `.preview-document-header-profile`
  - `.preview-document-header-code`
  - `.preview-document-header-version`
  - `.preview-document-title-block`
  - `.preview-document-title`

- [ ] **Step 3: Typecheck and build**

  ```bash
  cd frontend/apps/web && npx tsc --noEmit 2>&1 | head -40
  ```

  Expected: zero errors.

- [ ] **Step 4: Verify in browser**

  Rebuild: `cd frontend/apps/web && npm run build`

  Open the content-builder (create a new doc or open an existing one). The preview panel right side should now show a two-row colored header table — purple row with the document type code and title on the left, Código and Versão on the right; teal row below with the profile code and Status.

- [ ] **Step 5: Commit**

  ```bash
  git add \
    frontend/apps/web/src/components/content-builder/preview/PreviewDocumentPage.tsx \
    frontend/apps/web/src/styles.css
  git commit -m "feat(preview): replace flat header with branded two-row table header"
  ```

---

## Task 4: Redesign preview field and section rendering

**Files:**
- Modify: `frontend/apps/web/src/features/documents/runtime/DynamicEditor.module.css`
- Modify: `frontend/apps/web/src/features/documents/runtime/DynamicPreview.tsx`

- [ ] **Step 1: Update `.previewValue` and add `.previewSectionMode` in `DynamicEditor.module.css`**

  **Change 1 — remove `.previewValue` from the shared border rule.** Find:

  ```css
  .control,
  .previewValue,
  .tableShell,
  .repeatShell,
  .richShell {
    border: 1px solid color-mix(in srgb, var(--vinho) 14%, var(--cinza-300) 86%);
    border-radius: 16px;
    background: var(--white);
  }
  ```

  Replace with (`.previewValue` removed from the shared rule):

  ```css
  .control,
  .tableShell,
  .repeatShell,
  .richShell {
    border: 1px solid color-mix(in srgb, var(--vinho) 14%, var(--cinza-300) 86%);
    border-radius: 16px;
    background: var(--white);
  }
  ```

  **Change 2 — rewrite `.previewValue` rule.** Find the existing `.previewValue` rule block:

  ```css
  .previewValue {
    display: flex;
    align-items: center;
    min-height: 2.8rem;
    padding: 0.75rem 0.9rem;
    color: var(--vinho-d);
  }
  ```

  Replace with:

  ```css
  .previewValue {
    display: block;
    padding: 0.15rem 0 0.35rem;
    color: var(--vinho-d);
    font-size: 0.88rem;
    line-height: 1.5;
  }
  ```

  **Change 3 — add `.previewSectionMode` override at the end of the file** (before the `@media` block):

  ```css
  /* ── Preview-mode section overrides (DynamicPreview only) ── */
  .previewSectionMode.section {
    border: none;
    box-shadow: none;
    background: transparent;
    border-radius: 0;
    border-left: 3px solid color-mix(in srgb, var(--vinho) 45%, transparent);
    padding-left: 0;
    overflow: visible;
  }

  .previewSectionMode .sectionHeader {
    background: color-mix(in srgb, var(--vinho-soft) 55%, white 45%);
    border-bottom: none;
    padding: 0.5rem 0.85rem;
    border-radius: 0;
    cursor: default;
    pointer-events: none;
  }

  .previewSectionMode .sectionBadge {
    display: none;
  }
  ```

- [ ] **Step 2: Apply `.previewSectionMode` to sections in `DynamicPreview.tsx`**

  In `DynamicPreview.tsx`, find the `<section>` element inside the `runtimeSchema.sections.map(...)` call:

  ```tsx
  <section
    key={section.key}
    data-preview-section={section.key}
    className={`${styles.section} ${isActive ? styles.sectionActive : ""}`}
  >
  ```

  Replace with:

  ```tsx
  <section
    key={section.key}
    data-preview-section={section.key}
    className={`${styles.section} ${styles.previewSectionMode} ${isActive ? styles.sectionActive : ""}`}
  >
  ```

- [ ] **Step 3: Typecheck**

  ```bash
  cd frontend/apps/web && npx tsc --noEmit 2>&1 | head -40
  ```

  Expected: zero errors.

- [ ] **Step 4: Verify in browser**

  Rebuild: `cd frontend/apps/web && npm run build`

  Open the content-builder. In the "Ao vivo" panel:
  - Section headers should now show a colored left border bar with no rounded card or "PREVIEW" badge.
  - Scalar field values (OBJETIVO, ESCOPO, etc.) should show as plain text below the label — no border box.
  - Table and repeat fields should still show their bordered shells.
  - Type something in OBJETIVO → value should appear as plain text in the preview, no box around it.

- [ ] **Step 5: Commit**

  ```bash
  git add \
    frontend/apps/web/src/features/documents/runtime/DynamicEditor.module.css \
    frontend/apps/web/src/features/documents/runtime/DynamicPreview.tsx
  git commit -m "feat(preview): document-like section headers and borderless field values"
  ```

---

## Task 5: Verify TipTap rich field end-to-end

**Files:**
- No changes planned — verify first, fix only if broken.

- [ ] **Step 1: Add an etapa item and inspect the Descrição field**

  In the content-builder, click section 3 "Processo" in the left sidebar. Scroll to the ETAPAS field. Click "Adicionar item". A new repeat item should expand.

  Inside the new item, find the **Descrição** field. It should render a TipTap editor with a toolbar row (B, I, S, Paragrafo, H1, H2, H3, Lista, Numerada, Citar, Tabela, Imagem, Undo, Redo, color picker).

  **If the editor area is a blank white box with no toolbar:**
  The `useEditor` hook may be returning null or the TipTap extension CSS is missing. In `RichField.tsx`, check that `EditorContent` is receiving a non-null `editor` prop. If editor is null on mount (expected — TipTap is async), the toolbar shows disabled buttons but the area should not be completely blank. If it is blank, add a null guard:

  In `RichField.tsx` inside `RichEditor`, after the `useEditor` call:

  ```tsx
  if (editor === null) {
    return (
      <div className={styles.richRoot}>
        <div className={editorStyles.fieldLabel}>
          <span>{label}</span>
          {field.required && <span className={editorStyles.requiredMark}>*</span>}
        </div>
        <div className={styles.editorShell}>
          <div className={styles.editorBody} style={{ minHeight: "7.5rem" }} />
        </div>
      </div>
    );
  }
  ```

- [ ] **Step 2: Type in the Descrição field and verify live preview updates**

  Click into the Descrição editor area and type: `Teste do campo rich`.

  Look at the right panel ("Ao vivo"). In section 3 Processo, the etapa item should show the typed text.

  **If the preview does not update:**
  The `onChange` chain through `RepeatField` may be broken. Trace it:
  1. `RichField.onUpdate` → calls `onChange?.(tiptapEditor.getHTML())`
  2. `RepeatField.renderField` → calls the `onChange` callback that updates the item in `items`
  3. `DynamicEditor.renderRuntimeField` → passes `onChange` to `RepeatField`
  4. `DynamicEditor.onChange` → dispatches `set_draft` in `ContentBuilderView`

  Check that `RepeatField` passes `onChange` to nested `renderField` calls (lines 52–59 in `RepeatField.tsx`). It currently does this correctly — but if it's not working, add a `console.log` at step 1 to confirm `onChange` fires.

- [ ] **Step 3: If any fixes were made — typecheck and commit**

  ```bash
  cd frontend/apps/web && npx tsc --noEmit 2>&1 | head -40
  git add frontend/apps/web/src/features/documents/runtime/fields/RichField.tsx
  git commit -m "fix(rich-field): guard null editor during TipTap async init"
  ```

  If no fixes were needed, no commit required.

---

## Self-Review Checklist

- [x] **Spec coverage:** Export fix (Fix 1 ✓), Preview header redesign (Fix 2 ✓), Preview field rendering (Fix 2 ✓), TipTap verification (Fix 3 ✓)
- [x] **No placeholders:** All steps contain exact code
- [x] **Type consistency:** `documentStatus: string` added and threaded consistently across Tasks 2–3; `styles.previewSectionMode` defined in Task 4 Step 1 and used in Task 4 Step 2
- [x] **Prop chain complete:** ContentBuilderView → PreviewPanel → DocumentPreviewRenderer → DynamicPreview → PreviewDocumentPage
