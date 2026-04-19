# PDF-Only Preview Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the "Ao Vivo" React rendering stack with the existing PDF preview pipeline — always show the Carbone-generated PDF, delete all live preview components.

**Architecture:** Remove the "Ao Vivo" / "PDF" tab distinction. PreviewPanel always renders PdfPreview. After each auto-save (3s debounce, already implemented), the backend returns a signed pdfUrl which triggers a re-render. All DynamicPreview/PreviewTemplate React components are deleted along with their CSS. PdfPreview width becomes responsive to the panel width via ResizeObserver.

**Tech Stack:** React 18, TypeScript, react-pdf (existing PdfPreview.tsx), CSS.

---

## File map

- Modify: `frontend/apps/web/src/components/content-builder/preview/PreviewPanel.tsx` — rewrite to PDF-only
- Modify: `frontend/apps/web/src/components/content-builder/ContentBuilderView.tsx` — simplify preview props
- Modify: `frontend/apps/web/src/styles.css` — delete dead preview CSS, add overlay styles
- Modify: `frontend/apps/web/src/features/documents/runtime/DynamicEditor.module.css` — delete preview-only CSS
- Unchanged: `frontend/apps/web/src/components/create/widgets/PdfPreview.tsx` — PreviewPanel owns width via ResizeObserver and passes explicit `width` prop; PdfPreview is NOT modified
- Delete: 15 files (listed in Task 3)

---

### Task 1: Rewrite PreviewPanel to PDF-only

**Files:**
- Modify: `frontend/apps/web/src/components/content-builder/preview/PreviewPanel.tsx`

- [ ] **Step 1: Replace the entire PreviewPanel component**

Replace the full contents of `PreviewPanel.tsx` with:

```tsx
import { useEffect, useRef, useState } from "react";
import { PdfPreview } from "../../create/widgets/PdfPreview";

type PreviewPanelProps = {
  pdfUrl: string;
  isDirty: boolean;
  isBusy: boolean;
  collapsed: boolean;
  onToggleCollapse: (collapsed: boolean) => void;
};

export function PreviewPanel({
  pdfUrl,
  isDirty,
  isBusy,
  collapsed,
  onToggleCollapse,
}: PreviewPanelProps) {
  const bodyRef = useRef<HTMLDivElement | null>(null);
  const [bodyWidth, setBodyWidth] = useState(0);

  useEffect(() => {
    const el = bodyRef.current;
    if (!el) return;
    const observer = new ResizeObserver((entries) => {
      for (const entry of entries) {
        setBodyWidth(Math.floor(entry.contentRect.width));
      }
    });
    observer.observe(el);
    return () => observer.disconnect();
  }, []);

  if (collapsed) {
    return (
      <aside className="content-builder-preview is-collapsed">
        <button
          type="button"
          className="content-builder-preview-toggle is-collapsed"
          onClick={() => onToggleCollapse(false)}
          aria-label="Expandir preview"
        >
          <svg width="14" height="14" viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M13 5l-6 5 6 5" strokeLinecap="round" strokeLinejoin="round" />
          </svg>
        </button>
      </aside>
    );
  }

  return (
    <aside className="content-builder-preview">
      <div className="content-builder-preview-inner">
        <div className="preview-panel-header">
          <button
            type="button"
            className="content-builder-preview-toggle"
            onClick={() => onToggleCollapse(true)}
            aria-label="Recolher preview"
          >
            <svg width="14" height="14" viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M7 5l6 5-6 5" strokeLinecap="round" strokeLinejoin="round" />
            </svg>
          </button>
          <span className="preview-panel-title-text">Preview</span>
        </div>

        <div className="preview-panel-body" ref={bodyRef}>
          {!pdfUrl ? (
            <div className="content-builder-preview-empty">
              <div className="content-builder-preview-empty-icon" aria-hidden="true">
                <svg width="18" height="18" viewBox="0 0 18 18" fill="none" stroke="currentColor" strokeWidth="1.4">
                  <path d="M4 2h7l3 3v11H4V2z" strokeLinejoin="round" />
                  <path d="M11 2v3h3" strokeLinejoin="round" />
                  <path d="M6 9h6M6 12h4" strokeLinecap="round" />
                </svg>
              </div>
              <strong>Nenhum preview disponivel</strong>
              <span>Salve o documento para gerar o preview.</span>
            </div>
          ) : (
            <div className="preview-panel-pdf-wrapper">
              {(isDirty || isBusy) && (
                <div className={`preview-panel-overlay ${isBusy ? "is-saving" : "is-stale"}`}>
                  {isBusy ? (
                    <>
                      <span className="preview-panel-spinner" />
                      <span>Atualizando preview...</span>
                    </>
                  ) : (
                    <span className="preview-panel-stale-badge">Alteracoes nao salvas</span>
                  )}
                </div>
              )}
              {bodyWidth > 0 && (
                <PdfPreview url={pdfUrl} width={bodyWidth} />
              )}
            </div>
          )}
        </div>
      </div>
    </aside>
  );
}
```

