# CK5 Plan C — Production Readiness Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use `superpowers-extended-cc:subagent-driven-development` to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship CK5 as the sole production document engine. Build DOCX/PDF export (standalone `apps/ck5-export/` service + Go proxy), ship template publish gate (`draft → pending_review → published`), delete BlockNote/MDDM stack.

**Architecture:** Two-PR rollout. PR1 = additive (`apps/ck5-export/` Node.js Hono service replaces `apps/docgen/`, Go proxies via `/export/ck5/docx|pdf`, UI adds ExportMenu + PublishButton). PR2 = destructive (template publish DB migration + state machine, full deletion of `mddm-editor/` + `apps/docgen/` + `@blocknote/*`). CK5 HTML = single source of truth; IR ephemeral on export only.

**Tech Stack:** Node 20 + Hono + linkedom + docx (npm) for `ck5-export`. Go API proxies via `net/http`. Gotenberg (existing) handles PDF conversion. React + CKEditor5 on frontend.

**Execution model:** Codex (`gpt-5.3-codex`, reasoning_effort=high) writes all code via `mcp__codex__codex` subagent dispatch. Opus reviews every Phase completion via `superpowers-extended-cc:code-reviewer` before advancing.

**Conventions:**
- All tasks dispatched to Codex subagent via `mcp__codex__codex` with `sandbox: workspace-write`, `approval-policy: on-failure`.
- TDD enforced: failing test → implementation → passing test → commit.
- Commits scoped `feat(ck5-export):`, `feat(ck5-api):`, `feat(ck5-ui):`, `chore(ck5):`, `test(ck5):`.
- Working directory: worktree root `../MetalDocs-ck5-plan-c`.
- Use `rtk` prefix on terminal commands.
- After each Phase: controller invokes `superpowers-extended-cc:code-reviewer` before Phase N+1.

**Caveman directive:** Task goal lines + subagent prompts written terse. Code blocks = normal. Commit messages = normal.

---

## Phase 0 — Worktree Setup

### Task 0: Create worktree and branch

**Goal:** Isolated worktree for Plan C on top of Plan B merged `main`.

**Files:** None (git only).

**Acceptance Criteria:**
- [ ] Worktree `../MetalDocs-ck5-plan-c` exists on branch `migrate/ck5-plan-c`
- [ ] Clean `git status`

**Verify:** `cd ../MetalDocs-ck5-plan-c && rtk git status` → `On branch migrate/ck5-plan-c, nothing to commit`

**Steps:**

- [ ] **Step 1: Verify clean main**

Run: `rtk git status`
Expected: `On branch main`, no conflicts.

- [ ] **Step 2: Create worktree**

```bash
rtk git worktree add ../MetalDocs-ck5-plan-c -b migrate/ck5-plan-c
```

Expected: `Preparing worktree (new branch 'migrate/ck5-plan-c')`.

- [ ] **Step 3: Verify worktree**

```bash
cd ../MetalDocs-ck5-plan-c && rtk git status && ls apps/ frontend/apps/web/src/features/documents/ck5/
```

Expected: branch `migrate/ck5-plan-c`, `ck5/` directory present.

### Phase 0 Review Checkpoint

Controller invokes `superpowers-extended-cc:code-reviewer`:
- Worktree path correct
- Branch off current `main`
- No residual Plan B uncommitted files

**Gate:** Reviewer APPROVE → Phase 1. Else → fix and re-review.

---

## Phase 1 — `apps/ck5-export/` Service Skeleton

**Goal:** Empty Hono service on port 9001, no routes yet. Scaffolding only. Phase ends with `pnpm --filter @metaldocs/ck5-export dev` booting.

### Task 1: Scaffold package + tsconfig

**Goal:** `apps/ck5-export/package.json` + `tsconfig.json` + `tsconfig.build.json`.

**Files:**
- Create: `apps/ck5-export/package.json`
- Create: `apps/ck5-export/tsconfig.json`
- Create: `apps/ck5-export/tsconfig.build.json`
- Create: `apps/ck5-export/vitest.config.ts`
- Create: `apps/ck5-export/.gitignore`

**Acceptance Criteria:**
- [ ] `pnpm install` at repo root resolves `@metaldocs/ck5-export` workspace
- [ ] `pnpm --filter @metaldocs/ck5-export typecheck` passes on empty src
- [ ] `dist/` and `node_modules/` gitignored

**Verify:**
```bash
cd apps/ck5-export && rtk pnpm install && rtk pnpm typecheck
```
Expected: zero errors.

**Subagent prompt (Codex gpt-5.3-codex):**
```
Scaffold apps/ck5-export/ Node workspace package.
deps: hono@^4, linkedom@^0.18, docx@^9
devDeps: @types/node@^25, typescript@^5.4, vitest@^4.1, tsx@^4
scripts:
  dev: "tsx watch src/server.ts"
  build: "tsc -p tsconfig.build.json"
  typecheck: "tsc --noEmit"
  test: "vitest run"
  start: "node dist/server.js"
tsconfig extends apps/docgen/tsconfig.json shape (NodeNext, strict, target ES2022).
Add minimal src/server.ts placeholder exporting a Hono app with no routes.
Add .gitignore (dist, node_modules, coverage).
```

**Steps:**

- [ ] **Step 1: Dispatch Codex**

Tool: `mcp__codex__codex`
- model: `gpt-5.3-codex`
- sandbox: `workspace-write`
- prompt: (above)

- [ ] **Step 2: Run typecheck**

```bash
cd apps/ck5-export && rtk pnpm install && rtk pnpm typecheck
```
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
rtk git add apps/ck5-export/package.json apps/ck5-export/tsconfig*.json apps/ck5-export/vitest.config.ts apps/ck5-export/.gitignore apps/ck5-export/src/server.ts pnpm-lock.yaml
rtk git commit -m "chore(ck5-export): scaffold workspace package"
```

### Task 2: Hono server boot + health route

**Goal:** `GET /health` returns 200 `{ok:true}`. Port from `PORT` env, default 9001.

**Files:**
- Modify: `apps/ck5-export/src/server.ts`
- Create: `apps/ck5-export/src/__tests__/server.health.test.ts`

**Acceptance Criteria:**
- [ ] Health test passes (serve Hono via `app.fetch`)
- [ ] `pnpm dev` boots without crash

**Verify:**
```bash
cd apps/ck5-export && rtk pnpm vitest run src/__tests__/server.health.test.ts
```
Expected: 1 pass.

**Subagent prompt (Codex):**
```
In apps/ck5-export/src/server.ts:
- Use `hono` + `@hono/node-server` (add to deps).
- Export `app: Hono` and a `start(port)` fn.
- Register GET /health → JSON {ok: true, service: "ck5-export"}.
- If run as main module, read PORT env (default 9001), call start.

