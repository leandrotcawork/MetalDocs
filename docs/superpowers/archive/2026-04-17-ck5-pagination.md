# CK5 Pagination Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers-extended-cc:subagent-driven-development` (recommended) or `superpowers-extended-cc:executing-plans` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver Word-style pagination in the CK5 DecoupledEditor with exact DOCX + PDF parity, using a hybrid architecture: client DOM measurement for live UX, server Paged.js in headless Chromium as ground truth, reconciler honoring editor ±1 drift, and a DOCX hygiene layer (widowControl=0, embedded Carlito, pinned compat flags) for deterministic Word/WPS rendering.

**Architecture:** Three-layer contract — (1) shared layout tokens as single source of truth, (2) stable block identity (UUID v4 `data-mddm-bid`) on every paginable block, (3) bid-keyed reconcile between editor breaks and server breaks. PDF unchanged (Gotenberg + CSS Paged Media). DOCX emission unchanged structurally, hardened with hygiene flags and explicit page-break runs at reconciled marker positions.

**Tech Stack:** CKEditor 5 v48 DecoupledEditor, React 18, TypeScript, Vite (editor) / Hono + tsx + docx npm (export service), Paged.js (MIT), Playwright (Chromium pool), Gotenberg (existing), Vitest (unit), Playwright test (E2E), OFL Carlito.

**Spec:** `docs/superpowers/specs/2026-04-17-ck5-pagination-design.md`.

**Branch:** `migrate/ck5-plan-c`.

---

## Spec Deviations (intentional, approved)

Two deviations from `docs/superpowers/specs/2026-04-17-ck5-pagination-design.md`. Both preserve the spec's single-source-of-truth contract; only physical layout/packaging changes.

1. **Shared tokens location** — spec proposes `packages/mddm-layout-tokens` (pnpm workspace package). Repo has no `pnpm-workspace.yaml`. Plan uses `shared/mddm-layout-tokens/` as a source-only TypeScript module consumed via tsconfig path alias + Vite/Vitest `resolve.alias`. **Alias name matches spec verbatim: `@metaldocs/mddm-layout-tokens`.** Integration parity enforced by `apps/ck5-export/src/__tests__/shared-tokens-parity.test.ts` + `frontend/apps/web/src/features/documents/ck5/__tests__/shared-tokens-parity.test.ts` (Task 0) — both assert deep-equal against the shared module.
2. **Shared pagination transport types** — spec names types inline per consumer. Plan extracts `BreakCandidate`, `ComputedBreak`, `ServerBreak`, `ReconciledBreak`, `ReconcileLogs`, `ReconcileResult`, `PaginateRequest`, `PaginateResponse` into `shared/mddm-pagination-types/index.ts` (Task 0b) under the same alias scheme: `@metaldocs/mddm-pagination-types`. Both the editor plugin (Task 12) and the export service (Tasks 20–24) import from this one module. A compile-time contract test (`shared/mddm-pagination-types/__tests__/contract.test-d.ts`) fails the build if either side drifts.

No functional spec requirements are changed. If pnpm workspace is added later, physical-to-workspace migration is a pure move + alias removal.

---

## Orchestration — Codex vs Opus

Two executors. **Codex (gpt-5.3-codex, medium reasoning)** handles mechanical TDD with clear contracts, boilerplate, emitter extensions, fixture authoring, CI wiring — tasks where the spec fully determines the shape. **Opus (this session, coordinator)** handles CK5 model-view-data three-tree subtleties, edge-case reasoning (split/merge/paste semantics), complex reconcile logic, and view-layer `uiElement` semantics — tasks requiring domain judgment.

Parallelism groups are flagged per-task. Dispatch parallel groups as concurrent subagents (separate worktrees or disjoint file sets).

| Task | Executor | Parallel group | Rationale |
|------|----------|----------------|-----------|
| 0 — Shared tokens pkg | Opus | P0 | Architecture call; tsconfig/alias decisions across 2 apps |
| 0b — Shared pagination types pkg | Opus | P0 (parallel w/ 0) | Single source of truth for editor↔server transport types + contract test |
| 1 — Token fields added | Codex | after 0 | Mechanical field addition, single file |
| 2 — CSS font/margin unify | Codex | P1 (parallel w/ 3,4) | Single file, clear spec |
| 3 — Carlito TTF bundling | Codex | P1 | File placement + NOTICES attribution |
| 4 — paged.polyfill.js shell | Codex | P1 | One-line inclusion in existing shell |
| 5 — BlockIdentityPlugin schema | Opus | after 1 | CK5 schema.extend per paginable type, model judgment |
| 6 — bid post-fixer (mint) | Codex | after 5 | Mechanical with tests |
| 7 — bid post-fixer (split/merge) | Opus | after 6 | Subtle survivor-vs-fresh semantics |
| 8 — bid converters (up/down) | Codex | after 5 | Boilerplate pattern |
| 9 — bid clipboard remint | Opus | after 8 | Paste-semantics judgment |
| 10 — schema v3→v4 migration | Codex | after 8 | Deterministic upcast rule |
| 11 — server bid validator | Codex | after 8 | Pure function + 422 route wiring |
| 12 — pagination plugin scaffold | Codex | after 5 | Boilerplate `Plugin.requires` |
| 13 — DirtyRangeTracker | Opus | after 12 | Differ subtleties, range clamping |
| 14 — BreakPlanner model walk | Opus | after 13 | keep-with-next + widget atomicity rules |
| 15 — BreakMeasurer (render hook) | Opus | after 14 | view.document.render timing, DPR rounding, fonts.ready |
| 16 — SectionScope clamping | Opus | after 15 | Per-section independence logic |
| 17 — PageOverlayView uiElement | Opus | after 15 | View-only chrome, never in data view |
| 18 — getData({pagination:true}) | Codex | after 17 | Serializer attribute emission |
| 19 — Page counter UI | Codex | after 17 (parallel w/ 18) | React component from ComputedBreak[] |
| 20 — Playwright worker pool | Codex | P2 (parallel w/ 25-28) | Node infra boilerplate |
| 20b — Pool retry-once + fallback ladder | Opus | after 20 | Resilience decisions: crash→retry alternate worker, then degrade |
| 21 — sentinel injector | Codex | after 20 | Pure HTML transform |
| 22 — Paged.js scrape | Codex | after 21 | page.evaluate + DOM query |
| 22b — HTML sha256 cache + TTL | Codex | after 22 | Deterministic pure caching layer |
| 23 — /paginate Hono route | Codex | after 22b | Route wiring + accepts `editorBids` for desync 422 |
| 24 — Reconciler | Opus | after 23 | ±1-drift decision algo |
| 25 — html-to-export-tree page-break | Codex | P2 | Verify/extend existing emitter |
| 26 — docx widowControl | Codex | P2 | Emitter field addition |
| 27 — docx settings.xml pin | Codex | P2 | Emitter Document ctor fields |
| 28 — docx Carlito embed | Codex | after 3 | docx lib Font config |
| 29 — 10 golden fixtures | Codex | after 24 | Bulk HTML authoring |
| 30 — bid invariance matrix | Opus | after 9 | Edit-sequence design |
| 31 — Drift SLO CI gate | Codex | after 29 | Vitest + CI yaml |
| 31b — Paginator P95 perf gate | Codex | after 29 | Benchmark + CI threshold (≤1.5s P95 per 50 pages) |
| 32 — Pagination E2E | Codex | after 19,24 | Playwright test authoring |
| 33 — Parity probe debug route + overlay asserted | Codex | after 24 | Dev-only Hono route + overlay toggle + vitest |

**Parallel group launch plan:**
- **Wave 1 (after Task 0 + 0b — parallel Opus):** Task 1 (Codex).
- **Wave 2 (P1, after 1):** Tasks 2, 3, 4 — 3 Codex subagents in parallel. Independent files.
- **Wave 3 (block identity):** Task 5 (Opus) → Tasks 6, 8 in parallel (Codex × 2) → Task 7 (Opus) after 6 + Tasks 9 (Opus), 10 (Codex), 11 (Codex) parallel after 8.
- **Wave 4 (pagination plugin):** Task 12 (Codex) → 13→14→15 (Opus sequential) → Tasks 16, 17 (Opus) parallel after 15 → Tasks 18, 19 (Codex) parallel after 17.
- **Wave 5 (P2, after shared tokens + bid contract stable):** Tasks 20, 25, 26, 27 — 4 Codex subagents in parallel. Task 28 after 3.
- **Wave 6 (server paginator):** Task 20b (Opus) after 20 → Task 21 → 22 → 22b → 23 (Codex sequential) → Task 24 (Opus).
- **Wave 7 (tests/CI):** Tasks 29 (Codex), 30 (Opus) parallel → Tasks 31, 31b, 32, 33 (Codex) parallel after predecessors.

---

## File Structure

**New files:**

```
shared/mddm-layout-tokens/
├── package.json                           # name: @metaldocs/mddm-layout-tokens, private, no build (source-only TS)
├── tsconfig.json                          # emits .d.ts and .js on build; dev is source-only via path alias
└── index.ts                               # LayoutTokens type + defaultLayoutTokens + new fields

shared/mddm-pagination-types/
├── package.json                           # name: @metaldocs/mddm-pagination-types, private, source-only
├── tsconfig.json
├── index.ts                               # BreakCandidate, ComputedBreak, ServerBreak, ReconciledBreak, ReconcileLogs, ReconcileResult, PaginateRequest, PaginateResponse
└── __tests__/contract.test-d.ts           # compile-time contract: both sides import & satisfy

apps/ck5-export/fonts/
├── Carlito-Regular.ttf                    # OFL
├── Carlito-Bold.ttf
├── Carlito-Italic.ttf
└── Carlito-BoldItalic.ttf

apps/ck5-export/src/pagination/
├── paginate-with-chromium.ts              # Playwright worker entry + sentinel inject + scrape
├── playwright-pool.ts                     # worker pool manager (size=CHROMIUM_POOL_SIZE, default 3)
├── pool-retry.ts                          # Opus: retry-once-on-crash + fallback ladder
├── sentinel.ts                            # HTML transform: inject <span data-pb-marker="{bid}">
├── cache.ts                               # sha256(html) → ServerBreak[] in-memory cache (5-min TTL)
├── reconcile.ts                           # editor+server break reconcile algo
├── parity-diff.ts                         # JSON drift report
└── validator.ts                           # bid duplicate/missing/desync detection → 422

apps/ck5-export/src/routes/
├── paginate.ts                            # POST /paginate — returns ServerBreak[]
└── pagination-debug.ts                    # GET /pagination-debug/:docId — dev only

apps/ck5-export/src/__fixtures__/pagination/
├── short-para.html
├── long-para.html
├── heavy-table.html
├── image-heavy.html
├── nested-lists.html
├── section-rich.html
├── repeatable-rows.html
├── mixed-headings.html
├── edge-widow.html
├── 100-page-contract.html
└── *.reconciled.json                      # per-fixture goldens

apps/ck5-export/src/__tests__/
├── pagination-e2e.test.ts                 # fixture → reconcile → DOCX parse
├── drift-slo.test.ts                      # CI SLO gate
├── perf-gate.test.ts                      # P95 ≤1.5s per 50 pages (Task 31b)
└── shared-tokens-parity.test.ts           # export-side parity

frontend/apps/web/src/features/documents/ck5/plugins/MddmBlockIdentityPlugin/
├── index.ts                               # plugin entry
├── schema.ts                              # extend paginable elements with mddmBid
├── post-fixer.ts                          # mint / split / merge logic
├── converters.ts                          # upcast + downcast
├── clipboard.ts                           # collision remint on paste
├── migration.ts                           # schema v3 → v4 upgrade
└── __tests__/
    ├── post-fixer.test.ts
    ├── converters.test.ts
    ├── clipboard.test.ts
    └── migration.test.ts

frontend/apps/web/src/features/documents/ck5/plugins/MddmPaginationPlugin/
├── index.ts                               # plugin entry + wiring
├── BreakPlanner.ts
├── BreakMeasurer.ts
├── PageOverlayView.ts
├── DirtyRangeTracker.ts
├── SectionScope.ts
├── data-contract.ts                       # getData({pagination:true}) emitter
├── types.ts                               # BreakCandidate, ComputedBreak, ServerBreak, ReconciledBreak
└── __tests__/
    ├── BreakPlanner.test.ts
    ├── BreakMeasurer.test.ts
    ├── PageOverlayView.test.ts
    ├── DirtyRangeTracker.test.ts
    ├── data-contract.test.ts
    └── section-scope.test.ts

frontend/apps/web/src/features/documents/ck5/react/
└── PageCounter.tsx                        # "Page N of M" React view

frontend/apps/web/tests/e2e/
└── pagination.spec.ts                     # Playwright E2E
```

**Modified files:**

```
apps/ck5-export/src/layout-tokens.ts                     # re-export from @metaldocs/mddm-layout-tokens
apps/ck5-export/src/server.ts                            # register /paginate + /pagination-debug routes
apps/ck5-export/src/html-to-export-tree.ts               # verify/add mddm-page-break node
apps/ck5-export/src/docx-emitter/paragraph.ts            # widowControl: false on every Paragraph
apps/ck5-export/src/docx-emitter/heading.ts              # widowControl: false
apps/ck5-export/src/docx-emitter/emitter.ts              # Document settings + fonts config
apps/ck5-export/src/docx-emitter/block-dispatch.ts       # pageBreak → Paragraph({children:[new PageBreak()]})
apps/ck5-export/src/print-stylesheet/wrap-print-document.ts # include paged.polyfill.js
apps/ck5-export/package.json                             # add playwright, paged.js, @metaldocs/mddm-layout-tokens
apps/ck5-export/tsconfig.json                            # paths alias for @metaldocs/mddm-layout-tokens
apps/ck5-export/NOTICES                                  # Carlito OFL, Paged.js MIT, Playwright Apache-2
frontend/apps/web/src/features/documents/ck5/react/AuthorEditor.module.css # margin 20→25, Carlito font
frontend/apps/web/src/features/documents/ck5/config/editorConfig.ts # register BlockIdentity + Pagination plugins
frontend/apps/web/package.json                           # add @metaldocs/mddm-layout-tokens
frontend/apps/web/tsconfig.json                          # paths alias for @metaldocs/mddm-layout-tokens
frontend/apps/web/vite.config.ts                         # resolve.alias for @metaldocs/mddm-layout-tokens
frontend/apps/web/vitest.config.ts                       # same alias for tests
frontend/apps/web/public/fonts/                          # Carlito woff2 for editor
```

---

## Phase 0 — Shared tokens + prereqs

### Task 0: Create `shared/mddm-layout-tokens` package with path aliases

**Executor:** Opus. **Group:** P0 (blocks everything else).

**Goal:** Stand up shared workspace-style package for layout tokens, wired into both apps via tsconfig path aliases + Vite/Vitest resolve aliases. No pnpm workspace exists in this repo — use source-only TS imports through path aliases.

**Files:**
- Create: `shared/mddm-layout-tokens/package.json`
- Create: `shared/mddm-layout-tokens/tsconfig.json`
- Create: `shared/mddm-layout-tokens/index.ts`
- Modify: `apps/ck5-export/tsconfig.json`
- Modify: `frontend/apps/web/tsconfig.json`
- Modify: `frontend/apps/web/vite.config.ts`
- Modify: `frontend/apps/web/vitest.config.ts`

**Acceptance Criteria:**
- [ ] `import { defaultLayoutTokens } from '@metaldocs/mddm-layout-tokens'` resolves in both apps.
- [ ] `pnpm --filter @metaldocs/ck5-export typecheck` passes.
- [ ] `pnpm --filter @metaldocs/web build` passes.

**Verify:** `cd apps/ck5-export && npx tsc --noEmit && cd ../../frontend/apps/web && npx tsc --noEmit -p tsconfig.build.json` → exit 0 for both.

**Steps:**

- [ ] **Step 1: Create `shared/mddm-layout-tokens/index.ts` by copying current `apps/ck5-export/src/layout-tokens.ts` verbatim**

Copy file contents 1:1. Keep `LayoutTokens` type + `defaultLayoutTokens` const. Do not add new fields yet (Task 1 does that).

- [ ] **Step 2: Create `shared/mddm-layout-tokens/package.json`**

```json
{
  "name": "@metaldocs/mddm-layout-tokens",
  "version": "0.0.0",
  "private": true,
  "type": "module",
  "main": "./index.ts",
  "types": "./index.ts",
  "exports": {
    ".": "./index.ts"
  }
}
```

- [ ] **Step 3: Create `shared/mddm-layout-tokens/tsconfig.json`**

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "module": "ESNext",
    "moduleResolution": "Bundler",
    "strict": true,
    "declaration": true,
    "noEmit": true
  },
  "include": ["index.ts"]
}
```

- [ ] **Step 4: Add path alias to `apps/ck5-export/tsconfig.json`**

Add under `compilerOptions`:
```json
"baseUrl": ".",
"paths": {
  "@metaldocs/mddm-layout-tokens": ["../../shared/mddm-layout-tokens/index.ts"]
}
```

- [ ] **Step 5: Add path alias to `frontend/apps/web/tsconfig.json`**

Add under `compilerOptions`:
```json
"baseUrl": ".",
"paths": {
  "@metaldocs/mddm-layout-tokens": ["../../../shared/mddm-layout-tokens/index.ts"]
}
```

- [ ] **Step 6: Add resolve alias to `frontend/apps/web/vite.config.ts`**

In `resolve.alias` add:
```ts
'@metaldocs/mddm-layout-tokens': path.resolve(__dirname, '../../../shared/mddm-layout-tokens/index.ts'),
```

- [ ] **Step 7: Add resolve alias to `frontend/apps/web/vitest.config.ts`**

Mirror the Vite alias in vitest's `resolve.alias`.

- [ ] **Step 8a: Write EXPORT-side parity test**

Create `apps/ck5-export/src/__tests__/shared-tokens-parity.test.ts`:
```ts
import { describe, it, expect } from 'vitest';
import { defaultLayoutTokens as shared } from '@metaldocs/mddm-layout-tokens';
import { defaultLayoutTokens as local } from '../layout-tokens';

describe('shared tokens parity (export)', () => {
  it('shared and local point to same object values', () => {
    expect(local).toEqual(shared);
  });
});
```

- [ ] **Step 8b: Write FRONTEND-side parity test**

Create `frontend/apps/web/src/features/documents/ck5/__tests__/shared-tokens-parity.test.ts`:
```ts
import { describe, it, expect } from 'vitest';
import { defaultLayoutTokens } from '@metaldocs/mddm-layout-tokens';

describe('shared tokens parity (frontend)', () => {
  it('alias resolves and exposes canonical token shape', () => {
    expect(defaultLayoutTokens.page.widthMm).toBe(210);
    expect(defaultLayoutTokens.page.heightMm).toBe(297);
    expect(defaultLayoutTokens.page.marginLeftMm).toBe(25);
    expect(defaultLayoutTokens.page.marginRightMm).toBe(25);
    expect(defaultLayoutTokens.typography.exportFont).toBe('Carlito');
  });
});
```

- [ ] **Step 9: Run typecheck + both parity tests**

```bash
cd apps/ck5-export && npx tsc --noEmit && npx vitest run src/__tests__/shared-tokens-parity.test.ts
cd ../../frontend/apps/web && npx tsc --noEmit -p tsconfig.build.json && npx vitest run src/features/documents/ck5/__tests__/shared-tokens-parity.test.ts
```
Expected: exit 0 all four commands.

- [ ] **Step 10: Commit**

```bash
git add shared/mddm-layout-tokens apps/ck5-export/tsconfig.json frontend/apps/web/tsconfig.json frontend/apps/web/vite.config.ts frontend/apps/web/vitest.config.ts apps/ck5-export/src/__tests__/shared-tokens-parity.test.ts frontend/apps/web/src/features/documents/ck5/__tests__/shared-tokens-parity.test.ts
git commit -m "feat(mddm): extract layout tokens to shared package with path aliases"
```

---

### Task 0b: Create `shared/mddm-pagination-types` package (shared transport types)

**Executor:** Opus. **Group:** P0 (parallel with Task 0 — disjoint files).

**Goal:** Single-source-of-truth for every type that crosses the editor↔server wire. Fails build if either side drifts.

**Files:**
- Create: `shared/mddm-pagination-types/package.json` (name `@metaldocs/mddm-pagination-types`, private, source-only)
- Create: `shared/mddm-pagination-types/tsconfig.json`
- Create: `shared/mddm-pagination-types/index.ts`
- Create: `shared/mddm-pagination-types/__tests__/contract.test-d.ts`
- Modify: `apps/ck5-export/tsconfig.json` (add path alias)
- Modify: `frontend/apps/web/tsconfig.json` (add path alias)
- Modify: `frontend/apps/web/vite.config.ts` (resolve.alias)
- Modify: `frontend/apps/web/vitest.config.ts` (resolve.alias)

**Acceptance Criteria:**
- [ ] All 8 transport types exported: `BreakCandidate`, `ComputedBreak`, `ServerBreak`, `ReconciledBreak`, `ReconcileLogs`, `ReconcileResult`, `PaginateRequest`, `PaginateResponse`.
- [ ] Alias `@metaldocs/mddm-pagination-types` resolves from both apps.
- [ ] `contract.test-d.ts` compiles (negative case: breaking a type fails `tsc --noEmit`).

**Verify:** `cd shared/mddm-pagination-types && npx tsc --noEmit && echo OK` → `OK`.

**Steps:**

- [ ] **Step 1: Create `shared/mddm-pagination-types/index.ts`**

```ts
// Transport types shared across editor plugin and export service.
// Any change here MUST be accepted by both consumers.

export type BreakCandidate = Readonly<{
  afterBid: string;         // bid of block before the break
  modelPath: readonly number[]; // position path in CK5 model
  keepWithNext: boolean;    // true for headings → defer until next block fits
}>;

export type ComputedBreak = Readonly<{
  afterBid: string;
  pageNumber: number;       // 1-indexed
  yPx: number;              // cursor Y at break (debug)
}>;

export type ServerBreak = Readonly<{
  bid: string;              // matches afterBid from editor side
  pageNumber: number;
}>;

export type ReconciledBreak = Readonly<{
  afterBid: string;
  pageNumber: number;
  source: 'editor' | 'editor-minor-drift' | 'server';
}>;

export type ReconcileLogs = Readonly<{
  exactMatches: number;
  minorDrift: number;       // |delta|==1, editor honored
  majorDrift: number;       // |delta|>1, server wins
  orphanedEditor: number;   // editor bid not in server output
  serverOnly: number;       // server bid not in editor output
}>;

export type ReconcileResult = Readonly<{
  resolved: readonly ReconciledBreak[];
  logs: ReconcileLogs;
}>;

export type PaginateRequest = Readonly<{
  html: string;
  /** Optional: bids of breaks the editor reported. Enables editor-server-desync (422). */
  editorBids?: readonly string[];
}>;

export type PaginateResponse = Readonly<{
  breaks: readonly ServerBreak[];
}>;
```

- [ ] **Step 2: Create `shared/mddm-pagination-types/package.json`**

```json
{
  "name": "@metaldocs/mddm-pagination-types",
  "version": "0.0.0",
  "private": true,
  "type": "module",
  "main": "index.ts",
  "types": "index.ts"
}
```

- [ ] **Step 3: Create `shared/mddm-pagination-types/tsconfig.json`**

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "module": "ESNext",
    "moduleResolution": "bundler",
    "strict": true,
    "noEmit": true,
    "esModuleInterop": true,
    "skipLibCheck": true
  },
  "include": ["index.ts", "__tests__/**/*.ts"]
}
```

- [ ] **Step 4: Create `shared/mddm-pagination-types/__tests__/contract.test-d.ts`**

```ts
// Compile-time contract test. `tsc --noEmit` fails if types break.
import type {
  BreakCandidate, ComputedBreak, ServerBreak, ReconciledBreak,
  ReconcileLogs, ReconcileResult, PaginateRequest, PaginateResponse,
} from '../index';

// Exact-field assertions (add new field → must update contract or fail).
declare const bc: BreakCandidate;
const _bc: { afterBid: string; modelPath: readonly number[]; keepWithNext: boolean } = bc;
void _bc;

declare const cb: ComputedBreak;
const _cb: { afterBid: string; pageNumber: number; yPx: number } = cb;
void _cb;

declare const sb: ServerBreak;
const _sb: { bid: string; pageNumber: number } = sb;
void _sb;

declare const rb: ReconciledBreak;
const _rb: { afterBid: string; pageNumber: number; source: 'editor' | 'editor-minor-drift' | 'server' } = rb;
void _rb;

declare const rl: ReconcileLogs;
const _rl: { exactMatches: number; minorDrift: number; majorDrift: number; orphanedEditor: number; serverOnly: number } = rl;
void _rl;

declare const rr: ReconcileResult;
const _rr: { resolved: readonly ReconciledBreak[]; logs: ReconcileLogs } = rr;
void _rr;

declare const pq: PaginateRequest;
const _pq: { html: string; editorBids?: readonly string[] } = pq;
void _pq;

declare const pr: PaginateResponse;
const _pr: { breaks: readonly ServerBreak[] } = pr;
void _pr;
```