- [ ] **Step 2: Build to check for TypeScript errors**

```bash
cd frontend/apps/web && npx tsc --noEmit 2>&1 | head -30
```

Expected: Errors in `ContentBuilderView.tsx` because it still passes old props to PreviewPanel. This is expected — Task 2 fixes it.

- [ ] **Step 3: Commit**

```bash
git add frontend/apps/web/src/components/content-builder/preview/PreviewPanel.tsx
git commit -m "feat(preview): rewrite PreviewPanel to always show PDF

Remove Ao Vivo/PDF tabs. Always render PdfPreview with stale badge
and saving overlay. Width is responsive via ResizeObserver."
```

---

### Task 2: Simplify ContentBuilderView props

**Files:**
- Modify: `frontend/apps/web/src/components/content-builder/ContentBuilderView.tsx`

- [ ] **Step 1: Update the PreviewPanel usage**

Find the `previewPane` block (lines ~441-456):

```tsx
  const previewPane = (
    <PreviewPanel
      schema={schema}
      contentDraft={contentDraft}
      pdfUrl={pdfUrl}
      profileCode={profileCode}
      documentCode={documentCode}
      documentTitle={documentTitle}
      documentStatus={props.document?.status ?? "DRAFT"}
      version={version}
      activeSectionKey={activeSectionKey}
      isDirty={status === "dirty"}
      collapsed={previewCollapsed}
      onToggleCollapse={(collapsed) => dispatch({ type: "set_preview", payload: { collapsed } })}
    />
  );
```

Replace with:

```tsx
  const previewPane = (
    <PreviewPanel
      pdfUrl={pdfUrl}
      isDirty={status === "dirty"}
      isBusy={autoSave.isSaving || status === "saving" || status === "rendering"}
      collapsed={previewCollapsed}
      onToggleCollapse={(collapsed) => dispatch({ type: "set_preview", payload: { collapsed } })}
    />
  );
```

Note: `isBusy` combines auto-save, manual save ("Salvar rascunho"), and PDF render ("Gerar PDF") states so the overlay shows during all save paths, not just auto-save.

- [ ] **Step 2: Remove the unused import**

Find and remove the import of `normalizeDocumentTypeSchema` if it's only used for the preview schema threading. Also remove the `DocumentPreviewRenderer` import if present.

Check which imports are now unused:

```bash
cd frontend/apps/web && npx tsc --noEmit 2>&1 | grep "is declared but"
```

Remove any that show up.

- [ ] **Step 3: Build to verify**

```bash
cd frontend/apps/web && npm run build 2>&1 | tail -5
```

Expected: `✓ built in X.XXs` with no errors.

- [ ] **Step 4: Commit**

```bash
git add frontend/apps/web/src/components/content-builder/ContentBuilderView.tsx
git commit -m "refactor(preview): simplify ContentBuilderView preview props

Remove schema, contentDraft, profileCode, documentCode, documentTitle,
documentStatus, version, activeSectionKey props. Pass only pdfUrl,
isDirty, isSaving, collapsed, onToggleCollapse."
```

---

### Task 3: Delete dead preview files