Write test apps/ck5-export/src/__tests__/server.health.test.ts:
- import app
- res = await app.request("/health")
- expect res.status === 200
- expect await res.json() === {ok: true, service: "ck5-export"}
```

**Steps:**

- [ ] **Step 1: Dispatch Codex**
- [ ] **Step 2: Run `rtk pnpm vitest run`** → PASS
- [ ] **Step 3: Commit**

```bash
rtk git add apps/ck5-export/src/server.ts apps/ck5-export/src/__tests__/server.health.test.ts apps/ck5-export/package.json pnpm-lock.yaml
rtk git commit -m "feat(ck5-export): hono health route + bootstrapping"
```

### Task 3: Add `ck5-export` entry to launch.json

**Goal:** `preview_start ck5-plan-c-export` boots service on port 9001.

**Files:**
- Modify: `.claude/launch.json`

**Acceptance Criteria:**
- [ ] `launch.json` contains new entry `ck5-plan-c-export`
- [ ] JSON validates

**Verify:**
```bash
node -e "JSON.parse(require('fs').readFileSync('.claude/launch.json'))"
```
Expected: no throw.

**Subagent prompt (Codex):**
```
Add configuration entry to .claude/launch.json (KEEP existing entries):
{
  "name": "ck5-plan-c-export",
  "runtimeExecutable": "pnpm",
  "runtimeArgs": ["--dir", "../MetalDocs-ck5-plan-c/apps/ck5-export", "run", "dev"],
  "port": 9001
}
Also add ck5-plan-c-api (port 8083) and ck5-plan-c-web (port 4175) mirroring Plan B entries but pointed at the plan-c worktree.
```

**Steps:**
- [ ] **Step 1: Dispatch Codex**
- [ ] **Step 2: Verify JSON parse**
- [ ] **Step 3: Commit** `chore(ck5): launch.json entries for plan-c worktrees`

### Phase 1 Review Checkpoint

Controller invokes `superpowers-extended-cc:code-reviewer`:
- package.json deps match spec (hono, linkedom, docx, vitest, tsx)
- tsconfig strict mode enabled
- `pnpm dev` starts on 9001
- Health route returns expected shape

**Gate:** APPROVE → Phase 2.

---

## Phase 2 — Move Engine Modules from mddm-editor

**Goal:** Relocate `docx-emitter/`, `asset-resolver/`, `print-stylesheet/`, `inline-asset-rewriter.ts` from `mddm-editor/engine/` to `apps/ck5-export/src/`. Keep mddm-editor importable until PR2 deletion. Tests pass in new location.

### Task 4: Copy `docx-emitter/` tree

**Goal:** Verbatim copy under `apps/ck5-export/src/docx-emitter/` incl. `__tests__/`. Rewrite relative imports. Tests green.

**Files:**
- Create (copy): `apps/ck5-export/src/docx-emitter/**`
- Modify: imports inside copied files

**Acceptance Criteria:**
- [ ] All tests under `apps/ck5-export/src/docx-emitter/__tests__/` pass
- [ ] No cross-package imports back to `frontend/apps/web/**`

**Verify:**
```bash
cd apps/ck5-export && rtk pnpm vitest run src/docx-emitter/
```
Expected: all tests pass (same count as mddm-editor version).

**Subagent prompt (Codex):**
```
Task: Copy docx-emitter module from mddm-editor to ck5-export.

Source: frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/
Target: apps/ck5-export/src/docx-emitter/

Steps:
1. Copy every file and __tests__/ subfolder verbatim.
2. Rewrite imports: any import referencing "../asset-resolver", "../layout-ir", "../helpers", "../editor-tokens" etc MUST be resolved. Rules:
   - ../asset-resolver → ../asset-resolver (copy that too IF imported)
   - ../layout-ir → for now keep as relative reference; if layout-ir is NOT part of the move, create a local type file apps/ck5-export/src/layout-ir/index.ts that RE-EXPORTS the types from an inlined copy. Prefer inline copy — do NOT leave dangling imports back to mddm-editor.
   - ../helpers, ../editor-tokens, ../guards → inline copies needed by docx-emitter into apps/ck5-export/src/shared/ (create as needed, only pull in what is imported).
3. After moving, run vitest. Fix any type errors. Do NOT modify test assertions.
4. The original mddm-editor copy stays untouched for now (deleted in Phase 12).

Constraints:
- Zero imports from mddm-editor to ck5-export.
- Zero imports from ck5-export back to frontend/apps/web.
- Keep test goldens (.docx / .xml bytes) byte-identical.
```

**Steps:**
- [ ] **Step 1: Dispatch Codex**
- [ ] **Step 2: Run tests** → PASS
- [ ] **Step 3: Commit** `chore(ck5-export): move docx-emitter from mddm-editor`

### Task 5: Copy `asset-resolver/` tree

**Goal:** Relocate with tests passing.

**Files:**
- Create (copy): `apps/ck5-export/src/asset-resolver/**`

**Acceptance Criteria:**
- [ ] Tests pass in new location
- [ ] Uses `node-fetch` or native `fetch` for URL download

**Verify:**
```bash
cd apps/ck5-export && rtk pnpm vitest run src/asset-resolver/
```
Expected: all tests pass.

**Subagent prompt (Codex):**
```
Copy asset-resolver from mddm-editor/engine/asset-resolver/ to apps/ck5-export/src/asset-resolver/.
Same import-rewrite rules as Task 4.
Tests must pass in new location without modification.
```

**Steps:**
- [ ] **Step 1: Dispatch Codex**
- [ ] **Step 2: Tests PASS**
- [ ] **Step 3: Commit** `chore(ck5-export): move asset-resolver from mddm-editor`

### Task 6: Copy `print-stylesheet/` tree

**Goal:** Relocate with `print-css.ts` + `wrap-print-document.ts` + tests.

**Files:**
- Create (copy): `apps/ck5-export/src/print-stylesheet/**`

**Acceptance Criteria:**
- [ ] Tests pass
- [ ] `wrapInPrintDocument(html)` returns full HTML string with embedded CSS

**Verify:**
```bash
cd apps/ck5-export && rtk pnpm vitest run src/print-stylesheet/
```
Expected: all tests pass.

**Subagent prompt:** Same pattern as Tasks 4-5, scoped to `print-stylesheet/`.

**Steps:**
- [ ] Dispatch Codex → tests PASS → commit `chore(ck5-export): move print-stylesheet from mddm-editor`

### Task 7: Copy `inline-asset-rewriter.ts`

**Goal:** Relocate single file from `mddm-editor/engine/export/inline-asset-rewriter.ts` to `apps/ck5-export/src/inline-asset-rewriter.ts`. Tests (if any) move too.

**Files:**
- Create (copy): `apps/ck5-export/src/inline-asset-rewriter.ts`
- Create (copy): `apps/ck5-export/src/__tests__/inline-asset-rewriter.test.ts` (if exists in source)

**Acceptance Criteria:**
- [ ] Tests pass
- [ ] Uses relocated `asset-resolver/`

**Verify:**
```bash
cd apps/ck5-export && rtk pnpm vitest run src/__tests__/inline-asset-rewriter.test.ts
```
Expected: pass (or skip if no source test).

**Subagent prompt:** Analogous.

**Steps:**
- [ ] Dispatch Codex → PASS → commit `chore(ck5-export): move inline-asset-rewriter from mddm-editor`

### Phase 2 Review Checkpoint

Controller invokes `superpowers-extended-cc:code-reviewer`:
- Zero imports from `apps/ck5-export/**` to `frontend/apps/web/**`
- Tests moved, counts match source
- `rtk pnpm --filter @metaldocs/ck5-export test` green
- `rtk pnpm --filter @metaldocs/web test` still green (mddm-editor still intact, pre-PR2)

**Verify cmd (controller runs):**
```bash
rtk grep "from.*mddm-editor" apps/ck5-export/src/
```
Expected: 0 results.

**Gate:** APPROVE → Phase 3.

---

## Phase 3 — `html-to-export-tree.ts`

**Goal:** New module in `apps/ck5-export/src/` that walks CK5 HTML → `ExportNode` tree. Unit-tested with HTML fixtures.

### Task 8: Define `ExportNode` types

**Goal:** `apps/ck5-export/src/export-node.ts` with full discriminated union matching spec's shape mapping table.

**Files:**
- Create: `apps/ck5-export/src/export-node.ts`
- Create: `apps/ck5-export/src/__tests__/export-node.test.ts` (compile-time exhaustiveness check)

**Acceptance Criteria:**
- [ ] Types exported: `ExportNode`, `Section`, `Repeatable`, `RepeatableItem`, `Table`, `TableRow`, `TableCell`, `Field`, `Heading`, `Paragraph`, `List`, `ListItem`, `Image`, `Hyperlink`, `Text`, `LineBreak`, `Blockquote`
- [ ] Discriminated by `kind: "..."` literal

**Verify:**
```bash
cd apps/ck5-export && rtk pnpm typecheck && rtk pnpm vitest run src/__tests__/export-node.test.ts
```
Expected: PASS.

**Subagent prompt (Codex):**
```
Create apps/ck5-export/src/export-node.ts defining the ExportNode discriminated union.

Kinds (each is a TS interface):
- section { kind:"section"; variant:"solid"|"bordered"|"plain"; header?: ExportNode[]; body: ExportNode[] }
- repeatable { kind:"repeatable"; items: RepeatableItem[] }
- repeatableItem { kind:"repeatableItem"; children: ExportNode[] }
- table { kind:"table"; variant:"fixed"|"dynamic"; rows: TableRow[] }
- tableRow { kind:"tableRow"; cells: TableCell[] }
- tableCell { kind:"tableCell"; isHeader: boolean; children: ExportNode[]; colspan?: number; rowspan?: number }
- field { kind:"field"; id: string; fieldType: "text"|"number"|"date"|"boolean"|"select"; value: string }
- heading { kind:"heading"; level: 1|2|3|4|5|6; children: ExportNode[] }
- paragraph { kind:"paragraph"; align?: "left"|"right"|"center"|"justify"; children: ExportNode[] }
- list { kind:"list"; ordered: boolean; items: ListItem[] }
- listItem { kind:"listItem"; children: ExportNode[] }
- image { kind:"image"; src: string; alt?: string; width?: number; height?: number }
- hyperlink { kind:"hyperlink"; href: string; children: ExportNode[] }
- text { kind:"text"; value: string; marks?: ("bold"|"italic"|"underline"|"strike")[] }
- lineBreak { kind:"lineBreak" }
- blockquote { kind:"blockquote"; children: ExportNode[] }

Export a top-level type ExportNode = Section | Repeatable | ... | Blockquote;

Test file src/__tests__/export-node.test.ts: compile-time-only assertions using satisfies operator — confirm a nested literal can be typed as ExportNode.
```

**Steps:**
- [ ] Dispatch Codex → typecheck PASS → commit `feat(ck5-export): define ExportNode discriminated union`

### Task 9: HTML fixture library

**Goal:** Minimal HTML strings covering each CK5 shape.

**Files:**
- Create: `apps/ck5-export/src/__fixtures__/section-with-fields.html`
- Create: `apps/ck5-export/src/__fixtures__/table-fixed.html`
- Create: `apps/ck5-export/src/__fixtures__/table-dynamic.html`
- Create: `apps/ck5-export/src/__fixtures__/repeatable.html`
- Create: `apps/ck5-export/src/__fixtures__/rich-block.html`
- Create: `apps/ck5-export/src/__fixtures__/nested-formatting.html`

**Acceptance Criteria:**
- [ ] Each fixture = standalone snippet matching shape-mapping table
- [ ] No `<html>`/`<body>` wrappers

**Verify:** Visual inspection; each fixture non-empty.

**Subagent prompt (Codex):**
```
Create HTML fixtures under apps/ck5-export/src/__fixtures__/. Each matches spec shape-mapping table.

section-with-fields.html:
<section class="mddm-section" data-variant="bordered">
  <div class="mddm-section-header"><h2>Client Info</h2></div>
  <div class="mddm-section-body">
    <p>Name: <span class="mddm-field" data-field-id="client_name" data-field-type="text">ACME Corp</span></p>
    <p>Date: <span class="mddm-field" data-field-id="order_date" data-field-type="date">2026-04-16</span></p>
  </div>
</section>

table-fixed.html:
<figure class="table" data-variant="fixed">
  <table>
    <thead><tr><th>Item</th><th>Qty</th></tr></thead>
    <tbody><tr><td>Widget</td><td>10</td></tr></tbody>
  </table>
</figure>

table-dynamic.html: same as fixed but data-variant="dynamic".

repeatable.html:
<ol class="mddm-repeatable" data-field-id="items">
  <li><p>Row one</p></li>
  <li><p>Row two</p></li>
</ol>

rich-block.html:
<div class="mddm-rich-block">
  <p>Wrapped paragraph should be unwrapped.</p>
  <ul><li>One</li><li>Two</li></ul>
</div>

nested-formatting.html:
<p>Hello <strong>bold <em>italic</em></strong> and <a href="https://example.com">link</a>.<br>Line 2.</p>
```

**Steps:** Dispatch → commit `test(ck5-export): add HTML fixtures for export tree`.

### Task 10: Implement `htmlToExportTree`

**Goal:** DOM walker. Uses `linkedom` `parseHTML`. Returns `ExportNode[]`.

**Files:**
- Create: `apps/ck5-export/src/html-to-export-tree.ts`
- Create: `apps/ck5-export/src/__tests__/html-to-export-tree.test.ts`

**Acceptance Criteria:**
- [ ] Fn signature `export function htmlToExportTree(html: string): ExportNode[]`
- [ ] Each fixture in Task 9 → snapshot assertion passes
- [ ] Unwraps `<div class="mddm-rich-block">` (emits children in place)
- [ ] Unwraps `<span class="restricted-editing-exception">`
- [ ] Preserves text order, handles mixed inline content

**Verify:**
```bash
cd apps/ck5-export && rtk pnpm vitest run src/__tests__/html-to-export-tree.test.ts
```
Expected: all snapshots pass.

**Subagent prompt (Codex):**
```
Implement htmlToExportTree(html: string): ExportNode[] in apps/ck5-export/src/html-to-export-tree.ts.

Requirements:
1. Parse via `linkedom` parseHTML.
2. Walk childNodes of document.body recursively.
3. For each element, map to ExportNode per this table:
   - <section class="mddm-section"> → section { variant from data-variant, header: from .mddm-section-header, body: from .mddm-section-body }
   - <ol class="mddm-repeatable"> → repeatable { items: map each <li> to repeatableItem }
   - <figure class="table"><table> → table { variant from data-variant }, rows from <tr>, cells from <th>/<td>
   - <span class="mddm-field"> → field { id from data-field-id, fieldType from data-field-type, value=textContent }
   - <div class="mddm-rich-block"> → unwrap (emit children, do not produce a node)
   - <span class="restricted-editing-exception"> → unwrap
   - <h1>..<h6> → heading { level }
   - <p> → paragraph { align from style.textAlign if set }
   - <ul>/<ol> → list { ordered }
   - <li> → listItem
   - <img> → image { src, alt, width/height from attrs if numeric }
   - <a> → hyperlink { href }
   - <strong>/<b>, <em>/<i>, <u>, <s>/<strike> → wrap text children with marks
   - <br> → lineBreak
   - <blockquote> → blockquote
   - text node → text { value }

4. Tests: for each fixture (section-with-fields, table-fixed, table-dynamic, repeatable, rich-block, nested-formatting), read fixture, call htmlToExportTree, compare to expected ExportNode literal (not snapshot — explicit deep equal).

5. No eslint-disable. No any except the linkedom untyped interface wrapper.
```

**Steps:**
- [ ] Dispatch Codex (round 1: failing tests via explicit assertions)
- [ ] Tests PASS
- [ ] Commit `feat(ck5-export): html-to-export-tree DOM walker + tests`

### Task 10b: Golden parity — dual-path AST equivalence

**Goal:** For every golden under `apps/ck5-export/src/docx-emitter/__tests__/` that was migrated from `mddm-editor`, assert the new ExportNode path produces the same `docx` AST as the legacy IR path. Retire IR-only fixtures with logged reason in commit body.

**Files:**
- Create: `apps/ck5-export/src/__tests__/golden-parity.test.ts`
- Create: `apps/ck5-export/__fixtures-ir__/` — copy of legacy IR-shaped inputs (read-only, temp for parity window)
- Create: `docs/superpowers/plans/2026-04-16-ck5-plan-c-production-readiness-parity-log.md`

**Acceptance Criteria:**
- [ ] For each legacy golden: pair (IR-input → emitDocx) and (HTML-input → htmlToExportTree → emitDocx) → deep-equal via `packer`'s intermediate AST (or file-content hash if simpler)
- [ ] Goldens with no HTML equivalent → skipped with `.todo` + entry added to parity log explaining why (feature dead, handled differently in CK5, etc.)
- [ ] Parity log committed alongside tests

**Verify:**
```bash
cd apps/ck5-export && rtk pnpm vitest run src/__tests__/golden-parity.test.ts
```
Expected: every golden either passes equivalence OR is explicitly `todo`-skipped with log entry referencing it.

**Subagent prompt (Codex):**
```
For each golden under apps/ck5-export/src/docx-emitter/__tests__/ (inherited from mddm-editor):

1. Inventory: list every *.test.ts golden and whether it has an HTML fixture equivalent under apps/ck5-export/src/__fixtures__/. If missing, create the HTML equivalent that would produce the same logical document.

2. For each (IR input, HTML input) pair:
   - Build via old path: emitDocx(irInput) → doc1
   - Build via new path: emitDocx(htmlToExportTree(htmlInput)) → doc2
   - Compare: serialize both to XML via Packer.toString or similar, assert equality (ignoring RSID/timestamp fields — scrub via regex).

3. For goldens WITHOUT an HTML equivalent (e.g., IR-only features not supported by CK5):
   - Mark with test.todo("retired: <reason>")
   - Append one-line entry to docs/superpowers/plans/2026-04-16-ck5-plan-c-production-readiness-parity-log.md:
     "<golden-name>: retired — <reason>"

4. Goal: zero silent regressions between IR-era and CK5-HTML-era DOCX output.

Legacy IR inputs: if not preserved as fixtures in mddm-editor, pull from git history (mddm-editor tests). Create apps/ck5-export/__fixtures-ir__/ as a temporary snapshot directory.
```

**Steps:**
- [ ] Dispatch Codex (round 1: inventory)
- [ ] Review inventory output — confirm HTML equivalents exist or retirement justified
- [ ] Dispatch Codex (round 2: implement parity tests + log)
- [ ] Run tests → all PASS or `todo`
- [ ] Commit `test(ck5-export): dual-path golden parity + retirement log`

### Phase 3 Review Checkpoint

Controller invokes `code-reviewer`. Check:
- Unwrap behavior for `rich-block` + `restricted-editing-exception`
- Shape-mapping table full coverage
- Test count ≥ 6 (one per fixture) + additional inline-mark edge cases
- No `any` beyond linkedom wrapper
- Idempotent + pure (no side effects)
- **Parity log present; every retired golden has documented reason; active parity tests all PASS**

**Gate:** APPROVE → Phase 4.

---

## Phase 4 — `ck5-export` Routes

**Goal:** `POST /render/docx` + `POST /render/pdf-html` functional. Integration tests with golden `.docx` bytes + wrapped HTML strings.

### Task 11: `POST /render/docx` route

**Goal:** Route wires `htmlToExportTree → asset collect → AssetResolver.resolveAll → emitDocx → Packer.toBuffer`. Returns binary bytes.

**Files:**
- Modify: `apps/ck5-export/src/server.ts` (add route)
- Create: `apps/ck5-export/src/routes/render-docx.ts`
- Create: `apps/ck5-export/src/__tests__/render-docx.test.ts`

**Acceptance Criteria:**
- [ ] `POST /render/docx {html}` → 200 + `application/vnd.openxmlformats-officedocument.wordprocessingml.document`
- [ ] Empty body → 400
- [ ] Non-JSON body → 400
- [ ] Valid fixture HTML → returns non-empty Buffer; first 4 bytes = ZIP magic `PK\x03\x04`

**Verify:**
```bash
cd apps/ck5-export && rtk pnpm vitest run src/__tests__/render-docx.test.ts
```
Expected: PASS (≥ 3 test cases).

**Subagent prompt (Codex):**
```
Implement POST /render/docx in apps/ck5-export/src/routes/render-docx.ts.

Handler logic:
1. Parse JSON body { html: string }. If html missing or not string → 400 {error: "html required"}.
2. const tree = htmlToExportTree(html)
3. const urls = collectImageUrls(tree)   // helper in this file; walks tree collecting image.src
4. const assetMap = await AssetResolver.resolveAll(urls)  // import from ../asset-resolver
5. const doc = emitDocx(tree, assetMap)   // import from ../docx-emitter
6. const buf = await Packer.toBuffer(doc)  // import Packer from "docx"
7. Return c.body(buf, 200, { "Content-Type": "application/vnd.openxmlformats-officedocument.wordprocessingml.document" })

Error handling: try/catch → on error log + return 500 {error: message}.

Wire route in src/server.ts: app.post("/render/docx", renderDocxHandler).

Test src/__tests__/render-docx.test.ts:
- Case 1: POST with section-with-fields fixture → 200, first 4 bytes = [0x50, 0x4b, 0x03, 0x04]
- Case 2: POST {} → 400, error body contains "html required"
- Case 3: POST with malformed JSON → 400
- Mock asset-resolver fetch to return empty buffer for any URL.
```

**Steps:**
- [ ] Dispatch Codex → PASS → commit `feat(ck5-export): POST /render/docx route`

### Task 12: `POST /render/pdf-html` route

**Goal:** Returns wrapped HTML (text/html) that Gotenberg consumes.

**Files:**
- Create: `apps/ck5-export/src/routes/render-pdf-html.ts`
- Modify: `apps/ck5-export/src/server.ts`
- Create: `apps/ck5-export/src/__tests__/render-pdf-html.test.ts`

**Acceptance Criteria:**
- [ ] `POST /render/pdf-html {html}` → 200 + `text/html`
- [ ] Response body contains `<!DOCTYPE html>` + inlined `<style>` block (from `print-stylesheet/print-css.ts`)
- [ ] Image URLs inlined as data URIs via `inlineAssetRewriter`

**Verify:**
```bash
cd apps/ck5-export && rtk pnpm vitest run src/__tests__/render-pdf-html.test.ts
```
Expected: ≥ 2 tests PASS.

**Subagent prompt (Codex):**
```
Implement POST /render/pdf-html in apps/ck5-export/src/routes/render-pdf-html.ts.

Handler logic:
1. Parse body { html }. If missing → 400.
2. const withAssets = await inlineAssetRewriter(html)   // import from ../inline-asset-rewriter
3. const wrapped = wrapInPrintDocument(withAssets)      // import from ../print-stylesheet
4. c.body(wrapped, 200, { "Content-Type": "text/html; charset=utf-8" })

Wire in server.ts.

Tests:
- Case 1: POST with <p>Hello</p> → 200, body includes "<!DOCTYPE html>" AND "<style>"
- Case 2: POST {} → 400
- Mock inlineAssetRewriter to be identity for test.
```

**Steps:**
- [ ] Dispatch Codex → PASS → commit `feat(ck5-export): POST /render/pdf-html route`

### Task 13: Server integration smoke test

**Goal:** Boot server on ephemeral port, hit both routes via real HTTP.

**Files:**
- Create: `apps/ck5-export/src/__tests__/server.integration.test.ts`

**Acceptance Criteria:**
- [ ] Boots via `@hono/node-server` on port 0 (ephemeral)
- [ ] Real HTTP POST to both routes returns 200 with correct content-types
- [ ] Graceful shutdown in `afterAll`

**Verify:**
```bash
cd apps/ck5-export && rtk pnpm vitest run src/__tests__/server.integration.test.ts
```
Expected: PASS.

**Subagent prompt (Codex):**
```
Write integration test booting the Hono server on port 0 via @hono/node-server's serve.
Inside beforeAll: const server = serve({ fetch: app.fetch, port: 0 }); capture assigned port from server.address().
Inside afterAll: await new Promise(r => server.close(r)).

Test cases:
- POST http://127.0.0.1:{port}/render/docx with {html: "<p>Hi</p>"} → 200, content-type docx mime.
- POST http://127.0.0.1:{port}/render/pdf-html with same → 200, content-type text/html.
- GET /health → 200.
```

**Steps:**
- [ ] Dispatch Codex → PASS → commit `test(ck5-export): server integration smoke`

### Phase 4 Review Checkpoint

Controller runs `code-reviewer`:
- Both routes respect spec (content-types, 400 on missing body)
- Error handler returns structured JSON on 4xx
- Timeout-safe (AssetResolver has ceiling from `asset-resolver/ceilings.ts`)
- No process.exit or unhandled-rejection sinks

Controller also runs **manually:**
```bash
cd apps/ck5-export && rtk pnpm build && node dist/server.js &
curl -X POST http://localhost:9001/render/docx -H "Content-Type: application/json" -d '{"html":"<p>Hi</p>"}' -o /tmp/smoke.docx
file /tmp/smoke.docx  # should say "Microsoft Word 2007+"
```

**Gate:** APPROVE → Phase 5.

---

## Phase 5 — Go DOCX + PDF Export Proxy

**Goal:** Two new Go handlers on Go API (port 8083). Auth via existing middleware. Proxy to `ck5-export` + Gotenberg.

### Task 14: `CK5ExportClient` service

**Goal:** Thin wrapper around `http.Client` targeting `ck5-export` base URL. Two methods: `RenderDocx(ctx, html) ([]byte, error)`, `RenderPDFHtml(ctx, html) (string, error)`.

**Files:**
- Create: `internal/modules/documents/application/service_ck5_export_client.go`
- Create: `internal/modules/documents/application/service_ck5_export_client_test.go`

**Acceptance Criteria:**
- [ ] 30s timeout
- [ ] Non-200 from ck5-export → returns typed error with status code
- [ ] Tests use `httptest.NewServer` to stub `ck5-export`

**Verify:**
```bash
cd internal/modules/documents/application && rtk go test -run "CK5ExportClient" ./...
```
Expected: PASS.

**Subagent prompt (Codex):**
```
Create Go file internal/modules/documents/application/service_ck5_export_client.go.

package application

type CK5ExportClient struct {
    baseURL string
    http    *http.Client
}

func NewCK5ExportClient(baseURL string) *CK5ExportClient {
    return &CK5ExportClient{ baseURL: baseURL, http: &http.Client{ Timeout: 30 * time.Second } }
}

type CK5ExportError struct {
    Status int
    Body   string
}
func (e *CK5ExportError) Error() string { return fmt.Sprintf("ck5-export returned %d: %s", e.Status, e.Body) }

func (c *CK5ExportClient) RenderDocx(ctx context.Context, html string) ([]byte, error):
  - POST baseURL + "/render/docx" with JSON {html}
  - If resp.StatusCode != 200 → return nil, &CK5ExportError{Status: resp.StatusCode, Body: readAll(resp.Body)}
  - Return body bytes.

func (c *CK5ExportClient) RenderPDFHtml(ctx context.Context, html string) (string, error):
  - POST baseURL + "/render/pdf-html" with JSON {html}
  - On non-200 same error handling
  - Return string(body).

Test file service_ck5_export_client_test.go:
- Stub via httptest.NewServer.
- TestRenderDocx_OK: stub returns 200 + bytes "PK\x03\x04stub" → client returns same bytes, no error.
- TestRenderDocx_Non200: stub returns 500 → client returns *CK5ExportError with Status==500.
- TestRenderDocx_Timeout: stub sleeps 40s, client has 500ms timeout override via small test-only constructor; assert ctx deadline error.
- Same three tests for RenderPDFHtml.
```

**Steps:**
- [ ] Dispatch Codex → `rtk go test` PASS → commit `feat(ck5-api): CK5ExportClient service`

### Task 15: `GetCK5DocumentContent` augmentation

**Goal:** Service method `GetCK5DocumentContent(ctx, docID) (html string, title string, err error)` that fetches latest version where `ContentSource = ck5_browser`. Returns `ErrNotFound` sentinel if no such version.

**Files:**
- Modify: `internal/modules/documents/application/service_ck5.go` (extend Plan B's service)
- Modify: `internal/modules/documents/application/service_ck5_test.go`

**Acceptance Criteria:**
- [ ] Returns latest ck5_browser version HTML + document title
- [ ] Returns `ErrNotFound` if document absent OR no ck5_browser version exists
- [ ] Existing Plan B tests still pass

**Verify:**
```bash
cd internal/modules/documents/application && rtk go test -run "GetCK5DocumentContent" ./...
```
Expected: PASS.

**Subagent prompt (Codex):**
```
Extend internal/modules/documents/application/service_ck5.go with:

func (s *Service) GetCK5DocumentContent(ctx context.Context, docID string) (html, title string, err error):
- Fetch document by ID. If not found → return "", "", ErrDocumentNotFound.
- Query versions where DocumentID==docID AND ContentSource=="ck5_browser" ORDER BY Number DESC LIMIT 1.
- If no version → return "", "", ErrDocumentNotFound.
- Return version.Content + document.Title + nil.

Add test cases:
- TestGetCK5DocumentContent_OK: seed doc with two versions (native + ck5_browser) → returns ck5_browser content.
- TestGetCK5DocumentContent_NoCK5Version: seed only native → ErrDocumentNotFound.
- TestGetCK5DocumentContent_MissingDoc: → ErrDocumentNotFound.

Keep Plan B's GetCK5DocumentContentAuthorized untouched.
```

**Steps:**
- [ ] Dispatch Codex → PASS → commit `feat(ck5-api): add GetCK5DocumentContent service method`

### Task 16: DOCX export handler

**Goal:** `GET /api/v1/documents/{id}/export/ck5/docx` per spec.

**Files:**
- Create: `internal/modules/documents/delivery/http/handler_ck5_export.go`
- Create: `internal/modules/documents/delivery/http/handler_ck5_export_test.go`
- Modify: `internal/modules/documents/delivery/http/handler.go` (route wire)

**Acceptance Criteria:**
- [ ] Auth: `isAllowed(CapabilityDocumentView)` → 404 if denied (no info leak)
- [ ] No ck5_browser version → 404
- [ ] Calls `CK5ExportClient.RenderDocx(ctx, html)`
- [ ] 200 + `Content-Type: application/vnd.openxml…` + `Content-Disposition: attachment; filename="{title}.docx"`
- [ ] ck5-export 5xx/timeout → 502
- [ ] ck5-export 400 → 422

**Verify:**
```bash
cd internal/modules/documents/delivery/http && rtk go test -run "CK5ExportDocx" ./...
```
Expected: ≥ 5 tests PASS.

**Subagent prompt (Codex):**
```
Create handler_ck5_export.go. Route: GET /api/v1/documents/{id}/export/ck5/docx.

func (h *Handler) handleDocumentExportCK5Docx(w http.ResponseWriter, r *http.Request):
1. docID := r.PathValue("id")  (or however mux extracts, follow existing handlers)
2. Require auth → capability check (pattern: isAllowed(ctx, CapabilityDocumentView, docID)). If denied → 404.
3. html, title, err := h.service.GetCK5DocumentContent(ctx, docID). If ErrDocumentNotFound → 404.
4. bytes, err := h.ck5Export.RenderDocx(ctx, html).
   - If err is *CK5ExportError and Status >= 400 && Status < 500 → 422 + error msg.
   - If err != nil → 502 + "upstream ck5-export error".
5. w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
6. w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s.docx"`, sanitize(title)))
7. w.Write(bytes)

sanitize: replace any non-[A-Za-z0-9-_] with "_", cap length 128.

Wire in handler.go inside handleDocumentsSubRoutes.

Tests handler_ck5_export_test.go:
- TestExportDocx_OK_200: seed doc+ck5_browser version, stub CK5ExportClient.RenderDocx returns bytes → 200, correct Content-Type, Content-Disposition contains title.
- TestExportDocx_NoAuth_404: unauth request → 404.
- TestExportDocx_NoCK5Version_404: doc with only native version → 404.
- TestExportDocx_Upstream500_502: stub returns CK5ExportError{500} → 502.
- TestExportDocx_Upstream400_422: stub returns CK5ExportError{400} → 422.
```

**Steps:**
- [ ] Dispatch Codex → PASS → commit `feat(ck5-api): GET /documents/{id}/export/ck5/docx`

### Task 17: PDF export handler + Gotenberg proxy

**Goal:** `GET /api/v1/documents/{id}/export/ck5/pdf` — proxies wrapped HTML into existing Gotenberg client.

**Files:**
- Modify: `internal/modules/documents/delivery/http/handler_ck5_export.go` (add second handler)
- Modify: `internal/modules/documents/delivery/http/handler_ck5_export_test.go`
- Modify: `internal/modules/documents/delivery/http/handler.go` (route wire)

**Acceptance Criteria:**
- [ ] Flow: fetch ck5_browser HTML → `ck5Export.RenderPDFHtml` → existing `GotenbergClient.ConvertHTML(wrapped, printCss)` → stream PDF
- [ ] Content-Type `application/pdf`, Content-Disposition `attachment; filename="{title}.pdf"`
- [ ] Error mapping identical to DOCX: ck5-export 4xx → 422, ck5-export 5xx/timeout → 502, Gotenberg 5xx/timeout → 502
- [ ] Explicit tests for each mapping (not just happy path)

**Verify:**
```bash
cd internal/modules/documents/delivery/http && rtk go test -run "CK5ExportPdf" ./...
```
Expected: ≥ 6 tests PASS.

**Subagent prompt (Codex):**
```
Add handleDocumentExportCK5PDF to handler_ck5_export.go.

Flow:
1. Auth/fetch same as DOCX.
2. wrappedHTML, err := h.ck5Export.RenderPDFHtml(ctx, html). Map errors as in DOCX.
3. Locate existing GotenbergClient (search for "gotenberg" in internal/modules/documents/**; Plan B/docgen may have it). If not found, dispatch a sub-question — do NOT invent. For now assume h.gotenberg.ConvertHTML(ctx, wrappedHTML, printCSS) returns []byte.
4. printCSS := loaded from apps/ck5-export/src/print-stylesheet/print-css.ts… NO — in Go we embed a copy in internal/modules/documents/infrastructure/ck5_print_css.go as a string constant. Copy the CSS content verbatim from print-css.ts.
5. Stream PDF bytes with correct headers.

Wire route. Add tests (full error-mapping parity with DOCX):
- TestExportPdf_OK_200: stub ck5Export + gotenberg → 200 + application/pdf.
- TestExportPdf_NoCK5Version_404.
- TestExportPdf_NoAuth_404.
- TestExportPdf_CK5Export400_422: ck5Export returns CK5ExportError{Status:400} → 422.
- TestExportPdf_CK5Export500_502: ck5Export returns CK5ExportError{Status:500} → 502.
- TestExportPdf_CK5ExportTimeout_502: ck5Export returns ctx.DeadlineExceeded-wrapped error → 502.
- TestExportPdf_Gotenberg500_502: gotenberg returns 500 → 502.
- TestExportPdf_GotenbergTimeout_502: gotenberg times out → 502.
```

**Steps:**
- [ ] Dispatch Codex → PASS → commit `feat(ck5-api): GET /documents/{id}/export/ck5/pdf`

### Phase 5 Review Checkpoint

Controller runs `code-reviewer`:
- No info leak on auth failure (404 not 401/403)
- Correct error mapping (422 vs 502)
- Filename sanitization safe from path traversal
- `go build ./...` clean
- `go test ./internal/modules/documents/...` green
- Route wired in both handler.go and actually reachable (integration test covers)

**Gate:** APPROVE → Phase 6.

---

## Phase 6 — Frontend Export UI

**Goal:** `ExportMenu.tsx` + `exportApi.ts` with DOCX/PDF buttons + client-side print preview.

### Task 18: `exportApi.ts`

**Goal:** API client + `clientPrint` helper.

**Files:**
- Create: `frontend/apps/web/src/features/documents/ck5/persistence/exportApi.ts`
- Create: `frontend/apps/web/src/features/documents/ck5/persistence/__tests__/exportApi.test.ts`

**Acceptance Criteria:**
- [ ] `triggerExport(docId, fmt)` performs fetch → blob → anchor-download flow
- [ ] `clientPrint(editor)` opens hidden iframe, writes wrapped HTML, invokes print
- [ ] Throws `ExportError` with status on non-ok
- [ ] Vitest mocks fetch + `URL.createObjectURL`

**Verify:**
```bash
cd frontend/apps/web && rtk pnpm vitest run src/features/documents/ck5/persistence/__tests__/exportApi.test.ts
```
Expected: ≥ 4 tests PASS.

**Subagent prompt (Codex):**
```
Create frontend/apps/web/src/features/documents/ck5/persistence/exportApi.ts.

Exports:
- class ExportError extends Error { constructor(public status: number, msg?: string) }
- async function triggerExport(docId: string, fmt: "docx" | "pdf"): Promise<void>
- function clientPrint(editor: DecoupledEditor): void    // DecoupledEditor from @ckeditor/ckeditor5-editor-decoupled

triggerExport impl: verbatim from spec lines 165-175.

clientPrint impl: verbatim from spec lines 180-188. Import wrapInPrintDocument from a LOCAL copy — do NOT reach into apps/ck5-export/. Create frontend/apps/web/src/features/documents/ck5/print/wrap-print-document.ts that re-exports a minimal in-browser wrapper (reuse the CSS string from ck5-export via duplication — they will diverge minimally; acceptable because browser wrapper is a subset).

Actually simpler: import wrapInPrintDocument from "../print/wrap-print-document" — create that file with a minimal inline version.

Test exportApi.test.ts:
- Mock fetch: returns { ok: true, blob: async () => new Blob(["x"]) }. Call triggerExport → no throw, createObjectURL called, anchor.click called.
- Mock fetch returns ok: false, status: 500 → throws ExportError with status 500.
- clientPrint: stub document.createElement iframe, assert iframe.contentWindow.print called.
```

**Steps:**
- [ ] Dispatch Codex → PASS → commit `feat(ck5-ui): exportApi + clientPrint helpers`

### Task 19: `ExportMenu.tsx` component

**Goal:** Three buttons (DOCX, PDF, Print Preview). Calls `exportApi` functions.

**Files:**
- Create: `frontend/apps/web/src/features/documents/ck5/react/components/ExportMenu.tsx`
- Create: `frontend/apps/web/src/features/documents/ck5/react/components/__tests__/ExportMenu.test.tsx`
- Create: `frontend/apps/web/src/features/documents/ck5/react/components/ExportMenu.module.css`

**Acceptance Criteria:**
- [ ] Props: `{ docId: string; editor: DecoupledEditor | null; disabled?: boolean }`
- [ ] Three buttons; disabled when `editor==null` or `disabled`
- [ ] DOCX → `triggerExport(docId, "docx")`; PDF → `triggerExport(docId, "pdf")`; Print → `clientPrint(editor)`
- [ ] Shows inline error text on catch

**Verify:**
```bash
cd frontend/apps/web && rtk pnpm vitest run src/features/documents/ck5/react/components/__tests__/ExportMenu.test.tsx
```
Expected: ≥ 4 tests PASS.

**Subagent prompt (Codex):**
```
Create ExportMenu.tsx React component.

Props: { docId: string; editor: DecoupledEditor | null; disabled?: boolean }

Render three <button>s inside a <div className={styles.menu}>:
- "Export DOCX" → onClick: triggerExport(docId, "docx").catch(setErr)
- "Export PDF" → onClick: triggerExport(docId, "pdf").catch(setErr)
- "Print Preview" → onClick: editor && clientPrint(editor)

Error state via useState<string|null>. Render <span role="alert"> if err.

All buttons disabled if !editor || disabled.

CSS Module minimal: .menu { display: flex; gap: 8px; } .btn { ... }

Tests ExportMenu.test.tsx (@testing-library/react):
- Renders 3 buttons.
- Buttons disabled when editor prop is null.
- Click "Export DOCX" calls mocked triggerExport with (docId, "docx").
- Click "Print Preview" calls mocked clientPrint with editor.
- Error from triggerExport shows role="alert" text.
```

**Steps:**
- [ ] Dispatch Codex → PASS → commit `feat(ck5-ui): ExportMenu component`

### Task 20: Wire `ExportMenu` into AuthorPage + FillPage

**Goal:** Replace any placeholder export UI; ExportMenu in toolbar region.

**Files:**
- Modify: `frontend/apps/web/src/features/documents/ck5/react/AuthorPage.tsx`
- Modify: `frontend/apps/web/src/features/documents/ck5/react/FillPage.tsx`

**Acceptance Criteria:**
- [ ] AuthorPage renders `<ExportMenu docId={...} editor={editor} />` when in fill mode (template preview), disabled otherwise OR always shown but disabled until editor ready
- [ ] FillPage renders `<ExportMenu docId={docId} editor={editor} />` always
- [ ] Existing tests still green

**Verify:**
```bash
cd frontend/apps/web && rtk pnpm vitest run src/features/documents/ck5/react/
```
Expected: existing tests green + new wiring confirmed.

**Subagent prompt (Codex):**
```
Wire ExportMenu into AuthorPage.tsx and FillPage.tsx.

AuthorPage: it's a template editor. There's no docId for the template itself in the same shape — but the page has a testing "preview-as-doc" case, or the Publish flow covers that in Phase 10. For Plan C Phase 6: render ExportMenu ONLY in fill-mode preview subviews if present; otherwise defer. PREFERRED: read current AuthorPage.tsx first, find the toolbar region, add ExportMenu with docId={authorCtx.previewDocId ?? ""} disabled={!authorCtx.previewDocId}.

FillPage: always render <ExportMenu docId={docId} editor={editor} /> near the save button area. Read FillPage.tsx first to locate the right insertion point.

Update existing tests that snapshot these pages (expect the new buttons to appear).
```

**Steps:**
- [ ] Dispatch Codex (will read files itself)
- [ ] PASS → commit `feat(ck5-ui): wire ExportMenu into AuthorPage + FillPage`

### Phase 6 Review Checkpoint

Controller runs `code-reviewer`. Check:
- No hard-coded URLs (uses relative `/api/v1/...`)
- ExportError correctly surfaced to user (not swallowed)
- Disabled state driven by editor readiness
- iframe cleanup on blur/afterprint (memory leak check)

Controller manual check:
```bash
cd frontend/apps/web && rtk pnpm typecheck && rtk pnpm vitest run src/features/documents/ck5/
```

**Gate:** APPROVE → Phase 7.

---

## Phase 7 — PR1 Preview Smoke Validation

**Goal:** End-to-end validate export flow works across full stack in preview env. Merge PR1 on green.

### Task 21: Preview smoke — Author flow

**Goal:** Controller runs preview validation manually. Results recorded in PR description.

**Files:** None (validation only).

**Acceptance Criteria:**
- [ ] `preview_start ck5-plan-c-export` OK (port 9001)
- [ ] `preview_start ck5-plan-c-api` OK (port 8083)
- [ ] `preview_start ck5-plan-c-web` OK (port 4175)
- [ ] Author flow: navigate → save → click Export DOCX → blob download observed → `preview_network` shows GET /export/ck5/docx 200 + POST /render/docx 200
- [ ] Click Print Preview → `window.print` fires, no network call
- [ ] Zero console errors

**Verify:** Checklist filled in PR description.

**Steps:**

- [ ] **Step 1: Start services**

```
mcp__Claude_Preview__preview_start ck5-plan-c-export
mcp__Claude_Preview__preview_start ck5-plan-c-api
mcp__Claude_Preview__preview_start ck5-plan-c-web
```

Expected: three `serverId`s.

- [ ] **Step 2: Navigate & save**

```
preview_eval window.location.hash = "#/test-harness/ck5?mode=author&tpl=sandbox"
preview_snapshot   → verify toolbar + ExportMenu visible
preview_eval await window.__ck5.save()
preview_network   → assert PUT /api/v1/templates/sandbox/ck5-draft 200
```

- [ ] **Step 3: Export DOCX**

```
preview_click button[data-testid="ck5-export-docx"]   (or similar per Task 19 wiring)
preview_network   → assert GET /api/v1/documents/.../export/ck5/docx 200
                    assert POST http://localhost:9001/render/docx 200
```

- [ ] **Step 4: Print Preview**

```
preview_click button[data-testid="ck5-print-preview"]
preview_network   → assert ZERO new requests
```

- [ ] **Step 5: Zero console errors**

```
preview_console_logs level=error   → expect empty
```

- [ ] **Step 6: Log results in PR description**

### Task 22: Preview smoke — Fill flow + PDF

**Files:** None.

**Acceptance Criteria:**
- [ ] Fill flow navigates, fields editable, save 201
- [ ] Export PDF → GET /export/ck5/pdf 200 → POST /render/pdf-html 200 → Gotenberg call 200
- [ ] Blob downloaded, content-type application/pdf

**Steps:**
- [ ] Navigate `#/test-harness/ck5?mode=fill&tpl=sandbox&doc=sandbox-doc`
- [ ] `window.__ck5.save()` → POST /documents/sandbox-doc/content/ck5 201
- [ ] Click Export PDF button → confirm network chain
- [ ] Confirm blob download (file size > 1KB)

### Task 23: PR1 merge

**Goal:** Open PR1 with scope = additive only (no deletes yet).

**Files:** None (git only).

**Acceptance Criteria:**
- [ ] PR opened on `migrate/ck5-plan-c` targeting `main`
- [ ] Title: `feat(ck5): Plan C PR1 — ck5-export service + DOCX/PDF export + ExportMenu`
- [ ] Body lists tasks 0-22 + preview smoke evidence

**Steps:**
- [ ] Push branch, `gh pr create`
- [ ] Request review

### Phase 7 Review Checkpoint

Controller runs `code-reviewer` + `superpowers-extended-cc:verification-before-completion`:
- All smoke checks green
- No regressions in existing `frontend/apps/web` tests
- `rtk go test ./...` green
- Bundle still contains `@blocknote/*` (PR2 scope — do NOT remove yet)

**Gate:** APPROVE + PR1 merged → Phase 8 (start of destructive PR2).

---

## Phase 8 — Template Publish Schema + State Machine

**Goal:** DB migration + state machine (`draft → pending_review → published`) + service methods.

### Task 24: DB migration 0077

**Goal:** Adds `published_html TEXT` + `status TEXT` columns with check constraint.

**Files:**
- Create: `migrations/0077_add_template_publish_state.sql`

**Acceptance Criteria:**
- [ ] Migration file exists
- [ ] Runs clean against empty template_drafts
- [ ] Existing rows get `status='draft'`, `published_html=NULL`

**Verify:**
```bash
rtk go run ./cmd/migrate up
rtk psql $DATABASE_URL -c "\d metaldocs.template_drafts"
```
Expected: columns `published_html`, `status` visible.

**Subagent prompt (Codex):**
```
Create migrations/0077_add_template_publish_state.sql:

-- 0077_add_template_publish_state.sql
-- Adds template publish state machine (draft → pending_review → published).

ALTER TABLE metaldocs.template_drafts
  ADD COLUMN IF NOT EXISTS published_html TEXT,
  ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'draft'
    CHECK (status IN ('draft', 'pending_review', 'published'));

CREATE INDEX IF NOT EXISTS idx_template_drafts_status ON metaldocs.template_drafts(status);
```

**Steps:**
- [ ] Dispatch Codex → run migrate → commit `feat(ck5-api): migration 0077 add template publish state`

### Task 25: Repository publish state methods

**Goal:** `UpdateTemplateStatus`, `PublishTemplate` in memory + postgres repos.

**Files:**
- Modify: `internal/modules/documents/infrastructure/memory/template_drafts_repo.go`
- Modify: `internal/modules/documents/infrastructure/postgres/template_drafts_repo.go` (if exists; else locate real impl)
- Modify: `internal/modules/documents/domain/model.go` (add `TemplateStatus` enum type)
- Create: infrastructure tests

**Acceptance Criteria:**
- [ ] `TemplateStatus` enum: `draft|pending_review|published`
- [ ] `UpdateTemplateStatus(ctx, key, newStatus)` CAS-safe (concurrent callers serialize)
- [ ] `PublishTemplate(ctx, key, publishedHTML)` atomic (writes both fields)
- [ ] In-memory + postgres impls covered

**Verify:**
```bash
cd internal/modules/documents/infrastructure/memory && rtk go test ./...
```
Expected: PASS.

**Subagent prompt (Codex):**
```
Extend template_drafts repo interface with:
  UpdateTemplateStatus(ctx, key, newStatus TemplateStatus) error
  PublishTemplate(ctx, key, publishedHTML string) error  // sets status=published + published_html

Add TemplateStatus type in domain/model.go:
type TemplateStatus string
const (
  TemplateStatusDraft          TemplateStatus = "draft"
  TemplateStatusPendingReview  TemplateStatus = "pending_review"
  TemplateStatusPublished      TemplateStatus = "published"
)

Add Status field to TemplateDraft struct: Status TemplateStatus `json:"status"`.
Add PublishedHTML *string to TemplateDraft (nullable).

Tests for memory repo:
- UpdateTemplateStatus on non-existent key → error.
- UpdateTemplateStatus draft→pending_review → reflected on next Get.
- PublishTemplate on pending_review → status=published + published_html stored.

Postgres repo: do NOT hit real DB in test; use sqlmock if present, else leave postgres changes verified via integration test in Task 28.
```

**Steps:**
- [ ] Dispatch Codex → PASS → commit `feat(ck5-api): repo methods for template publish state`

### Task 26: Publish service + state machine

**Goal:** `PublishTemplateForReview(ctx, key, actor)` + `ApproveTemplate(ctx, key, actor)` with proper state transitions + capability checks.

**Files:**
- Create: `internal/modules/documents/application/service_ck5_template_publish.go`
- Create: `internal/modules/documents/application/service_ck5_template_publish_test.go`

**Acceptance Criteria:**
- [ ] `PublishTemplateForReview`: status must == draft (else `ErrInvalidTemplateStatus`); contentHtml must not be empty (else `ErrEmptyTemplateContent`); transitions to pending_review
- [ ] `ApproveTemplate`: status must == pending_review; copies blocks_json._ck5.contentHtml → published_html; transitions to published
- [ ] Cap checks delegated via capability adapter (CapabilityTemplateEdit for publish, CapabilityTemplatePublish for approve)

**Verify:**
```bash
cd internal/modules/documents/application && rtk go test -run "TemplatePublish|TemplateApprove" ./...
```
Expected: ≥ 6 tests PASS.

**Subagent prompt (Codex):**
```
Create service_ck5_template_publish.go with two service methods.

func (s *Service) PublishTemplateForReview(ctx, key string) error:
  draft, err := s.repo.GetTemplateDraft(ctx, key) → if not found → ErrTemplateNotFound
  if draft.Status != TemplateStatusDraft → return ErrInvalidTemplateStatus
  html := extractCK5ContentHtml(draft.BlocksJSON)   // reads _ck5.contentHtml from JSON bytes
  if strings.TrimSpace(html) == "" → return ErrEmptyTemplateContent
  return s.repo.UpdateTemplateStatus(ctx, key, TemplateStatusPendingReview)

func (s *Service) ApproveTemplate(ctx, key string) error:
  draft, err := s.repo.GetTemplateDraft(ctx, key) → if not found → ErrTemplateNotFound
  if draft.Status != TemplateStatusPendingReview → return ErrInvalidTemplateStatus
  html := extractCK5ContentHtml(draft.BlocksJSON)
  return s.repo.PublishTemplate(ctx, key, html)

Export sentinels:
var ErrInvalidTemplateStatus = errors.New("invalid template status for operation")
var ErrEmptyTemplateContent  = errors.New("template contentHtml is empty")

Tests:
- PublishTemplateForReview_OK (draft+non-empty html → OK, status=pending_review).
- PublishTemplateForReview_EmptyHtml → ErrEmptyTemplateContent.
- PublishTemplateForReview_WrongStatus (already pending) → ErrInvalidTemplateStatus.
- ApproveTemplate_OK (pending → published, published_html set).
- ApproveTemplate_WrongStatus (still draft) → ErrInvalidTemplateStatus.
- ApproveTemplate_NotFound → ErrTemplateNotFound.
```

**Steps:**
- [ ] Dispatch Codex → PASS → commit `feat(ck5-api): template publish/approve service methods`

### Task 27: Publish + Approve handlers

**Goal:** `POST /api/v1/templates/{key}/publish` + `POST /api/v1/templates/{key}/approve`.

**Files:**
- Create: `internal/modules/documents/delivery/http/handler_ck5_template_publish.go`
- Create: `internal/modules/documents/delivery/http/handler_ck5_template_publish_test.go`
- Modify: `internal/modules/documents/delivery/http/template_admin_handler.go` (wire routes)

**Acceptance Criteria (auth contract — strict 401/403, NO 404 masking):**
- [ ] `/publish`: auth=CapabilityTemplateEdit; 401 when no session; 403 insufficient cap; 404 when templateKey missing; 409 on wrong status; 400 on empty content; 200 OK + `{status:"pending_review"}`
- [ ] `/approve`: auth=CapabilityTemplatePublish; 401 when no session; 403 insufficient cap; 404 when templateKey missing; 409 on wrong status; 200 OK + `{status:"published"}`
- [ ] Auth semantics DIFFER from the document-view endpoints (which mask 403 as 404 to prevent enumeration). Template keys are admin-facing non-secret identifiers; enumeration protection not required.
- [ ] Both return JSON `{status: "..."}` with new status

**Verify:**
```bash
cd internal/modules/documents/delivery/http && rtk go test -run "TemplatePublish|TemplateApprove" ./...
```
Expected: ≥ 8 tests PASS.

**Subagent prompt (Codex):**
```
Create handler_ck5_template_publish.go with two http.HandlerFunc methods on *Handler.

AUTH CONTRACT (strict, no masking):
- No session → 401 Unauthorized.
- Session lacks required capability → 403 Forbidden.
- templateKey not found in repo → 404 Not Found.
DO NOT mask 403 as 404. This differs from document-view endpoints.

handleTemplatePublish:
  - method=POST
  - key := r.PathValue("key")
  - If request has no auth session → 401.
  - If session.Capabilities lacks CapabilityTemplateEdit for key → 403.
  - err := s.PublishTemplateForReview(ctx, key)
  - map errors:
    ErrTemplateNotFound → 404
    ErrInvalidTemplateStatus → 409
    ErrEmptyTemplateContent → 400
    nil → 200 + body {"status": "pending_review"}
    default → 500

handleTemplateApprove: analogous with CapabilityTemplatePublish + ApproveTemplate, same strict 401/403/404/409/200 contract.

Wire in template_admin_handler.go's handleTemplatesSubRoutes:
  case "publish" + POST → handleTemplatePublish
  case "approve" + POST → handleTemplateApprove

Tests (each handler):
- 200 happy path.
- 401 when request has no session.
- 403 when session lacks the specific capability (CapabilityTemplateEdit for publish, CapabilityTemplatePublish for approve).
- 404 when templateKey missing from repo.
- 409 when status transition invalid.
Publish-only: 400 when contentHtml empty.
```

**Steps:**
- [ ] Dispatch Codex → PASS → commit `feat(ck5-api): template publish/approve handlers`

### Task 28: Update GET /ck5-draft to return published snapshot

**Goal:** Fill mode sees frozen `published_html`; author/draft mode sees live draft.

**Files:**
- Modify: `internal/modules/documents/delivery/http/handler_ck5_template.go` (created in Plan B)
- Modify: `internal/modules/documents/delivery/http/handler_ck5_template_test.go`
- Possibly: `internal/modules/documents/application/service_ck5_template.go`

**Acceptance Criteria:**
- [ ] Query param `?mode=fill` AND status=published → return `published_html`
- [ ] Otherwise → return live draft `blocks_json._ck5.contentHtml` (Plan B behavior)
- [ ] Tests cover both branches

**Verify:**
```bash
cd internal/modules/documents/delivery/http && rtk go test -run "CK5TemplateDraft" ./...
```
Expected: PASS.

**Subagent prompt (Codex):**
```
Extend GET /api/v1/templates/{key}/ck5-draft from Plan B:

New behavior:
- Read query param mode: "author" (default) or "fill".
- If mode=fill AND template.Status == "published" AND template.PublishedHTML != nil → return { html: *template.PublishedHTML, manifest: ... }.
- Otherwise → existing Plan B behavior (live blocks_json._ck5.contentHtml).

Tests:
- GET ?mode=fill on published → returns published_html (which differs from live contentHtml).
- GET ?mode=fill on draft → returns live draft (backward-compatible).
- GET ?mode=author (default) → always live draft.
```

**Steps:**
- [ ] Dispatch Codex → PASS → commit `feat(ck5-api): fill mode uses published template snapshot`

### Phase 8 Review Checkpoint

Controller runs `code-reviewer`:
- Migration idempotent (IF NOT EXISTS)
- State machine enforced in service AND repo (defense in depth)
- No direct SQL status update bypassing state check
- Cap boundaries respected (edit vs publish)
- `rtk go test ./...` green

**Gate:** APPROVE → Phase 9.

---

## Phase 9 — Frontend Publish UI

**Goal:** `PublishButton.tsx` component; three visual states; wired into AuthorPage.

### Task 29: `templatePublishApi.ts`

**Goal:** Thin client for `/publish` + `/approve`.

**Files:**
- Create: `frontend/apps/web/src/features/documents/ck5/persistence/templatePublishApi.ts`
- Create: `frontend/apps/web/src/features/documents/ck5/persistence/__tests__/templatePublishApi.test.ts`

**Acceptance Criteria:**
- [ ] `publishTemplate(key)` → POST `/api/v1/templates/{key}/publish`, returns `{status}`
- [ ] `approveTemplate(key)` → POST `/api/v1/templates/{key}/approve`, returns `{status}`
- [ ] `getTemplateStatus(key)` → GET `/api/v1/templates/{key}` parse status field
- [ ] Throws typed `PublishError` with status on non-ok

**Verify:**
```bash
cd frontend/apps/web && rtk pnpm vitest run src/features/documents/ck5/persistence/__tests__/templatePublishApi.test.ts
```
Expected: ≥ 4 tests PASS.

**Subagent prompt (Codex):**
```
Create templatePublishApi.ts:

export class PublishError extends Error { constructor(public status: number, msg?: string) {} }

export async function publishTemplate(key: string): Promise<{status: string}> {
  const res = await fetch(`/api/v1/templates/${encodeURIComponent(key)}/publish`, { method: "POST", credentials: "include" });
  if (!res.ok) throw new PublishError(res.status, await res.text());
  return res.json();
}

export async function approveTemplate(key: string): Promise<{status: string}> { ... /* analogous */ }