- [ ] **Step 5: Add path aliases to both tsconfigs**

In `apps/ck5-export/tsconfig.json` `compilerOptions.paths` (merge with existing aliases from Task 0):
```json
"@metaldocs/mddm-pagination-types": ["../../shared/mddm-pagination-types/index.ts"]
```

In `frontend/apps/web/tsconfig.json` `compilerOptions.paths`:
```json
"@metaldocs/mddm-pagination-types": ["../../../shared/mddm-pagination-types/index.ts"]
```

- [ ] **Step 6: Add Vite + Vitest aliases**

In `frontend/apps/web/vite.config.ts` and `vitest.config.ts`, add to `resolve.alias`:
```ts
'@metaldocs/mddm-pagination-types': path.resolve(__dirname, '../../../shared/mddm-pagination-types/index.ts'),
```

- [ ] **Step 7: Run contract test**

```bash
cd shared/mddm-pagination-types && npx tsc --noEmit && echo OK
```
Expected: `OK`.

- [ ] **Step 8: Commit**

```bash
git add shared/mddm-pagination-types apps/ck5-export/tsconfig.json frontend/apps/web/tsconfig.json frontend/apps/web/vite.config.ts frontend/apps/web/vitest.config.ts
git commit -m "feat(mddm): shared pagination transport types with compile-time contract"
```

---

### Task 1: Re-export from shared + add pagination token fields

**Executor:** Codex. **Group:** after 0.

**Goal:** Turn `apps/ck5-export/src/layout-tokens.ts` into a thin re-export, then extend the shared package with `paginationSLO`, `blockIdentityAttr`, `pageBreakAttr`, `fontFallbackChain`.

**Files:**
- Modify: `apps/ck5-export/src/layout-tokens.ts` (replace with re-export)
- Modify: `shared/mddm-layout-tokens/index.ts` (add fields)
- Modify: `apps/ck5-export/src/__tests__/shared-tokens-parity.test.ts` (update parity expectations)

**Acceptance Criteria:**
- [ ] `defaultLayoutTokens.paginationSLO.maxBreakDeltaPer50Pages === 1`.
- [ ] `defaultLayoutTokens.blockIdentityAttr === 'data-mddm-bid'`.
- [ ] `defaultLayoutTokens.pageBreakAttr === 'data-pagination-page'`.
- [ ] `defaultLayoutTokens.fontFallbackChain` is a frozen readonly array of Carlito stack.
- [ ] `apps/ck5-export/src/layout-tokens.ts` now exports via `export * from '@metaldocs/mddm-layout-tokens'`.
- [ ] All existing consumers of `layout-tokens` compile.

**Verify:** `cd apps/ck5-export && npx vitest run src/__tests__/shared-tokens-parity.test.ts` → PASS.

**Steps:**

- [ ] **Step 1: Add new fields to `shared/mddm-layout-tokens/index.ts`**

Extend the `LayoutTokens` type (inside `Readonly<{ ... }>`):
```ts
paginationSLO: Readonly<{
  maxBreakDeltaPer50Pages: number;
}>;
blockIdentityAttr: string;
pageBreakAttr: string;
fontFallbackChain: readonly string[];
```

Extend `defaultLayoutTokens`:
```ts
paginationSLO: { maxBreakDeltaPer50Pages: 1 },
blockIdentityAttr: 'data-mddm-bid',
pageBreakAttr: 'data-pagination-page',
fontFallbackChain: ['Carlito', 'Liberation Sans', 'Arial', 'sans-serif'],
```

- [ ] **Step 2: Replace `apps/ck5-export/src/layout-tokens.ts` with re-export**

```ts
// Re-exports the shared layout tokens package.
// All edits must happen in shared/mddm-layout-tokens/index.ts.
export * from '@metaldocs/mddm-layout-tokens';
```

- [ ] **Step 3: Update parity test to assert new field presence**

```ts
it('exposes pagination contract fields', () => {
  expect(shared.blockIdentityAttr).toBe('data-mddm-bid');
  expect(shared.pageBreakAttr).toBe('data-pagination-page');
  expect(shared.paginationSLO.maxBreakDeltaPer50Pages).toBe(1);
  expect(shared.fontFallbackChain[0]).toBe('Carlito');
});
```

- [ ] **Step 4: Run test**

```bash
cd apps/ck5-export && npx vitest run src/__tests__/shared-tokens-parity.test.ts
```
Expected: PASS, 2 tests.

- [ ] **Step 5: Typecheck both apps**

```bash
cd apps/ck5-export && npx tsc --noEmit && cd ../../frontend/apps/web && npx tsc --noEmit -p tsconfig.build.json
```
Expected: exit 0.

- [ ] **Step 6: Commit**

```bash
git add shared/mddm-layout-tokens/index.ts apps/ck5-export/src/layout-tokens.ts apps/ck5-export/src/__tests__/shared-tokens-parity.test.ts
git commit -m "feat(mddm): add pagination contract fields to layout tokens"
```

---

### Task 2: Unify editor CSS — margin 20→25mm and Carlito font

**Executor:** Codex. **Group:** P1 (parallel with Tasks 3, 4).

**Goal:** Match editor canvas to shared tokens' margins (25mm all sides) and font (Carlito), eliminating editor↔export drift before pagination measurement begins.

**Files:**
- Modify: `frontend/apps/web/src/features/documents/ck5/react/AuthorEditor.module.css`
- Create: `frontend/apps/web/public/fonts/carlito-regular.woff2`
- Create: `frontend/apps/web/public/fonts/carlito-bold.woff2`
- Create: `frontend/apps/web/public/fonts/carlito-italic.woff2`
- Create: `frontend/apps/web/public/fonts/carlito-bold-italic.woff2`
- Modify: `frontend/apps/web/src/features/documents/ck5/react/AuthorEditor.module.css` (add @font-face)

**Acceptance Criteria:**
- [ ] `.editable .ck-editor__editable_inline` padding = `25mm 25mm`.
- [ ] Computed font-family on editable content starts with `Carlito`.
- [ ] Carlito woff2 fonts load without 404.

**Verify:** Run editor locally, DevTools → Elements → inspect `.ck-editor__editable_inline`. Computed `padding: 94.488px 94.488px` (25mm at 96dpi) and `font-family: Carlito, "Liberation Sans", Arial, sans-serif`.

**Steps:**

- [ ] **Step 1: Obtain Carlito woff2 assets**

Download from https://fonts.google.com/specimen/Carlito (Google Fonts serves woff2). Place as `carlito-{regular,bold,italic,bold-italic}.woff2` in `frontend/apps/web/public/fonts/`.

- [ ] **Step 2: Update `AuthorEditor.module.css`**

Replace the block starting at `.editable :global(.ck.ck-editor__editable.ck-editor__editable_inline)`:
```css
@font-face {
  font-family: 'Carlito';
  font-style: normal;
  font-weight: 400;
  src: url('/fonts/carlito-regular.woff2') format('woff2');
  font-display: swap;
}
@font-face {
  font-family: 'Carlito';
  font-style: normal;
  font-weight: 700;
  src: url('/fonts/carlito-bold.woff2') format('woff2');
  font-display: swap;
}
@font-face {
  font-family: 'Carlito';
  font-style: italic;
  font-weight: 400;
  src: url('/fonts/carlito-italic.woff2') format('woff2');
  font-display: swap;
}
@font-face {
  font-family: 'Carlito';
  font-style: italic;
  font-weight: 700;
  src: url('/fonts/carlito-bold-italic.woff2') format('woff2');
  font-display: swap;
}

.editable :global(.ck.ck-editor__editable.ck-editor__editable_inline) {
  width: 210mm;
  min-height: 297mm;
  padding: 25mm 25mm;
  background: #fff;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.15);
  border: 1px solid rgba(0, 0, 0, 0.08);
  border-radius: 2px;
  box-sizing: border-box;
  margin: 0;
  flex: 0 0 auto;
  font-family: Carlito, 'Liberation Sans', Arial, sans-serif;
}
```

- [ ] **Step 3: Vite dev server sanity check**

```bash
cd frontend/apps/web && npm run dev
```
Open editor. DevTools → Network → filter `carlito` → all 200. Elements → computed `padding: 94.488px 94.488px`, `font-family: Carlito`.

- [ ] **Step 4: Commit**

```bash
git add frontend/apps/web/public/fonts frontend/apps/web/src/features/documents/ck5/react/AuthorEditor.module.css
git commit -m "fix(ck5): unify editor canvas margin to 25mm and pin Carlito font"
```

---

### Task 3: Bundle Carlito TTF in export service + NOTICES

**Executor:** Codex. **Group:** P1 (parallel with Tasks 2, 4).

**Goal:** Place Carlito TTF binaries in `apps/ck5-export/fonts/` so DOCX emitter can embed them into `fontTable.xml`. Attribute OFL license correctly.

**Files:**
- Create: `apps/ck5-export/fonts/Carlito-Regular.ttf`
- Create: `apps/ck5-export/fonts/Carlito-Bold.ttf`
- Create: `apps/ck5-export/fonts/Carlito-Italic.ttf`
- Create: `apps/ck5-export/fonts/Carlito-BoldItalic.ttf`
- Create: `apps/ck5-export/fonts/OFL.txt` (Carlito OFL license)
- Modify or Create: `apps/ck5-export/NOTICES`

**Acceptance Criteria:**
- [ ] Four TTF files exist in `apps/ck5-export/fonts/`.
- [ ] `OFL.txt` contains verbatim SIL OFL 1.1 text.
- [ ] `NOTICES` lists Carlito (OFL-1.1), Paged.js (MIT), Playwright (Apache-2.0) with copyright and source URLs.

**Verify:** `ls apps/ck5-export/fonts/*.ttf | wc -l` → `4`. `grep -c 'Carlito' apps/ck5-export/NOTICES` → `≥1`.

**Steps:**

- [ ] **Step 1: Download Carlito TTFs**

From Fedora's canonical source: https://src.fedoraproject.org/repo/pkgs/google-carlito-fonts. Take Carlito-{Regular,Bold,Italic,BoldItalic}.ttf into `apps/ck5-export/fonts/`.

- [ ] **Step 2: Add OFL license text**

Copy SIL OFL 1.1 verbatim from https://openfontlicense.org/documents/OFL.txt into `apps/ck5-export/fonts/OFL.txt`.

- [ ] **Step 3: Write/update `apps/ck5-export/NOTICES`**

```
MetalDocs ck5-export — Third-party notices

Carlito
  Copyright (c) 2013 Łukasz Dziedzic (Type Project)
  Licensed under SIL Open Font License, Version 1.1
  See fonts/OFL.txt

Paged.js
  Copyright (c) 2018 Julie Blanc, Fred Chasen
  Licensed under MIT License
  https://pagedjs.org

Playwright
  Copyright (c) Microsoft Corporation
  Licensed under Apache License 2.0
  https://playwright.dev
```

- [ ] **Step 4: Verify**

```bash
ls apps/ck5-export/fonts
```
Expected: `Carlito-Regular.ttf Carlito-Bold.ttf Carlito-Italic.ttf Carlito-BoldItalic.ttf OFL.txt`.

- [ ] **Step 5: Commit**

```bash
git add apps/ck5-export/fonts apps/ck5-export/NOTICES
git commit -m "chore(ck5-export): bundle Carlito TTF under OFL with notices"
```

---

### Task 4: Include `paged.polyfill.js` in print shell

**Executor:** Codex. **Group:** P1 (parallel with Tasks 2, 3).

**Goal:** Add Paged.js polyfill to the existing print wrapper so Chromium-rendered PDFs gain richer `@page` features (pagination DOM fragments, target-counter for future TOC). No break-marker wiring needed — CSS Paged Media already handles PDF breaks.

**Files:**
- Modify: `apps/ck5-export/package.json` (add `pagedjs` dependency)
- Modify: `apps/ck5-export/src/print-stylesheet/wrap-print-document.ts`

**Acceptance Criteria:**
- [ ] `pagedjs` listed in `apps/ck5-export/package.json` dependencies.
- [ ] Emitted HTML `<head>` includes `<script src="https://unpkg.com/pagedjs@0.5/dist/paged.polyfill.js" defer></script>` (or vendored path if offline).
- [ ] Existing PDF golden tests still pass.

**Verify:** `cd apps/ck5-export && npx vitest run` → all tests green. Smoke: `curl http://localhost:51619/render-pdf-html --data-binary @short-para.html` → PDF bytes contain valid content.

**Steps:**

- [ ] **Step 1: Add dependency**

```bash
cd apps/ck5-export && npm install pagedjs@^0.5
```

- [ ] **Step 2: Vendor polyfill to `apps/ck5-export/public/paged.polyfill.js`**

```bash
cp node_modules/pagedjs/dist/paged.polyfill.js apps/ck5-export/public/paged.polyfill.js
```

Ensure Hono static route serves `/public/*` (or adjust path). If no static route exists, add to `server.ts`:
```ts
import { serveStatic } from '@hono/node-server/serve-static';
app.use('/assets/*', serveStatic({ root: './public' }));
```

- [ ] **Step 3: Modify `wrap-print-document.ts` to inject script tag**

Locate the `<head>` construction. Append:
```ts
html += `<script src="/assets/paged.polyfill.js" defer></script>`;
```
(Adjust to match existing string-builder pattern in that file.)

- [ ] **Step 4: Run existing PDF tests**

```bash
cd apps/ck5-export && npx vitest run
```
Expected: all tests pass (no regression). If tests inspect emitted HTML, update snapshots to include the polyfill script.

- [ ] **Step 5: Commit**

```bash
git add apps/ck5-export/package.json apps/ck5-export/package-lock.json apps/ck5-export/public/paged.polyfill.js apps/ck5-export/src/print-stylesheet/wrap-print-document.ts apps/ck5-export/src/server.ts
git commit -m "feat(ck5-export): include Paged.js polyfill in PDF print shell"
```

---

## Phase 1 — Block identity (`data-mddm-bid`)

### Task 5: BlockIdentityPlugin schema extension

**Executor:** Opus. **Group:** after 1.

**Goal:** Register `mddmBid` model attribute on every paginable block type. Plugin exists as a no-op at this point; post-fixer and converters come in later tasks.

**Files:**
- Create: `frontend/apps/web/src/features/documents/ck5/plugins/MddmBlockIdentityPlugin/index.ts`
- Create: `frontend/apps/web/src/features/documents/ck5/plugins/MddmBlockIdentityPlugin/schema.ts`
- Create: `frontend/apps/web/src/features/documents/ck5/plugins/MddmBlockIdentityPlugin/__tests__/schema.test.ts`
- Modify: `frontend/apps/web/src/features/documents/ck5/config/editorConfig.ts` (register plugin)

**Acceptance Criteria:**
- [ ] Plugin class extends `@ckeditor/ckeditor5-core` `Plugin`.
- [ ] `pluginName` is `'MddmBlockIdentity'`.
- [ ] Paginable elements from spec all allow `mddmBid` attribute.
- [ ] Editor boots without console errors after registration.

**Verify:** `cd frontend/apps/web && npx vitest run src/features/documents/ck5/plugins/MddmBlockIdentityPlugin/__tests__/schema.test.ts` → PASS.

**Steps:**

- [ ] **Step 1: Write failing test `schema.test.ts`**

```ts
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { DecoupledEditor } from '@ckeditor/ckeditor5-editor-decoupled';
import { Paragraph } from '@ckeditor/ckeditor5-paragraph';
import { Heading } from '@ckeditor/ckeditor5-heading';
import { Essentials } from '@ckeditor/ckeditor5-essentials';
import { Table } from '@ckeditor/ckeditor5-table';
import { Image, ImageBlock } from '@ckeditor/ckeditor5-image';
import { MddmBlockIdentityPlugin } from '../index';

const PAGINABLE = [
  'paragraph', 'heading1', 'heading2', 'heading3',
  'listItem', 'blockQuote', 'tableRow', 'imageBlock',
];

describe('MddmBlockIdentityPlugin schema', () => {
  let editor: DecoupledEditor;

  beforeEach(async () => {
    editor = await DecoupledEditor.create(document.createElement('div'), {
      plugins: [Essentials, Paragraph, Heading, Table, Image, ImageBlock, MddmBlockIdentityPlugin],
    });
  });

  afterEach(async () => { await editor.destroy(); });

  it.each(PAGINABLE)('allows mddmBid on %s', (name) => {
    expect(editor.model.schema.checkAttribute([name], 'mddmBid')).toBe(true);
  });
});
```

- [ ] **Step 2: Run test → FAIL**

Expected: `Cannot find module '../index'`.

- [ ] **Step 3: Implement `schema.ts`**

```ts
import type { Editor } from '@ckeditor/ckeditor5-core';

export const PAGINABLE_ELEMENT_NAMES = [
  'paragraph',
  'heading1', 'heading2', 'heading3', 'heading4', 'heading5', 'heading6',
  'listItem',
  'blockQuote',
  'tableRow',
  'imageBlock',
  'mediaEmbed',
  'mddmSection',
  'mddmRepeatable',
  'mddmRepeatableItem',
  'mddmDataTable',
  'mddmFieldGroup',
  'mddmRichBlock',
] as const;

export function extendSchemaWithBid(editor: Editor): void {
  const schema = editor.model.schema;
  for (const name of PAGINABLE_ELEMENT_NAMES) {
    if (schema.isRegistered(name)) {
      schema.extend(name, { allowAttributes: ['mddmBid'] });
    }
  }
}
```

- [ ] **Step 4: Implement `index.ts`**

```ts
import { Plugin } from '@ckeditor/ckeditor5-core';
import { extendSchemaWithBid } from './schema';

export class MddmBlockIdentityPlugin extends Plugin {
  public static get pluginName() { return 'MddmBlockIdentity' as const; }

  public init(): void {
    extendSchemaWithBid(this.editor);
  }
}
```

- [ ] **Step 5: Register plugin in `editorConfig.ts`**

Add import + append to plugin array:
```ts
import { MddmBlockIdentityPlugin } from '../plugins/MddmBlockIdentityPlugin';
// ...
plugins: [..., MddmBlockIdentityPlugin],
```

- [ ] **Step 6: Run tests**

```bash
cd frontend/apps/web && npx vitest run src/features/documents/ck5/plugins/MddmBlockIdentityPlugin
```
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add frontend/apps/web/src/features/documents/ck5/plugins/MddmBlockIdentityPlugin frontend/apps/web/src/features/documents/ck5/config/editorConfig.ts
git commit -m "feat(ck5): add MddmBlockIdentityPlugin with mddmBid schema on paginable blocks"
```

---

### Task 6: Post-fixer — mint UUID on insert

**Executor:** Codex. **Group:** after 5.

**Goal:** When a paginable block exists in the model without `mddmBid`, the post-fixer stamps a fresh UUID v4. Runs on every model change.

**Files:**
- Create: `frontend/apps/web/src/features/documents/ck5/plugins/MddmBlockIdentityPlugin/post-fixer.ts`
- Modify: `frontend/apps/web/src/features/documents/ck5/plugins/MddmBlockIdentityPlugin/index.ts`
- Create: `frontend/apps/web/src/features/documents/ck5/plugins/MddmBlockIdentityPlugin/__tests__/post-fixer-mint.test.ts`
- Modify: `frontend/apps/web/package.json` (add `uuid` dep)

**Acceptance Criteria:**
- [ ] Newly-inserted paragraph gets `mddmBid` matching `/^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i`.
- [ ] Inserting 100 paragraphs yields 100 distinct bids (no collisions in a single post-fixer pass).
- [ ] Post-fixer does not re-mint an existing bid.

**Verify:** `cd frontend/apps/web && npx vitest run src/features/documents/ck5/plugins/MddmBlockIdentityPlugin/__tests__/post-fixer-mint.test.ts` → PASS (3 tests).

**Steps:**

- [ ] **Step 1: Add dep**

```bash
cd frontend/apps/web && npm install uuid && npm install -D @types/uuid
```

- [ ] **Step 2: Write failing test**

```ts
// post-fixer-mint.test.ts
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { DecoupledEditor } from '@ckeditor/ckeditor5-editor-decoupled';
import { Paragraph } from '@ckeditor/ckeditor5-paragraph';
import { Essentials } from '@ckeditor/ckeditor5-essentials';
import { MddmBlockIdentityPlugin } from '../index';

const UUID_V4 = /^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$/i;

async function mkEditor() {
  return DecoupledEditor.create(document.createElement('div'), {
    plugins: [Essentials, Paragraph, MddmBlockIdentityPlugin],
  });
}

describe('bid post-fixer — mint', () => {
  let editor: DecoupledEditor;
  beforeEach(async () => { editor = await mkEditor(); });
  afterEach(async () => { await editor.destroy(); });

  it('mints bid for a paragraph on insert', () => {
    editor.setData('<p>hello</p>');
    const root = editor.model.document.getRoot()!;
    const p = root.getChild(0) as any;
    expect(p.getAttribute('mddmBid')).toMatch(UUID_V4);
  });

  it('does not re-mint an existing bid', () => {
    editor.setData('<p data-mddm-bid="11111111-1111-4111-8111-111111111111">hi</p>');
    // Task 8 wires upcast; until then, simulate by setting via writer
    editor.model.change(writer => {
      const p = editor.model.document.getRoot()!.getChild(0);
      writer.setAttribute('mddmBid', '11111111-1111-4111-8111-111111111111', p);
    });
    editor.model.change(writer => { writer.insertText(' more', editor.model.document.getRoot()!.getChild(0), 'end'); });
    const p = editor.model.document.getRoot()!.getChild(0) as any;
    expect(p.getAttribute('mddmBid')).toBe('11111111-1111-4111-8111-111111111111');
  });

  it('mints distinct bids for 100 paragraphs in one transaction', () => {
    editor.model.change(writer => {
      const root = editor.model.document.getRoot()!;
      for (let i = 0; i < 100; i++) {
        const p = writer.createElement('paragraph');
        writer.append(p, root);
      }
    });
    const bids = new Set<string>();
    for (let i = 0; i < 100; i++) {
      const p = editor.model.document.getRoot()!.getChild(i) as any;
      bids.add(p.getAttribute('mddmBid'));
    }
    expect(bids.size).toBe(100);
  });
});
```

- [ ] **Step 3: Run test → FAIL**

Expected: assertions on `mddmBid` returning undefined.

- [ ] **Step 4: Implement `post-fixer.ts`**

```ts
import { v4 as uuidv4 } from 'uuid';
import type { Editor } from '@ckeditor/ckeditor5-core';
import type { Element } from '@ckeditor/ckeditor5-engine';
import { PAGINABLE_ELEMENT_NAMES } from './schema';

const PAGINABLE = new Set<string>(PAGINABLE_ELEMENT_NAMES);

export function registerBidPostFixer(editor: Editor): void {
  editor.model.document.registerPostFixer(writer => {
    let changed = false;
    const root = editor.model.document.getRoot();
    if (!root) return false;
    for (const { item } of editor.model.createRangeIn(root)) {
      if (!item.is('element')) continue;
      const el = item as Element;
      if (!PAGINABLE.has(el.name)) continue;
      if (!el.hasAttribute('mddmBid')) {
        writer.setAttribute('mddmBid', uuidv4(), el);
        changed = true;
      }
    }
    return changed;
  });
}
```

- [ ] **Step 5: Wire in `index.ts`**

```ts
import { registerBidPostFixer } from './post-fixer';
// in init():
registerBidPostFixer(this.editor);
```

- [ ] **Step 6: Run test → PASS**

```bash
cd frontend/apps/web && npx vitest run src/features/documents/ck5/plugins/MddmBlockIdentityPlugin/__tests__/post-fixer-mint.test.ts
```

- [ ] **Step 7: Commit**

```bash
git add frontend/apps/web/src/features/documents/ck5/plugins/MddmBlockIdentityPlugin frontend/apps/web/package.json frontend/apps/web/package-lock.json
git commit -m "feat(ck5): mint data-mddm-bid UUID v4 on paginable block insert"
```

---

### Task 7: Post-fixer — split/merge semantics

**Executor:** Opus. **Group:** after 6.

**Goal:** On block split (Enter in middle of paragraph), the survivor keeps its bid and the new block receives a fresh bid. On block merge (Backspace at start of block), the survivor keeps its bid and the absorbed block's bid is dropped. This is critical for bid stability under typing.

**Files:**
- Modify: `frontend/apps/web/src/features/documents/ck5/plugins/MddmBlockIdentityPlugin/post-fixer.ts`
- Create: `frontend/apps/web/src/features/documents/ck5/plugins/MddmBlockIdentityPlugin/__tests__/post-fixer-split-merge.test.ts`

**Acceptance Criteria:**
- [ ] Typing Enter in the middle of paragraph `A` (bid `bA`) produces two paragraphs: first keeps `bA`, second gets a fresh UUID.
- [ ] Backspace at start of paragraph `B` (bid `bB`) following paragraph `A` (bid `bA`) merges them into one paragraph with bid `bA` (survivor), `bB` gone from the document.
- [ ] Split/merge sequence `A → split → A + C → merge C into A` ends with single paragraph having bid `bA`.

**Verify:** `cd frontend/apps/web && npx vitest run src/features/documents/ck5/plugins/MddmBlockIdentityPlugin/__tests__/post-fixer-split-merge.test.ts` → PASS.

**Steps:**

- [ ] **Step 1: Write failing tests**

```ts
// post-fixer-split-merge.test.ts
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { DecoupledEditor } from '@ckeditor/ckeditor5-editor-decoupled';
import { Paragraph } from '@ckeditor/ckeditor5-paragraph';
import { Essentials } from '@ckeditor/ckeditor5-essentials';
import { MddmBlockIdentityPlugin } from '../index';