**Files to delete:**
- `frontend/apps/web/src/features/documents/runtime/DynamicPreview.tsx`
- `frontend/apps/web/src/components/content-builder/preview/DocumentPreviewRenderer.tsx`
- `frontend/apps/web/src/components/content-builder/preview/PreviewDocumentPage.tsx`
- `frontend/apps/web/src/components/content-builder/preview/PreviewSectionBlock.tsx`
- `frontend/apps/web/src/components/content-builder/preview/PreviewFieldRenderer.tsx`
- `frontend/apps/web/src/components/content-builder/preview/PreviewTextField.tsx`
- `frontend/apps/web/src/components/content-builder/preview/PreviewArrayField.tsx`
- `frontend/apps/web/src/components/content-builder/preview/PreviewChecklistField.tsx`
- `frontend/apps/web/src/components/content-builder/preview/PreviewTableField.tsx`
- `frontend/apps/web/src/components/content-builder/preview/templates/PreviewTemplatePO.tsx`
- `frontend/apps/web/src/components/content-builder/preview/templates/PreviewTemplateIT.tsx`
- `frontend/apps/web/src/components/content-builder/preview/templates/PreviewTemplateRG.tsx`
- `frontend/apps/web/src/components/content-builder/preview/templates/PreviewTemplateFM.tsx`
- `frontend/apps/web/src/components/content-builder/preview/templates/PreviewTemplateGeneric.tsx`
- `frontend/apps/web/src/components/content-builder/preview/templates/templateRegistry.ts`

- [ ] **Step 1: Delete all files**

```bash
cd frontend/apps/web/src
rm features/documents/runtime/DynamicPreview.tsx
rm components/content-builder/preview/DocumentPreviewRenderer.tsx
rm components/content-builder/preview/PreviewDocumentPage.tsx
rm components/content-builder/preview/PreviewSectionBlock.tsx
rm components/content-builder/preview/PreviewFieldRenderer.tsx
rm components/content-builder/preview/PreviewTextField.tsx
rm components/content-builder/preview/PreviewArrayField.tsx
rm components/content-builder/preview/PreviewChecklistField.tsx
rm components/content-builder/preview/PreviewTableField.tsx
rm -rf components/content-builder/preview/templates/
```

- [ ] **Step 2: Search for orphaned imports**

```bash
cd frontend/apps/web/src
grep -r "DynamicPreview\|DocumentPreviewRenderer\|PreviewDocumentPage\|PreviewSectionBlock\|PreviewFieldRenderer\|PreviewTextField\|PreviewArrayField\|PreviewChecklistField\|PreviewTableField\|templateRegistry\|PreviewTemplate" --include="*.tsx" --include="*.ts" .
```

Expected: No matches. If any file still imports a deleted module, remove that import.

- [ ] **Step 3: Build to verify**

```bash
cd frontend/apps/web && npm run build 2>&1 | tail -10
```

Expected: `✓ built in X.XXs`. If there are import errors, the build output will name the offending file and the missing module — fix those imports.

- [ ] **Step 4: Commit**

```bash
cd frontend/apps/web/../../..
git add -A frontend/apps/web/src/features/documents/runtime/DynamicPreview.tsx
git add -A frontend/apps/web/src/components/content-builder/preview/DocumentPreviewRenderer.tsx
git add -A frontend/apps/web/src/components/content-builder/preview/PreviewDocumentPage.tsx
git add -A frontend/apps/web/src/components/content-builder/preview/PreviewSectionBlock.tsx
git add -A frontend/apps/web/src/components/content-builder/preview/PreviewFieldRenderer.tsx
git add -A frontend/apps/web/src/components/content-builder/preview/PreviewTextField.tsx
git add -A frontend/apps/web/src/components/content-builder/preview/PreviewArrayField.tsx
git add -A frontend/apps/web/src/components/content-builder/preview/PreviewChecklistField.tsx
git add -A frontend/apps/web/src/components/content-builder/preview/PreviewTableField.tsx
git add -A frontend/apps/web/src/components/content-builder/preview/templates/
git commit -m "chore(preview): delete 15 dead Ao Vivo rendering files

DynamicPreview, DocumentPreviewRenderer, PreviewDocumentPage,
PreviewSectionBlock, PreviewFieldRenderer, PreviewTextField,
PreviewArrayField, PreviewChecklistField, PreviewTableField,
and all 6 template files — replaced by PdfPreview."
```

---

### Task 4: Delete dead CSS

**Files:**
- Modify: `frontend/apps/web/src/styles.css`
- Modify: `frontend/apps/web/src/features/documents/runtime/DynamicEditor.module.css`

- [ ] **Step 1: Identify dead CSS selectors via search**

Run a repo-wide search for CSS class names that were only referenced by the deleted preview files:

```bash
cd frontend/apps/web/src
# Find all preview-doc and preview-document class usages remaining in .tsx/.ts files
grep -r "preview-document\|preview-doc-\|preview-panel-tab\|previewSectionMode\|previewValue\|previewEmpty" --include="*.tsx" --include="*.ts" .
```