export async function getTemplateStatus(key: string): Promise<string> {
  const res = await fetch(`/api/v1/templates/${encodeURIComponent(key)}`, { credentials: "include" });
  if (!res.ok) throw new PublishError(res.status);
  const body = await res.json();
  return body.status;
}

Tests with fetch-mock: happy paths + non-ok throws + encodes key.
```

**Steps:**
- [ ] Dispatch Codex → PASS → commit `feat(ck5-ui): templatePublishApi client`

### Task 30: `PublishButton.tsx` component

**Goal:** Three visual states per spec: `draft → "Publish for Review"`, `pending_review → "Awaiting Approval" + admin "Approve"`, `published → "Published" badge`.

**Files:**
- Create: `frontend/apps/web/src/features/documents/ck5/react/components/PublishButton.tsx`
- Create: `frontend/apps/web/src/features/documents/ck5/react/components/PublishButton.module.css`
- Create: `frontend/apps/web/src/features/documents/ck5/react/components/__tests__/PublishButton.test.tsx`

**Acceptance Criteria:**
- [ ] Props `{ templateKey: string; status: "draft"|"pending_review"|"published"; canApprove: boolean; onStatusChange?: (next: string) => void }`
- [ ] draft → primary button "Publish for Review"; on click calls `publishTemplate`, on success invokes `onStatusChange`
- [ ] pending_review + `canApprove=true` → shows "Awaiting Approval" badge + "Approve" button; on click `approveTemplate` → `onStatusChange`
- [ ] pending_review + `canApprove=false` → badge only
- [ ] published → "Published" badge, no buttons
- [ ] Loading + error states during async

**Verify:**
```bash
cd frontend/apps/web && rtk pnpm vitest run src/features/documents/ck5/react/components/__tests__/PublishButton.test.tsx
```
Expected: ≥ 6 tests PASS.

**Subagent prompt (Codex):**
```
Implement PublishButton.tsx per props/states in description. Use @testing-library/react patterns consistent with existing project tests.