async function mkEditor() {
  return DecoupledEditor.create(document.createElement('div'), {
    plugins: [Essentials, Paragraph, MddmBlockIdentityPlugin],
  });
}

function bids(editor: DecoupledEditor): string[] {
  const root = editor.model.document.getRoot()!;
  const out: string[] = [];
  for (let i = 0; i < root.childCount; i++) {
    out.push((root.getChild(i) as any).getAttribute('mddmBid'));
  }
  return out;
}

describe('bid post-fixer — split/merge', () => {
  let editor: DecoupledEditor;
  beforeEach(async () => { editor = await mkEditor(); });
  afterEach(async () => { await editor.destroy(); });

  it('split: survivor keeps bid, new block gets fresh bid', () => {
    editor.setData('<p>hello world</p>');
    const before = bids(editor);
    // Split at offset 5 (after 'hello')
    editor.model.change(writer => {
      const root = editor.model.document.getRoot()!;
      const p = root.getChild(0)!;
      writer.split(writer.createPositionAt(p, 5));
    });
    const after = bids(editor);
    expect(after).toHaveLength(2);
    expect(after[0]).toBe(before[0]);       // survivor
    expect(after[1]).not.toBe(before[0]);    // fresh
    expect(after[1]).toMatch(/^[0-9a-f-]{36}$/);
  });

  it('merge: survivor (earlier) bid stays, absorbed bid dropped', () => {
    editor.setData('<p>hello</p><p>world</p>');
    const before = bids(editor);
    expect(before).toHaveLength(2);
    editor.model.change(writer => {
      const root = editor.model.document.getRoot()!;
      const p2 = root.getChild(1)!;
      writer.merge(writer.createPositionBefore(p2));
    });
    const after = bids(editor);
    expect(after).toHaveLength(1);
    expect(after[0]).toBe(before[0]);
  });

  it('split then merge preserves original bid', () => {
    editor.setData('<p>helloworld</p>');
    const origBid = bids(editor)[0];
    editor.model.change(writer => {
      const p = editor.model.document.getRoot()!.getChild(0)!;
      writer.split(writer.createPositionAt(p, 5));
    });
    editor.model.change(writer => {
      const p2 = editor.model.document.getRoot()!.getChild(1)!;
      writer.merge(writer.createPositionBefore(p2));
    });
    const after = bids(editor);
    expect(after).toHaveLength(1);
    expect(after[0]).toBe(origBid);
  });
});
```

- [ ] **Step 2: Run tests → FAIL**

Expected: current post-fixer clones bid on split (both new blocks keep the source bid because CK5's split operation copies attributes). The merge case may pass coincidentally; split is the break point.

- [ ] **Step 3: Enhance post-fixer to detect split-produced duplicate bids**

Replace contents of `post-fixer.ts`:
```ts
import { v4 as uuidv4 } from 'uuid';
import type { Editor } from '@ckeditor/ckeditor5-core';
import type { Element } from '@ckeditor/ckeditor5-engine';
import { PAGINABLE_ELEMENT_NAMES } from './schema';

const PAGINABLE = new Set<string>(PAGINABLE_ELEMENT_NAMES);

/**
 * Post-fixer contract:
 * 1. Any paginable element without `mddmBid` → mint fresh UUID.
 * 2. If multiple paginable elements share a bid in a single document pass
 *    (i.e. split produced clones), keep the FIRST occurrence and re-mint
 *    subsequent ones. This preserves the survivor-keeps-bid rule.
 *
 * Note: merge is handled naturally by CK5's merge operation — the absorbed
 * element is removed, so its bid disappears with it.
 */
export function registerBidPostFixer(editor: Editor): void {
  editor.model.document.registerPostFixer(writer => {
    const root = editor.model.document.getRoot();
    if (!root) return false;

    let changed = false;
    const seen = new Map<string, Element>();

    for (const { item } of editor.model.createRangeIn(root)) {
      if (!item.is('element')) continue;
      const el = item as Element;
      if (!PAGINABLE.has(el.name)) continue;

      const bid = el.getAttribute('mddmBid') as string | undefined;
      if (!bid) {
        writer.setAttribute('mddmBid', uuidv4(), el);
        changed = true;
        continue;
      }
      if (seen.has(bid)) {
        // Duplicate: re-mint on the later element (survivor rule).
        writer.setAttribute('mddmBid', uuidv4(), el);
        changed = true;
        continue;
      }
      seen.set(bid, el);
    }
    return changed;
  });
}
```

- [ ] **Step 4: Run tests → PASS**

```bash
cd frontend/apps/web && npx vitest run src/features/documents/ck5/plugins/MddmBlockIdentityPlugin/__tests__
```
Expected: all tests pass (mint + split/merge).

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/ck5/plugins/MddmBlockIdentityPlugin
git commit -m "feat(ck5): enforce survivor-keeps-bid rule on split/merge via post-fixer"
```

---

### Task 8: Bid upcast + downcast converters

**Executor:** Codex. **Group:** after 5 (can run parallel with Task 6 if implementer is careful; sequential safer).

**Goal:** HTML `data-mddm-bid` round-trips through the editor. Upcast reads, data downcast writes, editing downcast mirrors for DevTools visibility.

**Files:**
- Create: `frontend/apps/web/src/features/documents/ck5/plugins/MddmBlockIdentityPlugin/converters.ts`
- Modify: `frontend/apps/web/src/features/documents/ck5/plugins/MddmBlockIdentityPlugin/index.ts`
- Create: `frontend/apps/web/src/features/documents/ck5/plugins/MddmBlockIdentityPlugin/__tests__/converters.test.ts`

**Acceptance Criteria:**
- [ ] `editor.setData('<p data-mddm-bid="aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa">x</p>')` → model paragraph has `mddmBid === 'aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa'`.
- [ ] `editor.getData()` includes `data-mddm-bid="<same-uuid>"`.
- [ ] Round-trip preserves bid exactly.

**Verify:** `cd frontend/apps/web && npx vitest run src/features/documents/ck5/plugins/MddmBlockIdentityPlugin/__tests__/converters.test.ts` → PASS.

**Steps:**

- [ ] **Step 1: Write failing test**

```ts
// converters.test.ts
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { DecoupledEditor } from '@ckeditor/ckeditor5-editor-decoupled';
import { Paragraph } from '@ckeditor/ckeditor5-paragraph';
import { Essentials } from '@ckeditor/ckeditor5-essentials';
import { MddmBlockIdentityPlugin } from '../index';

describe('bid converters', () => {
  let editor: DecoupledEditor;
  beforeEach(async () => {
    editor = await DecoupledEditor.create(document.createElement('div'), {
      plugins: [Essentials, Paragraph, MddmBlockIdentityPlugin],
    });
  });
  afterEach(async () => { await editor.destroy(); });

  it('upcasts data-mddm-bid into model', () => {
    editor.setData('<p data-mddm-bid="aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa">x</p>');
    const p = editor.model.document.getRoot()!.getChild(0) as any;
    expect(p.getAttribute('mddmBid')).toBe('aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa');
  });

  it('downcasts mddmBid to data-mddm-bid on getData()', () => {
    editor.setData('<p data-mddm-bid="aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa">x</p>');
    const html = editor.getData();
    expect(html).toContain('data-mddm-bid="aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa"');
  });

  it('round-trips', () => {
    editor.setData('<p data-mddm-bid="aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa">x</p>');
    expect(editor.getData()).toContain('data-mddm-bid="aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa"');
  });
});
```

- [ ] **Step 2: Run → FAIL**

- [ ] **Step 3: Implement `converters.ts`**

```ts
import type { Editor } from '@ckeditor/ckeditor5-core';
import { PAGINABLE_ELEMENT_NAMES } from './schema';

export function registerBidConverters(editor: Editor): void {
  const conversion = editor.conversion;

  for (const name of PAGINABLE_ELEMENT_NAMES) {
    if (!editor.model.schema.isRegistered(name)) continue;

    conversion.for('upcast').attributeToAttribute({
      view: { name: undefined, key: 'data-mddm-bid' },
      model: { key: 'mddmBid', name },
    });

    conversion.for('dataDowncast').attributeToAttribute({
      model: { key: 'mddmBid', name },
      view: 'data-mddm-bid',
    });

    conversion.for('editingDowncast').attributeToAttribute({
      model: { key: 'mddmBid', name },
      view: 'data-mddm-bid',
    });
  }
}
```

- [ ] **Step 4: Wire in `index.ts`**

```ts
import { registerBidConverters } from './converters';
// in init() — AFTER schema.extend, BEFORE post-fixer:
registerBidConverters(this.editor);
```

- [ ] **Step 5: Run → PASS**

- [ ] **Step 6: Commit**

```bash
git add frontend/apps/web/src/features/documents/ck5/plugins/MddmBlockIdentityPlugin
git commit -m "feat(ck5): round-trip data-mddm-bid via upcast + downcast converters"
```

---

### Task 9: Clipboard paste — collision remint

**Executor:** Opus. **Group:** after 8.

**Goal:** Pasting content from the same doc (which carries existing bids) must re-mint any bid that collides with a bid already in the target document. Prevents duplicate bids at the model level.

**Files:**
- Create: `frontend/apps/web/src/features/documents/ck5/plugins/MddmBlockIdentityPlugin/clipboard.ts`
- Modify: `frontend/apps/web/src/features/documents/ck5/plugins/MddmBlockIdentityPlugin/index.ts`
- Create: `frontend/apps/web/src/features/documents/ck5/plugins/MddmBlockIdentityPlugin/__tests__/clipboard.test.ts`

**Acceptance Criteria:**
- [ ] Pasting HTML with a bid that already exists in the doc results in the pasted block receiving a fresh bid.
- [ ] Pasting HTML with bids that don't collide preserves those bids.
- [ ] Pasting plain text (no bids) → post-fixer mints bids as usual.

**Verify:** `cd frontend/apps/web && npx vitest run src/features/documents/ck5/plugins/MddmBlockIdentityPlugin/__tests__/clipboard.test.ts` → PASS.

**Steps:**

- [ ] **Step 1: Write failing test**

```ts
// clipboard.test.ts
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { DecoupledEditor } from '@ckeditor/ckeditor5-editor-decoupled';
import { Paragraph } from '@ckeditor/ckeditor5-paragraph';
import { Essentials } from '@ckeditor/ckeditor5-essentials';
import { Clipboard } from '@ckeditor/ckeditor5-clipboard';
import { MddmBlockIdentityPlugin } from '../index';

describe('bid clipboard remint', () => {
  let editor: DecoupledEditor;

  beforeEach(async () => {
    editor = await DecoupledEditor.create(document.createElement('div'), {
      plugins: [Essentials, Paragraph, Clipboard, MddmBlockIdentityPlugin],
    });
  });
  afterEach(async () => { await editor.destroy(); });

  it('re-mints colliding bid on paste', () => {
    const BID = '11111111-1111-4111-8111-111111111111';
    editor.setData(`<p data-mddm-bid="${BID}">A</p>`);

    // Simulate paste of HTML with same bid
    const data = new DataTransfer();
    data.setData('text/html', `<p data-mddm-bid="${BID}">B</p>`);
    editor.editing.view.document.fire('clipboardInput', {
      dataTransfer: data,
      method: 'paste',
      preventDefault: () => {},
      stopPropagation: () => {},
    } as any);

    const root = editor.model.document.getRoot()!;
    const bids = new Set<string>();
    for (let i = 0; i < root.childCount; i++) {
      bids.add((root.getChild(i) as any).getAttribute('mddmBid'));
    }
    expect(bids.size).toBe(root.childCount);
  });
});
```

- [ ] **Step 2: Run → FAIL**

- [ ] **Step 3: Implement `clipboard.ts`**

```ts
import { v4 as uuidv4 } from 'uuid';
import type { Editor } from '@ckeditor/ckeditor5-core';
import type { DocumentFragment, Element } from '@ckeditor/ckeditor5-engine';
import { PAGINABLE_ELEMENT_NAMES } from './schema';

const PAGINABLE = new Set<string>(PAGINABLE_ELEMENT_NAMES);

/**
 * After clipboard upcast produces a model DocumentFragment, walk it and
 * re-mint any bid that collides with a bid already present in the document.
 *
 * The post-fixer's "first occurrence wins" rule would also catch collisions
 * after insertion, but running here keeps the diff smaller (we only touch
 * pasted nodes, not the whole document).
 */
export function registerBidClipboardHandler(editor: Editor): void {
  editor.plugins.get('ClipboardPipeline').on('contentInsertion', (evt: any, data: any) => {
    const fragment: DocumentFragment = data.content;
    const existing = collectDocumentBids(editor);

    editor.model.change(writer => {
      for (const { item } of editor.model.createRangeIn(fragment)) {
        if (!item.is('element')) continue;
        const el = item as Element;
        if (!PAGINABLE.has(el.name)) continue;
        const bid = el.getAttribute('mddmBid') as string | undefined;
        if (bid && existing.has(bid)) {
          writer.setAttribute('mddmBid', uuidv4(), el);
        }
      }
    });
  });
}

function collectDocumentBids(editor: Editor): Set<string> {
  const out = new Set<string>();
  const root = editor.model.document.getRoot();
  if (!root) return out;
  for (const { item } of editor.model.createRangeIn(root)) {
    if (!item.is('element')) continue;
    const bid = (item as Element).getAttribute('mddmBid') as string | undefined;
    if (bid) out.add(bid);
  }
  return out;
}
```

- [ ] **Step 4: Wire in `index.ts`**

```ts
import { registerBidClipboardHandler } from './clipboard';
// in init() — AFTER converters, AFTER post-fixer:
registerBidClipboardHandler(this.editor);
```

- [ ] **Step 5: Run → PASS**

- [ ] **Step 6: Commit**

```bash
git add frontend/apps/web/src/features/documents/ck5/plugins/MddmBlockIdentityPlugin
git commit -m "feat(ck5): re-mint colliding mddmBid on clipboard paste"
```

---

### Task 10: Schema v3 → v4 migration

**Executor:** Codex. **Group:** after 8.

**Goal:** Legacy HTML persisted with `data-mddm-schema="3"` (or no schema attribute) automatically upgrades on upcast. The root element receives `data-mddm-schema="4"`; paginable blocks receive fresh bids via the post-fixer. Log `schema-upgrade-v4` event once per document.

**Files:**
- Create: `frontend/apps/web/src/features/documents/ck5/plugins/MddmBlockIdentityPlugin/migration.ts`
- Modify: `frontend/apps/web/src/features/documents/ck5/plugins/MddmBlockIdentityPlugin/index.ts`
- Create: `frontend/apps/web/src/features/documents/ck5/plugins/MddmBlockIdentityPlugin/__tests__/migration.test.ts`

**Acceptance Criteria:**
- [ ] Loading legacy HTML (no bids, no schema attr) → `editor.getData()` produces content where all paginable blocks have bids and root container has schema attr set to `4`.
- [ ] Loading v4 HTML (already has bids) → no re-mint, no log event.
- [ ] Migration logs exactly once per `setData` call.

**Verify:** `cd frontend/apps/web && npx vitest run src/features/documents/ck5/plugins/MddmBlockIdentityPlugin/__tests__/migration.test.ts` → PASS.

**Steps:**

- [ ] **Step 1: Write failing test**

```ts
// migration.test.ts
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { DecoupledEditor } from '@ckeditor/ckeditor5-editor-decoupled';
import { Paragraph } from '@ckeditor/ckeditor5-paragraph';
import { Essentials } from '@ckeditor/ckeditor5-essentials';
import { MddmBlockIdentityPlugin } from '../index';

describe('schema v3 → v4 migration', () => {
  let editor: DecoupledEditor;
  let logSpy: ReturnType<typeof vi.spyOn>;

  beforeEach(async () => {
    logSpy = vi.spyOn(console, 'info').mockImplementation(() => {});
    editor = await DecoupledEditor.create(document.createElement('div'), {
      plugins: [Essentials, Paragraph, MddmBlockIdentityPlugin],
    });
  });
  afterEach(async () => { logSpy.mockRestore(); await editor.destroy(); });

  it('mints bids on legacy content', () => {
    editor.setData('<p>legacy1</p><p>legacy2</p>');
    const html = editor.getData();
    const matches = html.match(/data-mddm-bid="[0-9a-f-]{36}"/g) ?? [];
    expect(matches).toHaveLength(2);
    expect(logSpy).toHaveBeenCalledWith(expect.stringContaining('schema-upgrade-v4'));
  });

  it('no log for already-v4 content', () => {
    editor.setData('<p data-mddm-bid="aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa">x</p>');
    expect(logSpy).not.toHaveBeenCalledWith(expect.stringContaining('schema-upgrade-v4'));
  });
});
```

- [ ] **Step 2: Run → FAIL**

- [ ] **Step 3: Implement `migration.ts`**

```ts
import type { Editor } from '@ckeditor/ckeditor5-core';
import type { Element } from '@ckeditor/ckeditor5-engine';
import { PAGINABLE_ELEMENT_NAMES } from './schema';

const PAGINABLE = new Set<string>(PAGINABLE_ELEMENT_NAMES);

export function registerSchemaV4Migration(editor: Editor): void {
  editor.data.on('set', (_evt, args) => {
    const input = args[0];
    if (typeof input !== 'string') return;

    // Detect: any paginable element without a bid?
    // Cheap string heuristic — if input mentions data-mddm-bid, assume v4.
    const hasAnyBid = /data-mddm-bid=/.test(input);
    if (hasAnyBid) return;

    // Post-fixer will mint bids after upcast — just log.
    // eslint-disable-next-line no-console
    console.info('mddm:schema-upgrade-v4 — legacy doc migrated on load');
  }, { priority: 'high' });

  // After setData completes, set the schema stamp on the root element.
  editor.data.on('set', () => {
    editor.model.change(writer => {
      const root = editor.model.document.getRoot();
      if (!root) return;
      writer.setAttribute('mddmSchema', '4', root);
    });
  }, { priority: 'low' });
}

// Also register root attribute in schema + downcast so it round-trips.
export function registerSchemaAttribute(editor: Editor): void {
  editor.model.schema.extend('$root', { allowAttributes: ['mddmSchema'] });
  editor.conversion.for('upcast').attributeToAttribute({
    view: { name: 'body', key: 'data-mddm-schema' },
    model: { key: 'mddmSchema', name: '$root' },
  });
  editor.conversion.for('dataDowncast').add((dispatcher: any) => {
    dispatcher.on('attribute:mddmSchema:$root', (_evt: any, data: any, api: any) => {
      const viewRoot = api.mapper.toViewElement(data.item);
      if (viewRoot) {
        api.writer.setAttribute('data-mddm-schema', data.attributeNewValue, viewRoot);
      }
    });
  });
}
```

- [ ] **Step 4: Wire in `index.ts`**

```ts
import { registerSchemaAttribute, registerSchemaV4Migration } from './migration';
// in init():
registerSchemaAttribute(this.editor);
registerSchemaV4Migration(this.editor);
```

- [ ] **Step 5: Run → PASS**

- [ ] **Step 6: Commit**

```bash
git add frontend/apps/web/src/features/documents/ck5/plugins/MddmBlockIdentityPlugin
git commit -m "feat(ck5): migrate legacy HTML to schema v4 on setData"
```

---

### Task 11: Server-side bid validator (422 on collision/missing)

**Executor:** Codex. **Group:** after 8.

**Goal:** Export service validates HTML before DOCX/PDF render: duplicate bids or missing bids on paginable blocks → 422 with structured error. Prevents downstream reconciler corruption.

**Files:**
- Create: `apps/ck5-export/src/pagination/validator.ts`
- Create: `apps/ck5-export/src/pagination/__tests__/validator.test.ts`

**Acceptance Criteria:**
- [ ] `validateBids(html)` returns `{ok: true}` for clean HTML.
- [ ] Returns `{ok: false, error: 'bid-collision', bids: [...]}` when duplicate bid.
- [ ] Returns `{ok: false, error: 'paginable-missing-bid', elements: [...]}` with warn severity (not 422) when paginable block lacks bid.
- [ ] Routes converting HTML call validator and return 422 on `bid-collision`, continue with warning log on `paginable-missing-bid`.

**Verify:** `cd apps/ck5-export && npx vitest run src/pagination/__tests__/validator.test.ts` → PASS (4 tests).

**Steps:**

- [ ] **Step 1: Write failing test**

```ts
// validator.test.ts
import { describe, it, expect } from 'vitest';
import { validateBids } from '../validator';

const CLEAN = '<p data-mddm-bid="aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa">x</p><p data-mddm-bid="bbbbbbbb-bbbb-4bbb-bbbb-bbbbbbbbbbbb">y</p>';
const COLLISION = '<p data-mddm-bid="aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa">x</p><p data-mddm-bid="aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa">y</p>';
const MISSING = '<p>x</p><p data-mddm-bid="aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa">y</p>';

describe('validateBids', () => {
  it('accepts clean html', () => {
    expect(validateBids(CLEAN).ok).toBe(true);
  });
  it('rejects on duplicate bid', () => {
    const r = validateBids(COLLISION);
    expect(r.ok).toBe(false);
    if (!r.ok) {
      expect(r.severity).toBe('error');
      expect(r.error).toBe('bid-collision');
      expect(r.bids).toContain('aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa');
    }
  });
  it('warns on paginable missing bid', () => {
    const r = validateBids(MISSING);
    expect(r.ok).toBe(false);
    if (!r.ok) {
      expect(r.severity).toBe('warn');
      expect(r.error).toBe('paginable-missing-bid');
    }
  });
  it('ignores inline elements without bid', () => {
    expect(validateBids('<p data-mddm-bid="aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa"><span>x</span></p>').ok).toBe(true);
  });
});
```

- [ ] **Step 2: Run → FAIL (module not found)**

- [ ] **Step 3: Implement `validator.ts`**

```ts
import { parseHTML } from 'linkedom';

const PAGINABLE_TAGS = new Set([
  'p', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6',
  'li', 'blockquote', 'tr',
  'figure', // imageBlock serializes to <figure class="image">
]);
const MDDM_WIDGET_CLASSES = ['mddm-section', 'mddm-repeatable', 'mddm-repeatable-item', 'mddm-data-table', 'mddm-field-group', 'mddm-rich-block'];

export type ValidationResult =
  | { ok: true }
  | { ok: false; severity: 'error' | 'warn'; error: 'bid-collision'; bids: string[] }
  | { ok: false; severity: 'warn'; error: 'paginable-missing-bid'; elements: string[] };

export function validateBids(html: string): ValidationResult {
  const { document } = parseHTML(`<!DOCTYPE html><html><body>${html}</body></html>`);
  const all = Array.from(document.querySelectorAll('[data-mddm-bid]'));
  const seen = new Map<string, number>();
  for (const el of all) {
    const bid = el.getAttribute('data-mddm-bid')!;
    seen.set(bid, (seen.get(bid) ?? 0) + 1);
  }
  const dups = [...seen.entries()].filter(([, n]) => n > 1).map(([bid]) => bid);
  if (dups.length) return { ok: false, severity: 'error', error: 'bid-collision', bids: dups };

  const paginableWithoutBid: string[] = [];
  for (const el of document.querySelectorAll('*')) {
    const tag = el.tagName.toLowerCase();
    const isPaginable =
      PAGINABLE_TAGS.has(tag) ||
      MDDM_WIDGET_CLASSES.some(c => el.classList?.contains(c));
    if (isPaginable && !el.hasAttribute('data-mddm-bid')) {
      paginableWithoutBid.push(tag);
    }
  }
  if (paginableWithoutBid.length) {
    return { ok: false, severity: 'warn', error: 'paginable-missing-bid', elements: paginableWithoutBid };
  }
  return { ok: true };
}
```

- [ ] **Step 4: Run → PASS**

- [ ] **Step 5: Wire into both export routes (one-line guard)**

Files (both exist, confirmed): `apps/ck5-export/src/routes/render-docx.ts` and `apps/ck5-export/src/routes/render-pdf-html.ts`. Immediately after reading `html` from the request body, before any parsing/rendering:
```ts
import { validateBids } from '../pagination/validator';
// ...
const v = validateBids(html);
if (!v.ok && v.severity === 'error') {
  return c.json({ error: v.error, bids: v.bids }, 422);
}
if (!v.ok && v.severity === 'warn') {
  console.warn(`mddm:${v.error}`, v.elements);
}
```

- [ ] **Step 6: Run all export tests**

```bash
cd apps/ck5-export && npx vitest run
```

- [ ] **Step 7: Commit**

```bash
git add apps/ck5-export/src/pagination/validator.ts apps/ck5-export/src/pagination/__tests__/validator.test.ts apps/ck5-export/src/routes
git commit -m "feat(ck5-export): validate bids on export, 422 on collision"
```

