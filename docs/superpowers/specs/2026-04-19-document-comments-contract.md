# Document Comments Contract — docx-v2

**Status:** Phase 5.0 precondition for `2026-04-19-docx-editor-word-parity.md`. Blocks all P5.1 implementation.
**Date:** 2026-04-19
**Library reference:** `@eigenpal/docx-js-editor@0.0.34`.

## 1. Library `Comment` type (ground truth)

Extracted from `dist/agentApi-*.d.ts`:

```ts
interface Comment {
  id: number;                // matches commentRangeStart/End marker IDs in the .docx
  author: string;
  initials?: string;
  date?: string;             // ISO-8601
  content: Paragraph[];      // rich content (not plain text)
  parentId?: number;         // presence => reply; id points at parent thread root
  done?: boolean;            // true === resolved
}
```

Range anchors live **inside the .docx** as `CommentRangeStart` / `CommentRangeEnd` markers referencing `id`. They are **not** part of the `Comment` object the library emits via `comments` / `onCommentAdd` / `onCommentsChange`.

## 2. Field mapping → `document_comments` table

| Library field | Column                    | Type                   | Nullable | Notes |
|---------------|---------------------------|------------------------|----------|-------|
| `id`          | `library_comment_id`      | `INTEGER`              | no       | Author-assigned by library; scope = (document_id, library_comment_id) unique |
| (surrogate)   | `id`                      | `UUID PK`              | no       | Internal stable key (library id can shift on doc rewrite; UUID carries our foreign keys) |
| `author`      | `author_display`          | `TEXT`                 | no       | Free-form name the library stored |
| (actor)       | `author_id`               | `TEXT`                 | no       | Our IAM id (matches migrations 0114–0117 TEXT actor columns) |
| `initials`    | not persisted             | —                      | —        | Derivable from `author_display`; library recomputes |
| `date`        | `created_at`              | `TIMESTAMPTZ`          | no       | Server stamps on insert; library `date` used for display only |
| (server)      | `updated_at`              | `TIMESTAMPTZ`          | no       | Bumped on any mutation |
| `content`     | `content_json`            | `JSONB`                | no       | Raw library-emitted `Paragraph[]` — round-trip as-is |
| `parentId`    | `parent_library_id`       | `INTEGER`              | yes      | NULL = root thread; non-null = reply pointing at a sibling's `library_comment_id` |
| `done`        | (derived) `resolved_at`   | `TIMESTAMPTZ`          | yes      | `done = (resolved_at IS NOT NULL)` |
| (resolver)    | `resolved_by`             | `TEXT`                 | yes      | IAM id of the user who set `done=true`; NULL when `resolved_at` NULL |
| (scope)       | `document_id`             | `UUID`                 | no       | FK → `documents(id)` |
| (scope)       | `tenant_id`               | `UUID`                 | no       | Enforced by RLS |

### Not persisted (why)

- `initials` — derivable; library regenerates on display.
- `content` as flattened text — lossy. We persist the full `Paragraph[]` JSONB.
- PM `from` / `to` positions — we never persist range offsets. They live inside the `.docx` as `CommentRangeStart/End` markers and ride autosave round-trips automatically.

## 3. Threading model

The library uses a **flat parent pointer** (`parentId?`), not a `thread_id`:

- Root thread: `parentId === undefined`.
- Reply: `parentId = <root.id>`. No nested replies (one level only).

We mirror it exactly:

- `document_comments.parent_library_id` is the only threading column.
- No `thread_id`, no `reply_of_id`, no recursive CTEs.
- Reply metadata (author, timestamps, resolved state) lives on the reply row itself. Parent is untouched when a reply is added.

Loading to the library:
```sql
SELECT ... FROM document_comments WHERE document_id = $1 ORDER BY created_at;
```
API hand-serializes into the flat `Comment[]` shape the library expects.

## 4. Resolved state

Library: `done?: boolean`. Toggling fires `onCommentResolve(c)` where `c.done === true` (or a subsequent mutation with `done === false`).

Our DB: `resolved_at timestamptz NULL` + `resolved_by TEXT NULL`.

- Load: `done = resolved_at IS NOT NULL`.
- `onCommentResolve` handler: `UPDATE document_comments SET resolved_at = now(), resolved_by = $userID WHERE id = $internalID`.
- Un-resolve: `UPDATE ... SET resolved_at = NULL, resolved_by = NULL`.

We never expose `resolved_at` to the library — it only sees `done`.