Expected: No matches (the files that used these classes were deleted in Task 3). If any matches remain, investigate before deleting the CSS.

- [ ] **Step 2: Delete dead CSS from styles.css**

Delete these blocks from `styles.css`:

1. The entire `/* LIVE EDITOR — Document Preview (A4 page rendering) */` comment section and all rules below it up to the next major section comment. This includes `.preview-document`, `.preview-document-page`, `@container`, all `.preview-doc-*` rules, `.preview-document-content`, `.preview-document-footer`.

2. Remove `container-type: inline-size` from `.preview-panel-body`.

3. Remove the `.preview-panel-tabs` and `.preview-panel-tab` CSS rules (the tab switcher is gone).

- [ ] **Step 2: Add new CSS for stale badge, saving overlay, and spinner**

Add these new rules in `styles.css` after the `.preview-panel-body` block:

```css
.preview-panel-title-text {
  font-size: 0.82rem;
  font-weight: 700;
  color: var(--vinho-d);
  text-transform: uppercase;
  letter-spacing: 0.06em;
}

.preview-panel-pdf-wrapper {
  position: relative;
  min-height: 200px;
}

.preview-panel-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 0.5rem;
  z-index: 2;
  pointer-events: none;
}

.preview-panel-overlay.is-saving {
  background: rgba(255, 255, 255, 0.8);
  color: var(--vinho-d);
  font-size: 0.82rem;
  font-weight: 600;
}

.preview-panel-overlay.is-stale {
  align-items: flex-start;
  justify-content: flex-end;
  padding: 8px 10px;
}

.preview-panel-stale-badge {
  background: rgba(61, 28, 30, 0.75);
  color: #fff;
  font-size: 0.7rem;
  font-weight: 600;
  padding: 4px 10px;
  border-radius: 999px;
}

.preview-panel-spinner {
  width: 16px;
  height: 16px;
  border: 2px solid color-mix(in srgb, var(--vinho) 25%, transparent);
  border-top-color: var(--vinho);
  border-radius: 50%;
  animation: preview-spin 0.6s linear infinite;
}

@keyframes preview-spin {
  to { transform: rotate(360deg); }
}
```

- [ ] **Step 3: Delete preview-only CSS from DynamicEditor.module.css**

In `DynamicEditor.module.css`, delete these blocks:

- `.previewSectionMode.section` and all its rules
- `.previewSectionMode .sectionHeader`
- `.previewSectionMode .sectionBadge`
- `.previewValue`
- `.previewEmpty`

- [ ] **Step 4: Build to verify**

```bash
cd frontend/apps/web && npm run build 2>&1 | tail -5
```

Expected: `✓ built in X.XXs` with no errors.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/styles.css frontend/apps/web/src/features/documents/runtime/DynamicEditor.module.css
git commit -m "style(preview): delete dead Ao Vivo CSS, add PDF overlay styles

Remove .preview-document*, .preview-doc-*, .preview-panel-tab*,
container-type, previewSectionMode, previewValue, previewEmpty.
Add stale badge, saving overlay, and spinner for PDF preview."
```

---

### Task 5: Verify end-to-end

- [ ] **Step 1: Final build**

```bash
cd frontend/apps/web && npm run build 2>&1 | tail -5
```

Expected: `✓ built in X.XXs` with no errors.

- [ ] **Step 2: Verify no orphaned references**

```bash
cd frontend/apps/web/src
# Check no deleted component names remain in any source file
grep -r "DynamicPreview\|PreviewDocumentPage\|PreviewSectionBlock\|PreviewFieldRenderer\|previewSectionMode\|previewValue\|previewEmpty" --include="*.tsx" --include="*.ts" --include="*.css" .
```

Expected: No matches.

- [ ] **Step 3: Open the content builder in the browser**

Navigate to `http://127.0.0.1:4173/#/content-builder` and open a document.

Verify:
- The preview panel shows the PDF (no "Ao Vivo" / "PDF" tabs)
- The header says "Preview" with a collapse button
- If the document has been saved before, the PDF renders
- If not, the empty state shows "Nenhum preview disponivel"
- Type in a field — after 3 seconds (auto-save debounce), the preview should refresh with the new PDF
- The stale badge appears while typing, the saving overlay appears during the save, then the fresh PDF loads
- Drag the split handle — the PDF width adjusts to fill the panel
- Collapse/expand the preview panel works

- [ ] **Step 4: Commit any fixes found during verification**

If any issues are found, fix and commit them.