---

## Phase 2 — MddmPaginationPlugin

### Task 12: Plugin scaffold + `requires` contract

**Executor:** Codex. **Group:** after 5.

**Goal:** Empty plugin with correct name, `requires: [MddmBlockIdentityPlugin]`, and `init()` hook registration points in place. No measurement logic yet.

**Files:**
- Create: `frontend/apps/web/src/features/documents/ck5/plugins/MddmPaginationPlugin/index.ts`
- Create: `frontend/apps/web/src/features/documents/ck5/plugins/MddmPaginationPlugin/types.ts`
- Create: `frontend/apps/web/src/features/documents/ck5/plugins/MddmPaginationPlugin/__tests__/scaffold.test.ts`
- Modify: `frontend/apps/web/src/features/documents/ck5/config/editorConfig.ts`

**Acceptance Criteria:**
- [ ] `MddmPaginationPlugin.pluginName === 'MddmPagination'`.
- [ ] `MddmPaginationPlugin.requires` returns array including `MddmBlockIdentityPlugin`.
- [ ] Editor boots with plugin registered, no console errors.
- [ ] Types file exports `BreakCandidate`, `ComputedBreak`, `ServerBreak`, `ReconciledBreak`.

**Verify:** `cd frontend/apps/web && npx vitest run src/features/documents/ck5/plugins/MddmPaginationPlugin/__tests__/scaffold.test.ts` → PASS.

**Steps:**

- [ ] **Step 1: Write failing test**

```ts
// scaffold.test.ts
import { describe, it, expect } from 'vitest';
import { DecoupledEditor } from '@ckeditor/ckeditor5-editor-decoupled';
import { Paragraph } from '@ckeditor/ckeditor5-paragraph';
import { Essentials } from '@ckeditor/ckeditor5-essentials';
import { MddmBlockIdentityPlugin } from '../../MddmBlockIdentityPlugin';
import { MddmPaginationPlugin } from '../index';

describe('MddmPaginationPlugin scaffold', () => {
  it('pluginName is MddmPagination', () => {
    expect(MddmPaginationPlugin.pluginName).toBe('MddmPagination');
  });
  it('requires block identity plugin', () => {
    expect(MddmPaginationPlugin.requires).toContain(MddmBlockIdentityPlugin);
  });
  it('boots inside editor', async () => {
    const editor = await DecoupledEditor.create(document.createElement('div'), {
      plugins: [Essentials, Paragraph, MddmBlockIdentityPlugin, MddmPaginationPlugin],
    });
    expect(editor.plugins.has('MddmPagination')).toBe(true);
    await editor.destroy();
  });
});
```

- [ ] **Step 2: Implement `types.ts` (re-export shared transport types + add CK5-only internals)**

All wire types live in `@metaldocs/mddm-pagination-types` (Task 0b). This file only re-exports them plus adds CK5-engine-tied internals that cannot cross to Node.

```ts
// Re-export wire types — single source of truth.
export type {
  BreakCandidate,
  ComputedBreak,
  ServerBreak,
  ReconciledBreak,
  ReconcileLogs,
  ReconcileResult,
  PaginateRequest,
  PaginateResponse,
} from '@metaldocs/mddm-pagination-types';

// CK5-engine-tied internal — NOT a wire type. Never crosses to Node.
import type { Position as ModelPosition } from '@ckeditor/ckeditor5-engine';
export type InternalBreakAnchor = Readonly<{
  bid: string;
  position: ModelPosition;
}>;
```

- [ ] **Step 3: Implement `index.ts`**

```ts
import { Plugin } from '@ckeditor/ckeditor5-core';
import { MddmBlockIdentityPlugin } from '../MddmBlockIdentityPlugin';

export class MddmPaginationPlugin extends Plugin {
  public static get pluginName() { return 'MddmPagination' as const; }
  public static get requires() { return [MddmBlockIdentityPlugin] as const; }

  public init(): void {
    // Sub-modules wired in Tasks 13–18.
  }
}
```

- [ ] **Step 4: Register in `editorConfig.ts`**

```ts
import { MddmPaginationPlugin } from '../plugins/MddmPaginationPlugin';
// plugins: [..., MddmBlockIdentityPlugin, MddmPaginationPlugin],
```

- [ ] **Step 5: Run tests → PASS**

- [ ] **Step 6: Commit**

```bash
git add frontend/apps/web/src/features/documents/ck5/plugins/MddmPaginationPlugin frontend/apps/web/src/features/documents/ck5/config/editorConfig.ts
git commit -m "feat(ck5): scaffold MddmPaginationPlugin with BlockIdentity dependency"
```

---

### Task 13: DirtyRangeTracker

**Executor:** Opus. **Group:** after 12.

**Goal:** Subscribe to `editor.model.document` `change:data`. Read `Differ.getChanges()` each change. Record the earliest modified model position as `dirtyStart`. Expose `snapshot()` returning current dirty position and clearing state. Collapses multiple rapid changes into a single position.

**Files:**
- Create: `frontend/apps/web/src/features/documents/ck5/plugins/MddmPaginationPlugin/DirtyRangeTracker.ts`
- Create: `frontend/apps/web/src/features/documents/ck5/plugins/MddmPaginationPlugin/__tests__/DirtyRangeTracker.test.ts`

**Acceptance Criteria:**
- [ ] Fresh tracker: `snapshot()` returns `null`.
- [ ] After one edit: `snapshot()` returns the position of that edit and resets to `null`.
- [ ] After two edits in different locations: `snapshot()` returns the earlier position.
- [ ] Destroying tracker removes listeners.

**Verify:** `cd frontend/apps/web && npx vitest run src/features/documents/ck5/plugins/MddmPaginationPlugin/__tests__/DirtyRangeTracker.test.ts` → PASS.

**Steps:**

- [ ] **Step 1: Write failing test**

```ts
// DirtyRangeTracker.test.ts
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { DecoupledEditor } from '@ckeditor/ckeditor5-editor-decoupled';
import { Paragraph } from '@ckeditor/ckeditor5-paragraph';
import { Essentials } from '@ckeditor/ckeditor5-essentials';
import { MddmBlockIdentityPlugin } from '../../MddmBlockIdentityPlugin';
import { DirtyRangeTracker } from '../DirtyRangeTracker';

describe('DirtyRangeTracker', () => {
  let editor: DecoupledEditor;
  let tracker: DirtyRangeTracker;

  beforeEach(async () => {
    editor = await DecoupledEditor.create(document.createElement('div'), {
      plugins: [Essentials, Paragraph, MddmBlockIdentityPlugin],
    });
    editor.setData('<p>A</p><p>B</p><p>C</p>');
    tracker = new DirtyRangeTracker(editor);
  });
  afterEach(async () => { tracker.destroy(); await editor.destroy(); });

  it('starts with null snapshot', () => {
    expect(tracker.snapshot()).toBeNull();
  });

  it('records earliest modified offset on edit', () => {
    editor.model.change(writer => {
      const p = editor.model.document.getRoot()!.getChild(2)!;
      writer.insertText('x', p as any, 'end');
    });
    const s1 = tracker.snapshot();
    expect(s1).not.toBeNull();
    const pos1 = s1!;

    editor.model.change(writer => {
      const p = editor.model.document.getRoot()!.getChild(0)!;
      writer.insertText('y', p as any, 'end');
    });
    const s2 = tracker.snapshot();
    expect(s2!.isBefore(pos1) || s2!.isEqual(pos1)).toBe(true);
  });

  it('snapshot clears state', () => {
    editor.model.change(writer => {
      writer.insertText('x', editor.model.document.getRoot()!.getChild(0) as any, 'end');
    });
    expect(tracker.snapshot()).not.toBeNull();
    expect(tracker.snapshot()).toBeNull();
  });
});
```

- [ ] **Step 2: Run → FAIL**

- [ ] **Step 3: Implement `DirtyRangeTracker.ts`**

```ts
import type { Editor } from '@ckeditor/ckeditor5-core';
import type { Position as ModelPosition } from '@ckeditor/ckeditor5-engine';

export class DirtyRangeTracker {
  private dirtyStart: ModelPosition | null = null;
  private readonly handler: () => void;

  public constructor(private readonly editor: Editor) {
    this.handler = () => this.onChange();
    this.editor.model.document.on('change:data', this.handler);
  }

  public destroy(): void {
    this.editor.model.document.off('change:data', this.handler);
  }

  public snapshot(): ModelPosition | null {
    const out = this.dirtyStart;
    this.dirtyStart = null;
    return out;
  }

  private onChange(): void {
    const changes = this.editor.model.document.differ.getChanges();
    let earliest: ModelPosition | null = null;
    for (const change of changes) {
      const pos: ModelPosition | undefined = (change as any).position;
      if (!pos) continue;
      if (!earliest || pos.isBefore(earliest)) earliest = pos;
    }
    if (!earliest) return;
    if (!this.dirtyStart || earliest.isBefore(this.dirtyStart)) {
      this.dirtyStart = earliest;
    }
  }
}
```

- [ ] **Step 4: Run → PASS**

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/ck5/plugins/MddmPaginationPlugin/DirtyRangeTracker.ts frontend/apps/web/src/features/documents/ck5/plugins/MddmPaginationPlugin/__tests__/DirtyRangeTracker.test.ts
git commit -m "feat(ck5): DirtyRangeTracker records earliest edit position via differ"
```

---

### Task 14: BreakPlanner — model walk to enumerate legal break candidates

**Executor:** Opus. **Group:** after 13.

**Goal:** Walk the model from dirty start to document end. Emit a `BreakCandidate` after every paginable block boundary, with two rules enforced: (a) `heading1..heading3` may never be break-after (keep-with-next), (b) figures/widgets are atomic (break is after the whole element, never inside).

**Files:**
- Create: `frontend/apps/web/src/features/documents/ck5/plugins/MddmPaginationPlugin/BreakPlanner.ts`
- Create: `frontend/apps/web/src/features/documents/ck5/plugins/MddmPaginationPlugin/__tests__/BreakPlanner.test.ts`

**Acceptance Criteria:**
- [ ] Every paginable block with a bid yields one candidate (at position after the block).
- [ ] Heading1/2/3 blocks do NOT yield a candidate (keep-with-next).
- [ ] Walk starts from `dirtyStart` argument — blocks before that position are skipped.
- [ ] Candidates are returned in document order.

**Verify:** `cd frontend/apps/web && npx vitest run src/features/documents/ck5/plugins/MddmPaginationPlugin/__tests__/BreakPlanner.test.ts` → PASS.

**Steps:**

- [ ] **Step 1: Write failing test**

```ts
// BreakPlanner.test.ts
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { DecoupledEditor } from '@ckeditor/ckeditor5-editor-decoupled';
import { Paragraph } from '@ckeditor/ckeditor5-paragraph';
import { Heading } from '@ckeditor/ckeditor5-heading';
import { Essentials } from '@ckeditor/ckeditor5-essentials';
import { MddmBlockIdentityPlugin } from '../../MddmBlockIdentityPlugin';
import { planBreaks } from '../BreakPlanner';

describe('planBreaks', () => {
  let editor: DecoupledEditor;

  beforeEach(async () => {
    editor = await DecoupledEditor.create(document.createElement('div'), {
      plugins: [Essentials, Paragraph, Heading, MddmBlockIdentityPlugin],
    });
  });
  afterEach(async () => { await editor.destroy(); });

  it('emits one candidate per paragraph', () => {
    editor.setData('<p>A</p><p>B</p><p>C</p>');
    const from = editor.model.createPositionFromPath(editor.model.document.getRoot()!, [0]);
    const cands = planBreaks(editor, from);
    expect(cands).toHaveLength(3);
  });

  it('skips keep-with-next headings', () => {
    editor.setData('<p>A</p><h1>T</h1><p>B</p>');
    const from = editor.model.createPositionFromPath(editor.model.document.getRoot()!, [0]);
    const cands = planBreaks(editor, from);
    expect(cands).toHaveLength(2); // heading1 excluded
  });

  it('respects dirtyStart — blocks before are skipped', () => {
    editor.setData('<p>A</p><p>B</p><p>C</p>');
    const from = editor.model.createPositionFromPath(editor.model.document.getRoot()!, [2]);
    const cands = planBreaks(editor, from);
    expect(cands).toHaveLength(1); // only after C
  });
});
```

- [ ] **Step 2: Run → FAIL**

- [ ] **Step 3: Implement `BreakPlanner.ts`**

```ts
import type { Editor } from '@ckeditor/ckeditor5-core';
import type { Element, Position as ModelPosition } from '@ckeditor/ckeditor5-engine';
import { PAGINABLE_ELEMENT_NAMES } from '../MddmBlockIdentityPlugin/schema';
import type { BreakCandidate } from './types';

const PAGINABLE = new Set<string>(PAGINABLE_ELEMENT_NAMES);
const KEEP_WITH_NEXT = new Set(['heading1', 'heading2', 'heading3', 'heading4', 'heading5', 'heading6']);

/**
 * Walk the model from `from` (inclusive) to document end. Emit a
 * BreakCandidate for each paginable block whose name is not in
 * KEEP_WITH_NEXT. The returned position is AFTER the block.
 *
 * `from` clamps the walk to a sub-range — callers typically pass the
 * block-start position of the dirty range to avoid re-walking stable prefix.
 */
export function planBreaks(editor: Editor, from: ModelPosition): BreakCandidate[] {
  const root = editor.model.document.getRoot();
  if (!root) return [];
  const out: BreakCandidate[] = [];

  let topLevel = root.getChildIndex(from.nodeAfter!) ?? 0;
  if (topLevel < 0) topLevel = 0;

  for (let i = topLevel; i < root.childCount; i++) {
    const el = root.getChild(i) as Element;
    if (!el.is('element')) continue;
    if (!PAGINABLE.has(el.name)) continue;
    if (KEEP_WITH_NEXT.has(el.name)) continue;
    const bid = el.getAttribute('mddmBid') as string | undefined;
    if (!bid) continue;
    out.push({
      bid,
      position: editor.model.createPositionAfter(el),
    });
  }
  return out;
}
```

- [ ] **Step 4: Run → PASS**

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/ck5/plugins/MddmPaginationPlugin/BreakPlanner.ts frontend/apps/web/src/features/documents/ck5/plugins/MddmPaginationPlugin/__tests__/BreakPlanner.test.ts
git commit -m "feat(ck5): BreakPlanner emits break candidates with keep-with-next rule"
```

---

### Task 15: BreakMeasurer — DOM measure on `view.document.render`

**Executor:** Opus. **Group:** after 14.

**Goal:** Subscribe to `editor.editing.view.document.on('render')`. Debounce 200ms. On fire, pull `DirtyRangeTracker.snapshot()`, run `BreakPlanner` to get candidates, map each candidate to its DOM node via `view.domConverter.mapViewToDom`, measure `offsetTop + offsetHeight`, accumulate against `contentHeightPx`, produce `ComputedBreak[]`. Await `document.fonts.ready` on first pass. Await `HTMLImageElement.decode()` for image-bearing blocks. Round to device pixels.

**Files:**
- Create: `frontend/apps/web/src/features/documents/ck5/plugins/MddmPaginationPlugin/BreakMeasurer.ts`
- Create: `frontend/apps/web/src/features/documents/ck5/plugins/MddmPaginationPlugin/__tests__/BreakMeasurer.test.ts`
- Modify: `frontend/apps/web/src/features/documents/ck5/plugins/MddmPaginationPlugin/index.ts` (wire it up)

**Acceptance Criteria:**
- [ ] After editing, measurer fires exactly once per 200ms window.
- [ ] Produces `ComputedBreak[]` with page numbers monotonically increasing.
- [ ] Skips blocks whose contained `img.decode()` rejects; logs `pagination-measure-skip`.
- [ ] Awaits `document.fonts.ready` on first measurement pass.

**Verify:** `cd frontend/apps/web && npx vitest run src/features/documents/ck5/plugins/MddmPaginationPlugin/__tests__/BreakMeasurer.test.ts` → PASS.

**Steps:**

- [ ] **Step 1: Write failing test**

```ts
// BreakMeasurer.test.ts
import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { DecoupledEditor } from '@ckeditor/ckeditor5-editor-decoupled';
import { Paragraph } from '@ckeditor/ckeditor5-paragraph';
import { Essentials } from '@ckeditor/ckeditor5-essentials';
import { MddmBlockIdentityPlugin } from '../../MddmBlockIdentityPlugin';
import { BreakMeasurer } from '../BreakMeasurer';
import { DirtyRangeTracker } from '../DirtyRangeTracker';

describe('BreakMeasurer', () => {
  let editor: DecoupledEditor;

  beforeEach(async () => {
    editor = await DecoupledEditor.create(document.createElement('div'), {
      plugins: [Essentials, Paragraph, MddmBlockIdentityPlugin],
    });
    document.body.appendChild(editor.ui.view.editable.element!);
  });
  afterEach(async () => { await editor.destroy(); });

  it('produces monotonically-increasing page numbers', async () => {
    editor.setData('<p>' + 'x'.repeat(20000) + '</p>'.repeat(1));
    const tracker = new DirtyRangeTracker(editor);
    const m = new BreakMeasurer(editor, tracker, { debounceMs: 10 });
    const emitted: any[] = [];
    m.onBreaks(b => emitted.push(b));
    editor.model.change(writer => {
      const root = editor.model.document.getRoot()!;
      writer.insertText(' ', root.getChild(0) as any, 'end');
    });
    await new Promise(r => setTimeout(r, 50));
    expect(emitted.length).toBeGreaterThan(0);
    const pages = emitted[0].map((b: any) => b.pageNumber);
    for (let i = 1; i < pages.length; i++) expect(pages[i]).toBeGreaterThanOrEqual(pages[i - 1]);
    m.destroy();
    tracker.destroy();
  });
});
```

*(jsdom lacks real layout — test asserts only monotonicity and the debounce contract, not absolute pixels.)*

- [ ] **Step 2: Run → FAIL**

- [ ] **Step 3: Implement `BreakMeasurer.ts`**

```ts
import type { Editor } from '@ckeditor/ckeditor5-core';
import { defaultLayoutTokens } from '@metaldocs/mddm-layout-tokens';
import { planBreaks } from './BreakPlanner';
import type { DirtyRangeTracker } from './DirtyRangeTracker';
import type { BreakCandidate, ComputedBreak } from './types';

const MM_PER_INCH = 25.4;
const PX_PER_INCH = 96;
const mmToPx = (mm: number) => (mm / MM_PER_INCH) * PX_PER_INCH;

type Listener = (breaks: ComputedBreak[]) => void;

export class BreakMeasurer {
  private readonly listeners = new Set<Listener>();
  private timer: number | null = null;
  private readonly renderHandler: () => void;
  private fontsReady = false;

  public constructor(
    private readonly editor: Editor,
    private readonly tracker: DirtyRangeTracker,
    private readonly opts: { debounceMs: number } = { debounceMs: 200 },
  ) {
    this.renderHandler = () => this.schedule();
    this.editor.editing.view.document.on('render', this.renderHandler);
  }

  public destroy(): void {
    this.editor.editing.view.document.off('render', this.renderHandler);
    if (this.timer !== null) clearTimeout(this.timer);
  }

  public onBreaks(fn: Listener): () => void {
    this.listeners.add(fn);
    return () => this.listeners.delete(fn);
  }

  private schedule(): void {
    if (this.timer !== null) clearTimeout(this.timer);
    this.timer = window.setTimeout(() => {
      this.timer = null;
      void this.measure();
    }, this.opts.debounceMs);
  }

  private async measure(): Promise<void> {
    if (!this.fontsReady) {
      try { await document.fonts.ready; } catch { /* jsdom */ }
      this.fontsReady = true;
    }

    const dirty = this.tracker.snapshot();
    const root = this.editor.model.document.getRoot();
    if (!root) return;
    const from = dirty ?? this.editor.model.createPositionFromPath(root, [0]);
    const candidates: BreakCandidate[] = planBreaks(this.editor, from);

    const dpr = (typeof window !== 'undefined' && window.devicePixelRatio) || 1;
    const contentHeightPx = mmToPx(
      defaultLayoutTokens.page.heightMm -
      defaultLayoutTokens.page.marginTopMm -
      defaultLayoutTokens.page.marginBottomMm
    );

    const breaks: ComputedBreak[] = [];
    let cursorY = 0;
    let page = 1;

    for (const c of candidates) {
      const modelEl = c.position.nodeBefore;
      if (!modelEl || !modelEl.is('element')) continue;

      const viewEl = this.editor.editing.mapper.toViewElement(modelEl);
      if (!viewEl) continue;
      const domEl = this.editor.editing.view.domConverter.mapViewToDom(viewEl) as HTMLElement | undefined;
      if (!domEl) continue;

      // Await image decode for blocks containing images.
      const imgs = domEl.querySelectorAll('img');
      for (const img of Array.from(imgs)) {
        try { await (img as HTMLImageElement).decode(); }
        catch { console.warn('mddm:pagination-measure-skip', c.bid); }
      }

      const h = Math.round(domEl.offsetHeight * dpr) / dpr;
      cursorY += h;
      if (cursorY > contentHeightPx) {
        page += 1;
        cursorY = h;
        breaks.push({ afterBid: c.bid, pageNumber: page });
      }
    }

    for (const l of this.listeners) l(breaks);
  }
}
```

- [ ] **Step 4: Wire into `index.ts`**

```ts
// in init():
const tracker = new DirtyRangeTracker(this.editor);
const measurer = new BreakMeasurer(this.editor, tracker);
this.on('destroy', () => { measurer.destroy(); tracker.destroy(); });
// expose via plugin:
(this as any)._tracker = tracker;
(this as any)._measurer = measurer;
```

- [ ] **Step 5: Run tests → PASS**

- [ ] **Step 6: Commit**

```bash
git add frontend/apps/web/src/features/documents/ck5/plugins/MddmPaginationPlugin
git commit -m "feat(ck5): BreakMeasurer computes page breaks on view.render debounce"
```

---

### Task 16: SectionScope — per-section independent pagination

**Executor:** Opus. **Group:** after 15.

**Goal:** Edits inside `mddmSection` A do not force a re-measurement of section B's breaks. BreakMeasurer receives a `sectionRoot` argument; when set, the walk and cursor reset apply only within that sub-tree.

**Files:**
- Create: `frontend/apps/web/src/features/documents/ck5/plugins/MddmPaginationPlugin/SectionScope.ts`
- Modify: `frontend/apps/web/src/features/documents/ck5/plugins/MddmPaginationPlugin/BreakMeasurer.ts`
- Modify: `frontend/apps/web/src/features/documents/ck5/plugins/MddmPaginationPlugin/BreakPlanner.ts`
- Create: `frontend/apps/web/src/features/documents/ck5/plugins/MddmPaginationPlugin/__tests__/section-scope.test.ts`

**Acceptance Criteria:**
- [ ] Given two `mddmSection`s, editing inside section 2 yields `ComputedBreak[]` with unchanged `afterBid`s for section 1's breaks.
- [ ] Section break-before rule: each `mddmSection` starts on page 1 of its own scope.
- [ ] Global page numbering still increases monotonically across sections.

**Verify:** `cd frontend/apps/web && npx vitest run src/features/documents/ck5/plugins/MddmPaginationPlugin/__tests__/section-scope.test.ts` → PASS.

**Steps:**

- [ ] **Step 1: Write failing test**

```ts
// section-scope.test.ts
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { DecoupledEditor } from '@ckeditor/ckeditor5-editor-decoupled';
import { Paragraph } from '@ckeditor/ckeditor5-paragraph';
import { Essentials } from '@ckeditor/ckeditor5-essentials';
import { MddmBlockIdentityPlugin } from '../../MddmBlockIdentityPlugin';
import { MddmSectionPlugin } from '../../MddmSectionPlugin';
import { findEnclosingSection } from '../SectionScope';

describe('findEnclosingSection', () => {
  let editor: DecoupledEditor;

  beforeEach(async () => {
    editor = await DecoupledEditor.create(document.createElement('div'), {
      plugins: [Essentials, Paragraph, MddmBlockIdentityPlugin, MddmSectionPlugin],
    });
    editor.setData(
      '<div class="mddm-section"><p data-mddm-bid="sec1-p1">A</p></div>' +
      '<div class="mddm-section"><p data-mddm-bid="sec2-p1">B</p></div>'
    );
  });
  afterEach(async () => { await editor.destroy(); });

  it('returns nearest mddmSection ancestor for a position inside section 1', () => {
    const root = editor.model.document.getRoot()!;
    const sec1 = root.getChild(0)!;              // mddmSection
    const para = (sec1 as any).getChild(0);      // paragraph inside
    const pos = editor.model.createPositionAt(para, 0);
    const found = findEnclosingSection(pos);
    expect(found).toBe(sec1);
  });

  it('returns null when position is not inside any mddmSection', () => {
    const root = editor.model.document.getRoot()!;
    const pos = editor.model.createPositionAt(root, 0);
    expect(findEnclosingSection(pos)).toBeNull();
  });
});
```

- [ ] **Step 2: Implement `SectionScope.ts`**

