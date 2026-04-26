# Plan: Eigenpal — OutlinePlugin Port + Metadata Badges

**Date:** 2026-04-24
**Feature:** Port OutlinePlugin (T7) from eigenpal spike into MetalDocs `packages/editor-ui`; add document code + status badges to the editor title bar.

**Scope basis:** Spike RESULTS.md final decision — keep OutlinePlugin + metadata fields in existing toolbar; drop T5/T6/T8/T9.

**Why narrow:** `@eigenpal/docx-js-editor` v0.0.34 is already installed. `MetalDocsEditor.tsx` already wraps eigenpal with `PluginHost` + `templatePlugin` + `externalPlugins` prop. Scope = add one plugin + render existing `StateBadge` inside the page's existing `renderTitleBarRight` callback. No eigenpal version change needed (subject to P0 API verify).

---

## File Map

```
packages/editor-ui/
  src/
    plugins/
      OutlinePlugin.tsx          ← NEW (Codex — P1)
      mergefieldPlugin.ts        (unchanged)
      sidebarModelBridge.ts      (unchanged)
    MetalDocsEditor.tsx          ← EDIT (Sonnet — P2)  ← no P3 edit
    index.ts                     ← EDIT (Haiku — P2)
    types.ts                     ← UNCHANGED (no new props needed)

frontend/apps/web/src/features/documents/v2/
  DocumentEditorPage.tsx         ← EDIT (Sonnet — P3)  ← chips inside existing renderTitleBarRight
```

---

## Phase P0 — Verify eigenpal v0.0.34 API surface (Haiku)

**Why:** Spike RESULTS.md was captured on `@eigenpal/docx-js-editor` v0.0.35; MetalDocs has v0.0.34 installed (`package-lock.json:511,7146`). The OutlinePlugin uses `EditorPlugin`, `PluginPanelProps`, `panelConfig`, `onStateChange(view)`, `view.state.doc.descendants`, and `props.renderedDomContext.getCoordinatesForPosition` / `pagesContainer`. If any of these are missing or differ in v0.0.34, P1 will fail.

**Task P0.1:**

1. `cd packages/editor-ui && rtk tsc --noEmit` baseline (no edits) — note current error count.
2. Open `node_modules/@eigenpal/docx-js-editor/dist/index.d.ts` (or wherever types ship). Grep for: `EditorPlugin`, `PluginPanelProps`, `panelConfig`, `renderedDomContext`, `getCoordinatesForPosition`, `pagesContainer`, `scrollToPosition`.
3. Confirm all 7 symbols exported and shape matches the spike usage in `eigenpal-spike/src/plugins/outline/index.tsx`.

**Decision tree:**
- All match → proceed to P1 unchanged.
- Any missing/different → bump dep to v0.0.35 (`pnpm add @eigenpal/docx-js-editor@0.0.35` in `packages/editor-ui`) and re-run baseline. If bump introduces unrelated TSC errors, stop and escalate.

**Verify:** Symbol checklist passes OR dep bumped and baseline still clean.

---

## Phase P1 — Create OutlinePlugin (Codex)

### Task P1.1 — Write `packages/editor-ui/src/plugins/OutlinePlugin.tsx`

**What:** Port `OutlinePlugin` from spike `eigenpal-spike/src/plugins/outline/index.tsx`.

**Fix:** The spike uses a module-level `let cachedDoc: any | null = null`, which breaks if two editor instances exist simultaneously (second editor sees stale headings). Fix by exporting a factory function and using `useMemo` per editor instance.

**Implementation:**

