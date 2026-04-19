# MetalDocs docx-editor — Word-parity Integration Plan

**Scope:** Make our MetalDocs editor render and behave identically to the reference `@eigenpal/docx-js-editor` live demo at `https://www.docx-editor.dev/editor`. No forking, no re-skinning, no custom toolbar — use the library as shipped. Only integrate it into our session/autosave/checkpoint/export back-end.

**Why this plan exists:** current `MetalDocsEditor` wrapper renders a naked ProseMirror instance — no toolbar, no ruler, no page cards — because the library's own stylesheet is never imported and session-lost state forces `readonly`, which hides the built-in toolbar. We also pass only 4 of the ~40 available props, so we're using <10% of the library's UI surface.

**Out of scope (defer to separate plans):**
- pt-BR localization contribution (library ships en/de/pl only).
- Yjs / realtime collab wiring (`externalPlugins: [ySyncPlugin]` path).
- Mobile responsive tuning.
- Track-changes approval workflow UI (library supports `mode='suggesting'` — we'll enable the mode, but the approval/review UX is its own project).

**Reference:** `@eigenpal/docx-js-editor` 0.0.34 — the canonical API surface is the package's public exports (`import type { DocxEditorProps, DocxEditorRef, Comment, EditorPlugin, Translations } from '@eigenpal/docx-js-editor'`). Do **not** reference hashed internal dist files (`dist/react-*.d.ts`) from code or tests — those paths change per build.

## Library-as-is boundary (hard rule)

User directive: use `@eigenpal/docx-js-editor` exactly as shipped. No custom toolbar, no custom page chrome, no re-skinning. The only sanctioned custom UI is whatever we inject through the library's own extension slots:
- `renderTitleBarRight` — used **only** for MetalDocs-specific actions with no library equivalent: **Finalize**, **Export menu**, **Checkpoints toggle**.
- `toolbarExtra` — unused in this plan.
- Plugins via `PluginHost` — `templatePlugin` for mergefield chips and one thin bridge plugin that maps our existing `computeSidebarModel` into `ReactSidebarItem`s.

Anything else the user currently sees (banners, status badges, custom header, floating action buttons outside the editor) is deleted. Autosave status, session-lost notifications, and other transient messages are surfaced through `sonner` toasts (already bundled with the library — `package.json:106`), not custom DOM.

## Success criteria

1. Editor renders with the same visible chrome as the live demo: File/Format/Insert/Help menu bar, full rich-text toolbar, ruler, page cards, zoom control, outline toggle, print button, document title inline in the title bar.
2. Mergefield tokens (`{client_name}`, `{total_amount}`) render as highlighted chips, not plain text.
3. Typing → autosave round-trip (presign → PUT → commit) fires every ~16.5 s without session-lost bounces during a continuous editing session.
4. Finalize / checkpoint / export actions are reachable from the editor's own title-bar-right area (not a separate header above the editor).
5. Comments panel only appears when the document has comments (library default behavior). Adding a comment via the library UI persists to a new `document_comments` table and re-renders on reload.
6. Deleting the current custom `header` / `status` / `banner` JSX from `DocumentEditorPage.tsx` does not regress any finalize/checkpoint flow — those actions still work, now surfaced through library-native slots.
7. Existing vitest suites for `useDocumentAutosave`, `useDocumentSession`, and `MetalDocsEditor` pass; new tests added for comments sync cover add/resolve/delete/reply round-trip.

## File map

```
packages/editor-ui/
  src/
    MetalDocsEditor.tsx           REWRITE — kitchen-sink prop pass-through
    types.ts                      EDIT — expand MetalDocsEditorProps surface
    index.ts                      EDIT — re-export library types we expose
    overrides.css                 DELETE (superseded by library styles.css)
    plugins/
      mergefieldPlugin.ts         KEEP — rewrap as eigenpal-compatible plugin
      (new) sidebarModelBridge.ts NEW — bridge computeSidebarModel → ReactSidebarItem

frontend/apps/web/src/features/documents/v2/
  DocumentEditorPage.tsx          REWRITE — delete custom header, use renderTitleBarRight
  hooks/
    useDocumentSession.ts         EDIT — debounce acquire, stop StrictMode re-acquire leak
    useDocumentAutosave.ts        EDIT — carry form snapshot via onSelectionChange (not every change)
    useDocumentComments.ts        NEW — CRUD hook backed by document_comments API
  api/
    documentsV2.ts                EDIT — add comments CRUD endpoints
  styles/
    DocumentEditorPage.module.css EDIT — page becomes a pure 100% viewport frame

internal/modules/documents_v2/
  domain/model.go                 EDIT — add Comment, CommentReply types
  repository/repository.go        EDIT — add CRUD for document_comments
  application/service.go          EDIT — wire comment CRUD
  delivery/http/handler.go        EDIT — add REST endpoints

migrations/
  0118_docx_v2_document_comments.sql   NEW  (root migrations/, current head is 0117)

docs/superpowers/plans/
  2026-04-19-docx-editor-word-parity.md  THIS FILE
```

## Phases

### Phase 0 — Diagnosis freeze (non-coding, 10 min)

Before any edit, confirm current bug state is a pure CSS+props problem and not a library-version regression.

- **Task 0.1** Hard-reload `http://localhost:4174/#/documents-v2/99883d92-36d4-41f7-9f24-693729354892` with devtools open. Clear `metaldocs_docs_v2` IndexedDB. Confirm one session/acquire (201) returns `last_ack_revision_id: 64d5cd09-6db9-4bfe-b22a-12b6b504f131`.
  - Verify: no "Session lost" banner after acquire. If banner still appears, Phase 0 fails and we investigate `useDocumentSession.ts:28` (heartbeat 409 path) before proceeding.
- **Task 0.2** In devtools, evaluate `!!document.querySelector('.layout-page')`. Expected: `true`. This confirms the library is rendering pages but they're unstyled. If `false`, library is broken — escalate.
- **Task 0.3** `grep -l "styles.css" packages/editor-ui/src` — expected empty. Confirms missing CSS import is real.

### Phase 1 — Restore native chrome (CSS + props — 30 min)

**Task 1.1** `packages/editor-ui/src/MetalDocsEditor.tsx` — replace `import './overrides.css'` with `import '@eigenpal/docx-js-editor/styles.css'`. Delete `overrides.css` file.

**Task 1.2** `packages/editor-ui/src/types.ts` — expand `MetalDocsEditorProps`:
```ts
import type { ReactNode } from 'react';
import type { Comment } from '@eigenpal/docx-js-editor';

export type EditorMode = 'template-draft' | 'document-edit' | 'readonly';

export interface MetalDocsEditorProps {
  documentId?: string;
  documentBuffer?: ArrayBuffer;
  mode: EditorMode;
  userId: string;
  author?: string;
  documentName?: string;
  documentNameEditable?: boolean;
  onDocumentNameChange?: (name: string) => void;
  comments?: Comment[];
  onCommentsChange?: (comments: Comment[]) => void;
  onCommentAdd?: (c: Comment) => void;
  onCommentResolve?: (c: Comment) => void;
  onCommentDelete?: (c: Comment) => void;
  onCommentReply?: (reply: Comment, parent: Comment) => void;
  renderTitleBarRight?: () => ReactNode;
  onAutoSave?: (buf: ArrayBuffer) => Promise<void>;
  onLockLost?: () => void;
}
```

**Task 1.3** `packages/editor-ui/src/MetalDocsEditor.tsx` — pass through every prop, translate `mode`:
```ts
const libMode = props.mode === 'readonly' ? 'viewing' : 'editing';
```
Render:
```tsx
<DocxEditor
  ref={inner}
  documentBuffer={props.documentBuffer}
  mode={libMode}
  author={props.author ?? props.userId}
  documentName={props.documentName}
  documentNameEditable={props.documentNameEditable ?? (libMode === 'editing')}
  onDocumentNameChange={props.onDocumentNameChange}
  comments={props.comments}
  onCommentsChange={props.onCommentsChange}
  onCommentAdd={props.onCommentAdd}
  onCommentResolve={props.onCommentResolve}
  onCommentDelete={props.onCommentDelete}
  onCommentReply={props.onCommentReply}
  renderTitleBarRight={props.renderTitleBarRight}
  showRuler
  showMarginGuides
  showOutlineButton
  showPrintButton
  showZoomControl
  onChange={handleChange}
/>
```
Keep `showToolbar` unset so library default (`true`) wins.

**Task 1.4** `packages/editor-ui/src/index.ts` — re-export `Comment` from lib so consumers don't import it directly:
```ts
export type { Comment } from '@eigenpal/docx-js-editor';
```

**Task 1.5** Manual verify: reload browser → confirm Word-like chrome visible. Toolbar present. Ruler present. Page cards with drop-shadow. Outline button top-left. Zoom bottom-right. No custom red header remaining (that's Phase 2).

**Task 1.6** vitest `packages/editor-ui/test/MetalDocsEditor.mount.test.tsx` — update assertions to reflect new DOM. Remove any reference to `.metaldocs-editor` custom classes we deleted. Add assertions:
- Toolbar visible when `mode='document-edit'`.
- Toolbar hidden when `mode='readonly'` (library translates `mode='viewing'` → toolbar off).
- Ruler visible.
- `renderTitleBarRight` slot renders exactly the nodes we pass.

Also add `packages/editor-ui/test/props.contract.test.tsx` — a compile-only contract test that imports `DocxEditorProps` from the package root and asserts every field on `MetalDocsEditorProps` that maps 1:1 is type-compatible. This catches API drift on future library bumps.

### Phase 2 — Move page chrome into the editor title bar (60 min)

Reference live demo carries all actions in the editor's own title bar. Our current `DocumentEditorPage` owns a separate `<header>` with document name, status, Finalize. Move all of it into `renderTitleBarRight` so the page becomes a thin frame.

**Task 2.1** `frontend/apps/web/src/features/documents/v2/DocumentEditorPage.tsx` — delete the `<header>` block (current lines 135-145). Delete the "readonly" and "lost" banners and the `error` banner. They are replaced by `sonner` toasts fired from the relevant hooks (`useDocumentSession` on phase transitions, `useDocumentAutosave` on terminal error states).

**Task 2.2** Add `<Toaster />` from `sonner` once at the app shell root (if not already present). Autosave status is no longer rendered as persistent UI — it becomes a single toast on `error`/`session_lost`/`stale`. `saved` is silent (library already surfaces save state through its own save button).

**Task 2.3** Create `renderTitleBarRight` callback passed to `MetalDocsEditor`. The returned fragment contains exactly three buttons, in this order:
```tsx
renderTitleBarRight={() => (
  <>
    <button onClick={toggleCheckpoints} aria-label="Checkpoints">Checkpoints</button>
    <ExportMenuButton documentID={documentID} />
    <button
      onClick={handleFinalize}
      disabled={session.state.phase !== 'writer' || docStatus !== 'draft'}
    >Finalize</button>
  </>
)}
```
No other custom controls. No autosave badge (library renders its own save indicator in the toolbar).

**Task 2.4** Pass `documentName={documentName}` and `onDocumentNameChange` — the callback hits `PATCH /api/v2/documents/:id` with `{ name }`. Add that endpoint if missing.

**Task 2.5** `CheckpointsPanel` — move out of the page layout. Render it inside a `<Dialog>` (Radix — already a library transitive dep) triggered by the Checkpoints button from Task 2.3. No off-canvas drawer, no split layout. This keeps the editor filling the viewport.

**Task 2.6** `ExportMenuButton` — thin wrapper around the existing `ExportMenu`, rendered as a single button with a dropdown. Same action set as today.

**Task 2.7** `styles/DocumentEditorPage.module.css` — strip nearly everything. Keep only a `.page { position:fixed; inset:0; }` rule so the library fills the viewport.

**Task 2.8** Manual verify against `https://www.docx-editor.dev/editor`: editor fills viewport, library title bar is the only chrome, our three buttons (Checkpoints, Export, Finalize) sit in the title-bar-right slot. No banners. No header above the editor.

### Phase 3 — Session hygiene (45 min)

Eliminate the "Session lost" bounces so the editor stays in writer mode across normal activity.

**Task 3.1** `useDocumentSession.ts` — guard against StrictMode double-acquire leaking heartbeat timers:
- Track acquire-in-flight via a ref. Second acquire short-circuits if the first is pending.
- `stopHeartbeat` already called from `startHeartbeat`, but add an abort check in the interval callback so a timer fired between unmount and cleanup does not set state.

**Task 3.2** Promote `heartbeat 409` recovery: instead of transitioning to `phase: 'lost'`, attempt one silent re-acquire. If re-acquire returns `writer` with a matching userID, resume. Only surface lost state after 2 failed re-acquires.

**Task 3.3** Add a `visibilitychange` listener: when tab hides for >2 min, pause heartbeat; on visible, do one re-acquire. Prevents heartbeat-storm on backgrounded tabs.

**Task 3.4** Unit test `useDocumentSession.test.tsx` — add cases for StrictMode double mount, heartbeat 409 → silent retry, tab-backgrounded pause/resume.

### Phase 4 — Mergefield chips via templatePlugin (60 min)

Library ships `templatePlugin` — wire it so `{client_name}` renders as a chip.

**Task 4.1** `packages/editor-ui/src/plugins/sidebarModelBridge.ts` — new file. Export a ReactEditorPlugin that consumes the existing `computeSidebarModel` output and renders one `ReactSidebarItem` per mergefield group. Anchor each item at the first token occurrence in PM.

**Task 4.2** `MetalDocsEditor.tsx` — wrap `<DocxEditor>` in `<PluginHost plugins={[templatePlugin, sidebarModelBridge]}>`. Import `templatePlugin`, `PluginHost`, and `EditorPlugin` from the package root (`@eigenpal/docx-js-editor`). Do not import from internal dist paths.

**Task 4.3** Pass `externalPlugins` through so a consumer can still inject more plugins.

**Task 4.4** vitest at `packages/editor-ui/test/templatePlugin.wiring.test.tsx`: render wrapper with a doc containing `{foo}` — expect the library's token class to appear in the DOM (assert against the role/attribute exposed by `templatePlugin`, not a CSS class name, so we don't couple to internal naming). If not, fall back to the library's own sample doc to confirm plugin wiring is live before attributing failure to our doc.

### Phase 5 — Controlled comments, backed by new table

#### Phase 5.0 — Comment contract (design-only, must land before any code)

Before any CRUD code is written, produce a dedicated contract document at `docs/superpowers/specs/2026-04-19-document-comments-contract.md` capturing:

- **Field mapping** — enumerate every field on the library's `Comment` type (import from `@eigenpal/docx-js-editor`) and map each to a column in `document_comments`. Call out fields we do **not** persist and why.
- **Threading model** — confirm library uses `parent` pointer vs. flat `thread_id` and mirror that exactly. Reply metadata (author, timestamps) lives on the reply row, not aggregated on the parent.
- **Resolved state** — library uses `resolved: boolean`. We persist `resolved_at timestamptz NULL`; exposed to the library as `resolved = resolved_at IS NOT NULL`.
- **Anchor model** — library anchors comments to PM ranges via `from`/`to` that it stores inside the document (as range markers in the OOXML / PM doc). Our DB row stores **only thread metadata** (author, body, resolved, replies, created/updated). PM-range anchors stay inside the `.docx` — that way autosave-induced PM position shifts don't desync us. The contract doc must state this explicitly and show the round-trip.
- **Identity** — `author_id` is TEXT (matches migrations `0114-0117` moving all docs_v2 actor columns to TEXT). All other actor-referencing columns follow the same rule.
- **Load path** — on editor mount, we fetch thread metadata from our API and feed it to `MetalDocsEditor` via the `comments` prop. The library reconciles against range markers already in the `.docx`. Orphan threads (DB row with no matching anchor in the doc) are listed in the contract's error-handling section with the chosen behavior (drop / keep as resolved / show as anchorless).
- **Contract tests** — three Go tests + three Vitest tests must be described in the contract doc. The contract doc blocks Phase 5 implementation until the tests are written and failing.

#### Phase 5.1 — Implementation (after contract is committed)

**Task 5.1.1** `migrations/0118_docx_v2_document_comments.sql` — root-level migration, sequence 0118 (current head is `0117_docx_v2_user_ids_to_text.sql`). Actor columns are TEXT.
```sql
CREATE TABLE document_comments (
  id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  tenant_id      UUID NOT NULL,
  document_id    UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
  parent_id      UUID REFERENCES document_comments(id) ON DELETE CASCADE,
  library_thread_id TEXT NOT NULL,
  author_id      TEXT NOT NULL,
  author_name    TEXT NOT NULL,
  body           TEXT NOT NULL,
  resolved_at    TIMESTAMPTZ,
  created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_document_comments_doc ON document_comments (document_id, created_at);
CREATE UNIQUE INDEX idx_document_comments_thread ON document_comments (document_id, library_thread_id);
```
No PM range columns. Anchors stay inside the `.docx` per contract.

**Task 5.1.2** Go domain types `Comment`, `CommentReply`, `CommentCreateInput`, `CommentUpdateInput`. Identifier types align with the contract.

**Task 5.1.3** Repository CRUD + service + HTTP:
- `GET /api/v2/documents/:id/comments` — list threads + replies.
- `POST /api/v2/documents/:id/comments` — create thread or reply (`parent_id` optional).
- `POST /api/v2/documents/:id/comments/:cid/resolve` | `/reopen`.
- `DELETE /api/v2/documents/:id/comments/:cid` — cascades replies.

**Task 5.1.4** `useDocumentComments.ts` — load on mount, expose `comments`, `add`, `resolve`, `delete`, `reply`. Optimistic updates. Convert API rows ↔ library `Comment` shape per contract. Pass to `MetalDocsEditor` as controlled props.

**Task 5.1.5** Contract tests land green before UI tests. Then add an E2E playwright test: open doc → add comment via library UI → reload → comment persists + highlights the same range.

### Phase 6 — Polish and delete (30 min)

**Task 6.1** Delete `packages/editor-ui/src/overrides.css` (already done in Task 1.1 — final sweep, just confirm the file is gone and no imports reference it).

**Task 6.2** Delete `userId` requirement from `MetalDocsEditorProps` if unused after `author` substitution. Update callers.

**Task 6.3** Run full vitest + `go test ./...` + playwright smoke. Commit per task, one subagent per task. Final commit runs `pnpm --filter @metaldocs/web build` to catch TS errors.

**Not in this plan:** deletion of `frontend/apps/web/src/features/documents/ck5/`. That belongs to W5 cutover (`docs/superpowers/plans/2026-04-18-docx-editor-w5-cutover.md`). Keep scopes separate.

## Execution notes

- **Agent assignment:** Codex (`gpt-5.3-codex`) handles Phases 1, 2, 4, 5 implementation tasks. General-purpose subagent handles Phase 0 diagnostics + Phase 3 session hygiene (smaller diffs, lighter reasoning). Opus reviews each task and commits.
- **TDD enforcement:** Phase 3, 4, 5 introduce new logic — each starts with a failing test. Phase 1, 2, 6 are primarily integration/deletion — visual verification via preview browser replaces unit tests where no logic branches change.
- **Verification cadence:** after every phase, manual browser check against the reference screenshot at `https://www.docx-editor.dev/editor`. Phase does not advance until visual match is confirmed.
- **Rollback point:** each phase is one commit. If Phase N destabilizes, `git revert` gets us back to the last green phase.

## Open risks

- `templatePlugin` may clash with `computeSidebarModel`'s existing token rendering. Mitigation: Phase 4 Task 4.1 uses the library plugin as the single source of truth; our `computeSidebarModel` becomes a pure data transformer, not a DOM mutator.
- `renderTitleBarRight` real estate is narrow. If Checkpoints + Export + Finalize crowd it on smaller viewports, collapse under an overflow `⋯` menu. Autosave status stays out of this slot (toasts only, per boundary rule).
- Comment anchor drift: the library persists PM range anchors inside the `.docx` itself, so autosave round-trips preserve them automatically. Our DB stores thread metadata only — no `pm_range_from/to` columns, no `comments.reanchor` RPC. The residual risk is **orphan threads**: a DB thread whose anchor marker no longer exists in the document (e.g. user deletes the range). The Phase 5.0 contract doc picks the orphan policy (drop silently / surface as anchorless thread / mark resolved) and a contract test locks the chosen behavior.