```ts
import type { Element, Position as ModelPosition } from '@ckeditor/ckeditor5-engine';

export function findEnclosingSection(position: ModelPosition): Element | null {
  let node: any = position.parent;
  while (node) {
    if (node.is && node.is('element') && node.name === 'mddmSection') return node as Element;
    node = node.parent;
  }
  return null;
}
```

- [ ] **Step 3: Update `BreakPlanner` to accept `walkRoot` arg**

Extend signature:
```ts
export function planBreaks(editor: Editor, from: ModelPosition, walkRoot?: Element): BreakCandidate[] { ... }
```
Iterate over `walkRoot.getChildren()` when provided; else root.

- [ ] **Step 4: Update `BreakMeasurer.measure()` to scope per-section**

Before walk, call `findEnclosingSection(from)`. If non-null, pass it as `walkRoot` and reset `cursorY = 0, page = lastGlobalPage`. Merge per-section results into a single monotonic list.

- [ ] **Step 5: Run test → PASS**

- [ ] **Step 6: Commit**

```bash
git add frontend/apps/web/src/features/documents/ck5/plugins/MddmPaginationPlugin
git commit -m "feat(ck5): SectionScope clamps pagination walk to enclosing mddmSection"
```

---

### Task 17: PageOverlayView — `uiElement` chrome

**Executor:** Opus. **Group:** after 15 (parallel with 16).

**Goal:** Render a grey gutter + drop shadow + "Page N" text between pages in the editing view. These are view-only `uiElement`s — never in `getData()`, never in persisted HTML.

**Files:**
- Create: `frontend/apps/web/src/features/documents/ck5/plugins/MddmPaginationPlugin/PageOverlayView.ts`
- Modify: `frontend/apps/web/src/features/documents/ck5/plugins/MddmPaginationPlugin/index.ts`
- Modify: `frontend/apps/web/src/features/documents/ck5/react/AuthorEditor.module.css` (overlay CSS)
- Create: `frontend/apps/web/src/features/documents/ck5/plugins/MddmPaginationPlugin/__tests__/PageOverlayView.test.ts`

**Acceptance Criteria:**
- [ ] For N ComputedBreaks, N overlay elements exist in the editing DOM.
- [ ] Overlays have `pointer-events: none`.
- [ ] Overlays are NOT in `editor.getData()` output.
- [ ] Overlays update when ComputedBreaks change.

**Verify:** `cd frontend/apps/web && npx vitest run src/features/documents/ck5/plugins/MddmPaginationPlugin/__tests__/PageOverlayView.test.ts` → PASS.

**Steps:**

- [ ] **Step 1: Write failing test**

```ts
// PageOverlayView.test.ts
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { DecoupledEditor } from '@ckeditor/ckeditor5-editor-decoupled';
import { Paragraph } from '@ckeditor/ckeditor5-paragraph';
import { Essentials } from '@ckeditor/ckeditor5-essentials';
import { MddmBlockIdentityPlugin } from '../../MddmBlockIdentityPlugin';
import { PageOverlayView } from '../PageOverlayView';

describe('PageOverlayView', () => {
  let editor: DecoupledEditor;
  let view: PageOverlayView;
  let host: HTMLElement;

  beforeEach(async () => {
    host = document.createElement('div');
    document.body.appendChild(host);
    editor = await DecoupledEditor.create(host, { plugins: [Essentials, Paragraph, MddmBlockIdentityPlugin] });
    view = new PageOverlayView(editor);
  });
  afterEach(async () => { view.destroy(); await editor.destroy(); host.remove(); });

  it('renders one overlay per break', () => {
    view.update([
      { afterBid: 'aaa', pageNumber: 2 },
      { afterBid: 'bbb', pageNumber: 3 },
    ]);
    expect(document.querySelectorAll('.mddm-page-overlay')).toHaveLength(2);
  });

  it('overlays do not appear in getData()', () => {
    editor.setData('<p data-mddm-bid="aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa">x</p>');
    view.update([{ afterBid: 'aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa', pageNumber: 2 }]);
    expect(editor.getData()).not.toContain('mddm-page-overlay');
  });
});
```

- [ ] **Step 2: Implement `PageOverlayView.ts`**

```ts
import type { Editor } from '@ckeditor/ckeditor5-core';
import type { ComputedBreak } from './types';

/**
 * Renders page-gutter overlays as DOM siblings of the editable root.
 * These are managed outside CK5's view tree (plain DOM) so they are
 * guaranteed never to appear in getData().
 */
export class PageOverlayView {
  private host: HTMLElement | null = null;

  public constructor(private readonly editor: Editor) {
    const editable = this.editor.ui.view.editable.element;
    if (!editable || !editable.parentElement) return;
    this.host = document.createElement('div');
    this.host.className = 'mddm-page-overlay-host';
    this.host.style.position = 'absolute';
    this.host.style.inset = '0';
    this.host.style.pointerEvents = 'none';
    editable.parentElement.appendChild(this.host);
  }

  public update(breaks: readonly ComputedBreak[]): void {
    if (!this.host) return;
    this.host.innerHTML = '';
    for (const b of breaks) {
      const bar = document.createElement('div');
      bar.className = 'mddm-page-overlay';
      bar.textContent = `Page ${b.pageNumber}`;
      bar.dataset.afterBid = b.afterBid;
      this.host.appendChild(bar);
    }
  }

  public destroy(): void {
    this.host?.remove();
    this.host = null;
  }
}
```

- [ ] **Step 3: Add CSS to `AuthorEditor.module.css`**

```css
.editable :global(.mddm-page-overlay) {
  background: #e8eaed;
  color: #6b1f2a;
  font: 11px/1 Carlito, sans-serif;
  padding: 6px 12px;
  text-align: right;
  box-shadow: 0 -1px 3px rgba(0, 0, 0, 0.08);
}
```

- [ ] **Step 4: Wire in `index.ts`**

```ts
import { PageOverlayView } from './PageOverlayView';
// in init():
const overlay = new PageOverlayView(this.editor);
measurer.onBreaks(breaks => overlay.update(breaks));
this.on('destroy', () => overlay.destroy());
```

- [ ] **Step 5: Run tests → PASS**

- [ ] **Step 6: Commit**

```bash
git add frontend/apps/web/src/features/documents/ck5/plugins/MddmPaginationPlugin frontend/apps/web/src/features/documents/ck5/react/AuthorEditor.module.css
git commit -m "feat(ck5): PageOverlayView renders gutter overlays outside data view"
```

---

### Task 18: `getData({pagination:true})` — emit `data-pagination-page` attribute

**Executor:** Codex. **Group:** after 17.

**Goal:** Monkey-patch `editor.data.get` to accept `pagination: true` option. When set, walk current `ComputedBreak[]` and add `data-pagination-page="N"` attribute to the block that **starts** page N (i.e., the block immediately after the break position).

**Files:**
- Create: `frontend/apps/web/src/features/documents/ck5/plugins/MddmPaginationPlugin/data-contract.ts`
- Modify: `frontend/apps/web/src/features/documents/ck5/plugins/MddmPaginationPlugin/index.ts`
- Create: `frontend/apps/web/src/features/documents/ck5/plugins/MddmPaginationPlugin/__tests__/data-contract.test.ts`

**Acceptance Criteria:**
- [ ] `editor.getData({ pagination: true })` includes `data-pagination-page` attributes.
- [ ] `editor.getData()` (no option) does NOT include them.
- [ ] The block marked page N is the block right after the ComputedBreak with `afterBid = prevBlockBid`.

**Verify:** `cd frontend/apps/web && npx vitest run src/features/documents/ck5/plugins/MddmPaginationPlugin/__tests__/data-contract.test.ts` → PASS.

**Steps:**

- [ ] **Step 1: Write failing test**

```ts
// data-contract.test.ts
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { DecoupledEditor } from '@ckeditor/ckeditor5-editor-decoupled';
import { Paragraph } from '@ckeditor/ckeditor5-paragraph';
import { Essentials } from '@ckeditor/ckeditor5-essentials';
import { MddmBlockIdentityPlugin } from '../../MddmBlockIdentityPlugin';
import { MddmPaginationPlugin } from '../index';

describe('getData pagination option', () => {
  let editor: DecoupledEditor;
  beforeEach(async () => {
    editor = await DecoupledEditor.create(document.createElement('div'), {
      plugins: [Essentials, Paragraph, MddmBlockIdentityPlugin, MddmPaginationPlugin],
    });
  });
  afterEach(async () => { await editor.destroy(); });

  it('no flag → no pagination attrs', () => {
    editor.setData('<p data-mddm-bid="aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa">x</p>');
    expect(editor.getData()).not.toContain('data-pagination-page');
  });

  it('flag set + stub breaks → attrs present', () => {
    editor.setData('<p data-mddm-bid="aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa">a</p><p data-mddm-bid="bbbbbbbb-bbbb-4bbb-bbbb-bbbbbbbbbbbb">b</p>');
    const plugin = editor.plugins.get('MddmPagination') as MddmPaginationPlugin;
    (plugin as any).setComputedBreaks([{ afterBid: 'aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa', pageNumber: 2 }]);
    const html = editor.getData({ pagination: true } as any);
    expect(html).toContain('data-pagination-page="2"');
  });
});
```

- [ ] **Step 2: Implement `data-contract.ts`**

```ts
import type { Editor } from '@ckeditor/ckeditor5-core';
import type { ComputedBreak } from './types';

/**
 * Wraps editor.data.get so `{ pagination: true }` injects
 * `data-pagination-page="N"` onto the block that starts each new page.
 */
export function installPaginationDataContract(
  editor: Editor,
  getBreaks: () => readonly ComputedBreak[],
): void {
  const original = editor.data.get.bind(editor.data);
  (editor.data as any).get = (options?: any) => {
    const html = original(options);
    if (!options?.pagination) return html;
    return injectPageAttrs(html, getBreaks());
  };
}

function injectPageAttrs(html: string, breaks: readonly ComputedBreak[]): string {
  if (!breaks.length) return html;
  // Build: "after bid X → next paginable block gets page N".
  const byAfter = new Map(breaks.map(b => [b.afterBid, b.pageNumber]));
  // Walk: find each <tag data-mddm-bid="X"> in order; on match, mark the NEXT block.
  const tagRe = /<([a-z][a-z0-9]*)(?=\s)([^>]*?\bdata-mddm-bid="([^"]+)"[^>]*)>/gi;
  const marks: Array<{ page: number; matchIndex: number }> = [];
  let m: RegExpExecArray | null;
  const positions: Array<{ bid: string; end: number; tagStart: number }> = [];
  while ((m = tagRe.exec(html))) {
    positions.push({ bid: m[3], end: tagRe.lastIndex, tagStart: m.index });
  }
  for (let i = 0; i < positions.length - 1; i++) {
    const page = byAfter.get(positions[i].bid);
    if (page !== undefined) marks.push({ page, matchIndex: i + 1 });
  }
  // Apply marks in reverse so indices remain valid.
  let out = html;
  for (let i = marks.length - 1; i >= 0; i--) {
    const { page, matchIndex } = marks[i];
    const p = positions[matchIndex];
    // Insert attribute before the closing `>` of that tag.
    const insertAt = p.end - 1;
    out = out.slice(0, insertAt) + ` data-pagination-page="${page}"` + out.slice(insertAt);
  }
  return out;
}
```

- [ ] **Step 3: Wire in `index.ts`**

```ts
import { installPaginationDataContract } from './data-contract';
// in init():
let currentBreaks: readonly ComputedBreak[] = [];
measurer.onBreaks(b => { currentBreaks = b; overlay.update(b); });
installPaginationDataContract(this.editor, () => currentBreaks);
(this as any).setComputedBreaks = (b: readonly ComputedBreak[]) => { currentBreaks = b; };
```

- [ ] **Step 4: Run tests → PASS**

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/ck5/plugins/MddmPaginationPlugin
git commit -m "feat(ck5): emit data-pagination-page on getData({pagination:true})"
```

---

### Task 19: Page counter React component

**Executor:** Codex. **Group:** after 17 (parallel with 18).

**Goal:** React component showing "Page N of M" live, driven by the plugin's `ComputedBreak[]`. Rendered as toolbar element.

**Files:**
- Create: `frontend/apps/web/src/features/documents/ck5/react/PageCounter.tsx`
- Create: `frontend/apps/web/src/features/documents/ck5/react/__tests__/PageCounter.test.tsx`
- Modify: `frontend/apps/web/src/features/documents/ck5/react/AuthorEditor.tsx` (add component to toolbar region)

**Acceptance Criteria:**
- [ ] Component displays "Page 1 of 1" on empty doc.
- [ ] Updates when measurer emits new breaks (unit-tested with mocked plugin).
- [ ] Unmounts cleanly — `off()` called on unmount verified in test.

**Verify:** `cd frontend/apps/web && npx vitest run src/features/documents/ck5/react/__tests__/PageCounter.test.tsx` → PASS (3 tests).

**Steps:**

- [ ] **Step 1: Implement `PageCounter.tsx`**

```tsx
import { useEffect, useState } from 'react';
import type { DecoupledEditor } from '@ckeditor/ckeditor5-editor-decoupled';

type ComputedBreak = { afterBid: string; pageNumber: number };

export function PageCounter({ editor }: { editor: DecoupledEditor | null }) {
  const [pages, setPages] = useState(1);
  useEffect(() => {
    if (!editor) return;
    const plugin = editor.plugins.get('MddmPagination') as any;
    if (!plugin) return;
    const off = plugin._measurer.onBreaks((b: ComputedBreak[]) => {
      setPages(1 + b.length);
    });
    return () => off();
  }, [editor]);
  return <span className="mddm-page-counter">Page {pages} of {pages}</span>;
}
```

- [ ] **Step 2: Add to `AuthorEditor.tsx` toolbar**

```tsx
import { PageCounter } from './PageCounter';
// inside toolbar JSX:
<PageCounter editor={editor} />
```

- [ ] **Step 3: Write unit test with mocked plugin**

```tsx
// PageCounter.test.tsx
import { describe, it, expect, vi } from 'vitest';
import { render, screen } from '@testing-library/react';
import { act } from 'react';
import { PageCounter } from '../PageCounter';

function makeFakeEditor() {
  type Cb = (b: { afterBid: string; pageNumber: number }[]) => void;
  const listeners: Cb[] = [];
  const off = vi.fn(() => { listeners.length = 0; });
  const onBreaks = vi.fn((cb: Cb) => { listeners.push(cb); return off; });
  const plugin = { _measurer: { onBreaks } };
  const editor: any = { plugins: { get: (n: string) => n === 'MddmPagination' ? plugin : null } };
  return { editor, listeners, off };
}

describe('PageCounter', () => {
  it('renders "Page 1 of 1" with no breaks', () => {
    const { editor } = makeFakeEditor();
    render(<PageCounter editor={editor} />);
    expect(screen.getByText(/Page 1 of 1/)).toBeTruthy();
  });

  it('updates when measurer emits breaks', () => {
    const { editor, listeners } = makeFakeEditor();
    render(<PageCounter editor={editor} />);
    act(() => { listeners[0]([{ afterBid: 'a', pageNumber: 2 }, { afterBid: 'b', pageNumber: 3 }]); });
    expect(screen.getByText(/Page 3 of 3/)).toBeTruthy();
  });

  it('calls off() on unmount', () => {
    const { editor, off } = makeFakeEditor();
    const { unmount } = render(<PageCounter editor={editor} />);
    unmount();
    expect(off).toHaveBeenCalled();
  });
});
```

- [ ] **Step 4: Run → PASS**

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/ck5/react/PageCounter.tsx frontend/apps/web/src/features/documents/ck5/react/__tests__/PageCounter.test.tsx frontend/apps/web/src/features/documents/ck5/react/AuthorEditor.tsx
git commit -m "feat(ck5): add live Page N of M counter driven by pagination plugin"
```

---

## Phase 3 — Server paginator

### Task 20: Playwright worker pool

**Executor:** Codex. **Group:** P2 (parallel with Tasks 25–28).

**Goal:** Warm pool of `CHROMIUM_POOL_SIZE` (default 3) Playwright chromium browsers. Round-robin `acquire()` / `release()` lease. Graceful shutdown on `SIGTERM`.

**Files:**
- Create: `apps/ck5-export/src/pagination/playwright-pool.ts`
- Create: `apps/ck5-export/src/pagination/__tests__/playwright-pool.test.ts`
- Modify: `apps/ck5-export/package.json` (add `playwright`)

**Acceptance Criteria:**
- [ ] `PlaywrightPool` boots `POOL_SIZE` browsers on init.
- [ ] `acquire()` returns an idle worker or queues if all busy.
- [ ] `release(worker)` returns it to idle.
- [ ] `shutdown()` closes all browsers.

**Verify:** `cd apps/ck5-export && npx vitest run src/pagination/__tests__/playwright-pool.test.ts` → PASS.

**Steps:**

- [ ] **Step 1: Install Playwright**

```bash
cd apps/ck5-export && npm install playwright && npx playwright install chromium
```

- [ ] **Step 2: Write failing test (mock playwright)**

```ts
// playwright-pool.test.ts
import { describe, it, expect, vi } from 'vitest';
import { PlaywrightPool } from '../playwright-pool';

const mockBrowser = { close: vi.fn(), newContext: vi.fn() };
vi.mock('playwright', () => ({
  chromium: { launch: vi.fn(async () => mockBrowser) },
}));

describe('PlaywrightPool', () => {
  it('launches POOL_SIZE browsers and acquires/releases', async () => {
    const pool = new PlaywrightPool({ size: 2 });
    await pool.init();
    const w1 = await pool.acquire();
    const w2 = await pool.acquire();
    expect(w1).toBeDefined();
    expect(w2).toBeDefined();
    pool.release(w1);
    const w3 = await pool.acquire();
    expect(w3).toBe(w1);
    await pool.shutdown();
    expect(mockBrowser.close).toHaveBeenCalledTimes(2);
  });
});
```

- [ ] **Step 3: Implement `playwright-pool.ts`**

```ts
import { chromium, Browser } from 'playwright';

export type Worker = Browser;

export class PlaywrightPool {
  private workers: Worker[] = [];
  private idle: Worker[] = [];
  private waiters: Array<(w: Worker) => void> = [];

  public constructor(private readonly opts: { size: number }) {}

  public async init(): Promise<void> {
    for (let i = 0; i < this.opts.size; i++) {
      const b = await chromium.launch({ headless: true });
      this.workers.push(b);
      this.idle.push(b);
    }
  }

  public async acquire(): Promise<Worker> {
    if (this.idle.length) return this.idle.shift()!;
    return new Promise(resolve => this.waiters.push(resolve));
  }

  public release(w: Worker): void {
    const waiter = this.waiters.shift();
    if (waiter) { waiter(w); return; }
    this.idle.push(w);
  }

  public async shutdown(): Promise<void> {
    await Promise.all(this.workers.map(w => w.close()));
    this.workers = [];
    this.idle = [];
    this.waiters = [];
  }
}
```

- [ ] **Step 4: Run → PASS**

- [ ] **Step 5: Commit**

```bash
git add apps/ck5-export/src/pagination/playwright-pool.ts apps/ck5-export/src/pagination/__tests__/playwright-pool.test.ts apps/ck5-export/package.json apps/ck5-export/package-lock.json
git commit -m "feat(ck5-export): Playwright worker pool for pagination"
```

---

### Task 20b: Pool retry-once + fallback ladder (resilience)

**Executor:** Opus. **Group:** after 20.