```tsx
import { useEffect, useState } from 'react';
import type { EditorPlugin, PluginPanelProps } from '@eigenpal/docx-js-editor';

type OutlineHeading = {
  id: string;
  level: number;
  text: string;
  pos: number;
};

type OutlineState = {
  headings: OutlineHeading[];
  activeId: string | null;
};

function OutlinePanel(props: PluginPanelProps<OutlineState>) {
  const [activeId, setActiveId] = useState<string | null>(null);
  const headings = props.pluginState?.headings ?? [];

  useEffect(() => {
    const ctx = props.renderedDomContext;
    if (!ctx) {
      setActiveId((prev) => (prev === null ? prev : null));
      return;
    }

    let raf = 0;
    const tick = () => {
      raf = 0;
      if (headings.length === 0) {
        setActiveId((prev) => (prev === null ? prev : null));
      } else {
        const targetY = ctx.pagesContainer.scrollTop + 80;
        let bestId: string | null = null;
        let bestDistance = Number.POSITIVE_INFINITY;
        for (const heading of headings) {
          const coords = ctx.getCoordinatesForPosition(heading.pos);
          if (!coords) continue;
          const distance = Math.abs(coords.y - targetY);
          if (distance < bestDistance) {
            bestDistance = distance;
            bestId = heading.id;
          }
        }
        setActiveId((prev) => (prev === bestId ? prev : bestId));
      }
    };

    const scheduleTick = () => {
      if (raf !== 0) return;
      raf = window.requestAnimationFrame(tick);
    };

    ctx.pagesContainer.addEventListener('scroll', scheduleTick);
    scheduleTick();

    return () => {
      ctx.pagesContainer.removeEventListener('scroll', scheduleTick);
      if (raf !== 0) window.cancelAnimationFrame(raf);
    };
  }, [props.renderedDomContext, headings]);

  return (
    <div className="outline-panel">
      {headings.length === 0 ? (
        <div style={{ padding: 8, color: '#6b7280', fontSize: 12 }}>(no headings)</div>
      ) : (
        headings.map((heading) => (
          <button
            key={heading.id}
            type="button"
            className={`outline-item${activeId === heading.id ? ' outline-item-active' : ''}`}
            style={{ paddingLeft: `${Math.max(0, heading.level - 1) * 12}px` }}
            onClick={() => props.scrollToPosition(heading.pos)}
          >
            {heading.text}
          </button>
        ))
      )}
    </div>
  );
}

export function createOutlinePlugin(): EditorPlugin<OutlineState> {
  let cachedDoc: unknown = null;

  return {
    id: 'outline',
    name: 'Outline',
    panelConfig: {
      position: 'left',
      defaultSize: 260,
      minSize: 220,
      maxSize: 400,
      resizable: true,
      collapsible: true,
    },
    initialize: () => ({ headings: [], activeId: null }),
    onStateChange(view) {
      if (view.state.doc === cachedDoc) return;
      cachedDoc = view.state.doc;

      const headings: OutlineHeading[] = [];

      type ParagraphLike = {
        type?: { name?: string };
        attrs?: { outlineLevel?: unknown; styleId?: unknown };
        textContent?: string;
      };

      view.state.doc.descendants((rawNode: unknown, pos: number) => {
        const node = rawNode as ParagraphLike;
        if (node.type?.name !== 'paragraph') return;
        const outline = node.attrs?.outlineLevel;
        let level: number | null = null;
        if (outline != null) {
          level = typeof outline === 'number' ? outline + 1 : 1;
        } else {
          const styleId = String(node.attrs?.styleId ?? '');
          const styleMatch = styleId.match(/^(T[íi]tulo|Heading)(\d+)$/i);
          if (styleMatch) {
            const parsedLevel = Number.parseInt(styleMatch[2], 10);
            if (Number.isFinite(parsedLevel)) {
              level = Math.max(1, Math.min(6, parsedLevel));
            }
          }
        }
        if (level == null) return;
        const text = (node.textContent ?? '').trim() || 'Untitled heading';
        headings.push({ id: String(headings.length), level, text, pos });
      });

      return { headings, activeId: null };
    },
    Panel: OutlinePanel,
    styles: `
      .outline-panel { height: 100%; overflow-y: auto; box-sizing: border-box; padding: 8px; }
      .outline-item { display: block; width: 100%; border: 0; background: transparent; text-align: left; font-size: 13px; line-height: 1.4; padding-top: 6px; padding-bottom: 6px; border-radius: 6px; cursor: pointer; color: #1f2937; }
      .outline-item:hover { background: #f3f4f6; }
      .outline-item-active { background: #e8f0fe; color: #1d4ed8; font-weight: 500; }
    `,
  };
}
```

**Verify:** `cd packages/editor-ui && rtk tsc --noEmit` — 0 errors.

---

## Phase P2 — Barrel Export + Wire (Haiku + Sonnet)

### Task P2.1 — Export from barrel (Haiku)

**File:** `packages/editor-ui/src/index.ts`

**Change:** Add one line:
```ts
export { createOutlinePlugin } from './plugins/OutlinePlugin';
```

**Verify:** `rtk tsc --noEmit` — 0 errors.

---

### Task P2.2 — Wire OutlinePlugin into MetalDocsEditor (Sonnet)

**File:** `packages/editor-ui/src/MetalDocsEditor.tsx`

**Change:** Use `useMemo` to create one plugin instance per editor mount. Add to the plugins array when mode is not readonly.

At the top, add import:
```ts
import { useMemo } from 'react';
import { createOutlinePlugin } from './plugins/OutlinePlugin';
```

Replace existing `const plugins: ReactEditorPlugin[] = [...]` block with:
```ts
const outlinePlugin = useMemo(() => createOutlinePlugin(), []);

const plugins: ReactEditorPlugin[] = [
  templatePlugin,
  ...(props.mode !== 'readonly' ? [outlinePlugin] : []),
  ...(props.sidebarModel ? [buildSidebarModelPlugin(props.sidebarModel)] : []),
  ...(props.externalPlugins ?? []),
];
```

Note: `useMemo` with `[]` dep ensures same plugin instance across re-renders (no stale closure issue). New mount → new `cachedDoc` closure.

**Verify:** `rtk tsc --noEmit` — 0 errors. Start dev server, open a document, confirm left panel shows outline headings.

---

## Phase P3 — Metadata Chips in Title Bar (Sonnet)

**Decision:** Render chips inside the existing `renderTitleBarRight` callback in `DocumentEditorPage.tsx` (lines 202-217). No new props on `MetalDocsEditor`. No `types.ts` edit. No `wrappedTitleBarRight` wrapper. Reuse the existing `StateBadge` component (`frontend/apps/web/src/features/approval/components/StateBadge.tsx`) which already maps lowercase English status keys (`draft`, `under_review`, `approved`, `scheduled`, `published`, `superseded`, `rejected`, `obsolete`, `cancelled`) to Portuguese labels + colors.

### Task P3.1 — Render code + status chips in DocumentEditorPage (Sonnet)

**File:** `frontend/apps/web/src/features/documents/v2/DocumentEditorPage.tsx`

**Context:**
- `doc` is fetched via `getDocument(documentID)`. Has `doc.Code`/`doc.code` and `doc.Status`/`doc.status`.
- `docStatus` already computed at line 175: `const docStatus = doc?.Status ?? doc?.status ?? ''`.
- `StateBadge` typed as `state: ApprovalState` — verify `docStatus` value falls in that union before passing. If not guaranteed, narrow with a guard.
- Existing callback (line 202-217) currently renders `<button>Checkpoints</button>`, `<ExportMenuButton />`, `<button>Finalize</button>`.

**Change:** Inject code chip + `<StateBadge />` at the start of the fragment, before Checkpoints button:

```tsx
import { StateBadge } from '../../approval/components/StateBadge';
import type { ApprovalState } from '../../approval/api/approvalTypes';

// inside component, before return:
const docCode = doc?.Code ?? doc?.code ?? '';
const VALID_STATES: ApprovalState[] = ['draft','under_review','approved','scheduled','published','superseded','rejected','obsolete','cancelled'];
const badgeState: ApprovalState | null =
  (VALID_STATES as readonly string[]).includes(docStatus) ? (docStatus as ApprovalState) : null;

// inside renderTitleBarRight:
renderTitleBarRight={() => (
  <>
    {docCode && (
      <span style={{
        fontSize: 11, fontWeight: 600, padding: '2px 6px',
        borderRadius: 4, background: '#f1f5f9', color: '#475569',
        border: '1px solid #e2e8f0', marginRight: 6,
      }}>
        {docCode}
      </span>
    )}
    {badgeState && <StateBadge state={badgeState} size="sm" />}
    <button type="button" onClick={() => setCheckpointsOpen(true)}>Checkpoints</button>
    <ExportMenuButton
      documentID={documentID}
      canExport={sessionPhase === 'writer' || sessionPhase === 'readonly'}
    />
    <button
      type="button"
      onClick={() => void handleFinalize()}
      disabled={session.state.phase !== 'writer' || docStatus !== 'draft'}
    >
      Finalize
    </button>
  </>
)}
```

**Why a guard, not a cast:** `docStatus` is typed as `string` from `doc?.Status ?? doc?.status ?? ''`. Casting blind to `ApprovalState` loses runtime safety if backend ships a value the badge map doesn't cover (`STATE_CONFIG[cfg]` would be undefined → crash).

**Verify:**
1. `rtk tsc --noEmit` — 0 errors in `frontend/apps/web`.
2. Open document in browser → title bar shows `DOC-XXX` code chip + Portuguese status badge (e.g. "Rascunho" gray for `draft`) to the left of Checkpoints/Export/Finalize buttons.
3. Open a doc with an unknown/empty status → only code chip renders, no crash.

---

## Phase P4 — Opus Review

**Agent:** `nexus:code-reviewer`, model=`opus`

**Checklist:**
- P0: eigenpal API symbols verified or dep bumped cleanly to v0.0.35
- P1: `OutlinePlugin.tsx` — `cachedDoc` closure is per-instance (not module-level), no `any` leaks (descendants uses `unknown` + `ParagraphLike` cast), styles scoped, TSC clean
- P2: `MetalDocsEditor.tsx` — `useMemo(..., [])` creates stable plugin, OutlinePlugin absent in readonly mode, no infinite re-render risk
- P3: `StateBadge` reused (no inline status colors), runtime guard rejects unknown statuses cleanly, code chip renders when `doc.Code` empty handled gracefully
- No regressions to templatePlugin, sidebarModelBridge, or autosave

---

## Execution Order

| Phase | Task | Agent | Time est. |
|---|---|---|---|
| P0 | eigenpal v0.0.34 API verify | Haiku | 5 min |
| P1 | OutlinePlugin.tsx | Codex (medium) | 15 min |
| P2 | index.ts barrel | Haiku | 2 min |
| P2 | MetalDocsEditor.tsx wire | Sonnet | 5 min |
| P3 | DocumentEditorPage.tsx chips (StateBadge + code) | Sonnet | 5 min |
| P4 | Opus review | Opus | 10 min |

**Total estimate: ~42 min execution**

---

## Success Criteria

1. `rtk tsc --noEmit` clean in both `packages/editor-ui` and `frontend/apps/web`
2. Open any DOCX with headings → left panel shows outline, click → scrolls
3. OutlinePlugin absent in readonly mode (no left panel)
4. Title bar shows `DOC-XXX` code chip + Portuguese `StateBadge` (e.g. "Rascunho" for draft)
5. Document with unknown/empty status → no crash, badge omitted
6. Existing features unchanged: templatePlugin sidebar, comments, autosave, checkpoints, export