## 5. Anchor model (the critical invariant)

**Anchors live in the `.docx`, never in the DB.**

- Library inserts `CommentRangeStart` / `CommentRangeEnd` markers carrying `id` into the document's paragraph content.
- Autosave serializes the .docx (markers included) → S3. On reload, library re-parses and rebuilds PM ranges from markers.
- Our DB row stores **only thread metadata** (author, content, resolved, timestamps, parent pointer).

**Why:** If we stored PM `from`/`to` offsets in the DB, every autosave-induced position shift (insert text above a comment → shifts offsets) would desync DB ↔ doc. By keeping anchors inside the serialized .docx, the transform is atomic.

### Round-trip

```
user selects range + adds comment
  ↓
library emits onCommentAdd(c)
  ↓ (client)
POST /api/v2/documents/:id/comments { library_comment_id, author, content_json, parent_library_id? }
  ↓ (server)
INSERT into document_comments → return new internal UUID
  ↓ (client)
continues editing. next autosave captures .docx with CommentRangeStart/End markers.
  ↓
presign → PUT S3 → commit → revision_num++
  ↓
reload of same document
  ↓
GET /api/v2/documents/:id/comments → Comment[] assembled from DB rows
  ↓
<MetalDocsEditor comments={commentsFromDB} /> → library reconciles with markers in the .docx
```

### Orphan policy

An **orphan thread** = DB row has no matching `CommentRangeStart` marker in the current `.docx`. Can happen if:
- Author undid text containing the anchor before autosaving.
- Restore-from-checkpoint replaced the doc body but DB rows survived.

**Chosen behavior:** show orphan threads in a dedicated "Orphaned" section of the sidebar (client-side filter), read-only, with a single **Delete** action. Do not auto-delete on server. Do not attempt reanchoring. Reconciliation is one-way: `.docx` is ground truth for anchors.

Detection: on load, client computes `docMarkerIds = new Set(markers.map(m => m.id))` and partitions `threads` into `live = threads.filter(t => docMarkerIds.has(t.library_comment_id))` vs `orphans = rest`.

## 6. Identity

- `author_id` is `TEXT` (matches docs_v2 actor columns 0114–0117).
- `resolved_by` is `TEXT`.
- No FK to an internal `users` table — IAM owns the identity.
- RLS: `tenant_id = current_tenant()`.

## 7. API surface

```
GET    /api/v2/documents/:id/comments                → Comment[] (library-shaped, flat)
POST   /api/v2/documents/:id/comments                → { library_comment_id, author, content, parent_library_id? } → Comment
PATCH  /api/v2/documents/:id/comments/:libraryID     → partial update (content, done) → Comment
DELETE /api/v2/documents/:id/comments/:libraryID     → 204
```

- Address by `library_comment_id` (scoped to `document_id`), not internal UUID — keeps the client simple (it only knows library ids).
- Concurrency: last-write-wins; no optimistic locking on threads (threads are typically single-author anyway).
- Tenant + document session ACL enforced in middleware, not in the handler.

## 8. Contract tests

### Go tests (`internal/modules/documents_v2/...`)

1. `TestCreateComment_RoundTrip` — POST with library-shaped `content`, GET returns the same payload verbatim (JSONB stored as-is).
2. `TestResolveComment_DerivedDoneField` — PATCH with `{"done":true}` sets `resolved_at`/`resolved_by`; GET returns `done: true`, no leak of `resolved_at`.
3. `TestReplyThread_ParentLibraryID` — two POSTs: root (`parent_library_id=null`) then reply (`parent_library_id=rootLibID`). GET returns both; reply carries its own author/timestamps.

### Vitest (`frontend/apps/web/src/features/documents/v2/hooks/__tests__/`)

1. `useDocumentComments.load.test.tsx` — fetch on mount, hook returns `Comment[]` library-shaped (numeric `id`, `done` boolean).
2. `useDocumentComments.add.test.tsx` — calling `add()` POSTs, optimistic-appends, rolls back on failure.
3. `useDocumentComments.orphan.test.tsx` — given DB threads + a doc-marker set missing one id, hook returns `{ live, orphans }` partition.

**Convention:** all six tests must be written and failing (red) before any P5.1 implementation ships. This contract doc is the source of truth for their assertions.

## 9. Open questions — none

All previously-open questions (ID type, threading shape, resolved representation, anchor storage, orphan policy) are settled above. No TBDs. Any P5.1 deviation requires updating this doc first.