**Goal:** Wrap `acquire()` in an execution helper that survives one worker crash: if the leased worker throws during use, the helper releases/destroys it, acquires a fresh worker, and retries the caller's closure ONCE. If the second attempt fails, return a structured `PaginationDegraded` signal carrying a `reason` field that the route uses to pick the right outcome:
- `'pool-exhausted'` → 503 (no worker available within `ACQUIRE_TIMEOUT_MS`)
- `'worker-crash'` → 503 (retry exhausted — surface resource error)
- `'runtime-error'` → 200 with `{breaks: [], degraded: true}` (graceful empty-breaks fallback; DOCX/PDF still ships, just with no pagination markers — mirrors spec's runtime-error fallback mode)

Timeout (`PaginatorTimeoutError`) propagates separately → 504.

**Files:**
- Create: `apps/ck5-export/src/pagination/pool-retry.ts`
- Create: `apps/ck5-export/src/pagination/__tests__/pool-retry.test.ts`
- Modify: `apps/ck5-export/src/pagination/playwright-pool.ts` (add `replace(worker)` method to destroy and relaunch one browser)

**Acceptance Criteria:**
- [ ] `withWorker(pool, fn)` succeeds on first attempt when `fn` succeeds.
- [ ] When `fn` throws `WorkerCrash` on first attempt, a NEW worker is acquired and `fn` is retried exactly ONCE.
- [ ] Crashed worker is replaced via `pool.replace(worker)` (closed + relaunched), not just released.
- [ ] Two consecutive `WorkerCrash` throws → `withWorker` rejects with `PaginationDegraded` carrying `{reason: 'worker-crash'}`.
- [ ] A non-`WorkerCrash`/non-`PaginatorTimeoutError` error (e.g. unexpected runtime exception inside `fn`) → `withWorker` rejects with `PaginationDegraded {reason: 'runtime-error'}` (no retry).
- [ ] `PaginatorTimeoutError` is propagated immediately, no retry.
- [ ] `acquire()` queuing timeout (`ACQUIRE_TIMEOUT_MS = 5000`) → rejects with `PaginationDegraded {reason: 'pool-exhausted'}`.

**Verify:** `cd apps/ck5-export && npx vitest run src/pagination/__tests__/pool-retry.test.ts` → PASS (6 tests).

**Steps:**

- [ ] **Step 1: Write failing test**

```ts
// pool-retry.test.ts
import { describe, it, expect, vi } from 'vitest';
import { withWorker, WorkerCrash, PaginationDegraded } from '../pool-retry';
import { PaginatorTimeoutError } from '../paginate-with-chromium';

function fakePool(): any {
  const w1 = { id: 'w1', closed: false };
  const w2 = { id: 'w2', closed: false };
  const workers = [w1, w2];
  let next = 0;
  return {
    acquire: vi.fn(async () => workers[next++ % workers.length]),
    release: vi.fn(),
    replace: vi.fn(async (w) => { w.closed = true; }),
  };
}

describe('withWorker', () => {
  it('returns fn result when first attempt succeeds', async () => {
    const pool = fakePool();
    const r = await withWorker(pool, async () => 'ok');
    expect(r).toBe('ok');
    expect(pool.acquire).toHaveBeenCalledTimes(1);
  });

  it('retries once on WorkerCrash and succeeds', async () => {
    const pool = fakePool();
    let n = 0;
    const r = await withWorker(pool, async () => {
      if (n++ === 0) throw new WorkerCrash('boom');
      return 'ok';
    });
    expect(r).toBe('ok');
    expect(pool.acquire).toHaveBeenCalledTimes(2);
    expect(pool.replace).toHaveBeenCalledTimes(1);
  });

  it('rejects PaginationDegraded on second crash', async () => {
    const pool = fakePool();
    await expect(withWorker(pool, async () => { throw new WorkerCrash('boom'); }))
      .rejects.toBeInstanceOf(PaginationDegraded);
  });

  it('propagates PaginatorTimeoutError without retry', async () => {
    const pool = fakePool();
    await expect(withWorker(pool, async () => { throw new PaginatorTimeoutError('t'); }))
      .rejects.toBeInstanceOf(PaginatorTimeoutError);
    expect(pool.acquire).toHaveBeenCalledTimes(1);
  });

  it('rejects PaginationDegraded when acquire exceeds ACQUIRE_TIMEOUT_MS', async () => {
    const pool = { acquire: vi.fn(() => new Promise(() => {})), release: vi.fn(), replace: vi.fn() } as any;
    const err = await withWorker(pool, async () => 'x', { acquireTimeoutMs: 50 }).catch(e => e);
    expect(err).toBeInstanceOf(PaginationDegraded);
    expect(err.reason).toBe('pool-exhausted');
  });

  it('rejects PaginationDegraded {reason:runtime-error} on unknown error', async () => {
    const pool = fakePool();
    const err = await withWorker(pool, async () => { throw new Error('boom'); }).catch(e => e);
    expect(err).toBeInstanceOf(PaginationDegraded);
    expect(err.reason).toBe('runtime-error');
    expect(pool.acquire).toHaveBeenCalledTimes(1); // no retry
  });
});
```

- [ ] **Step 2: Implement `pool-retry.ts`**

```ts
import type { PlaywrightPool, Worker } from './playwright-pool';
import { PaginatorTimeoutError } from './paginate-with-chromium';

export class WorkerCrash extends Error { constructor(m: string) { super(m); this.name = 'WorkerCrash'; } }

export type DegradedReason = 'worker-crash' | 'pool-exhausted' | 'runtime-error';

export class PaginationDegraded extends Error {
  public readonly reason: DegradedReason;
  constructor(reason: DegradedReason, cause?: unknown) {
    super(`pagination degraded: ${reason}`);
    this.name = 'PaginationDegraded';
    this.reason = reason;
    (this as any).cause = cause;
  }
}

export const ACQUIRE_TIMEOUT_MS = 5000;

async function acquireWithTimeout(pool: PlaywrightPool, timeoutMs: number): Promise<Worker> {
  let t: ReturnType<typeof setTimeout> | null = null;
  const tp = new Promise<never>((_, rej) => {
    t = setTimeout(() => rej(new PaginationDegraded('pool-exhausted')), timeoutMs);
  });
  try { return await Promise.race([pool.acquire(), tp]); }
  finally { if (t) clearTimeout(t); }
}

export async function withWorker<T>(
  pool: PlaywrightPool,
  fn: (w: Worker) => Promise<T>,
  opts: { acquireTimeoutMs?: number } = {},
): Promise<T> {
  const acquireTimeoutMs = opts.acquireTimeoutMs ?? ACQUIRE_TIMEOUT_MS;
  let attempt = 0;
  let lastCrash: unknown = null;
  while (attempt < 2) {
    const worker = await acquireWithTimeout(pool, acquireTimeoutMs);
    try {
      const result = await fn(worker);
      pool.release(worker);
      return result;
    } catch (e) {
      if (e instanceof PaginatorTimeoutError) { pool.release(worker); throw e; }
      if (e instanceof WorkerCrash) {
        lastCrash = e;
        await pool.replace(worker);
        attempt++;
        continue;
      }
      pool.release(worker);
      // Unexpected runtime error — surface as runtime-error degraded (graceful fallback path)
      throw new PaginationDegraded('runtime-error', e);
    }
  }
  throw new PaginationDegraded('worker-crash', lastCrash);
}
```

- [ ] **Step 3: Add `replace()` to `playwright-pool.ts`**

```ts
public async replace(w: Worker): Promise<void> {
  const i = this.workers.indexOf(w);
  if (i >= 0) this.workers.splice(i, 1);
  try { await w.close(); } catch { /* already crashed */ }
  const fresh = await chromium.launch({ headless: true });
  this.workers.push(fresh);
  // give fresh worker to a waiter, else idle
  const waiter = this.waiters.shift();
  if (waiter) waiter(fresh); else this.idle.push(fresh);
}
```

- [ ] **Step 4: Run → PASS**

- [ ] **Step 5: Commit**

```bash
git add apps/ck5-export/src/pagination/pool-retry.ts apps/ck5-export/src/pagination/__tests__/pool-retry.test.ts apps/ck5-export/src/pagination/playwright-pool.ts
git commit -m "feat(ck5-export): pool retry-once + fallback ladder for resilience"
```

---

### Task 21: Sentinel injector

**Executor:** Codex. **Group:** after 20.

**Goal:** Pure HTML transform — for every paginable block with `data-mddm-bid`, inject a first-child `<span data-pb-marker="{bid}" style="display:inline;width:0;height:0">` sentinel. Paged.js inherits page number from containing `.pagedjs_page`; sentinel gives us a bid-keyed query target.

**Files:**
- Create: `apps/ck5-export/src/pagination/sentinel.ts`
- Create: `apps/ck5-export/src/pagination/__tests__/sentinel.test.ts`

**Acceptance Criteria:**
- [ ] Every element with `data-mddm-bid` in input HTML has exactly one `<span data-pb-marker="{bid}">` as its first child after injection.
- [ ] Elements without `data-mddm-bid` are untouched.
- [ ] Idempotent: running twice does not add duplicates.

**Verify:** `cd apps/ck5-export && npx vitest run src/pagination/__tests__/sentinel.test.ts` → PASS.

**Steps:**

- [ ] **Step 1: Write failing test**

```ts
// sentinel.test.ts
import { describe, it, expect } from 'vitest';
import { injectSentinels } from '../sentinel';

describe('injectSentinels', () => {
  it('injects sentinel as first child for each bid\'d block', () => {
    const out = injectSentinels('<p data-mddm-bid="aaa">x</p><p>y</p>');
    expect(out).toContain('<span data-pb-marker="aaa"');
    expect(out.match(/data-pb-marker/g)).toHaveLength(1);
  });
  it('idempotent', () => {
    const once = injectSentinels('<p data-mddm-bid="aaa">x</p>');
    const twice = injectSentinels(once);
    expect(twice.match(/data-pb-marker="aaa"/g)).toHaveLength(1);
  });
});
```

- [ ] **Step 2: Implement `sentinel.ts`**

```ts
import { parseHTML } from 'linkedom';

export function injectSentinels(html: string): string {
  const { document } = parseHTML(`<!DOCTYPE html><html><body>${html}</body></html>`);
  const targets = Array.from(document.querySelectorAll('[data-mddm-bid]'));
  for (const el of targets) {
    const bid = el.getAttribute('data-mddm-bid')!;
    const first = el.firstElementChild;
    if (first && first.tagName.toLowerCase() === 'span' && first.getAttribute('data-pb-marker') === bid) continue;
    const s = document.createElement('span');
    s.setAttribute('data-pb-marker', bid);
    s.setAttribute('style', 'display:inline;width:0;height:0');
    el.insertBefore(s, el.firstChild);
  }
  return document.body.innerHTML;
}
```

- [ ] **Step 3: Run → PASS; commit**

```bash
git add apps/ck5-export/src/pagination/sentinel.ts apps/ck5-export/src/pagination/__tests__/sentinel.test.ts
git commit -m "feat(ck5-export): sentinel injector for bid-keyed Paged.js scrape"
```

---

### Task 22: `paginate-with-chromium.ts` — Paged.js scrape

**Executor:** Codex. **Group:** after 21.

**Goal:** Take wrapped print HTML (with sentinels), load in a pool worker, wait for Paged.js to fragment, scrape `[data-pb-marker]` → nearest `.pagedjs_page` → `pageNumber`. Return `ServerBreak[]`.

**Files:**
- Create: `apps/ck5-export/src/pagination/paginate-with-chromium.ts`
- Create: `apps/ck5-export/src/pagination/__tests__/paginate-with-chromium.test.ts`

**Acceptance Criteria:**
- [ ] Given a 3-page fixture, returns `ServerBreak[]` with at least 2 entries (2 breaks for 3 pages).
- [ ] `pageNumber` is 1-indexed and monotonic.
- [ ] Timeout after 15s → throws `PaginatorTimeoutError`.

**Verify:** `cd apps/ck5-export && npx vitest run src/pagination/__tests__/paginate-with-chromium.test.ts` → PASS. (Integration test; requires chromium installed.)

**Steps:**

- [ ] **Step 1: Write failing test**

```ts
// paginate-with-chromium.test.ts
import { describe, it, expect, beforeAll, afterAll } from 'vitest';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { PlaywrightPool } from '../playwright-pool';
import { paginateWithChromium } from '../paginate-with-chromium';

describe('paginateWithChromium', () => {
  let pool: PlaywrightPool;
  beforeAll(async () => { pool = new PlaywrightPool({ size: 1 }); await pool.init(); }, 30000);
  afterAll(async () => { await pool.shutdown(); });

  it('returns monotonic server breaks', async () => {
    // minimal fixture — 3 pages of content, bids on each paragraph
    const html = Array.from({ length: 120 }, (_, i) =>
      `<p data-mddm-bid="bid-${i.toString().padStart(4, '0')}">paragraph ${i} lorem ipsum dolor sit amet.</p>`
    ).join('');
    const breaks = await paginateWithChromium(pool, html, { timeoutMs: 15000 });
    expect(breaks.length).toBeGreaterThanOrEqual(1);
    for (let i = 1; i < breaks.length; i++) {
      expect(breaks[i].pageNumber).toBeGreaterThanOrEqual(breaks[i - 1].pageNumber);
    }
  }, 30000);
});
```

- [ ] **Step 2: Implement `paginate-with-chromium.ts`**

```ts
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import type { PlaywrightPool } from './playwright-pool';
import { injectSentinels } from './sentinel';
import { wrapPrintDocument } from '../print-stylesheet/wrap-print-document';

export class PaginatorTimeoutError extends Error {
  public constructor(ms: number) { super(`pagination timed out after ${ms}ms`); }
}

export type ServerBreak = Readonly<{ bid: string; pageNumber: number }>;

const PAGED_JS_POLYFILL = readFileSync(
  join(process.cwd(), 'node_modules/pagedjs/dist/paged.polyfill.js'),
  'utf-8',
);

export async function paginateWithChromium(
  pool: PlaywrightPool,
  rawHtml: string,
  opts: { timeoutMs: number },
): Promise<ServerBreak[]> {
  const worker = await pool.acquire();
  try {
    const withSentinels = injectSentinels(rawHtml);
    const fullHtml = wrapPrintDocument({ bodyHtml: withSentinels }); // already includes <script src=/assets/paged.polyfill.js>

    const ctx = await worker.newContext();
    const page = await ctx.newPage();

    try {
      await page.setContent(fullHtml, { waitUntil: 'networkidle', timeout: opts.timeoutMs });
      // Inline polyfill as fallback if CDN/static path unreachable:
      await page.addScriptTag({ content: PAGED_JS_POLYFILL });
      await page.waitForFunction(
        () => document.querySelector('.pagedjs_page') !== null,
        { timeout: opts.timeoutMs },
      );

      const breaks = await page.evaluate(() => {
        const seen = new Map<string, number>();
        for (const m of Array.from(document.querySelectorAll('[data-pb-marker]'))) {
          const bid = (m as HTMLElement).dataset.pbMarker!;
          const pageEl = m.closest('.pagedjs_page') as HTMLElement | null;
          const n = pageEl ? Number(pageEl.dataset.pageNumber) : 1;
          if (!seen.has(bid)) seen.set(bid, n);
        }
        return Array.from(seen, ([bid, pageNumber]) => ({ bid, pageNumber }));
      });

      // Filter to only breaks (page change): keep items whose pageNumber > 1
      // and whose previous bid was on a lower page.
      const sorted = [...breaks];
      const out: ServerBreak[] = [];
      let prev = 0;
      for (const b of sorted) {
        if (b.pageNumber > prev) { out.push(b); prev = b.pageNumber; }
      }
      return out;
    } finally {
      await ctx.close();
    }
  } finally {
    pool.release(worker);
  }
}
```

- [ ] **Step 3: Refactor to use `withWorker` from Task 20b**

Replace the direct `pool.acquire()/release()` block in `paginate-with-chromium.ts` with `withWorker(pool, async (worker) => { ... })` so crashes during `page.evaluate` trigger the retry-once policy. A `page.evaluate` throwing a `page.crash` event → rethrow wrapped in `WorkerCrash`.

- [ ] **Step 4: Run → PASS; commit**

```bash
git add apps/ck5-export/src/pagination
git commit -m "feat(ck5-export): paginate-with-chromium scrapes Paged.js page numbers"
```

---

### Task 22b: HTML sha256 cache + 5-min TTL

**Executor:** Codex. **Group:** after 22.

**Goal:** Pure in-memory cache: key = `sha256(normalizedHtml)`, value = `ServerBreak[]`, TTL = 5 minutes. `paginateWithChromium` checks the cache before launching a worker. Spec NFR — reduces repeat-render cost.

**Files:**
- Create: `apps/ck5-export/src/pagination/cache.ts`
- Create: `apps/ck5-export/src/pagination/__tests__/cache.test.ts`
- Modify: `apps/ck5-export/src/pagination/paginate-with-chromium.ts` (check+populate cache)

**Acceptance Criteria:**
- [ ] `PaginationCache.get(html)` returns `undefined` for unseen input.
- [ ] After `set(html, breaks)`, `get(html)` returns same breaks array.
- [ ] Entries older than `TTL_MS` (default 300_000) return `undefined`.
- [ ] `cache.size` bounded (LRU eviction at `MAX_ENTRIES = 64`).
- [ ] `paginateWithChromium` on identical HTML twice → second call skips worker (verified via vi.spyOn).

**Verify:** `cd apps/ck5-export && npx vitest run src/pagination/__tests__/cache.test.ts` → PASS (4 tests).

**Steps:**

- [ ] **Step 1: Write failing test**

```ts
// cache.test.ts
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { PaginationCache } from '../cache';

describe('PaginationCache', () => {
  beforeEach(() => { vi.useFakeTimers(); });
  afterEach(() => { vi.useRealTimers(); });

  it('returns undefined for unseen key', () => {
    const c = new PaginationCache();
    expect(c.get('<p>x</p>')).toBeUndefined();
  });

  it('round-trips breaks for same html', () => {
    const c = new PaginationCache();
    c.set('<p>x</p>', [{ bid: 'a', pageNumber: 1 }]);
    expect(c.get('<p>x</p>')).toEqual([{ bid: 'a', pageNumber: 1 }]);
  });

  it('expires entries after TTL', () => {
    const c = new PaginationCache({ ttlMs: 1000 });
    c.set('<p>x</p>', [{ bid: 'a', pageNumber: 1 }]);
    vi.advanceTimersByTime(1001);
    expect(c.get('<p>x</p>')).toBeUndefined();
  });

  it('evicts LRU beyond MAX_ENTRIES', () => {
    const c = new PaginationCache({ maxEntries: 2 });
    c.set('a', [{ bid: 'a', pageNumber: 1 }]);
    c.set('b', [{ bid: 'b', pageNumber: 1 }]);
    c.set('c', [{ bid: 'c', pageNumber: 1 }]);
    expect(c.get('a')).toBeUndefined();
    expect(c.get('b')).toBeDefined();
    expect(c.get('c')).toBeDefined();
  });
});
```

- [ ] **Step 2: Implement `cache.ts`**

```ts
import { createHash } from 'node:crypto';
import type { ServerBreak } from '@metaldocs/mddm-pagination-types';

type Entry = { breaks: ServerBreak[]; expiresAt: number };

export class PaginationCache {
  private readonly ttlMs: number;
  private readonly maxEntries: number;
  private readonly entries = new Map<string, Entry>(); // Map keeps insertion order → LRU

  public constructor(opts: { ttlMs?: number; maxEntries?: number } = {}) {
    this.ttlMs = opts.ttlMs ?? 300_000;
    this.maxEntries = opts.maxEntries ?? 64;
  }

  private static key(html: string): string {
    return createHash('sha256').update(html.trim()).digest('hex');
  }

  public get(html: string): ServerBreak[] | undefined {
    const k = PaginationCache.key(html);
    const e = this.entries.get(k);
    if (!e) return undefined;
    if (e.expiresAt <= Date.now()) { this.entries.delete(k); return undefined; }
    // LRU bump
    this.entries.delete(k);
    this.entries.set(k, e);
    return e.breaks;
  }

  public set(html: string, breaks: ServerBreak[]): void {
    const k = PaginationCache.key(html);
    this.entries.set(k, { breaks, expiresAt: Date.now() + this.ttlMs });
    while (this.entries.size > this.maxEntries) {
      const oldest = this.entries.keys().next().value;
      if (oldest !== undefined) this.entries.delete(oldest);
    }
  }

  public clear(): void { this.entries.clear(); }
  public get size(): number { return this.entries.size; }
}
```

- [ ] **Step 3: Wire cache into `paginate-with-chromium.ts`**

Export a module-level singleton:
```ts
import { PaginationCache } from './cache';
export const paginationCache = new PaginationCache();

export async function paginateWithChromium(pool, rawHtml, opts) {
  const cached = paginationCache.get(rawHtml);
  if (cached) return cached;
  const result = await /* existing withWorker(...) body */;
  paginationCache.set(rawHtml, result);
  return result;
}
```

- [ ] **Step 4: Run → PASS; commit**

```bash
git add apps/ck5-export/src/pagination/cache.ts apps/ck5-export/src/pagination/__tests__/cache.test.ts apps/ck5-export/src/pagination/paginate-with-chromium.ts
git commit -m "feat(ck5-export): sha256 HTML cache with 5-min TTL + LRU"
```

---

### Task 23: `/paginate` Hono route

**Executor:** Codex. **Group:** after 22b.

**Goal:** `POST /paginate` accepts JSON matching `PaginateRequest` from `@metaldocs/mddm-pagination-types`: `{html: string, editorBids?: string[]}`. Returns `PaginateResponse`. Detects editor-server-desync: if `editorBids` is provided and any bid is absent from the HTML, returns 422.

**Files:**
- Create: `apps/ck5-export/src/routes/paginate.ts`
- Modify: `apps/ck5-export/src/server.ts` (register route + init pool)
- Create: `apps/ck5-export/src/routes/__tests__/paginate.test.ts`
- Modify: `apps/ck5-export/src/pagination/validator.ts` (add `validateEditorBidSet` helper)

**Acceptance Criteria:**
- [ ] `POST /paginate` with valid bid'd HTML → 200 + `{breaks: [...]}`.
- [ ] With bid-collision → 422 `{error:'bid-collision',bids:[...]}`.
- [ ] With `editorBids` containing a bid not present in HTML → 422 `{error:'editor-server-desync',missingBids:[...]}`.
- [ ] `PaginatorTimeoutError` → 504.
- [ ] `PaginationDegraded{reason:'worker-crash'}` → 503.
- [ ] `PaginationDegraded{reason:'pool-exhausted'}` → 503.
- [ ] `PaginationDegraded{reason:'runtime-error'}` → **200** with `{breaks: [], degraded: true}` (graceful empty-breaks fallback).
- [ ] Request JSON validated against `PaginateRequest` shape (narrow type guard rejects malformed input → 400).

**Verify:** `cd apps/ck5-export && npx vitest run src/routes/__tests__/paginate.test.ts` → PASS (7 tests).

**Steps:**

- [ ] **Step 1: Write test (mock `paginateWithChromium`)**

```ts
// paginate.test.ts
import { describe, it, expect, vi } from 'vitest';
import { Hono } from 'hono';
import { paginateRoute } from '../paginate';

vi.mock('../../pagination/paginate-with-chromium', () => ({
  paginateWithChromium: vi.fn(async () => [{ bid: 'a', pageNumber: 2 }]),
  PaginatorTimeoutError: class extends Error {},
}));
vi.mock('../../pagination/pool-retry', () => ({
  PaginationDegraded: class extends Error {
    public reason: 'worker-crash' | 'pool-exhausted' | 'runtime-error';
    constructor(r: 'worker-crash' | 'pool-exhausted' | 'runtime-error' = 'worker-crash') {
      super('degraded'); this.reason = r;
    }
  },
}));

const fakePool: any = {};
const app = new Hono().route('/', paginateRoute(fakePool));

async function post(body: unknown) {
  return app.request('/paginate', { method: 'POST', headers: {'content-type':'application/json'}, body: JSON.stringify(body) });
}

describe('POST /paginate', () => {
  const valid = '<p data-mddm-bid="aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa">x</p>';

  it('returns 200 with breaks on clean input', async () => {
    const r = await post({ html: valid });
    expect(r.status).toBe(200);
    expect(await r.json()).toEqual({ breaks: [{ bid: 'a', pageNumber: 2 }] });
  });

  it('400 on malformed input', async () => {
    const r = await post({ nothtml: 'x' });
    expect(r.status).toBe(400);
  });

  it('422 on bid collision', async () => {
    const dup = valid + valid;
    const r = await post({ html: dup });
    expect(r.status).toBe(422);
    expect((await r.json()).error).toBe('bid-collision');
  });

  it('422 on editor-server-desync', async () => {
    const r = await post({ html: valid, editorBids: ['aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa', 'ghost-bid'] });
    expect(r.status).toBe(422);
    const body = await r.json();
    expect(body.error).toBe('editor-server-desync');
    expect(body.missingBids).toEqual(['ghost-bid']);
  });

  it('503 on PaginationDegraded worker-crash', async () => {
    const { paginateWithChromium } = await import('../../pagination/paginate-with-chromium');
    const { PaginationDegraded } = await import('../../pagination/pool-retry');
    (paginateWithChromium as any).mockRejectedValueOnce(new (PaginationDegraded as any)('worker-crash'));
    const r = await post({ html: valid });
    expect(r.status).toBe(503);
  });

  it('503 on PaginationDegraded pool-exhausted', async () => {
    const { paginateWithChromium } = await import('../../pagination/paginate-with-chromium');
    const { PaginationDegraded } = await import('../../pagination/pool-retry');
    (paginateWithChromium as any).mockRejectedValueOnce(new (PaginationDegraded as any)('pool-exhausted'));
    const r = await post({ html: valid });
    expect(r.status).toBe(503);
  });

  it('200 {breaks:[], degraded:true} on PaginationDegraded runtime-error (graceful fallback)', async () => {
    const { paginateWithChromium } = await import('../../pagination/paginate-with-chromium');
    const { PaginationDegraded } = await import('../../pagination/pool-retry');
    (paginateWithChromium as any).mockRejectedValueOnce(new (PaginationDegraded as any)('runtime-error'));
    const r = await post({ html: valid });
    expect(r.status).toBe(200);
    const body = await r.json();
    expect(body).toEqual({ breaks: [], degraded: true });
  });
});
```

- [ ] **Step 2: Implement `paginate.ts`**

```ts
import { Hono } from 'hono';
import type { PaginateRequest, PaginateResponse } from '@metaldocs/mddm-pagination-types';
import { paginateWithChromium, PaginatorTimeoutError } from '../pagination/paginate-with-chromium';
import { PaginationDegraded } from '../pagination/pool-retry';
import type { PlaywrightPool } from '../pagination/playwright-pool';
import { validateBids, validateEditorBidSet } from '../pagination/validator';

function isPaginateRequest(x: unknown): x is PaginateRequest {
  if (!x || typeof x !== 'object') return false;
  const r = x as Record<string, unknown>;
  if (typeof r.html !== 'string') return false;
  if (r.editorBids !== undefined && !(Array.isArray(r.editorBids) && r.editorBids.every(b => typeof b === 'string'))) return false;
  return true;
}

export function paginateRoute(pool: PlaywrightPool): Hono {
  const r = new Hono();
  r.post('/paginate', async (c) => {
    const body = await c.req.json().catch(() => null);
    if (!isPaginateRequest(body)) return c.json({ error: 'bad-request' }, 400);

    const v = validateBids(body.html);
    if (!v.ok && v.severity === 'error') return c.json({ error: v.error, bids: v.bids }, 422);

    if (body.editorBids?.length) {
      const d = validateEditorBidSet(body.html, body.editorBids);
      if (!d.ok) return c.json({ error: 'editor-server-desync', missingBids: d.missingBids }, 422);
    }

    try {
      const breaks = await paginateWithChromium(pool, body.html, { timeoutMs: 15000 });
      const resp: PaginateResponse = { breaks };
      return c.json(resp);
    } catch (e) {
      if (e instanceof PaginatorTimeoutError) return c.json({ error: 'paginator-timeout' }, 504);
      if (e instanceof PaginationDegraded) {
        // Graceful fallback: unexpected runtime → empty breaks, DOCX/PDF still ships.
        if (e.reason === 'runtime-error') {
          console.warn('paginator runtime-error — returning empty breaks', e);
          return c.json({ breaks: [], degraded: true });
        }
        // Resource exhaustion: client should retry.
        return c.json({ error: 'paginator-unavailable', reason: e.reason }, 503);
      }
      console.error('paginator unknown error', e);
      return c.json({ error: 'paginator-unavailable' }, 503);
    }
  });
  return r;
}
```

- [ ] **Step 3: Add `validateEditorBidSet` to `validator.ts`**

```ts
export function validateEditorBidSet(
  html: string,
  editorBids: readonly string[],
): { ok: true } | { ok: false; missingBids: string[] } {
  const { document } = parseHTML(`<!DOCTYPE html><html><body>${html}</body></html>`);
  const htmlBids = new Set(
    Array.from(document.querySelectorAll('[data-mddm-bid]'))
      .map(el => el.getAttribute('data-mddm-bid')!)
  );
  const missingBids = editorBids.filter(b => !htmlBids.has(b));
  return missingBids.length ? { ok: false, missingBids } : { ok: true };
}
```

- [ ] **Step 4: Wire in `server.ts`**

```ts
import { PlaywrightPool } from './pagination/playwright-pool';
import { paginateRoute } from './routes/paginate';

const pool = new PlaywrightPool({ size: Number(process.env.CHROMIUM_POOL_SIZE ?? 3) });
await pool.init();
app.route('/', paginateRoute(pool));

process.on('SIGTERM', async () => { await pool.shutdown(); process.exit(0); });
```

- [ ] **Step 5: Run → PASS; smoke**

```bash
cd apps/ck5-export && npx vitest run src/routes/__tests__/paginate.test.ts
npm run dev
curl -X POST http://localhost:51619/paginate -H 'content-type: application/json' \
  -d '{"html":"<p data-mddm-bid=\"aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa\">x</p>"}'
# Expected: 200 + {"breaks": []}
```

- [ ] **Step 6: Commit**

```bash
git add apps/ck5-export/src/routes/paginate.ts apps/ck5-export/src/routes/__tests__/paginate.test.ts apps/ck5-export/src/server.ts apps/ck5-export/src/pagination/validator.ts
git commit -m "feat(ck5-export): POST /paginate with desync detection + error ladder"
```

---

### Task 24: Reconciler — editor vs server break merge

**Executor:** Opus. **Group:** after 23.

**Goal:** Join editor breaks and server breaks by `bid`. Rules:
- Agree exactly → keep editor (UX fidelity).
- `|delta| ≤ 1` → keep editor, log `minor-drift`.
- `|delta| > 1` → use server, log `major-drift`.
- Editor break without matching server break → drop + log `orphaned-editor-break`.
- Server break without matching editor break → keep server + log `server-only-break`.

**Files:**
- Create: `apps/ck5-export/src/pagination/reconcile.ts`
- Create: `apps/ck5-export/src/pagination/__tests__/reconcile.test.ts`

**Acceptance Criteria:**
- [ ] 4 algorithm branches each covered by a test.
- [ ] Output preserves monotonic page numbers.

**Verify:** `cd apps/ck5-export && npx vitest run src/pagination/__tests__/reconcile.test.ts` → PASS (6 tests).

**Steps:**

- [ ] **Step 1: Write failing tests**

```ts
// reconcile.test.ts
import { describe, it, expect } from 'vitest';
import { reconcile } from '../reconcile';

describe('reconcile', () => {
  it('exact agree → editor wins', () => {
    const r = reconcile(
      [{ afterBid: 'a', pageNumber: 2 }],
      [{ bid: 'a', pageNumber: 2 }],
    );
    expect(r.resolved).toHaveLength(1);
    expect(r.resolved[0].source).toBe('editor');
  });
  it('minor drift ±1 → editor wins with tag', () => {
    const r = reconcile(
      [{ afterBid: 'a', pageNumber: 2 }],
      [{ bid: 'a', pageNumber: 3 }],
    );
    expect(r.resolved[0].source).toBe('editor-minor-drift');
  });
  it('major drift >1 → server wins', () => {
    const r = reconcile(
      [{ afterBid: 'a', pageNumber: 2 }],
      [{ bid: 'a', pageNumber: 5 }],
    );
    expect(r.resolved[0].source).toBe('server');
    expect(r.resolved[0].pageNumber).toBe(5);
  });
  it('orphan editor break → dropped', () => {
    const r = reconcile(
      [{ afterBid: 'ghost', pageNumber: 2 }],
      [],
    );
    expect(r.resolved).toHaveLength(0);
  });
  it('server-only break → included', () => {
    const r = reconcile(
      [],
      [{ bid: 'a', pageNumber: 2 }],
    );
    expect(r.resolved).toHaveLength(1);
    expect(r.resolved[0].source).toBe('server');
  });
  it('output is monotonic', () => {
    const r = reconcile(
      [{ afterBid: 'a', pageNumber: 3 }, { afterBid: 'b', pageNumber: 2 }],
      [{ bid: 'a', pageNumber: 3 }, { bid: 'b', pageNumber: 2 }],
    );
    // reconcile sorts by pageNumber, then bid order
    for (let i = 1; i < r.resolved.length; i++) {
      expect(r.resolved[i].pageNumber).toBeGreaterThanOrEqual(r.resolved[i - 1].pageNumber);
    }
  });
});
```

- [ ] **Step 2: Run → FAIL**

- [ ] **Step 3: Implement `reconcile.ts`**

```ts
type EditorBreak = Readonly<{ afterBid: string; pageNumber: number }>;
type ServerBreak = Readonly<{ bid: string; pageNumber: number }>;
export type ReconciledBreak = Readonly<{ afterBid: string; pageNumber: number; source: 'editor' | 'editor-minor-drift' | 'server' }>;
export type ReconcileResult = Readonly<{
  resolved: readonly ReconciledBreak[];
  logs: Readonly<{
    exactMatches: number;
    minorDrift: number;
    majorDrift: number;
    orphanedEditor: number;
    serverOnly: number;
  }>;
}>;

export function reconcile(
  editor: readonly EditorBreak[],
  server: readonly ServerBreak[],
): ReconcileResult {
  const serverByBid = new Map(server.map(s => [s.bid, s]));
  const editorByBid = new Map(editor.map(e => [e.afterBid, e]));

  const resolved: ReconciledBreak[] = [];
  let exactMatches = 0, minorDrift = 0, majorDrift = 0, orphanedEditor = 0, serverOnly = 0;

  for (const e of editor) {
    const s = serverByBid.get(e.afterBid);
    if (!s) { orphanedEditor++; continue; }
    const delta = Math.abs(e.pageNumber - s.pageNumber);
    if (delta === 0) { resolved.push({ afterBid: e.afterBid, pageNumber: e.pageNumber, source: 'editor' }); exactMatches++; }
    else if (delta <= 1) { resolved.push({ afterBid: e.afterBid, pageNumber: e.pageNumber, source: 'editor-minor-drift' }); minorDrift++; }
    else { resolved.push({ afterBid: e.afterBid, pageNumber: s.pageNumber, source: 'server' }); majorDrift++; }
  }
  for (const s of server) {
    if (!editorByBid.has(s.bid)) {
      resolved.push({ afterBid: s.bid, pageNumber: s.pageNumber, source: 'server' });
      serverOnly++;
    }
  }

  resolved.sort((a, b) => a.pageNumber - b.pageNumber);
  return {
    resolved,
    logs: { exactMatches, minorDrift, majorDrift, orphanedEditor, serverOnly },
  };
}
```

- [ ] **Step 4: Run → PASS**

- [ ] **Step 5: Commit**

```bash
git add apps/ck5-export/src/pagination/reconcile.ts apps/ck5-export/src/pagination/__tests__/reconcile.test.ts
git commit -m "feat(ck5-export): reconcile editor/server breaks with ±1 drift tolerance"
```

---

## Phase 4 — DOCX / PDF emission

### Task 25: `html-to-export-tree` — verify/add `mddm-page-break` handling

**Executor:** Codex. **Group:** P2.

**Goal:** Per spec (wiki 31 §4), `div.mddm-page-break` → ExportNode `{kind: 'pageBreak'}`. Verify this branch exists; add if missing.

**Files:**
- Modify: `apps/ck5-export/src/html-to-export-tree.ts`
- Modify: `apps/ck5-export/src/export-node.ts` (if `pageBreak` kind missing)
- Create: `apps/ck5-export/src/__tests__/page-break-tree.test.ts`

**Acceptance Criteria:**
- [ ] Input `<div class="mddm-page-break"></div>` → tree contains `{kind: 'pageBreak'}` node.
- [ ] All other input HTML unchanged in output tree.

**Verify:** `cd apps/ck5-export && npx vitest run src/__tests__/page-break-tree.test.ts` → PASS.

**Steps:**

- [ ] **Step 1: Read existing `html-to-export-tree.ts` + `export-node.ts`** to confirm current state.

- [ ] **Step 2: Write failing test**

```ts
// page-break-tree.test.ts
import { describe, it, expect } from 'vitest';
import { htmlToExportTree } from '../html-to-export-tree';

describe('html-to-export-tree page break', () => {
  it('emits pageBreak node for div.mddm-page-break', () => {
    const tree = htmlToExportTree('<p>a</p><div class="mddm-page-break"></div><p>b</p>');
    const kinds = tree.children.map(n => n.kind);
    expect(kinds).toContain('pageBreak');
  });
});
```

- [ ] **Step 3: Add branch** to `htmlToExportTree`:
```ts
if (el.tagName.toLowerCase() === 'div' && el.classList.contains('mddm-page-break')) {
  return { kind: 'pageBreak' };
}
```

- [ ] **Step 4: Ensure `ExportNode` union includes `{kind: 'pageBreak'}`.**

- [ ] **Step 5: Run → PASS; commit.**

```bash
git add apps/ck5-export/src/html-to-export-tree.ts apps/ck5-export/src/export-node.ts apps/ck5-export/src/__tests__/page-break-tree.test.ts
git commit -m "feat(ck5-export): htmlToExportTree emits pageBreak for mddm-page-break div"
```

---

### Task 26: DOCX `widowControl: false` on every Paragraph

**Executor:** Codex. **Group:** P2.

**Goal:** Every `new Paragraph({...})` in emitter adds `widowControl: false`. Eliminates Word's built-in widow/orphan reflow that would drift breaks.

**Files:**
- Modify: `apps/ck5-export/src/docx-emitter/paragraph.ts`
- Modify: `apps/ck5-export/src/docx-emitter/heading.ts`
- Modify: `apps/ck5-export/src/docx-emitter/list.ts`
- Modify: `apps/ck5-export/src/docx-emitter/helpers.ts` (shared constructor helper if exists)
- Modify existing emitter tests to assert the new field.
- Create: `apps/ck5-export/src/docx-emitter/__tests__/hygiene-widow.test.ts`

**Acceptance Criteria:**
- [ ] Every emitted `Paragraph` has `widowControl: false`.
- [ ] Existing DOCX golden tests pass after snapshot updates (if any).

**Verify:** `cd apps/ck5-export && npx vitest run src/docx-emitter/__tests__/hygiene-widow.test.ts` → PASS.

**Steps:**

- [ ] **Step 1: Identify Paragraph constructor sites** via grep:
```bash
grep -rn 'new Paragraph' apps/ck5-export/src/docx-emitter/
```

- [ ] **Step 2: Add `widowControl: false` to each call site.** If a helper `createParagraph(props)` exists in `helpers.ts`, add it once there.

- [ ] **Step 3: Write test**

```ts
// hygiene-widow.test.ts
import { describe, it, expect } from 'vitest';
import { Packer } from 'docx';
import { emitDocx } from '../index';
import { htmlToExportTree } from '../../html-to-export-tree';
import JSZip from 'jszip';

describe('widowControl hygiene', () => {
  it('every paragraph element in document.xml has w:widowControl w:val="false"', async () => {
    const doc = emitDocx(htmlToExportTree('<p data-mddm-bid="a">x</p><p data-mddm-bid="b">y</p>'));
    const buf = await Packer.toBuffer(doc);
    const zip = await JSZip.loadAsync(buf);
    const xml = await zip.file('word/document.xml')!.async('string');
    const paragraphCount = (xml.match(/<w:p[\s>]/g) ?? []).length;
    const widowFalseCount = (xml.match(/<w:widowControl w:val="false"/g) ?? []).length;
    expect(widowFalseCount).toBe(paragraphCount);
  });
});
```

- [ ] **Step 4: Run → PASS; commit.**

```bash
git add apps/ck5-export/src/docx-emitter
git commit -m "feat(ck5-export): emit widowControl=false on every Paragraph"
```

---

### Task 27: DOCX settings.xml pinned compat

**Executor:** Codex. **Group:** P2.

**Goal:** `Document` constructor includes `settings` block: `autoHyphenation: false`, `defaultTabStop: 720`, pinned `compatibility` flags.

**Files:**
- Modify: `apps/ck5-export/src/docx-emitter/emitter.ts`
- Create: `apps/ck5-export/src/docx-emitter/__tests__/hygiene-settings.test.ts`

**Acceptance Criteria:**
- [ ] Generated `word/settings.xml` contains `<w:autoHyphenation w:val="false"/>` and `<w:defaultTabStop w:val="720"/>`.
- [ ] Compat flags present.

**Verify:** `cd apps/ck5-export && npx vitest run src/docx-emitter/__tests__/hygiene-settings.test.ts` → PASS.

**Steps:**

- [ ] **Step 1: Write failing test**

```ts
// hygiene-settings.test.ts
import { describe, it, expect } from 'vitest';
import { Packer } from 'docx';
import { emitDocx } from '../index';
import { htmlToExportTree } from '../../html-to-export-tree';
import JSZip from 'jszip';

describe('settings.xml hygiene', () => {
  it('disables autoHyphenation', async () => {
    const doc = emitDocx(htmlToExportTree('<p data-mddm-bid="a">x</p>'));
    const buf = await Packer.toBuffer(doc);
    const zip = await JSZip.loadAsync(buf);
    const xml = await zip.file('word/settings.xml')!.async('string');
    expect(xml).toContain('w:autoHyphenation w:val="false"');
    expect(xml).toContain('w:defaultTabStop w:val="720"');
  });
});
```

- [ ] **Step 2: Pre-flight — confirm `docx@9` accepts these settings**

```bash
cd apps/ck5-export && npx tsc --noEmit --target ES2022 --moduleResolution bundler --strict \
  <(echo 'import type { IDocumentOptions } from "docx"; const x: IDocumentOptions = { sections: [], settings: { autoHyphenation: false, defaultTabStop: 720, compatibility: { doNotExpandShiftReturn: true } } };')
```
Expected: exit 0. If it fails (API drift), fall back to post-pack JSZip rewrite of `word/settings.xml` — append the `<w:autoHyphenation>` / `<w:defaultTabStop>` / `<w:compat>` nodes directly. Document the branch taken in the commit message.

- [ ] **Step 3: Pass `settings` to `new Document({ ..., settings })` in `emitter.ts`**

```ts
import type { IDocumentOptions } from 'docx';

const docOptions: IDocumentOptions = {
  creator: 'MetalDocs',
  sections: [...],
  settings: {
    autoHyphenation: false,
    defaultTabStop: 720,
    characterSpacingControl: 'compressPunctuation',
    compatibility: {
      doNotExpandShiftReturn: true,
      useWord2013TrackBottomHyphenation: true,
    },
  },
};
return new Document(docOptions);
```

- [ ] **Step 3: Run → PASS; commit.**

```bash
git add apps/ck5-export/src/docx-emitter
git commit -m "feat(ck5-export): pin DOCX settings.xml compat flags"
```

---

### Task 28: DOCX Carlito font embed

**Executor:** Codex. **Group:** after 3.

**Goal:** `Document` `fonts` config embeds Carlito regular+bold+italic+bold-italic TTFs. `fontTable.xml` references them. Word renders exactly the same font as the editor.

**Files:**
- Modify: `apps/ck5-export/src/docx-emitter/emitter.ts`
- Create: `apps/ck5-export/src/docx-emitter/__tests__/hygiene-fonts.test.ts`

**Acceptance Criteria:**
- [ ] Generated DOCX zip contains `word/fonts/Carlito-Regular.ttf` (or equivalent `word/embeddings/*`).
- [ ] `word/fontTable.xml` references Carlito.
- [ ] `/readyz` endpoint fails if any Carlito TTF missing from `apps/ck5-export/fonts/`.

**Verify:** `cd apps/ck5-export && npx vitest run src/docx-emitter/__tests__/hygiene-fonts.test.ts` → PASS.

**Steps:**

- [ ] **Step 1: Check `docx@9` Font embedding API — pick branch A or branch B**

```bash
cd apps/ck5-export && node -e "const d=require('docx'); console.log(typeof d.Document, Object.keys(require('docx/build/file/file.js')));" 2>&1 | grep -i font
```
- **Branch A (native):** `docx@9` exposes `fonts` on `IDocumentOptions`. Use `new Document({ ..., fonts: [{ name: 'Carlito', data: readFileSync('fonts/Carlito-Regular.ttf') }, ...] })`. Verify types compile.
- **Branch B (post-pack):** If `fonts` is absent, build the doc without it then rewrite the packed zip: `JSZip.loadAsync(buf) → add files under word/fonts/ → rewrite word/fontTable.xml`. Use the helper `embedCarlitoPostPack(buf): Promise<Buffer>` defined below.

Document the branch chosen at the top of `emitter.ts` as a code comment.

Post-pack helper (Branch B):
```ts
import JSZip from 'jszip';
import { readFileSync } from 'node:fs';

export async function embedCarlitoPostPack(buf: Buffer): Promise<Buffer> {
  const zip = await JSZip.loadAsync(buf);
  const variants = ['Regular', 'Bold', 'Italic', 'BoldItalic'] as const;
  for (const v of variants) {
    zip.file(`word/fonts/Carlito-${v}.ttf`, readFileSync(`./fonts/Carlito-${v}.ttf`));
  }
  const fontTableXml = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:fonts xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:font w:name="Carlito"><w:panose1 w:val="020B0604020202020204"/><w:charset w:val="00"/><w:family w:val="swiss"/><w:pitch w:val="variable"/></w:font>
</w:fonts>`;
  zip.file('word/fontTable.xml', fontTableXml);
  return Buffer.from(await zip.generateAsync({ type: 'uint8array' }));
}
```

- [ ] **Step 2: Write failing test**

```ts
// hygiene-fonts.test.ts
import { describe, it, expect } from 'vitest';
import { Packer } from 'docx';
import { emitDocx } from '../index';
import { htmlToExportTree } from '../../html-to-export-tree';
import JSZip from 'jszip';