State transitions confirmed via onStatusChange callback.

Test cases:
- draft renders "Publish for Review" button; click calls publishTemplate(templateKey); onStatusChange("pending_review").
- pending_review + canApprove=false renders badge only (no Approve button).
- pending_review + canApprove=true shows Approve button; click calls approveTemplate; onStatusChange("published").
- published renders "Published" badge, no interactive buttons.
- Error from publishTemplate renders error text role="alert".
- Loading state: button disabled + shows "Publishing…" during async.
```

**Steps:**
- [ ] Dispatch Codex → PASS → commit `feat(ck5-ui): PublishButton component`

### Task 31: Wire PublishButton into AuthorPage

**Goal:** In template-edit mode, display PublishButton. Derive `canApprove` from session capabilities.

**Files:**
- Modify: `frontend/apps/web/src/features/documents/ck5/react/AuthorPage.tsx`

**Acceptance Criteria:**
- [ ] Page fetches template status on mount
- [ ] Renders `<PublishButton>` adjacent to ExportMenu
- [ ] `canApprove` sourced from current user capabilities (find existing session hook pattern)
- [ ] `onStatusChange` re-fetches or updates local state

**Verify:**
```bash
cd frontend/apps/web && rtk pnpm vitest run src/features/documents/ck5/react/
```
Expected: existing tests green + AuthorPage test updated with PublishButton expectation.

**Subagent prompt (Codex):**
```
Read AuthorPage.tsx. Integrate:
1. useEffect on mount: getTemplateStatus(templateKey) → setStatus.
2. Render <PublishButton templateKey={templateKey} status={status} canApprove={session.capabilities.includes("template.publish")} onStatusChange={setStatus} />.
3. Find existing session/capabilities hook; if absent, implement a minimal useSession() reading from /api/v1/session (or whatever pattern exists — search first, do not invent).
4. Update AuthorPage tests to include the PublishButton presence.
```

**Steps:**
- [ ] Dispatch Codex → PASS → commit `feat(ck5-ui): wire PublishButton into AuthorPage`

### Phase 9 Review Checkpoint

Controller runs `code-reviewer`:
- PublishButton not rendered in non-template mode
- `canApprove` not hard-coded true
- Error states surface to user, not swallowed
- Accessibility: buttons have aria-labels; badges have role="status"

**Gate:** APPROVE → Phase 10.

---

## Phase 10 — Delete BlockNote/MDDM Stack

**Goal:** Destructive cleanup. `mddm-editor/` directory removed. `@blocknote/*` removed from package.json. All references in Go/frontend excised.

### Task 32: Delete `mddm-editor/` directory

**Goal:** Full recursive delete.

**Files:** (deletes)
- Delete: `frontend/apps/web/src/features/documents/mddm-editor/` (entire)

**Acceptance Criteria:**
- [ ] Directory absent
- [ ] `rtk grep "from.*mddm-editor" frontend/apps/web/src/` → 0 results
- [ ] `rtk grep "mddm-editor" frontend/apps/web/src/` → 0 results

**Verify:**
```bash
rtk grep "mddm-editor" frontend/apps/web/src/ | wc -l
```
Expected: 0.

**Subagent prompt (Codex):**
```
Delete frontend/apps/web/src/features/documents/mddm-editor/ recursively.

Before deleting, grep the repo for any remaining importers. If any file outside mddm-editor/ imports from mddm-editor, report them — do NOT delete blindly. Controller will update those files in a follow-up (Task 33).

Expected importers per spec scan:
- DocumentsWorkspaceView.tsx (maybe)
- DocumentsHubView.tsx
- adapters/*
- runtime/*
Controller expects these to either be removed or rewritten to use ck5 stack. In Plan C scope, any residual mddm-editor importer MUST be deleted or pointed at ck5 before proceeding.

Action: first dry-run (list importers), THEN delete + report.
```

**Steps:**
- [ ] Dispatch Codex (dry-run) → review importers list
- [ ] Second Codex dispatch to fix importers (rewrite to ck5 or delete)
- [ ] Third Codex dispatch: delete mddm-editor directory
- [ ] Verify grep → 0
- [ ] Commit `feat(ck5)!: delete mddm-editor directory`

### Task 33: Delete `@blocknote/*` from package.json + lockfile

**Goal:** Remove four `@blocknote/*` deps.

**Files:**
- Modify: `frontend/apps/web/package.json`
- Modify: `pnpm-lock.yaml` (regenerated)

**Acceptance Criteria:**
- [ ] No `@blocknote/*` keys in `package.json`
- [ ] `rtk pnpm install` clean
- [ ] `rtk pnpm build` bundle contains zero `@blocknote/*` strings

**Verify:**
```bash
rtk grep "@blocknote" frontend/apps/web/package.json
rtk pnpm install && rtk pnpm build
rtk grep "@blocknote" frontend/apps/web/dist/ | wc -l
```
Expected: 0 everywhere.

**Subagent prompt (Codex):**
```
Remove @blocknote/core, @blocknote/mantine, @blocknote/react, @blocknote/server-util from frontend/apps/web/package.json dependencies.
Run pnpm install to update lockfile.
Also search repo for any remaining import from @blocknote/* — must be zero after mddm-editor deletion. If any found, surface them.
```

**Steps:**
- [ ] Dispatch Codex → verify bundle clean → commit `chore(ck5)!: remove @blocknote/* deps`

### Task 34: Delete Go docgen handlers + DocgenClient

**Goal:** Remove `handleDocumentExportDocx`, `handleDocumentContentDocx`, `handleDocumentTemplateDocx`, `handleDocumentContentRenderPDF`, `handleDocumentContentBrowserPost`, `DocgenClient`.

**Files:**
- Modify: `internal/modules/documents/delivery/http/handler.go`
- Modify: `internal/modules/documents/delivery/http/handler_content.go`
- Modify: `internal/modules/documents/delivery/http/handler_render_pdf.go`
- Delete: `internal/modules/documents/delivery/http/export_handler.go`
- Delete: `internal/modules/documents/delivery/http/export_handler_test.go`
- Delete: `internal/modules/documents/delivery/http/handler_render_pdf.go` if wholly obsolete
- Delete: `internal/modules/documents/delivery/http/handler_render_pdf_test.go`
- Modify: `internal/modules/documents/application/service_content_docx.go` (delete or gut)
- Modify: `internal/modules/documents/application/export_service.go` (delete or gut)

**Acceptance Criteria:**
- [ ] `rtk grep "DocgenClient|docgen" internal/` → 0 results (or only comments)
- [ ] `rtk grep "handleDocumentExportDocx\|handleDocumentContentDocx\|handleDocumentTemplateDocx\|handleDocumentContentRenderPDF\|handleDocumentContentBrowserPost" internal/` → 0
- [ ] `rtk go build ./...` clean
- [ ] `rtk go test ./...` green

**Verify:**
```bash
rtk go build ./... && rtk go test ./...
```
Expected: clean.

**Subagent prompt (Codex):**
```
Destructive cleanup of docgen-related Go code.

Delete these symbols (via Edit/Write on the files they live in):
- handleDocumentExportDocx (handler.go)
- handleDocumentContentDocx (handler_content.go)
- handleDocumentTemplateDocx (handler_content.go or handler.go)
- handleDocumentContentRenderPDF (handler_render_pdf.go)
- handleDocumentContentBrowserPost (handler_content.go) — KEEP handleDocumentContentBrowserGet
- DocgenClient struct + constructor (search: "DocgenClient")
- Any docgen URL config reader (viper key, env var)
- export_handler.go + export_handler_test.go → delete if they are entirely docgen-scoped (first scan — keep if contains still-used logic)
- handler_render_pdf.go + _test.go → same

Route wiring cleanup:
- Remove all references to these handlers in handleDocumentsSubRoutes and related routers.

Service layer:
- application/service_content_docx.go → delete (moved concerns to ck5-export).
- application/export_service.go → delete if solely for docgen path; else gut docgen-specific code.

After changes, run go build ./... and go test ./.... If any test references deleted handlers, delete those tests.

Do NOT delete:
- handleDocumentContentNativeGet/Post
- handleDocumentContentBrowserGet
- handleDocumentExportCK5Docx (Plan C)
- handleDocumentExportCK5Pdf (Plan C)
- service_ck5_export_client.go
```

**Steps:**
- [ ] Dispatch Codex (round 1: inventory to delete)
- [ ] Dispatch round 2: perform deletes
- [ ] Verify builds + tests green
- [ ] Commit `feat(ck5)!: delete docgen Go handlers + DocgenClient`

### Task 35: Delete `apps/docgen/` directory

**Goal:** Full removal.

**Files:**
- Delete: `apps/docgen/` (entire)
- Modify: `.claude/launch.json` (remove `metaldocs-docgen` entry)

**Acceptance Criteria:**
- [ ] `apps/docgen/` absent
- [ ] launch.json has no `metaldocs-docgen` entry
- [ ] `rtk pnpm install` clean at repo root
- [ ] `rtk grep "docgen" --exclude-dir=apps/ck5-export --exclude-dir=docs` returns zero OR only archival doc strings

**Verify:**
```bash
ls apps/
```
Expected: `api`, `ck5-export`, `ck5-studio`, `worker` (no `docgen`).

**Subagent prompt (Codex):**
```
Delete apps/docgen/ recursively.
Remove from .claude/launch.json: any entry with runtimeArgs referencing docgen.
Verify pnpm-workspace.yaml glob still matches remaining packages; fix if needed.
Run rtk pnpm install to regenerate lockfile.
```

**Steps:**
- [ ] Dispatch Codex → verify `pnpm install` → commit `chore(ck5)!: delete apps/docgen`

### Phase 10 Review Checkpoint

Controller runs `code-reviewer`:
- Zero mddm-editor / @blocknote / docgen references left (grep evidence)
- Go build + test green
- Frontend build + test green
- Bundle analysis: `@blocknote` absent
- No dangling imports

Manual controller checks (full pre-delete verification suite per spec):
```bash
rtk grep "mddm-editor" frontend/apps/web/src/ | wc -l       # 0
rtk grep "@blocknote" frontend/apps/web/src/ | wc -l        # 0
rtk grep "docgen" internal/ | wc -l                          # 0 (or archival only)
rtk go build ./...                                           # clean
rtk go test ./...                                            # green
cd frontend/apps/web && rtk vitest run                       # workspace-wide frontend tests green
cd frontend/apps/web && rtk pnpm build                       # clean
rtk pnpm -r test                                             # all workspace packages green (ck5-export + web + worker)
```

**Gate:** APPROVE (all checks above green) → Phase 11.

---

## Phase 11 — PR2 Preview Smoke Validation

**Goal:** End-to-end smoke covering publish workflow + deletion proof. Merge PR2 on green.

### Task 36: Template publish/approve preview smoke

**Files:** None (validation).

**Acceptance Criteria:**
- [ ] `preview_start ck5-plan-c-api|export|web` OK
- [ ] Navigate Author → click Publish → `POST /templates/sandbox/publish 200` → PublishButton shows "Awaiting Approval"
- [ ] Login as admin → click Approve → `POST /templates/sandbox/approve 200` → button shows "Published"
- [ ] Navigate Fill mode → `GET /ck5-draft?mode=fill` returns `published_html` (verify via network inspection)

**Steps:**
- [ ] Follow spec lines 372-378
- [ ] Record evidence in PR description

### Task 37: Deletion smoke

**Files:** None.

**Acceptance Criteria:**
- [ ] `preview_snapshot` of authored page → zero mddm-editor DOM markers
- [ ] `rtk pnpm build` → grep dist for `@blocknote` → 0
- [ ] `preview_console_logs level=error` → 0 errors referencing mddm-editor or docgen

**Steps:**
- [ ] Run each check → capture output → attach to PR

### Task 38: PR2 merge

**Goal:** Open PR2 scoped as destructive + publish flow. Merge after review.

**Files:** None (git only).

**Acceptance Criteria:**
- [ ] PR title: `feat(ck5)!: Plan C PR2 — template publish + delete mddm/docgen/@blocknote`
- [ ] Body lists tasks 24-37 + smoke evidence + migration note
- [ ] `!` denotes breaking change (backend + frontend both affected)

**Steps:**
- [ ] Push → `gh pr create` → request review → merge

### Phase 11 Review Checkpoint

Controller runs `code-reviewer` + `verification-before-completion`:
- Publish state machine end-to-end proven
- Deletion verified via grep + bundle analysis
- No regressions to CK5 author/fill export flow (Phase 7 re-validated)
- Migration 0077 applied in staging confirmed

Final pre-merge gate — controller runs all:
```bash
rtk go build ./...                       # clean
rtk go test ./...                        # green
cd frontend/apps/web && rtk vitest run   # workspace-wide frontend tests green
cd frontend/apps/web && rtk pnpm build   # clean
rtk pnpm -r test                         # all workspace packages green
```
Expected: every command exits 0.

**Gate:** APPROVE + all gate commands green + PR2 merged → Plan C complete.

---

## Self-Review Checklist (run before Codex hardening)

**1. Spec coverage:**
- [x] `apps/ck5-export/` service → Phase 1-4
- [x] Engine code moves → Phase 2
- [x] html-to-export-tree → Phase 3
- [x] Go DOCX + PDF handlers → Phase 5
- [x] Frontend ExportMenu + APIs → Phase 6
- [x] Preview smoke (Author + Fill) → Phase 7
- [x] Template publish state machine + handlers → Phase 8
- [x] PublishButton UI → Phase 9
- [x] BlockNote/MDDM deletion → Phase 10
- [x] Preview smoke for deletion → Phase 11
- [x] launch.json update → Tasks 3, 35

**2. Placeholder scan:** No TBD/TODO; all paths exact; subagent prompts contain concrete code or explicit "read first" instructions where file content required.

**3. Type consistency:** `CK5ExportClient` used in Tasks 14-17; `ExportNode` used in Tasks 8-10; `TemplateStatus` used in Tasks 25-28; `publishTemplate`/`approveTemplate` used in Tasks 29-31.

---

## Phase Review Workflow (controller uses)

For each Phase:
1. Dispatch Codex subagent per task (via `mcp__codex__codex`, gpt-5.3-codex, sandbox=workspace-write)
2. Run `Verify` command → must pass
3. Commit with conventional scope
4. At Phase end: invoke `superpowers-extended-cc:code-reviewer` agent with phase summary
5. On APPROVE → advance to next Phase
6. On REJECT/fixes → dispatch Codex to repair, re-review (max 1 retry per phase; escalate to user if second failure)

---

## File Inventory Snapshot (for PR descriptions)

### Created (Plan C total)

```
apps/ck5-export/package.json
apps/ck5-export/tsconfig.json
apps/ck5-export/tsconfig.build.json
apps/ck5-export/vitest.config.ts
apps/ck5-export/.gitignore
apps/ck5-export/src/server.ts
apps/ck5-export/src/export-node.ts
apps/ck5-export/src/html-to-export-tree.ts
apps/ck5-export/src/routes/render-docx.ts
apps/ck5-export/src/routes/render-pdf-html.ts
apps/ck5-export/src/__fixtures__/*.html (6 files)
apps/ck5-export/src/__tests__/*.test.ts (≥5 files)
apps/ck5-export/src/docx-emitter/**    (moved)
apps/ck5-export/src/asset-resolver/**  (moved)
apps/ck5-export/src/print-stylesheet/** (moved)
apps/ck5-export/src/inline-asset-rewriter.ts (moved)

internal/modules/documents/application/service_ck5_export_client.go
internal/modules/documents/application/service_ck5_template_publish.go
internal/modules/documents/delivery/http/handler_ck5_export.go
internal/modules/documents/delivery/http/handler_ck5_template_publish.go
internal/modules/documents/infrastructure/ck5_print_css.go
+ _test.go counterparts for each

migrations/0077_add_template_publish_state.sql

frontend/apps/web/src/features/documents/ck5/persistence/exportApi.ts
frontend/apps/web/src/features/documents/ck5/persistence/templatePublishApi.ts
frontend/apps/web/src/features/documents/ck5/react/components/ExportMenu.tsx
frontend/apps/web/src/features/documents/ck5/react/components/PublishButton.tsx
frontend/apps/web/src/features/documents/ck5/print/wrap-print-document.ts
+ test + CSS module counterparts
```

### Modified
```
.claude/launch.json
internal/modules/documents/delivery/http/handler.go
internal/modules/documents/delivery/http/template_admin_handler.go
internal/modules/documents/delivery/http/handler_ck5_template.go     (mode=fill branch)
internal/modules/documents/application/service_ck5.go                (GetCK5DocumentContent)
internal/modules/documents/application/service_ck5_template.go       (repo publish glue)
internal/modules/documents/infrastructure/memory/template_drafts_repo.go (+ postgres)
internal/modules/documents/domain/model.go                           (TemplateStatus enum)
frontend/apps/web/src/features/documents/ck5/react/AuthorPage.tsx
frontend/apps/web/src/features/documents/ck5/react/FillPage.tsx
frontend/apps/web/package.json                                       (- @blocknote/*)
```

### Deleted (PR2)
```
frontend/apps/web/src/features/documents/mddm-editor/    (entire dir)
apps/docgen/                                              (entire dir)
internal/modules/documents/delivery/http/export_handler.go + _test.go
internal/modules/documents/delivery/http/handler_render_pdf.go + _test.go
internal/modules/documents/application/service_content_docx.go + _test.go (if wholly docgen-scoped)
internal/modules/documents/application/export_service.go + _test.go (same condition)
```