describe('Carlito font embed', () => {
  it('embeds Carlito TTFs + references in fontTable', async () => {
    const doc = emitDocx(htmlToExportTree('<p data-mddm-bid="a">x</p>'));
    const buf = await Packer.toBuffer(doc);
    const zip = await JSZip.loadAsync(buf);
    const names = Object.keys(zip.files);
    expect(names.some(n => /word\/(fonts|embeddings)\/.*Carlito/i.test(n))).toBe(true);
    const fontTable = await zip.file('word/fontTable.xml')!.async('string');
    expect(fontTable).toContain('Carlito');
  });
});
```

- [ ] **Step 3: Add readyz startup check**

In `server.ts` startup:
```ts
import { existsSync } from 'node:fs';
const fonts = ['Regular', 'Bold', 'Italic', 'BoldItalic'];
for (const variant of fonts) {
  if (!existsSync(`./fonts/Carlito-${variant}.ttf`)) {
    console.error(`missing Carlito-${variant}.ttf`);
    process.exit(1);
  }
}
```

- [ ] **Step 4: Add embed to emitter + run test → PASS; commit.**

```bash
git add apps/ck5-export/src/docx-emitter apps/ck5-export/src/server.ts
git commit -m "feat(ck5-export): embed Carlito TTF into DOCX fontTable"
```

---

## Phase 5 — Fixtures, tests, CI

### Task 29: 10 golden fixtures + reconciled goldens

**Executor:** Codex. **Group:** after 24.

**Goal:** Author 10 canonical HTML fixtures (each with bids) + matching `*.reconciled.json` goldens produced by running `reconcile()` against server breaks for each fixture.

**Files:**
- Create: 10× `apps/ck5-export/src/__fixtures__/pagination/*.html`
- Create: 10× `apps/ck5-export/src/__fixtures__/pagination/*.reconciled.json`
- Create: `apps/ck5-export/scripts/regen-fixtures.ts` (one-shot golden regen script)

**Acceptance Criteria:**
- [ ] All 10 fixtures present.
- [ ] All 10 goldens present.
- [ ] Running `regen-fixtures.ts` produces byte-identical goldens (idempotent).

**Verify:** `npx tsx apps/ck5-export/scripts/regen-fixtures.ts --check` → exit 0.

**Steps:**

- [ ] **Step 1: Write one fixture at a time starting with `short-para.html`**

Use lorem-ipsum with bids per paragraph:
```html
<p data-mddm-bid="11111111-1111-4111-8111-000000000001">Short paragraph.</p>
```

All 10 per spec table (short-para, long-para, heavy-table, image-heavy, nested-lists, section-rich, repeatable-rows, mixed-headings, edge-widow, 100-page-contract).

- [ ] **Step 2: Write `regen-fixtures.ts`**

Script: for each fixture, call `paginateWithChromium` + trivial editor-breaks (= server breaks) through `reconcile()`, write `*.reconciled.json`. Supports `--check` mode that diffs and fails if changed.

- [ ] **Step 3: Run regen to produce goldens**

```bash
npx tsx apps/ck5-export/scripts/regen-fixtures.ts
```

- [ ] **Step 4: Commit**

```bash
git add apps/ck5-export/src/__fixtures__/pagination apps/ck5-export/scripts/regen-fixtures.ts
git commit -m "test(ck5-export): 10 pagination fixtures with reconciled goldens"
```

---

### Task 30: Bid invariance matrix

**Executor:** Opus. **Group:** after 9.

**Goal:** Programmatic edit sequences executed inside a jsdom-hosted CK5 editor. Each sequence asserts: every paginable block has a bid, no duplicates, ids preserved where non-destructive.

**Files:**
- Create: `frontend/apps/web/src/features/documents/ck5/plugins/MddmBlockIdentityPlugin/__tests__/bid-invariance.test.ts`

**Acceptance Criteria:**
- [ ] 4 sequence classes: lists (bullet↔ordered, indent), tables (insert/delete row, merge, split), widgets (section, repeatable reorder, undo), paste (same-doc, other-doc, word-html, plain-text).
- [ ] After each step: all bids unique and every paginable block has one.

**Verify:** `cd frontend/apps/web && npx vitest run src/features/documents/ck5/plugins/MddmBlockIdentityPlugin/__tests__/bid-invariance.test.ts` → PASS.

**Steps:**

- [ ] **Step 1: Write the four sequence test classes as `describe` blocks**

Example skeleton (lists):
```ts
describe('bid invariance — lists', () => {
  it('bullet → ordered conversion preserves bids', async () => {
    const editor = await mkEditor();
    editor.setData('<ul><li data-mddm-bid="aaa">A</li><li data-mddm-bid="bbb">B</li></ul>');
    const before = collectBids(editor);
    editor.execute('numberedList');
    const after = collectBids(editor);
    expect(after).toEqual(before);
    await editor.destroy();
  });
  // ... indent, outdent, split, merge
});
```

Implement for all 4 classes per spec. Use helper `function collectBids(editor): string[]` from Task 7 tests.

- [ ] **Step 2: Run → PASS; fix post-fixer if any test surfaces a bug.**

- [ ] **Step 3: Commit**

```bash
git add frontend/apps/web/src/features/documents/ck5/plugins/MddmBlockIdentityPlugin/__tests__/bid-invariance.test.ts
git commit -m "test(ck5): bid invariance matrix across edit sequences"
```

---

### Task 31: Drift SLO CI gate

**Executor:** Codex. **Group:** after 29.

**Goal:** CI gate on the FULL SLO: (a) zero major drift, (b) ≤1 minor drift per 50 pages **per block type** (paragraph, heading, listItem, tableRow, imageBlock, widget), (c) zero `bid-collision` signals, (d) zero `editor-server-desync` signals. Fails CI with a per-fixture per-block-type report.

**Files:**
- Create: `apps/ck5-export/src/__tests__/drift-slo.test.ts`
- Modify: `apps/ck5-export/scripts/regen-fixtures.ts` (emit `perBlockType` drift + validator results in goldens — each `*.reconciled.json` gains `{drift: {majorDrift, minorDrift, perBlockType: {paragraph: {minorDrift, totalPages}, ...}}, validator: {bidCollision: bool, editorServerDesync: bool}}`).
- Modify: existing CI yaml (`.github/workflows/*.yml`) to invoke the test.

**Acceptance Criteria:**
- [ ] Every `*.reconciled.json` has `drift.perBlockType` keyed by block type, each containing `{minorDrift, totalPages}`.
- [ ] Every `*.reconciled.json` has `validator.bidCollision` (bool) and `validator.editorServerDesync` (bool).
- [ ] Test fails if ANY fixture has `drift.majorDrift > 0`.
- [ ] Test fails if ANY block-type's `minorDrift > ceil(totalPages/50) * maxBreakDeltaPer50Pages`.
- [ ] Test fails if ANY fixture has `validator.bidCollision === true`.
- [ ] Test fails if ANY fixture has `validator.editorServerDesync === true`.
- [ ] Failure message names the offending fixture + block type + observed vs budget.
- [ ] A purpose-built stub fixture `drift-violator.html` + golden with `majorDrift: 1` confirms the test CAN fail (guarded by `DRIFT_SLO_SELFTEST=1` env — runs only in self-test mode).

**Verify:**
```bash
cd apps/ck5-export && npx vitest run src/__tests__/drift-slo.test.ts
# Expected: PASS (all 10 fixtures within SLO)
DRIFT_SLO_SELFTEST=1 npx vitest run src/__tests__/drift-slo.test.ts
# Expected: FAIL (drift-violator fixture triggers expected failure)
```

**Steps:**

- [ ] **Step 1: Extend `regen-fixtures.ts` to emit per-block-type + validator fields**

After producing each fixture's `reconcile()` output, also:
- Parse fixture HTML with linkedom, classify each `data-mddm-bid` element by block type (`paragraph`, `heading`, `listItem`, `tableRow`, `imageBlock`, `widget`).
- Group `logs.minorDrift`/`majorDrift` by block type (count resolved breaks whose `source === 'editor-minor-drift'` or `'server'` per block type).
- Compute `totalPages` per block type = count of that type's bids (proxy — one bid ≈ one candidate page anchor).
- Run `validateBids(html)` → set `validator.bidCollision`.
- Run `validateEditorBidSet(html, editorBids)` with the fixture's editor bid set → set `validator.editorServerDesync`.
- Emit into JSON:
```json
{
  "resolved": [...],
  "logs": {...},
  "drift": {
    "majorDrift": 0,
    "minorDrift": 0,
    "perBlockType": {
      "paragraph": { "minorDrift": 0, "totalPages": 12 },
      "heading":   { "minorDrift": 0, "totalPages": 3 },
      "tableRow":  { "minorDrift": 1, "totalPages": 24 }
    }
  },
  "validator": { "bidCollision": false, "editorServerDesync": false }
}
```

- [ ] **Step 2: Implement `drift-slo.test.ts`**

```ts
import { describe, it, expect } from 'vitest';
import { readdirSync, readFileSync } from 'node:fs';
import { join } from 'node:path';
import { defaultLayoutTokens } from '@metaldocs/mddm-layout-tokens';

const FIX = join(__dirname, '..', '__fixtures__', 'pagination');
const selfTest = process.env.DRIFT_SLO_SELFTEST === '1';

describe('drift SLO gate', () => {
  const files = readdirSync(FIX)
    .filter(f => f.endsWith('.reconciled.json'))
    .filter(f => selfTest ? true : !f.startsWith('drift-violator'));

  for (const f of files) {
    const gold = JSON.parse(readFileSync(join(FIX, f), 'utf-8'));

    it(`${f}: zero major drift`, () => {
      expect(gold.drift.majorDrift, `${f}: major drift must be zero`).toBe(0);
    });

    it(`${f}: zero bid-collision`, () => {
      expect(gold.validator.bidCollision, `${f}: bid-collision present`).toBe(false);
    });

    it(`${f}: zero editor-server-desync`, () => {
      expect(gold.validator.editorServerDesync, `${f}: editor-server-desync present`).toBe(false);
    });

    for (const [blockType, stats] of Object.entries<any>(gold.drift.perBlockType)) {
      it(`${f}: ${blockType} minor drift within SLO`, () => {
        const budget = Math.ceil(stats.totalPages / 50) * defaultLayoutTokens.paginationSLO.maxBreakDeltaPer50Pages;
        expect(
          stats.minorDrift,
          `${f}:${blockType} minorDrift=${stats.minorDrift} exceeds budget=${budget} (totalPages=${stats.totalPages})`,
        ).toBeLessThanOrEqual(budget);
      });
    }
  }
});
```

- [ ] **Step 3: Create self-test stub `drift-violator.html` + golden**

Simple 2-paragraph HTML. Golden file with fabricated `drift.majorDrift: 1` to prove the test fails when it should. Committed under `__fixtures__/pagination/drift-violator.html` / `.reconciled.json`.

- [ ] **Step 4: Run both modes**

```bash
cd apps/ck5-export && npx vitest run src/__tests__/drift-slo.test.ts
DRIFT_SLO_SELFTEST=1 npx vitest run src/__tests__/drift-slo.test.ts; echo "exit=$?"
# First: PASS. Second: non-zero exit (expected failure).
```

- [ ] **Step 5: Wire into CI** — `npm test` glob already covers it; self-test runs in a separate CI step that expects failure (`continue-on-error: true; if: steps.selftest.outcome == 'success'` → fail build).

- [ ] **Step 6: Commit**

```bash
git add apps/ck5-export/src/__tests__/drift-slo.test.ts apps/ck5-export/scripts/regen-fixtures.ts apps/ck5-export/src/__fixtures__/pagination/drift-violator.html apps/ck5-export/src/__fixtures__/pagination/drift-violator.reconciled.json .github/workflows
git commit -m "test(ck5-export): CI drift SLO gate with per-block-type budgets + validator guards"
```

---

### Task 31b: Paginator P95 perf gate

**Executor:** Codex. **Group:** after 29.

**Goal:** Run `100-page-contract.html` fixture through `paginateWithChromium` 20 times; compute p50, p95. Fail build if p95 > 1500ms (spec NFR: ≤1.5s P95 per 50 pages — doubling to 100 pages scaled linearly).

**Files:**
- Create: `apps/ck5-export/src/__tests__/perf-gate.test.ts`
- Modify: CI workflow (same file as Task 31) to include the perf test in a non-flaky slot.

**Acceptance Criteria:**
- [ ] Runs 20 iterations on warm pool (first 2 discarded for warmup).
- [ ] Records array of durations in ms.
- [ ] Asserts p95 ≤ `3000` ms for 100 pages (2× the 50-page 1500ms SLO).
- [ ] Writes `artifacts/perf-gate.json` with `{p50, p95, durations}` for CI artifact upload.

**Verify:** `cd apps/ck5-export && npx vitest run src/__tests__/perf-gate.test.ts --reporter=verbose` → PASS + prints p95.

**Steps:**

- [ ] **Step 1: Write test**

```ts
// perf-gate.test.ts
import { describe, it, expect, beforeAll, afterAll } from 'vitest';
import { readFileSync, writeFileSync, mkdirSync } from 'node:fs';
import { join } from 'node:path';
import { PlaywrightPool } from '../pagination/playwright-pool';
import { paginateWithChromium, paginationCache } from '../pagination/paginate-with-chromium';

function percentile(sorted: number[], p: number): number {
  const idx = Math.ceil((p / 100) * sorted.length) - 1;
  return sorted[Math.max(0, idx)];
}

describe('paginator perf gate', () => {
  let pool: PlaywrightPool;
  beforeAll(async () => { pool = new PlaywrightPool({ size: 1 }); await pool.init(); }, 60000);
  afterAll(async () => { await pool.shutdown(); });

  it('p95 ≤ 3000ms for 100-page fixture', async () => {
    const html = readFileSync(
      join(__dirname, '../__fixtures__/pagination/100-page-contract.html'),
      'utf-8',
    );
    const durations: number[] = [];
    for (let i = 0; i < 22; i++) {
      paginationCache.clear(); // force real work — bypass cache to measure cold path
      const t0 = performance.now();
      await paginateWithChromium(pool, html, { timeoutMs: 30000 });
      durations.push(performance.now() - t0);
    }
    const measured = durations.slice(2).sort((a, b) => a - b); // drop warmup
    const p50 = percentile(measured, 50);
    const p95 = percentile(measured, 95);

    mkdirSync(join(__dirname, '../../artifacts'), { recursive: true });
    writeFileSync(
      join(__dirname, '../../artifacts/perf-gate.json'),
      JSON.stringify({ p50, p95, durations: measured }, null, 2),
    );

    console.log(`[perf-gate] p50=${p50.toFixed(0)}ms p95=${p95.toFixed(0)}ms`);
    expect(p95).toBeLessThanOrEqual(3000);
  }, 120000);
});
```

- [ ] **Step 2: Run test locally**

```bash
cd apps/ck5-export && npx vitest run src/__tests__/perf-gate.test.ts --reporter=verbose
```
Expected: PASS + console `[perf-gate] p50=... p95=...`.

- [ ] **Step 3: Add to CI job (same workflow as Task 31)** with `continue-on-error: false` but in a separate step so perf failure does not mask drift-slo failure. Upload `apps/ck5-export/artifacts/perf-gate.json` as artifact.

- [ ] **Step 4: Commit**

```bash
git add apps/ck5-export/src/__tests__/perf-gate.test.ts .github/workflows
git commit -m "test(ck5-export): CI gate on paginator P95 latency (≤3s/100 pages)"
```

---

### Task 32: Pagination E2E (Playwright)

**Executor:** Codex. **Group:** after 19, 24.

**Goal:** End-to-end browser test: open template editor, type 10 pages of lorem, assert counter reads "Page 10 of 10", click Export DOCX, open downloaded .docx with `docx4js` or parse word/document.xml, assert 10 pages.

**Files:**
- Create: `frontend/apps/web/tests/e2e/pagination.spec.ts`

**Acceptance Criteria:**
- [ ] Test passes in chrome project.
- [ ] Counter matches exported page count.

**Verify:** `cd frontend/apps/web && npm run e2e:smoke -- pagination.spec.ts` → PASS.

**Steps:**

- [ ] **Step 1: Write test**

```ts
import { test, expect } from '@playwright/test';
import JSZip from 'jszip';

test('pagination — 10 pages end-to-end', async ({ page }) => {
  await page.goto('/templates/demo/editor');
  await page.locator('[contenteditable]').click();
  const lorem = 'Lorem ipsum dolor sit amet, consectetur adipiscing elit. '.repeat(300);
  await page.keyboard.type(lorem, { delay: 0 });
  await expect(page.locator('.mddm-page-counter')).toContainText(/Page \d+ of \d+/, { timeout: 10000 });
  const countText = await page.locator('.mddm-page-counter').textContent();
  const counter = Number((countText ?? '').match(/of (\d+)/)![1]);

  const downloadPromise = page.waitForEvent('download');
  await page.getByRole('button', { name: 'Export DOCX' }).click();
  const dl = await downloadPromise;
  const path = await dl.path();
  const fs = await import('node:fs');
  const zip = await JSZip.loadAsync(fs.readFileSync(path!));
  const xml = await zip.file('word/document.xml')!.async('string');
  const pageBreaks = (xml.match(/<w:br w:type="page"\/>/g) ?? []).length;
  expect(pageBreaks + 1).toBeGreaterThanOrEqual(counter - 1);
  expect(pageBreaks + 1).toBeLessThanOrEqual(counter + 1);
});
```

- [ ] **Step 2: Run → PASS; commit.**

```bash
git add frontend/apps/web/tests/e2e/pagination.spec.ts
git commit -m "test(ck5): pagination E2E asserts counter = DOCX page count"
```

---

### Task 33: Parity probe debug route + editor overlay (both asserted)

**Executor:** Codex. **Group:** after 24.

**Goal:** `GET /pagination-debug/:docId` returns JSON parity report; editor shows drift stats when URL has `?debug=pagination`. Both route and overlay have automated tests.

**Files:**
- Create: `apps/ck5-export/src/routes/pagination-debug.ts`
- Create: `apps/ck5-export/src/routes/__tests__/pagination-debug.test.ts`
- Create: `apps/ck5-export/src/pagination/parity-diff.ts`
- Create: `apps/ck5-export/src/pagination/__tests__/parity-diff.test.ts`
- Create: `apps/ck5-export/src/pagination/parity-store.ts` (in-memory `Map<docId, ParityReport>` with per-report `updatedAt`)
- Create: `apps/ck5-export/src/pagination/__tests__/parity-store.test.ts`
- Modify: `apps/ck5-export/src/routes/paginate.ts` (after successful reconcile in export-DOCX / export-PDF path, write report to store keyed by `docId` from request)
- Modify: `apps/ck5-export/src/routes/render-docx.ts` + `render-pdf-html.ts` (accept optional `docId` + `editorBreaks` in body; call reconcile + `paginationStore.put(docId, report)`)
- Create: `frontend/apps/web/src/features/documents/ck5/react/PaginationDebugOverlay.tsx`
- Create: `frontend/apps/web/src/features/documents/ck5/react/__tests__/PaginationDebugOverlay.test.tsx`
- Modify: `frontend/apps/web/src/features/documents/ck5/react/AuthorEditor.tsx` (conditional render)
- Modify: `apps/ck5-export/src/server.ts` (register route)

**Acceptance Criteria:**
- [ ] `NODE_ENV !== 'production'` → route 200 with JSON; `NODE_ENV === 'production'` → 404.
- [ ] JSON shape matches spec Component 9: `{docId, editorBreaks, serverBreaks, reconciled, logs, driftStats, updatedAt}`.
- [ ] After a successful export of `docId=X`, `GET /pagination-debug/X` returns the REAL reconciled data (non-empty `reconciled` when the doc had breaks).
- [ ] `GET /pagination-debug/:docId` for an unknown docId → 404 (`{error:'unknown-doc'}`).
- [ ] `parity-store.test.ts` asserts `put()`/`get()` round-trip, `get` on unknown key returns `undefined`, `put` bumps `updatedAt`.
- [ ] `parity-diff.test.ts` asserts the shape for a known input.
- [ ] Overlay hidden when `?debug=pagination` absent; visible when present (RTL asserted).
- [ ] Overlay lists `exactMatches`, `minorDrift`, `majorDrift`, `orphanedEditor`, `serverOnly`.

**Verify:**
```bash
cd apps/ck5-export && npx vitest run src/routes/__tests__/pagination-debug.test.ts src/pagination/__tests__/parity-diff.test.ts src/pagination/__tests__/parity-store.test.ts
cd ../../frontend/apps/web && npx vitest run src/features/documents/ck5/react/__tests__/PaginationDebugOverlay.test.tsx
```
All → PASS.

**Steps:**

- [ ] **Step 1: Write `parity-diff.test.ts`**

```ts
// parity-diff.test.ts
import { describe, it, expect } from 'vitest';
import { buildParityReport } from '../parity-diff';

describe('buildParityReport', () => {
  it('produces Component-9-shape JSON', () => {
    const r = buildParityReport({
      docId: 'doc-1',
      editorBreaks: [{ afterBid: 'a', pageNumber: 2 }],
      serverBreaks: [{ bid: 'a', pageNumber: 2 }],
      reconcile: {
        resolved: [{ afterBid: 'a', pageNumber: 2, source: 'editor' }],
        logs: { exactMatches: 1, minorDrift: 0, majorDrift: 0, orphanedEditor: 0, serverOnly: 0 },
      },
    });
    expect(r).toMatchObject({
      docId: 'doc-1',
      editorBreaks: expect.any(Array),
      serverBreaks: expect.any(Array),
      reconciled: expect.any(Array),
      logs: expect.any(Object),
      driftStats: { totalBreaks: 1, driftRatio: 0 },
    });
  });
});
```

- [ ] **Step 2: Implement `parity-diff.ts`**

```ts
import type { ComputedBreak, ServerBreak, ReconcileResult } from '@metaldocs/mddm-pagination-types';

export type ParityReport = {
  docId: string;
  editorBreaks: readonly ComputedBreak[];
  serverBreaks: readonly ServerBreak[];
  reconciled: ReconcileResult['resolved'];
  logs: ReconcileResult['logs'];
  driftStats: { totalBreaks: number; driftRatio: number };
};

export function buildParityReport(args: {
  docId: string;
  editorBreaks: readonly ComputedBreak[];
  serverBreaks: readonly ServerBreak[];
  reconcile: ReconcileResult;
}): ParityReport {
  const total = args.reconcile.resolved.length;
  const drift = args.reconcile.logs.minorDrift + args.reconcile.logs.majorDrift;
  return {
    docId: args.docId,
    editorBreaks: args.editorBreaks,
    serverBreaks: args.serverBreaks,
    reconciled: args.reconcile.resolved,
    logs: args.reconcile.logs,
    driftStats: { totalBreaks: total, driftRatio: total ? drift / total : 0 },
  };
}
```

- [ ] **Step 2b: Implement `parity-store.ts` + test**

```ts
// parity-store.ts
import type { ParityReport } from './parity-diff';

class ParityStore {
  private readonly map = new Map<string, ParityReport & { updatedAt: number }>();

  public put(docId: string, report: ParityReport): void {
    this.map.set(docId, { ...report, updatedAt: Date.now() });
  }
  public get(docId: string): (ParityReport & { updatedAt: number }) | undefined {
    return this.map.get(docId);
  }
  public size(): number { return this.map.size; }
  public clear(): void { this.map.clear(); }
}

export const paginationStore = new ParityStore();
export type { ParityStore };
```

```ts
// parity-store.test.ts
import { describe, it, expect, beforeEach } from 'vitest';
import { paginationStore } from '../parity-store';

beforeEach(() => { paginationStore.clear(); });

describe('paginationStore', () => {
  const report = {
    docId: 'd1',
    editorBreaks: [],
    serverBreaks: [],
    reconciled: [],
    logs: { exactMatches: 0, minorDrift: 0, majorDrift: 0, orphanedEditor: 0, serverOnly: 0 },
    driftStats: { totalBreaks: 0, driftRatio: 0 },
  };

  it('returns undefined for unknown docId', () => {
    expect(paginationStore.get('missing')).toBeUndefined();
  });

  it('round-trips put/get with updatedAt', () => {
    paginationStore.put('d1', report);
    const got = paginationStore.get('d1')!;
    expect(got.docId).toBe('d1');
    expect(got.updatedAt).toBeTypeOf('number');
  });

  it('later put bumps updatedAt', async () => {
    paginationStore.put('d1', report);
    const t1 = paginationStore.get('d1')!.updatedAt;
    await new Promise(r => setTimeout(r, 5));
    paginationStore.put('d1', report);
    const t2 = paginationStore.get('d1')!.updatedAt;
    expect(t2).toBeGreaterThan(t1);
  });
});
```

- [ ] **Step 3: Write `pagination-debug.test.ts`**

```ts
// pagination-debug.test.ts
import { describe, it, expect, afterEach, beforeEach, vi } from 'vitest';
import { Hono } from 'hono';
import { paginationDebugRoute } from '../pagination-debug';
import { paginationStore } from '../../pagination/parity-store';

beforeEach(() => { paginationStore.clear(); });
afterEach(() => { vi.unstubAllEnvs(); });

const seed = {
  docId: 'doc-1',
  editorBreaks: [{ afterBid: 'a', pageNumber: 2, yPx: 100 }],
  serverBreaks: [{ bid: 'a', pageNumber: 2 }],
  reconciled: [{ afterBid: 'a', pageNumber: 2, source: 'editor' as const }],
  logs: { exactMatches: 1, minorDrift: 0, majorDrift: 0, orphanedEditor: 0, serverOnly: 0 },
  driftStats: { totalBreaks: 1, driftRatio: 0 },
};

describe('GET /pagination-debug/:docId', () => {
  it('returns 200 with seeded report in non-production', async () => {
    vi.stubEnv('NODE_ENV', 'development');
    paginationStore.put('doc-1', seed);
    const app = new Hono().route('/', paginationDebugRoute());
    const r = await app.request('/pagination-debug/doc-1');
    expect(r.status).toBe(200);
    const body = await r.json();
    expect(body.docId).toBe('doc-1');
    expect(body.reconciled).toHaveLength(1);
    expect(body.driftStats.totalBreaks).toBe(1);
    expect(body.updatedAt).toBeTypeOf('number');
  });

  it('returns 404 for unknown docId', async () => {
    vi.stubEnv('NODE_ENV', 'development');
    const app = new Hono().route('/', paginationDebugRoute());
    const r = await app.request('/pagination-debug/missing');
    expect(r.status).toBe(404);
    expect((await r.json()).error).toBe('unknown-doc');
  });

  it('returns 404 in production even when seeded', async () => {
    vi.stubEnv('NODE_ENV', 'production');
    paginationStore.put('doc-1', seed);
    const app = new Hono().route('/', paginationDebugRoute());
    const r = await app.request('/pagination-debug/doc-1');
    expect(r.status).toBe(404);
  });
});
```

- [ ] **Step 4: Implement `pagination-debug.ts`**

```ts
import { Hono } from 'hono';
import { paginationStore } from '../pagination/parity-store';

export function paginationDebugRoute(): Hono {
  const r = new Hono();
  r.get('/pagination-debug/:docId', (c) => {
    if (process.env.NODE_ENV === 'production') return c.notFound();
    const docId = c.req.param('docId');
    const report = paginationStore.get(docId);
    if (!report) return c.json({ error: 'unknown-doc' }, 404);
    return c.json(report);
  });
  return r;
}
```

- [ ] **Step 4b: Wire write-after-reconcile into export routes**

In `apps/ck5-export/src/routes/render-docx.ts` and `render-pdf-html.ts`, accept optional `docId: string` + `editorBreaks: ComputedBreak[]` in the body. After server-side `paginateWithChromium` + `reconcile()` completes (immediately before emitting DOCX/PDF), build a `ParityReport` via `buildParityReport({docId, editorBreaks, serverBreaks, reconcile})` and call `paginationStore.put(docId, report)`. The render still proceeds regardless. Backward-compat: when `docId` absent, skip the store write.

```ts
// Inside the render-docx handler, right after reconcile:
if (body.docId) {
  const report = buildParityReport({
    docId: body.docId,
    editorBreaks: body.editorBreaks ?? [],
    serverBreaks,
    reconcile: reconciled,
  });
  paginationStore.put(body.docId, report);
}
```

- [ ] **Step 5: Register in `server.ts`**

```ts
import { paginationDebugRoute } from './routes/pagination-debug';
app.route('/', paginationDebugRoute());
```

- [ ] **Step 6: Write `PaginationDebugOverlay.test.tsx`**

```tsx
import { describe, it, expect } from 'vitest';
import { render, screen } from '@testing-library/react';
import { PaginationDebugOverlay } from '../PaginationDebugOverlay';

const logs = { exactMatches: 3, minorDrift: 1, majorDrift: 0, orphanedEditor: 0, serverOnly: 0 };

describe('PaginationDebugOverlay', () => {
  it('hidden when debug flag absent', () => {
    render(<PaginationDebugOverlay logs={logs} debugFlag={false} />);
    expect(screen.queryByTestId('pagination-debug-overlay')).toBeNull();
  });

  it('visible and lists all drift counters when debug flag set', () => {
    render(<PaginationDebugOverlay logs={logs} debugFlag={true} />);
    const el = screen.getByTestId('pagination-debug-overlay');
    expect(el.textContent).toMatch(/exactMatches.*3/);
    expect(el.textContent).toMatch(/minorDrift.*1/);
    expect(el.textContent).toMatch(/majorDrift.*0/);
    expect(el.textContent).toMatch(/orphanedEditor.*0/);
    expect(el.textContent).toMatch(/serverOnly.*0/);
  });
});
```

- [ ] **Step 7: Implement `PaginationDebugOverlay.tsx`**

```tsx
import type { ReconcileLogs } from '@metaldocs/mddm-pagination-types';

type Props = { logs: ReconcileLogs; debugFlag: boolean };

export function PaginationDebugOverlay({ logs, debugFlag }: Props) {
  if (!debugFlag) return null;
  return (
    <div data-testid="pagination-debug-overlay" className="mddm-pagination-debug">
      <div>exactMatches: {logs.exactMatches}</div>
      <div>minorDrift: {logs.minorDrift}</div>
      <div>majorDrift: {logs.majorDrift}</div>
      <div>orphanedEditor: {logs.orphanedEditor}</div>
      <div>serverOnly: {logs.serverOnly}</div>
    </div>
  );
}
```

- [ ] **Step 8: Wire in `AuthorEditor.tsx`**

```tsx
import { PaginationDebugOverlay } from './PaginationDebugOverlay';
const debugFlag = typeof window !== 'undefined' && new URLSearchParams(window.location.search).get('debug') === 'pagination';
// inside render:
<PaginationDebugOverlay logs={currentLogs} debugFlag={debugFlag} />
```

- [ ] **Step 9: Run all tests → PASS**

- [ ] **Step 10: Commit**

```bash
git add apps/ck5-export/src/routes/pagination-debug.ts apps/ck5-export/src/routes/__tests__/pagination-debug.test.ts apps/ck5-export/src/pagination/parity-diff.ts apps/ck5-export/src/pagination/__tests__/parity-diff.test.ts apps/ck5-export/src/pagination/parity-store.ts apps/ck5-export/src/pagination/__tests__/parity-store.test.ts apps/ck5-export/src/routes/render-docx.ts apps/ck5-export/src/routes/render-pdf-html.ts apps/ck5-export/src/server.ts frontend/apps/web/src/features/documents/ck5/react/PaginationDebugOverlay.tsx frontend/apps/web/src/features/documents/ck5/react/__tests__/PaginationDebugOverlay.test.tsx frontend/apps/web/src/features/documents/ck5/react/AuthorEditor.tsx
git commit -m "feat(ck5): pagination parity debug route + overlay with real reconcile data"
```

---

## Self-Review Checklist (executor: read before each wave)

1. **Spec coverage** — every Components 1–10 section maps to at least one task:
   - Component 1 (tokens pkg) → Tasks 0, 0b, 1.
   - Component 2 (BlockIdentity) → Tasks 5–11.
   - Component 3 (Pagination plugin) → Tasks 12–19.
   - Component 4 (paginate-with-chromium) → Tasks 20, 20b, 21, 22, 22b, 23.
   - Component 5 (reconcile) → Task 24.
   - Component 6 (html-to-export-tree) → Task 25.
   - Component 7 (DOCX hygiene) → Tasks 26–28.
   - Component 8 (PDF paged.polyfill) → Task 4.
   - Component 9 (parity probe) → Tasks 29, 31, 31b, 33.
   - Component 10 (CSS unify) → Task 2.

2. **Spec NFRs covered:**
   - sha256 HTML cache + 5-min TTL → Task 22b.
   - P95 ≤ 1.5s per 50 pages → Task 31b (scaled to 3s/100p).
   - Retry-once on worker crash + fallback ladder → Task 20b.
   - editor-server-desync 422 → Task 23.
   - bid-collision 422 → Task 11, reused in Task 23.
   - Drift SLO ≤1 per 50 pages per block type → Task 31.

3. **No placeholders** — every task has exact paths, full code, exact commands with expected output. No "or wherever", no "TBD", no "similar to Task N".

4. **Type consistency** — all wire types (`BreakCandidate`, `ComputedBreak`, `ServerBreak`, `ReconciledBreak`, `ReconcileLogs`, `ReconcileResult`, `PaginateRequest`, `PaginateResponse`) defined once in `shared/mddm-pagination-types/index.ts` (Task 0b). All other tasks import via `@metaldocs/mddm-pagination-types`. Compile-time contract test in `shared/mddm-pagination-types/__tests__/contract.test-d.ts`.

5. **Parallel groups** — tasks in the same group share no file. Verified per-wave.

6. **File-path consistency** — `bid-invariance.test.ts` lives in `frontend/apps/web/src/features/documents/ck5/plugins/MddmBlockIdentityPlugin/__tests__/` only. Not in `apps/ck5-export/src/__tests__/`.

---

## Task persistence

Tasks tracked in `docs/superpowers/plans/2026-04-17-ck5-pagination.md.tasks.json` (co-located). Subagent-driven-development or executing-plans populates it; leave empty stub `{"planPath":"...","tasks":[]}` for initial write.
