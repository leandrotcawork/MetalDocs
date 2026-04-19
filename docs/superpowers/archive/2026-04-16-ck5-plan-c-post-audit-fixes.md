# CK5 Plan C — Post-Audit Fixes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers-extended-cc:subagent-driven-development (recommended) or superpowers-extended-cc:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Repair four correctness bugs and four architectural defects surfaced by the outsider audit of the shipped Plan C implementation — without regressing any existing green test.

**Architecture:**
- Runtime `/render/docx` pipeline stays: `html → html-to-export-tree → ExportNode[] → docx-emitter → docx`. Bugs inside it are patched.
- Dead parallel pipeline (`codecs/` + `layout-interpreter/` + `docx-emitter/` dir from mddm-editor) is deleted outright — zero route imports it today.
- `layout-ir/` directory is collapsed to a single flat `layout-tokens.ts` (the plan-c spec explicitly forbade persistent IR).
- The 565-LOC god file `ck5-docx-emitter.ts` is split block-by-block into a fresh `docx-emitter/` module (name reused after the dead one is removed) to match the modularity the legacy path had before it was deleted.
- Walker loses the `inlineContext` parameter — inline runs are collected at emit time, not threaded top-down.

**Tech Stack:** TypeScript, vitest, Hono, linkedom, docx-npm (ck5-export service). Go + testify (API). React + vitest + CKEditor5 v48 (frontend).

**Execution Notes (for the dispatcher / subagent-driven-development coordinator):**
- **Implementation subagents:** Use Codex `gpt-5.3-codex` with `reasoning_effort: high` for all implementation tasks. Do NOT use Claude models for code changes — Codex is the implementer.
- **Review subagents:** Use Codex `gpt-5.3-codex` or the most capable Claude available for spec + quality review.
- **Prompting style:** Prefix all subagent prompts with `/caveman` (caveman mode, full intensity) to keep instructions terse and reduce token waste. Caveman drops filler/articles/hedging; technical substance stays.

---

## Scope Check

All work touches a single subsystem: `apps/ck5-export` + one frontend toolbar file + one Go comment + one plan doc. Single plan is correct — no split needed.

## File Structure (target state after plan)

```
apps/ck5-export/src/
  server.ts
  export-node.ts                       ← gain: Field.label field
  html-to-export-tree.ts               ← simplified: no inlineContext
  layout-tokens.ts                     ← NEW (replaces layout-ir/ dir)
  inline-asset-rewriter.ts             ← unchanged
  asset-resolver/                      ← unchanged
  print-stylesheet/
    print-css.ts                       ← import updated to layout-tokens
    wrap-print-document.ts             ← unchanged
  shared/
    helpers/
      units.ts                         ← unchanged (still used)
      color.ts                         ← unchanged (still used)
    (adapter.ts deleted if orphan — Task 6)
  docx-emitter/                        ← REBUILT from split
    index.ts                           ← barrel: emitDocxFromExportTree, collectImageUrls
    emitter.ts                         ← top-level emitDocxFromExportTree
    asset-collector.ts                 ← collectImageUrls walker
    inline.ts                          ← TextRun/inline collectors
    paragraph.ts
    heading.ts
    list.ts
    blockquote.ts
    image.ts
    field.ts
    table.ts
    section.ts
    repeatable.ts
  routes/
    render-docx.ts                     ← imports new docx-emitter/index
    render-pdf-html.ts                 ← unchanged
  __tests__/                           ← unchanged (route-level integration tests)
  __fixtures__/                        ← unchanged

DELETED:
  apps/ck5-export/src/ck5-docx-emitter.ts
  apps/ck5-export/src/codecs/            (entire dir, incl. tests)
  apps/ck5-export/src/layout-interpreter/ (entire dir, incl. tests)
  apps/ck5-export/src/layout-ir/         (entire dir — replaced by layout-tokens.ts)
  apps/ck5-export/src/docx-emitter/      (entire dead legacy dir — name freed, rebuilt in Task 9)

frontend/apps/web/src/features/documents/ck5/config/toolbars.ts ← Task 1

internal/modules/documents/application/service_template_lifecycle.go:677 ← Task 11
docs/superpowers/specs/2026-04-16-ck5-plan-c-production-readiness.md ← Task 12 (append)
```

## Working Directory

**All commands below assume CWD = `C:\Users\leandro.theodoro.MN-NTB-LEANDROT\Documents\MetalDocs-ck5-plan-c`** unless stated otherwise. That is the worktree for branch `migrate/ck5-plan-c` where Plan C was implemented.

---

## Phase 1 — Correctness Bug Fixes (TDD)

### Task 1: Add `restrictedEditingExceptionBlock` to `AUTHOR_TOOLBAR`

**Goal:** Make the already-failing `toolbars.test.ts` pass by registering the block-level exception command button.

**Files:**
- Modify: `frontend/apps/web/src/features/documents/ck5/config/toolbars.ts`
- Test: `frontend/apps/web/src/features/documents/ck5/config/__tests__/toolbars.test.ts` (existing, already asserts target)

**Acceptance Criteria:**
- [ ] `AUTHOR_TOOLBAR` contains both `'restrictedEditingException'` (inline) and `'restrictedEditingExceptionBlock'` (block).
- [ ] `FILL_TOOLBAR` does NOT contain `restrictedEditingExceptionBlock` (already enforced by existing test at line 21).
- [ ] `toolbars.test.ts` all four `it()` blocks pass.

**Verify:** `pnpm -C frontend/apps/web vitest run src/features/documents/ck5/config/__tests__/toolbars.test.ts` → exit 0, 4 tests pass.

**Steps:**

- [ ] **Step 1: Confirm the existing test already fails in red**

```bash
pnpm -C frontend/apps/web vitest run src/features/documents/ck5/config/__tests__/toolbars.test.ts
```

Expected: FAIL with `expected [ ... ] to include 'restrictedEditingExceptionBlock'` on line 15.

- [ ] **Step 2: Add block exception button to AUTHOR_TOOLBAR**

Edit `frontend/apps/web/src/features/documents/ck5/config/toolbars.ts`:

```ts
export const AUTHOR_TOOLBAR: readonly string[] = [
  'insertMddmSection',
  'insertMddmRepeatable',
  'insertMddmField',
  'insertMddmRichBlock',
  'insertTable',
  'imageUpload',
  '|',
  'heading',
  'bulletedList',
  'numberedList',
  'alignment',
  '|',
  'bold',
  'italic',
  'underline',
  'fontSize',
  'fontColor',
  'link',
  'removeFormat',
  '|',
  'restrictedEditingException',
  'restrictedEditingExceptionBlock',
  '|',
  'undo',
  'redo',
];
```

- [ ] **Step 3: Re-run target test — confirm green**

```bash
pnpm -C frontend/apps/web vitest run src/features/documents/ck5/config/__tests__/toolbars.test.ts
```

Expected: PASS 4/4.

- [ ] **Step 4: Run full CK5 frontend suite — confirm no regression**

```bash
pnpm -C frontend/apps/web vitest run src/features/documents/ck5
```

Expected: PASS all (was 87 pass + 1 fail → should now be 88 pass).

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/ck5/config/toolbars.ts
git commit -m "fix(ck5): add restrictedEditingExceptionBlock to AUTHOR_TOOLBAR

Memory of v48 restricted-editing plugin: exposes both inline (span) and
block (div wrapper) commands. Toolbar only had the inline variant, causing
the pre-existing toolbars.test.ts assertion to fail."
```

---

### Task 2: Emit field label (not raw id) in DOCX label cell

**Goal:** CK5 stores a human-readable label as `data-field-label` on each `span.mddm-field`. The export currently drops it on the floor and prints the machine `data-field-id` in the left column of the generated DOCX table. Fix so the label column shows the label.

**Files:**
- Modify: `apps/ck5-export/package.json` (declare `jszip` + `@types/jszip` as devDeps)
- Create: `apps/ck5-export/src/__tests__/helpers/read-docx-xml.ts` (shared DOCX-XML read helper, reused by Task 3, Task 8)
- Modify: `apps/ck5-export/src/export-node.ts`
- Modify: `apps/ck5-export/src/html-to-export-tree.ts`
- Modify: `apps/ck5-export/src/ck5-docx-emitter.ts` (this file dies in Task 9; edit is still needed now so Phase 2/3 can proceed from a green tree)
- Create: `apps/ck5-export/src/__tests__/field-label-round-trip.test.ts`

**Acceptance Criteria:**
- [ ] `jszip` + `@types/jszip` declared in `devDependencies`.
- [ ] `apps/ck5-export/src/__tests__/helpers/read-docx-xml.ts` exports `readDocxDocumentXml(doc: Document): Promise<string>`.
- [ ] `Field` interface in `export-node.ts` gains optional `label?: string`.
- [ ] Walker reads `data-field-label` attribute into `Field.label`.
- [ ] Emitter uses `label ?? id` for the DOCX label cell text.
- [ ] New test asserts round-trip: HTML `<span class="mddm-field" data-field-id="customer" data-field-label="Customer">Acme</span>` → `word/document.xml` contains `<w:t ...>Customer</w:t>` and does NOT contain `<w:t ...>customer</w:t>`.

**Verify:** `pnpm -C apps/ck5-export vitest run src/__tests__/field-label-round-trip.test.ts` → PASS.

**Steps:**

- [ ] **Step 0: Declare `jszip` devDep and create DOCX-XML helper**

`jszip` is already a transitive runtime dependency of the `docx` package (`apps/ck5-export/package-lock.json:1092`). We declare it explicitly in `devDependencies` so test helpers have first-class types without relying on transitive resolution.

Edit `apps/ck5-export/package.json` — replace the `devDependencies` block:

```json
"devDependencies": {
  "@types/jszip": "^3.4",
  "@types/node": "^20",
  "jszip": "^3.10",
  "tsx": "^4",
  "typescript": "^5.4",
  "vitest": "^2"
}
```

Install:
```bash
pnpm -C apps/ck5-export install
```

Create `apps/ck5-export/src/__tests__/helpers/read-docx-xml.ts`:

```ts
import JSZip from "jszip"
import { Packer, type Document } from "docx"

// Renders `doc` to a .docx ZIP buffer and returns the decoded
// `word/document.xml` body. Tests assert on this XML (not the raw ZIP
// bytes) because ZIP compression makes string-search over `buf.toString()`
// non-deterministic: the bytes we care about may be deflated.
export async function readDocxDocumentXml(doc: Document): Promise<string> {
  const buf = await Packer.toBuffer(doc)
  const zip = await JSZip.loadAsync(buf)
  const entry = zip.file("word/document.xml")
  if (!entry) {
    throw new Error("word/document.xml missing from generated .docx")
  }
  return entry.async("string")
}
```

Typecheck (should already pass — the helper is unused until Step 1):
```bash
pnpm -C apps/ck5-export typecheck
```

Expected: exit 0.

- [ ] **Step 1: Write failing round-trip test**

Create `apps/ck5-export/src/__tests__/field-label-round-trip.test.ts`:

```ts
import { describe, it, expect } from "vitest"
import { htmlToExportTree } from "../html-to-export-tree"
import { emitDocxFromExportTree } from "../ck5-docx-emitter"
import { defaultLayoutTokens } from "../layout-ir"
import { readDocxDocumentXml } from "./helpers/read-docx-xml"

describe("field label round-trip", () => {
  const html =
    '<span class="mddm-field" data-field-id="customer" data-field-type="text" data-field-label="Customer">Acme</span>'

  it("walker captures label from data-field-label", () => {
    const tree = htmlToExportTree(html)
    expect(tree).toHaveLength(1)
    const node = tree[0]
    expect(node.kind).toBe("field")
    if (node.kind !== "field") return
    expect(node.id).toBe("customer")
    expect(node.label).toBe("Customer")
    expect(node.value).toBe("Acme")
  })

  it("emitter prints label in DOCX label cell", async () => {
    const tree = htmlToExportTree(html)
    const doc = emitDocxFromExportTree(tree, defaultLayoutTokens, new Map())
    const xml = await readDocxDocumentXml(doc)
    // Label text must render in a w:t element.
    expect(xml).toMatch(/<w:t[^>]*>Customer<\/w:t>/)
    // Regression guard: machine id must NOT appear as rendered text.
    // (It may appear elsewhere as an XML attribute — we only forbid
    //  the w:t body form.)
    expect(xml).not.toMatch(/<w:t[^>]*>customer<\/w:t>/)
  })
})
```

- [ ] **Step 2: Run test — confirm red**

```bash
pnpm -C apps/ck5-export vitest run src/__tests__/field-label-round-trip.test.ts
```

Expected: FAIL, first on `expect(node.label).toBe("Customer")` because the walker does not populate `label`.

- [ ] **Step 3: Extend `Field` interface**

Edit `apps/ck5-export/src/export-node.ts`, replace the existing `Field` interface:

```ts
export interface Field {
  kind: "field"
  id: string
  label?: string
  fieldType: "text" | "number" | "date" | "boolean" | "select"
  value: string
}
```

- [ ] **Step 4: Populate `label` in walker**

Edit `apps/ck5-export/src/html-to-export-tree.ts`. Replace the `mddm-field` branch (current lines 111-120) with:

```ts
  if (tagName === "SPAN" && el.classList.contains("mddm-field")) {
    const label = el.getAttribute("data-field-label") ?? undefined
    return [
      {
        kind: "field",
        id: el.getAttribute("data-field-id") ?? "",
        label: label && label.length > 0 ? label : undefined,
        fieldType: (el.getAttribute("data-field-type") ?? "text") as Field["fieldType"],
        value: el.textContent ?? "",
      },
    ]
  }
```

- [ ] **Step 5: Use label in emitter**

Edit `apps/ck5-export/src/ck5-docx-emitter.ts`. Replace the `emitField` function (currently at lines 251-286). The new label cell text is `node.label ?? node.id`:

```ts
function emitField(node: ExportNode, tokens: LayoutTokens): Table[] {
  if (node.kind !== "field") {
    return []
  }

  const borderColor = hexToFill(tokens.theme.accentBorder)
  const borders = {
    top: { style: BorderStyle.SINGLE, size: 4, color: borderColor },
    bottom: { style: BorderStyle.SINGLE, size: 4, color: borderColor },
    left: { style: BorderStyle.SINGLE, size: 4, color: borderColor },
    right: { style: BorderStyle.SINGLE, size: 4, color: borderColor },
  }

  const labelText = node.label ?? node.id

  return [
    new Table({
      width: { size: 100, type: WidthType.PERCENTAGE },
      rows: [
        new TableRow({
          children: [
            new TableCell({
              width: { size: 35, type: WidthType.PERCENTAGE },
              shading: { fill: hexToFill(tokens.theme.accentLight), type: "clear", color: "auto" },
              borders,
              children: [new Paragraph({ children: [new TextRun({ text: labelText })] })],
            }),
            new TableCell({
              width: { size: 65, type: WidthType.PERCENTAGE },
              borders,
              children: [new Paragraph({ children: [new TextRun({ text: node.value })] })],
            }),
          ],
        }),
      ],
    }),
  ]
}
```

- [ ] **Step 6: Re-run test — confirm green**

```bash
pnpm -C apps/ck5-export vitest run src/__tests__/field-label-round-trip.test.ts
```

Expected: PASS 2/2.

- [ ] **Step 7: Run full ck5-export suite — confirm no regression**

```bash
pnpm -C apps/ck5-export test
```

Expected: PASS all (90+2 = 92 pass + 18 todo + 1 skipped).

- [ ] **Step 8: Commit**

```bash
git add apps/ck5-export/package.json apps/ck5-export/src/__tests__/helpers/read-docx-xml.ts apps/ck5-export/src/export-node.ts apps/ck5-export/src/html-to-export-tree.ts apps/ck5-export/src/ck5-docx-emitter.ts apps/ck5-export/src/__tests__/field-label-round-trip.test.ts
# If the monorepo lockfile changed, stage it as well (path may be root or apps/ck5-export):
git add pnpm-lock.yaml 2>/dev/null || git add apps/ck5-export/pnpm-lock.yaml 2>/dev/null || true

git commit -m "fix(ck5-export): render data-field-label in DOCX label cell

The walker was reading data-field-id only and dropping data-field-label.
The emitter printed the raw id in the left column, which is machine-
readable and not what end users see in the editor. Add optional
Field.label, read the attribute, and prefer label over id at emit time.

Also introduces a small readDocxDocumentXml test helper (unzips the .docx
and returns word/document.xml) so DOCX assertions don't have to scan raw
ZIP bytes."
```

---

### Task 3: Thread hyperlink color through LayoutTokens

**Goal:** Remove the hardcoded `"0563C1"` hyperlink color from the emitter (lines 113, 143) and source it from `LayoutTokens.theme.hyperlink` so a future theme override works in one place.

**Files:**
- Modify: `apps/ck5-export/src/layout-ir/index.ts` (token shape + default value)
- Modify: `apps/ck5-export/src/ck5-docx-emitter.ts` (read from tokens)
- Create: `apps/ck5-export/src/__tests__/hyperlink-token.test.ts`

**Acceptance Criteria:**
- [ ] `LayoutTokens.theme` gains `hyperlink: string` (hex).
- [ ] `defaultLayoutTokens.theme.hyperlink` = `"#0563C1"` (preserve current visual output).
- [ ] Emitter reads the color from tokens in both the `collectHyperlinkRuns` text branch and the `field` branch inside it.
- [ ] Unit test: passing a token override (`#FF0000`) produces `FF0000` somewhere in the DOCX XML and `0563C1` nowhere.

**Verify:** `pnpm -C apps/ck5-export vitest run src/__tests__/hyperlink-token.test.ts` → PASS.

**Steps:**

- [ ] **Step 1: Write failing test**

Create `apps/ck5-export/src/__tests__/hyperlink-token.test.ts` (reuses the `readDocxDocumentXml` helper created in Task 2 Step 0):

```ts
import { describe, it, expect } from "vitest"
import { htmlToExportTree } from "../html-to-export-tree"
import { emitDocxFromExportTree } from "../ck5-docx-emitter"
import { defaultLayoutTokens } from "../layout-ir"
import { readDocxDocumentXml } from "./helpers/read-docx-xml"

describe("hyperlink color sourced from LayoutTokens", () => {
  it("default tokens emit #0563C1 as a w:color attribute", async () => {
    const tree = htmlToExportTree('<p><a href="https://example.com">link</a></p>')
    const doc = emitDocxFromExportTree(tree, defaultLayoutTokens, new Map())
    const xml = await readDocxDocumentXml(doc)
    expect(xml).toMatch(/w:color[^>]*w:val="0563C1"/)
  })

  it("override token color appears in DOCX; old default does not", async () => {
    const tree = htmlToExportTree('<p><a href="https://example.com">link</a></p>')
    const override = {
      ...defaultLayoutTokens,
      theme: { ...defaultLayoutTokens.theme, hyperlink: "#FF0000" },
    }
    const doc = emitDocxFromExportTree(tree, override, new Map())
    const xml = await readDocxDocumentXml(doc)
    expect(xml).toMatch(/w:color[^>]*w:val="FF0000"/)
    expect(xml).not.toMatch(/w:color[^>]*w:val="0563C1"/)
  })
})
```

- [ ] **Step 2: Run test — confirm red**

```bash
pnpm -C apps/ck5-export vitest run src/__tests__/hyperlink-token.test.ts
```

Expected: FAIL on the override test — the hardcoded `"0563C1"` leaks through.

- [ ] **Step 3: Extend theme tokens**

Edit `apps/ck5-export/src/layout-ir/index.ts`. Add `hyperlink` to the `theme` shape and default:

Change the type (around line 34):
```ts
  theme: Readonly<{
    accent: string;
    accentLight: string;
    accentDark: string;
    accentBorder: string;
    hyperlink: string;
  }>;
```

Change the default (around line 141 inside `defaultLayoutTokens`):
```ts
  theme: {
    accent: "#6b1f2a",
    accentLight: "#f9f3f3",
    accentDark: "#3e1018",
    accentBorder: "#dfc8c8",
    hyperlink: "#0563C1",
  },
```

- [ ] **Step 4: Read color from tokens in emitter**

Edit `apps/ck5-export/src/ck5-docx-emitter.ts`. Replace `collectHyperlinkRuns` (lines 101-153). Both the `text` case and the `field` case read `tokens.theme.hyperlink` via `hexToFill`:

```ts
function collectHyperlinkRuns(nodes: ExportNode[], tokens: LayoutTokens): TextRun[] {
  const runs: TextRun[] = []
  const linkColor = hexToFill(tokens.theme.hyperlink)

  for (const node of nodes) {
    switch (node.kind) {
      case "text": {
        const set = new Set(node.marks ?? [])
        runs.push(
          new TextRun({
            text: node.value,
            font: tokens.typography.exportFont,
            size: ptToHalfPt(tokens.typography.baseSizePt),
            color: linkColor,
            bold: set.has("bold"),
            italics: set.has("italic"),
            underline: {},
            strike: set.has("strike"),
          }),
        )
        break
      }
      case "lineBreak":
        runs.push(new TextRun({ break: 1 }))
        break
      case "hyperlink":
        runs.push(...collectHyperlinkRuns(node.children, tokens))
        break
      case "paragraph":
      case "heading":
      case "blockquote":
      case "listItem":
      case "tableCell":
      case "repeatableItem":
        runs.push(...collectHyperlinkRuns(node.children, tokens))
        break
      case "field":
        runs.push(
          new TextRun({
            text: node.value,
            font: tokens.typography.exportFont,
            size: ptToHalfPt(tokens.typography.baseSizePt),
            color: linkColor,
            underline: {},
          }),
        )
        break
      default:
        break
    }
  }

  return runs
}
```

- [ ] **Step 5: Re-run test — confirm green**

```bash
pnpm -C apps/ck5-export vitest run src/__tests__/hyperlink-token.test.ts
```

Expected: PASS 2/2.

- [ ] **Step 6: Run full ck5-export suite — confirm no regression**

```bash
pnpm -C apps/ck5-export test
```

Expected: PASS all.

- [ ] **Step 7: Commit**

```bash
git add apps/ck5-export/src/layout-ir/index.ts apps/ck5-export/src/ck5-docx-emitter.ts apps/ck5-export/src/__tests__/hyperlink-token.test.ts
git commit -m "fix(ck5-export): source hyperlink color from LayoutTokens.theme.hyperlink

The hex #0563C1 was inline in ck5-docx-emitter.ts at two call sites.
Promote to tokens so theme overrides actually apply to link rendering."
```

---

## Phase 2 — Delete Dead IR Pipeline

### Task 4: Delete dead legacy `docx-emitter/` directory

**Goal:** Route `/render/docx` has never imported `apps/ck5-export/src/docx-emitter/`; the directory exists only because it was moved from `mddm-editor/` during Plan C and the parity retirement plan was abandoned. Delete the whole directory (source + tests). Spec §Testing step 3 explicitly allowed retirement once dual-path parity was no longer needed.

**Files:**
- Delete: `apps/ck5-export/src/docx-emitter/` (entire directory, ~976 LOC source + all `__tests__/`)

**Acceptance Criteria:**
- [ ] `apps/ck5-export/src/docx-emitter/` no longer exists.
- [ ] `pnpm -C apps/ck5-export build && pnpm -C apps/ck5-export test` pass.
- [ ] No file outside `docx-emitter/` imports from `docx-emitter/` (verified by grep).

**Verify:**
```
grep -R "from ['\"]\\.\\.\\?/docx-emitter" apps/ck5-export/src | grep -v ck5-docx-emitter
```
Expected: empty output.

**Steps:**

- [ ] **Step 1: Prove no live importers remain**

```bash
grep -RIn "docx-emitter" apps/ck5-export/src --include="*.ts" | grep -v "apps/ck5-export/src/docx-emitter/" | grep -v "ck5-docx-emitter" | grep -v "__tests__/__fixtures__"
```

Expected: empty output (every remaining occurrence is either inside the condemned dir itself or a comment in unrelated file).

If any line surfaces from outside the dead dir → STOP and investigate before deleting.

- [ ] **Step 2: Remove the directory**

```bash
git rm -r apps/ck5-export/src/docx-emitter
```

- [ ] **Step 3: Build + test to confirm zero regression**

```bash
pnpm -C apps/ck5-export typecheck
pnpm -C apps/ck5-export build
pnpm -C apps/ck5-export test
```

Expected all three: exit 0. `build` is required alongside `typecheck` because the package's emit step can fail (e.g., unresolved bundler imports) even when `typecheck` is green.

- [ ] **Step 4: Commit**

```bash
git commit -m "chore(ck5-export)!: delete dead legacy docx-emitter directory

Route /render/docx uses ck5-docx-emitter.ts exclusively. The legacy
directory moved in from mddm-editor during Plan C was never wired up
and the dual-path parity migration was abandoned. ~976 LOC + tests
removed.

BREAKING: none at runtime (no public re-exports). Internal refactor only."
```

---

### Task 5: Delete `codecs/` and `layout-interpreter/`

**Goal:** Same rationale — both exist only to support the dead legacy emitter removed in Task 4. Remove the two directories together.

**Files:**
- Delete: `apps/ck5-export/src/codecs/` (~739 LOC + tests)
- Delete: `apps/ck5-export/src/layout-interpreter/` (~294 LOC + tests)

**Acceptance Criteria:**
- [ ] Both directories removed.
- [ ] `pnpm -C apps/ck5-export typecheck && pnpm -C apps/ck5-export test` pass.

**Verify:**
```
grep -R "from ['\"]\\.\\.\\?/\\(codecs\\|layout-interpreter\\)" apps/ck5-export/src
```
Expected: empty.

**Steps:**

- [ ] **Step 1: Confirm no surviving importers**

```bash
grep -RIn "from [\"']\\.\\.\\?/codecs" apps/ck5-export/src --include="*.ts"
grep -RIn "from [\"']\\.\\.\\?/layout-interpreter" apps/ck5-export/src --include="*.ts"
```

Both expected: empty (Task 4 removed the only consumers — the legacy emitters).

- [ ] **Step 2: Remove both directories**

```bash
git rm -r apps/ck5-export/src/codecs
git rm -r apps/ck5-export/src/layout-interpreter
```

- [ ] **Step 3: Build + test**

```bash
pnpm -C apps/ck5-export typecheck
pnpm -C apps/ck5-export test
```

Expected both: exit 0.

- [ ] **Step 4: Commit**

```bash
git commit -m "chore(ck5-export)!: delete dead codecs + layout-interpreter

Both directories only fed the legacy docx-emitter removed in the previous
commit. No routes, no public entry points depend on them. ~1033 LOC +
tests removed.

Spec reminder: 'CK5 HTML is the single source of truth. IR is ephemeral
on export only.' These directories rebuilt a persistent IR pipeline
in violation of that principle."
```

---

### Task 6: Delete `shared/adapter.ts` if orphaned

**Goal:** `shared/adapter.ts` holds legacy `MDDMBlock` / `MDDMTextRun` types from the old mddm-editor IR. Verify it has no live importers after Tasks 4-5, then delete.

**Files:**
- Delete (conditional): `apps/ck5-export/src/shared/adapter.ts`

**Acceptance Criteria:**
- [ ] If orphan: file removed and tests pass.
- [ ] If live: file kept; note in commit log why.

**Verify:**
```
grep -R "shared/adapter" apps/ck5-export/src
```
Expected: empty → orphan confirmed.

**Steps:**

- [ ] **Step 1: Grep for importers**

```bash
grep -RIn "shared/adapter" apps/ck5-export/src --include="*.ts"
```

- [ ] **Step 2: If empty, delete**

```bash
git rm apps/ck5-export/src/shared/adapter.ts
pnpm -C apps/ck5-export typecheck
pnpm -C apps/ck5-export test
```

Expected: both exit 0.

If the grep in Step 1 surfaced any live importer outside deleted dirs, skip the delete and proceed to Step 3 instead — record why in the commit body.

- [ ] **Step 3: Commit**

```bash
git add -A apps/ck5-export
git commit -m "chore(ck5-export): remove orphaned shared/adapter.ts

MDDMBlock/MDDMTextRun types were only referenced by the deleted
codecs/ and layout-interpreter/. No surviving importer."
```

If kept:
```bash
git commit -m "chore(ck5-export): keep shared/adapter.ts — still imported by <path>" --allow-empty
```

---

### Task 7: Flatten `layout-ir/` → `layout-tokens.ts`

**Goal:** The surviving tokens (`LayoutTokens` type + `defaultLayoutTokens` const) belong in a single flat file. `ComponentRules` + `defaultComponentRules` become dead after Task 5 — delete them. The directory framing ("IR") was a spec violation.

**Files:**
- Create: `apps/ck5-export/src/layout-tokens.ts`
- Delete: `apps/ck5-export/src/layout-ir/` (entire directory)
- Modify: `apps/ck5-export/src/ck5-docx-emitter.ts` (import update)
- Modify: `apps/ck5-export/src/routes/render-docx.ts` (import update)
- Modify: `apps/ck5-export/src/print-stylesheet/print-css.ts` (import update)
- Modify: `apps/ck5-export/src/__tests__/field-label-round-trip.test.ts` (import update)
- Modify: `apps/ck5-export/src/__tests__/hyperlink-token.test.ts` (import update)

**Acceptance Criteria:**
- [ ] `layout-ir/` no longer exists.
- [ ] `layout-tokens.ts` exports `LayoutTokens` type and `defaultLayoutTokens` const — NOT `ComponentRules` or `defaultComponentRules`.
- [ ] All importers now reference `../layout-tokens` (or `./layout-tokens`).
- [ ] `typecheck` + `test` green.

**Verify:**
```
grep -R "layout-ir" apps/ck5-export/src
```
Expected: empty.

**Steps:**

- [ ] **Step 1: Create the flat tokens file**

Create `apps/ck5-export/src/layout-tokens.ts`. Copy the `LayoutTokens` type, `PAGE_*` / `MARGIN_*` constants, and `defaultLayoutTokens` const **exactly** as they exist in `layout-ir/index.ts` — including the `hyperlink` field added in Task 3. Do NOT copy:
- `SectionRule`, `FieldRule`, `FieldGroupRule`, `DataTableRule`, `RepeatableRule`, `RichBlockRule`, `ComponentRules` types
- `defaultComponentRules` const

The entire file is the tokens types block + `defaultLayoutTokens` and nothing else.

- [ ] **Step 2: Update importers**

Find all importers:
```bash
grep -RIln "layout-ir" apps/ck5-export/src --include="*.ts"
```

For each hit, change:
- `from "../layout-ir"` → `from "../layout-tokens"`
- `from "./layout-ir"` → `from "./layout-tokens"`

Known importers to update (verify with grep first):
- `apps/ck5-export/src/ck5-docx-emitter.ts`
- `apps/ck5-export/src/routes/render-docx.ts`
- `apps/ck5-export/src/print-stylesheet/print-css.ts`
- `apps/ck5-export/src/__tests__/field-label-round-trip.test.ts` (created in Task 2)
- `apps/ck5-export/src/__tests__/hyperlink-token.test.ts` (created in Task 3)

- [ ] **Step 3: Delete `layout-ir/` directory**

```bash
git rm -r apps/ck5-export/src/layout-ir
```

- [ ] **Step 4: typecheck + test**

```bash
pnpm -C apps/ck5-export typecheck
pnpm -C apps/ck5-export test
```

Both expected: exit 0. If `ComponentRules` import leaks anywhere unexpected, surface and delete the consumer.

- [ ] **Step 5: Commit**

```bash
git add -A apps/ck5-export/src
git commit -m "refactor(ck5-export): flatten layout-ir/ to layout-tokens.ts

Plan C spec required ephemeral IR. The layout-ir/ directory framed
design tokens as a persistent IR with ComponentRules scaffolding for
the (now deleted) layout-interpreter. Collapse to a single flat
layout-tokens.ts exporting only LayoutTokens type + defaultLayoutTokens.
ComponentRules + defaultComponentRules deleted — they had no surviving
consumer."
```

---

## Phase 3 — Refactor the New Emitter

### Task 8: Seed direct docx-emitter coverage before the split

**Goal:** Task 4 deleted the legacy `docx-emitter/__tests__/` suite (~15 test files). The only remaining coverage of the surviving emitter is the route-level smoke test at `src/__tests__/render-docx.test.ts`, which asserts the response ZIP magic bytes and error paths — it does not exercise any specific block family. Before we split the 565-LOC god file in Task 9, pin the behavior of every block family with direct unit tests against `ck5-docx-emitter.ts`. These tests will travel with the split: only the import path changes in Task 9 Step 14.

**Files:**
- Create: `apps/ck5-export/src/__tests__/emitter-blocks.test.ts` (one describe block per emitter concern)

**Acceptance Criteria:**
- [ ] `emitter-blocks.test.ts` has a describe block for each of: paragraph, heading, list (ordered + bulleted), blockquote, image, field, hyperlink, table, section, repeatable, `collectImageUrls`.
- [ ] Every test renders via `emitDocxFromExportTree` + `readDocxDocumentXml` and asserts against `word/document.xml` (no raw-ZIP `toString("utf8")`).
- [ ] All tests pass against the current `ck5-docx-emitter.ts` (green baseline).
- [ ] After Task 9 Step 14 updates the import path, they still pass without behavioral changes.

**Verify:** `pnpm -C apps/ck5-export vitest run src/__tests__/emitter-blocks.test.ts` → exit 0, all describe blocks green.

**Steps:**

- [ ] **Step 1: Create the test file**

Create `apps/ck5-export/src/__tests__/emitter-blocks.test.ts`:

```ts
import { describe, it, expect } from "vitest"
import { htmlToExportTree } from "../html-to-export-tree"
import { collectImageUrls, emitDocxFromExportTree } from "../ck5-docx-emitter"
import { defaultLayoutTokens } from "../layout-tokens"
import type { ResolvedAsset } from "../asset-resolver"
import { readDocxDocumentXml } from "./helpers/read-docx-xml"

const emptyAssets = new Map<string, ResolvedAsset>()

async function emit(html: string, assets = emptyAssets): Promise<string> {
  const tree = htmlToExportTree(html)
  const doc = emitDocxFromExportTree(tree, defaultLayoutTokens, assets)
  return readDocxDocumentXml(doc)
}

describe("emitter — paragraph", () => {
  it("renders a plain paragraph with text run", async () => {
    const xml = await emit("<p>Hello world</p>")
    expect(xml).toMatch(/<w:p[\s>]/)
    expect(xml).toMatch(/<w:t[^>]*>Hello world<\/w:t>/)
  })

  it("renders bold + italic + underline marks", async () => {
    const xml = await emit("<p><strong>b</strong><em>i</em><u>u</u></p>")
    expect(xml).toMatch(/<w:b[\s/]/)
    expect(xml).toMatch(/<w:i[\s/]/)
    expect(xml).toMatch(/<w:u[\s/]/)
  })
})

describe("emitter — heading", () => {
  it("emits heading style for H1..H3", async () => {
    const xml = await emit("<h1>one</h1><h2>two</h2><h3>three</h3>")
    expect(xml).toMatch(/w:pStyle[^>]*w:val="Heading1"/)
    expect(xml).toMatch(/w:pStyle[^>]*w:val="Heading2"/)
    expect(xml).toMatch(/w:pStyle[^>]*w:val="Heading3"/)
  })
})

describe("emitter — list", () => {
  it("bulleted list emits numPr/ilvl", async () => {
    const xml = await emit("<ul><li>a</li><li>b</li></ul>")
    expect(xml).toMatch(/<w:numPr>/)
    expect(xml).toMatch(/<w:t[^>]*>a<\/w:t>/)
    expect(xml).toMatch(/<w:t[^>]*>b<\/w:t>/)
  })

  it("ordered list emits numPr with numbering reference", async () => {
    const xml = await emit("<ol><li>a</li></ol>")
    expect(xml).toMatch(/<w:numPr>/)
  })
})

describe("emitter — blockquote", () => {
  it("blockquote renders with left indent", async () => {
    const xml = await emit("<blockquote>quote</blockquote>")
    expect(xml).toMatch(/<w:ind[^>]*w:left="720"/)
    expect(xml).toMatch(/<w:t[^>]*>quote<\/w:t>/)
  })
})

describe("emitter — image", () => {
  it("skips images whose asset was not resolved", async () => {
    const xml = await emit('<p><img src="/assets/missing.png"></p>')
    expect(xml).not.toMatch(/<w:drawing/)
  })

  it("renders a drawing when the asset resolves", async () => {
    const bytes = Buffer.from(
      // 1x1 transparent PNG
      "iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mNkYAAAAAYAAjCB0C8AAAAASUVORK5CYII=",
      "base64",
    )
    const assets = new Map<string, ResolvedAsset>([
      ["/assets/one.png", { bytes, mimeType: "image/png" }],
    ])
    const xml = await emit('<p><img src="/assets/one.png" width="10" height="10"></p>', assets)
    expect(xml).toMatch(/<w:drawing/)
  })
})

describe("emitter — field", () => {
  it("renders a 2-column table with label and value", async () => {
    const xml = await emit(
      '<span class="mddm-field" data-field-id="x" data-field-label="X" data-field-type="text">v</span>',
    )
    expect(xml).toMatch(/<w:tbl[\s>]/)
    expect(xml).toMatch(/<w:t[^>]*>X<\/w:t>/)
    expect(xml).toMatch(/<w:t[^>]*>v<\/w:t>/)
  })
})

describe("emitter — hyperlink", () => {
  it("renders a hyperlink text run", async () => {
    const xml = await emit('<p><a href="https://example.com">link</a></p>')
    expect(xml).toMatch(/<w:t[^>]*>link<\/w:t>/)
    expect(xml).toMatch(/w:color[^>]*w:val="0563C1"/)
  })
})

describe("emitter — table", () => {
  it("renders figure.table → w:tbl with rows and cells", async () => {
    const xml = await emit(
      '<figure class="table"><table><tbody>' +
        "<tr><th>h1</th><th>h2</th></tr>" +
        "<tr><td>a</td><td>b</td></tr>" +
        "</tbody></table></figure>",
    )
    expect(xml).toMatch(/<w:tbl[\s>]/)
    expect(xml).toMatch(/<w:t[^>]*>h1<\/w:t>/)
    expect(xml).toMatch(/<w:t[^>]*>h2<\/w:t>/)
    expect(xml).toMatch(/<w:t[^>]*>a<\/w:t>/)
    expect(xml).toMatch(/<w:t[^>]*>b<\/w:t>/)
  })
})

describe("emitter — section", () => {
  it("section with header + body emits two rows in a framing table", async () => {
    const xml = await emit(
      '<section class="mddm-section" data-variant="bordered">' +
        '<header class="mddm-section-header"><h2>Title</h2></header>' +
        '<div class="mddm-section-body"><p>body</p></div>' +
        "</section>",
    )
    expect(xml).toMatch(/<w:t[^>]*>Title<\/w:t>/)
    expect(xml).toMatch(/<w:t[^>]*>body<\/w:t>/)
    // Two rows (header + body) in the section's framing table.
    const rows = xml.match(/<w:tr[\s>]/g) ?? []
    expect(rows.length).toBeGreaterThanOrEqual(2)
  })
})

describe("emitter — repeatable", () => {
  it("each repeatable item emits its own framing table", async () => {
    const xml = await emit(
      '<ol class="mddm-repeatable">' + "<li><p>one</p></li><li><p>two</p></li>" + "</ol>",
    )
    expect(xml).toMatch(/<w:t[^>]*>one<\/w:t>/)
    expect(xml).toMatch(/<w:t[^>]*>two<\/w:t>/)
    // At least two tables (one per item).
    const tables = xml.match(/<w:tbl[\s>]/g) ?? []
    expect(tables.length).toBeGreaterThanOrEqual(2)
  })
})

describe("asset-collector — collectImageUrls", () => {
  it("collects urls from nested sections, repeatables, lists, tables", () => {
    const html =
      '<section class="mddm-section">' +
      '<div class="mddm-section-body">' +
      '<p><img src="/a.png"></p>' +
      '<ol class="mddm-repeatable"><li><p><img src="/b.png"></p></li></ol>' +
      "<ul><li><img src=\"/c.png\"></li></ul>" +
      '<figure class="table"><table><tr><td><img src="/d.png"></td></tr></table></figure>' +
      "</div></section>"
    const tree = htmlToExportTree(html)
    const urls = collectImageUrls(tree)
    expect(urls.sort()).toEqual(["/a.png", "/b.png", "/c.png", "/d.png"])
  })

  it("deduplicates repeated urls", () => {
    const html = '<p><img src="/same.png"></p><p><img src="/same.png"></p>'
    const tree = htmlToExportTree(html)
    expect(collectImageUrls(tree)).toEqual(["/same.png"])
  })
})
```

- [ ] **Step 2: Run — confirm green baseline**

```bash
pnpm -C apps/ck5-export vitest run src/__tests__/emitter-blocks.test.ts
```

Expected: all tests PASS against the current `ck5-docx-emitter.ts`. If any test fails, that is a real pre-existing bug — STOP and triage before proceeding.

- [ ] **Step 3: Run full ck5-export suite — confirm no regression**

```bash
pnpm -C apps/ck5-export test
```

Expected: exit 0.

- [ ] **Step 4: Commit**

```bash
git add apps/ck5-export/src/__tests__/emitter-blocks.test.ts
git commit -m "test(ck5-export): pin per-block emitter behavior before split

Task 4 removed the legacy docx-emitter/__tests__ suite. The surviving
emitter had no direct unit coverage — only a route-level ZIP-magic-bytes
smoke test. Seed per-block-family tests (paragraph, heading, list,
blockquote, image, field, hyperlink, table, section, repeatable) plus
collectImageUrls against ck5-docx-emitter.ts. The next task splits that
file; these tests travel with it and must stay green."
```

---

### Task 9: Split `ck5-docx-emitter.ts` into per-block `docx-emitter/` module

**Goal:** The surviving emitter is 565 LOC in one file. Restore block-per-file modularity (the shape the legacy dir had before Plan C deleted mddm-editor). Name `docx-emitter/` is free after Task 4. The `emitter-blocks.test.ts` suite seeded in Task 8 is the behavioral safety net — it must stay green across the split with only import-path updates.

**Architecture — how we avoid circular imports:**

Three leaf modules recurse into block children: `table.ts` (cells can contain paragraphs / lists / nested blocks), `section.ts` (header + body are block lists), `repeatable.ts` (items are block lists). They must not `import { emitBlocks } from "./block-dispatch"` while `block-dispatch.ts` imports them back — that is a cycle, and the "type-only import" workaround is a fiction (a type cannot be called at runtime).

Instead, we use **callback injection**: `helpers.ts` declares an `EmitBlocks` function type. The leaf modules that recurse accept `emitBlocks: EmitBlocks` as an extra parameter. `block-dispatch.ts` owns the single concrete `emitBlocks` implementation and passes itself into those leaves when calling them. Import graph becomes a DAG:

```
helpers.ts  (leaf — no sibling imports)
inline.ts   → helpers
paragraph.ts, heading.ts, list.ts, blockquote.ts, image.ts, field.ts, asset-collector.ts → helpers, inline
table.ts, section.ts, repeatable.ts → helpers, inline  (NO import from block-dispatch)
block-dispatch.ts → every leaf above (emitBlocks passes itself into table/section/repeatable)
emitter.ts → block-dispatch, list (for DEFAULT_NUMBERING_REFERENCE), helpers (for mmToTwip via shared)
index.ts   → emitter, asset-collector  (barrel)
```

No cycle. Every leaf module is independently unit-testable.

**Files:**
- Create: `apps/ck5-export/src/docx-emitter/` directory with split files (see below).
- Delete: `apps/ck5-export/src/ck5-docx-emitter.ts`
- Modify: `apps/ck5-export/src/routes/render-docx.ts` (import path change)
- Modify: `apps/ck5-export/src/__tests__/field-label-round-trip.test.ts` (import path change)
- Modify: `apps/ck5-export/src/__tests__/hyperlink-token.test.ts` (import path change)
- Modify: `apps/ck5-export/src/__tests__/emitter-blocks.test.ts` (import path change — the Task 8 safety net)

**Target module layout:**
```
apps/ck5-export/src/docx-emitter/
  index.ts           ← barrel: re-export emitDocxFromExportTree + collectImageUrls
  emitter.ts         ← emitDocxFromExportTree (document-level assembly)
  block-dispatch.ts  ← emitBlock + emitBlocks (the central switch; injects itself into leaves that recurse)
  inline.ts          ← buildTextRun, collectInlineRuns, collectHyperlinkRuns, mapHyperlinkRuns
  asset-collector.ts ← collectImageUrls
  paragraph.ts       ← emitParagraph, paragraphFromInlineChildren
  heading.ts         ← emitHeading, toHeadingLevel
  list.ts            ← emitList, DEFAULT_NUMBERING_REFERENCE const
  image.ts           ← emitImage, DEFAULT_IMAGE_* consts, toDocxImageType
  field.ts           ← emitField (the one Task 2 fixed)
  table.ts           ← emitTable, emitTableRow, emitTableCell (all accept EmitBlocks)
  section.ts         ← emitSection, getSectionBorder (accepts EmitBlocks)
  repeatable.ts      ← emitRepeatable, emitRepeatableItem, ensureCellChildren (accepts EmitBlocks)
  helpers.ts         ← hexToFill + type aliases DocxBlock, HeadingLevelValue, EmitBlocks
```

**Acceptance Criteria:**
- [ ] `ck5-docx-emitter.ts` deleted.
- [ ] Each new file holds a coherent block concern.
- [ ] `docx-emitter/index.ts` re-exports `emitDocxFromExportTree` and `collectImageUrls`.
- [ ] No new behavioral change — `emitter-blocks.test.ts` (Task 8), `field-label-round-trip.test.ts` (Task 2), `hyperlink-token.test.ts` (Task 3), and `render-docx.test.ts` stay green with only import-path updates.
- [ ] No circular imports. Verified by `madge` in Step 15, or by inspection of the dependency graph above.

**Verify:** `pnpm -C apps/ck5-export typecheck && pnpm -C apps/ck5-export test` → exit 0 both.

**Steps:**

- [ ] **Step 1: Create helpers module**

Create `apps/ck5-export/src/docx-emitter/helpers.ts`:

```ts
import { HeadingLevel, type Paragraph, type Table } from "docx"
import type { ExportNode } from "../export-node"
import type { LayoutTokens } from "../layout-tokens"
import type { ResolvedAsset } from "../asset-resolver"

export type DocxBlock = Paragraph | Table
export type HeadingLevelValue = (typeof HeadingLevel)[keyof typeof HeadingLevel]

// Callback type used to break the cycle between block-dispatch and the
// leaf emitters that recurse into child blocks (table, section, repeatable).
// Passed in by block-dispatch; never imported from block-dispatch by leaves.
export type EmitBlocks = (
  nodes: ExportNode[],
  tokens: LayoutTokens,
  assetMap: ReadonlyMap<string, ResolvedAsset>,
) => DocxBlock[]

export function hexToFill(hex: string): string {
  return hex.replace(/^#/, "").toUpperCase()
}
```

- [ ] **Step 2: Create inline module**

Create `apps/ck5-export/src/docx-emitter/inline.ts`. Copy `buildTextRun`, `mapHyperlinkRuns`, `collectHyperlinkRuns`, `collectInlineRuns` from `ck5-docx-emitter.ts:68-186` **verbatim** (including the hyperlink-color-from-tokens change landed in Task 3). Imports:

```ts
import { TextRun } from "docx"
import type { ExportNode, Hyperlink } from "../export-node"
import type { LayoutTokens } from "../layout-tokens"
import { ptToHalfPt } from "../shared/helpers/units"
import { hexToFill } from "./helpers"
```

Exports: `buildTextRun`, `mapHyperlinkRuns`, `collectHyperlinkRuns`, `collectInlineRuns`.

- [ ] **Step 3: Create paragraph module**

Create `apps/ck5-export/src/docx-emitter/paragraph.ts`. Move `paragraphFromInlineChildren` and `emitParagraph` from `ck5-docx-emitter.ts:188-210`. Imports:

```ts
import { Paragraph, TextRun, type IParagraphOptions } from "docx"
import type { ExportNode, Paragraph as ExportParagraph } from "../export-node"
import type { LayoutTokens } from "../layout-tokens"
import { collectInlineRuns } from "./inline"
```

Exports: `paragraphFromInlineChildren`, `emitParagraph`.

- [ ] **Step 4: Create heading module**

Create `apps/ck5-export/src/docx-emitter/heading.ts`. Move `toHeadingLevel` (`ck5-docx-emitter.ts:44-59`) and `emitHeading` (`200-206`). Imports:

```ts
import { HeadingLevel, type Paragraph } from "docx"
import type { Heading } from "../export-node"
import type { LayoutTokens } from "../layout-tokens"
import { paragraphFromInlineChildren } from "./paragraph"
import type { HeadingLevelValue } from "./helpers"
```

Exports: `toHeadingLevel`, `emitHeading`.

- [ ] **Step 5: Create list module**

Create `apps/ck5-export/src/docx-emitter/list.ts`. Move `emitList` + `DEFAULT_NUMBERING_REFERENCE` const (`ck5-docx-emitter.ts:35, 212-222`). Imports:

```ts
import type { Paragraph } from "docx"
import type { List } from "../export-node"
import type { LayoutTokens } from "../layout-tokens"
import { paragraphFromInlineChildren } from "./paragraph"

export const DEFAULT_NUMBERING_REFERENCE = "default-numbering"
```

Exports: `DEFAULT_NUMBERING_REFERENCE`, `emitList`.

- [ ] **Step 6: Create image module**

Create `apps/ck5-export/src/docx-emitter/image.ts`. Move `DEFAULT_IMAGE_*` consts, `toDocxImageType`, `emitImage` (`ck5-docx-emitter.ts:33-34, 61-66, 224-249`). Imports:

```ts
import { ImageRun, Paragraph } from "docx"
import type { Image } from "../export-node"
import type { ResolvedAsset } from "../asset-resolver"
```

Exports: `emitImage`.

- [ ] **Step 7: Create field module**

Create `apps/ck5-export/src/docx-emitter/field.ts`. Move `emitField` (`ck5-docx-emitter.ts:251-286`, the version already fixed in Task 2). Imports:

```ts
import { BorderStyle, Paragraph, Table, TableCell, TableRow, TextRun, WidthType } from "docx"
import type { ExportNode } from "../export-node"
import type { LayoutTokens } from "../layout-tokens"
import { hexToFill } from "./helpers"
```

Exports: `emitField`.

- [ ] **Step 8: Create table module (accepts `EmitBlocks` via injection)**

Create `apps/ck5-export/src/docx-emitter/table.ts`. Move `emitTableCell`, `emitTableRow`, `emitTable` (`ck5-docx-emitter.ts:288-331`). **Do NOT import from `./block-dispatch`.** Accept an `emitBlocks: EmitBlocks` parameter on the functions that recurse:

```ts
import { BorderStyle, Paragraph, Table, TableCell, TableRow, WidthType } from "docx"
import type {
  TableCell as ExportTableCell,
  TableRow as ExportTableRow,
  Table as ExportTable,
} from "../export-node"
import type { ResolvedAsset } from "../asset-resolver"
import type { LayoutTokens } from "../layout-tokens"
import { hexToFill, type EmitBlocks } from "./helpers"

export function emitTableCell(
  node: ExportTableCell,
  tokens: LayoutTokens,
  assetMap: ReadonlyMap<string, ResolvedAsset>,
  emitBlocks: EmitBlocks,
): TableCell {
  const borderColor = hexToFill(tokens.theme.accentBorder)
  const children = emitBlocks(node.children, tokens, assetMap)

  return new TableCell({
    columnSpan: node.colspan,
    rowSpan: node.rowspan,
    shading: node.isHeader
      ? { fill: hexToFill(tokens.theme.accentLight), type: "clear", color: "auto" }
      : undefined,
    borders: {
      top: { style: BorderStyle.SINGLE, size: 4, color: borderColor },
      bottom: { style: BorderStyle.SINGLE, size: 4, color: borderColor },
      left: { style: BorderStyle.SINGLE, size: 4, color: borderColor },
      right: { style: BorderStyle.SINGLE, size: 4, color: borderColor },
    },
    children: children.length > 0 ? children : [new Paragraph({ children: [] })],
  })
}

export function emitTableRow(
  node: ExportTableRow,
  tokens: LayoutTokens,
  assetMap: ReadonlyMap<string, ResolvedAsset>,
  emitBlocks: EmitBlocks,
): TableRow {
  return new TableRow({
    children: node.cells.map((cell) => emitTableCell(cell, tokens, assetMap, emitBlocks)),
  })
}

export function emitTable(
  node: ExportTable,
  tokens: LayoutTokens,
  assetMap: ReadonlyMap<string, ResolvedAsset>,
  emitBlocks: EmitBlocks,
): Table[] {
  return [
    new Table({
      width: { size: 100, type: WidthType.PERCENTAGE },
      rows: node.rows.map((row) => emitTableRow(row, tokens, assetMap, emitBlocks)),
    }),
  ]
}
```

- [ ] **Step 9: Create repeatable module (accepts `EmitBlocks`)**

Create `apps/ck5-export/src/docx-emitter/repeatable.ts`. Move `emitRepeatableItem`, `emitRepeatable`, `ensureCellChildren` (`ck5-docx-emitter.ts:333-370`). Accept `emitBlocks`:

```ts
import { BorderStyle, Paragraph, Table, TableCell, TableRow, WidthType } from "docx"
import type { Repeatable, RepeatableItem } from "../export-node"
import type { ResolvedAsset } from "../asset-resolver"
import type { LayoutTokens } from "../layout-tokens"
import { hexToFill, type DocxBlock, type EmitBlocks } from "./helpers"

export function ensureCellChildren(children: DocxBlock[]): DocxBlock[] {
  return children.length > 0 ? children : [new Paragraph({ children: [] })]
}

export function emitRepeatableItem(
  item: RepeatableItem,
  tokens: LayoutTokens,
  assetMap: ReadonlyMap<string, ResolvedAsset>,
  emitBlocks: EmitBlocks,
): Table[] {
  const borderColor = hexToFill(tokens.theme.accentBorder)
  return [
    new Table({
      width: { size: 100, type: WidthType.PERCENTAGE },
      rows: [
        new TableRow({
          children: [
            new TableCell({
              borders: {
                top: { style: BorderStyle.SINGLE, size: 4, color: borderColor },
                bottom: { style: BorderStyle.SINGLE, size: 4, color: borderColor },
                left: { style: BorderStyle.SINGLE, size: 8, color: hexToFill(tokens.theme.accent) },
                right: { style: BorderStyle.SINGLE, size: 4, color: borderColor },
              },
              children: ensureCellChildren(emitBlocks(item.children, tokens, assetMap)),
            }),
          ],
        }),
      ],
    }),
  ]
}

export function emitRepeatable(
  node: Repeatable,
  tokens: LayoutTokens,
  assetMap: ReadonlyMap<string, ResolvedAsset>,
  emitBlocks: EmitBlocks,
): Table[] {
  return node.items.flatMap((item) => emitRepeatableItem(item, tokens, assetMap, emitBlocks))
}
```

- [ ] **Step 10: Create section module (accepts `EmitBlocks`)**

Create `apps/ck5-export/src/docx-emitter/section.ts`. Move `getSectionBorder`, `emitSection` (`ck5-docx-emitter.ts:85-95, 373-417`). Accept `emitBlocks`:

```ts
import {
  BorderStyle,
  Table,
  TableCell,
  TableRow,
  WidthType,
  type IBorderOptions,
} from "docx"
import type { Section } from "../export-node"
import type { ResolvedAsset } from "../asset-resolver"
import type { LayoutTokens } from "../layout-tokens"
import { hexToFill, type EmitBlocks } from "./helpers"
import { ensureCellChildren } from "./repeatable"

export function getSectionBorder(variant: Section["variant"], tokens: LayoutTokens): IBorderOptions {
  if (variant === "plain") {
    return { style: BorderStyle.NONE, size: 0, color: "auto" }
  }
  if (variant === "solid") {
    return { style: BorderStyle.SINGLE, size: 8, color: hexToFill(tokens.theme.accent) }
  }
  return { style: BorderStyle.SINGLE, size: 4, color: hexToFill(tokens.theme.accentBorder) }
}

export function emitSection(
  node: Section,
  tokens: LayoutTokens,
  assetMap: ReadonlyMap<string, ResolvedAsset>,
  emitBlocks: EmitBlocks,
): Table[] {
  const border = getSectionBorder(node.variant, tokens)
  const headerFill =
    node.variant === "plain"
      ? undefined
      : { fill: hexToFill(tokens.theme.accent), type: "clear" as const, color: "auto" }

  const rows: TableRow[] = []

  if (node.header && node.header.length > 0) {
    rows.push(
      new TableRow({
        children: [
          new TableCell({
            shading: headerFill,
            borders: { top: border, bottom: border, left: border, right: border },
            children: ensureCellChildren(emitBlocks(node.header, tokens, assetMap)),
          }),
        ],
      }),
    )
  }

  rows.push(
    new TableRow({
      children: [
        new TableCell({
          borders: { top: border, bottom: border, left: border, right: border },
          children: ensureCellChildren(emitBlocks(node.body, tokens, assetMap)),
        }),
      ],
    }),
  )

  return [new Table({ width: { size: 100, type: WidthType.PERCENTAGE }, rows })]
}
```

- [ ] **Step 11: Create block-dispatch module (the only module that knows every leaf)**

Create `apps/ck5-export/src/docx-emitter/block-dispatch.ts`. Move `emitBlock` + `emitBlocks` (`ck5-docx-emitter.ts:419-460`). This is the **only** place that imports every sibling; leaves never import back:

```ts
import { Paragraph, Table, TableRow } from "docx"
import type { ExportNode } from "../export-node"
import type { ResolvedAsset } from "../asset-resolver"
import type { LayoutTokens } from "../layout-tokens"
import type { DocxBlock } from "./helpers"
import { collectInlineRuns } from "./inline"
import { paragraphFromInlineChildren } from "./paragraph"
import { emitParagraph } from "./paragraph"
import { emitHeading } from "./heading"
import { emitList } from "./list"
import { emitImage } from "./image"
import { emitField } from "./field"
import { emitTable, emitTableRow, emitTableCell } from "./table"
import { emitSection } from "./section"
import { emitRepeatable, emitRepeatableItem } from "./repeatable"

export function emitBlocks(
  nodes: ExportNode[],
  tokens: LayoutTokens,
  assetMap: ReadonlyMap<string, ResolvedAsset>,
): DocxBlock[] {
  return nodes.flatMap((node) => emitBlock(node, tokens, assetMap))
}

function emitBlock(
  node: ExportNode,
  tokens: LayoutTokens,
  assetMap: ReadonlyMap<string, ResolvedAsset>,
): DocxBlock[] {
  switch (node.kind) {
    case "paragraph":
      return emitParagraph(node, tokens)
    case "heading":
      return emitHeading(node, tokens)
    case "list":
      return emitList(node, tokens)
    case "listItem":
      return [paragraphFromInlineChildren(node.children, tokens)]
    case "image":
      return emitImage(node, assetMap)
    case "blockquote":
      return [paragraphFromInlineChildren(node.children, tokens, { indent: { left: 720 } })]
    case "section":
      return emitSection(node, tokens, assetMap, emitBlocks)
    case "table":
      return emitTable(node, tokens, assetMap, emitBlocks)
    case "tableRow":
      return [new Table({ rows: [emitTableRow(node, tokens, assetMap, emitBlocks)] })]
    case "tableCell":
      return [
        new Table({
          rows: [new TableRow({ children: [emitTableCell(node, tokens, assetMap, emitBlocks)] })],
        }),
      ]
    case "field":
      return emitField(node, tokens)
    case "repeatable":
      return emitRepeatable(node, tokens, assetMap, emitBlocks)
    case "repeatableItem":
      return emitRepeatableItem(node, tokens, assetMap, emitBlocks)
    case "hyperlink":
    case "text":
    case "lineBreak":
      return [new Paragraph({ children: collectInlineRuns([node], tokens) })]
  }
}
```

- [ ] **Step 12: Create emitter + asset-collector**

Create `apps/ck5-export/src/docx-emitter/emitter.ts`. Move `emitDocxFromExportTree` (`ck5-docx-emitter.ts:462-505`). Imports:

```ts
import { Document } from "docx"
import type { ExportNode } from "../export-node"
import type { ResolvedAsset } from "../asset-resolver"
import type { LayoutTokens } from "../layout-tokens"
import { mmToTwip } from "../shared/helpers/units"
import { emitBlocks } from "./block-dispatch"
import { DEFAULT_NUMBERING_REFERENCE } from "./list"
```

Export `emitDocxFromExportTree`.

Create `apps/ck5-export/src/docx-emitter/asset-collector.ts`. Move `collectImageUrls` (`ck5-docx-emitter.ts:507-565`) verbatim. No siblings imported. Imports only types from `../export-node`. Export `collectImageUrls`.

- [ ] **Step 13: Create barrel**

Create `apps/ck5-export/src/docx-emitter/index.ts`:

```ts
export { emitDocxFromExportTree } from "./emitter"
export { collectImageUrls } from "./asset-collector"
```

- [ ] **Step 14: Delete monolith + update every caller**

```bash
git rm apps/ck5-export/src/ck5-docx-emitter.ts
```

Update `apps/ck5-export/src/routes/render-docx.ts` — change the import from `../ck5-docx-emitter` to `../docx-emitter`:

```ts
import { collectImageUrls, emitDocxFromExportTree } from "../docx-emitter"
```

Update imports in these four test files — replace `from "../ck5-docx-emitter"` with `from "../docx-emitter"`:

- `apps/ck5-export/src/__tests__/field-label-round-trip.test.ts` (Task 2)
- `apps/ck5-export/src/__tests__/hyperlink-token.test.ts` (Task 3)
- `apps/ck5-export/src/__tests__/emitter-blocks.test.ts` (Task 8 — the safety-net suite; its green status after this step is the regression gate)
- Any other file the grep below surfaces.

Grep sweep:
```bash
grep -RIn "ck5-docx-emitter" apps/ck5-export/src --include="*.ts"
```

Expected after all edits: empty.

- [ ] **Step 15: typecheck + test (behavioral safety-net must stay green)**

```bash
pnpm -C apps/ck5-export typecheck
pnpm -C apps/ck5-export test
```

Both expected: exit 0. In particular, `emitter-blocks.test.ts` must pass unchanged (aside from the import path updated in Step 14) — that is the regression signal. If any block-family test fails, the split introduced a behavioral change: STOP and diff the new leaf module against the original function before committing.

Optional cycle check (if `madge` is installed globally):
```bash
npx madge --circular apps/ck5-export/src/docx-emitter
```
Expected: "No circular dependency found!"

- [ ] **Step 16: Commit**

```bash
git add -A apps/ck5-export/src
git commit -m "refactor(ck5-export): split ck5-docx-emitter.ts into per-block module

565 LOC god file → docx-emitter/ with one file per block kind. Behavior
unchanged; Task 8's emitter-blocks.test.ts stays green on the same
inputs. Restores the modularity the legacy dir had before Plan C removed
mddm-editor.

No circular imports: table.ts / section.ts / repeatable.ts receive an
EmitBlocks callback instead of importing emitBlocks from block-dispatch.
block-dispatch.ts is the single module that knows every leaf."
```

---

### Task 10: Simplify walker — drop `inlineContext` parameter

**Goal:** `walkNode` threads `inlineContext: boolean` through every recursion to decide whether whitespace-only text nodes are kept or dropped. Replace with two entrypoints dispatched by the element type itself:

- `walkInline` — called from the TRUE inline containers only: `<p>`, `<h1..h6>`, `<a>`, and the inline-mark elements `<b>/<strong>`, `<i>/<em>`, `<u>`, `<s>/<strike>`. Preserves all text nodes (including whitespace-only) so `"hello <b>world</b>"` doesn't collapse to `"helloworld"`.
- `walkBlocks` — called everywhere else, including `<blockquote>`, `<li>`, `<td>/<th>`, the `<li>` inside `mddm-repeatable`, `mddm-section-header` / `mddm-section-body`. Drops pure-whitespace text nodes between block elements. Non-whitespace text nodes ARE still kept; the emitter later coerces them to inline runs via `paragraphFromInlineChildren`.

> **Important classification reminder** (this is what the original `inlineContext=true`/`false` flag tracked): blockquote / listItem / tableCell / repeatableItem are **block** containers in the walker's eyes. The fact that their content is re-serialized as inline runs at emit time does NOT make them inline walkers. Matching the original behavior (`walkChildren(..., false, ...)`) means they dispatch through `walkBlocks`, not `walkInline`.

**Files:**
- Modify: `apps/ck5-export/src/html-to-export-tree.ts`
- Create: `apps/ck5-export/src/__tests__/walker-whitespace.test.ts` (baseline + post-refactor coverage)

**Acceptance Criteria:**
- [ ] `walkNode`, `walkChildren`, and the `inlineContext` parameter are gone.
- [ ] Two new entrypoints `walkInline` / `walkBlocks` exist; each TRUE inline container (p, h1-6, a, b, strong, i, em, u, s, strike) dispatches to `walkInline`; every other container dispatches to `walkBlocks`.
- [ ] `buildTableCell` explicitly updated: its old call to `walkChildren(el, false, [])` is replaced with `walkBlocks(el, [])`. (Missing this is the most likely regression.)
- [ ] Context-preserving wrappers `SPAN.restricted-editing-exception` and `DIV.mddm-rich-block` each recurse via `walkInline`/`walkBlocks` depending on whether they were reached from an inline or block caller — verified by the new wrapper-parity cases in `walker-whitespace.test.ts`.
- [ ] `walker-whitespace.test.ts` passes BEFORE the refactor against the current walker (baseline) and AFTER the refactor against the new walker (behavior parity).
- [ ] All other existing tests in `apps/ck5-export/src/__tests__/` pass unchanged.

**Verify:** `pnpm -C apps/ck5-export test` → exit 0.

**Steps:**

- [ ] **Step 0: Write baseline whitespace-preservation tests (BEFORE refactor)**

Create `apps/ck5-export/src/__tests__/walker-whitespace.test.ts`. These tests are written against the CURRENT (pre-refactor) walker and must pass as-is. They're the behavior contract the refactor has to preserve:

```ts
import { describe, expect, it } from "vitest"
import { htmlToExportTree } from "../html-to-export-tree"

describe("walker whitespace — inline containers keep spaces", () => {
  it("paragraph preserves whitespace between inline elements", () => {
    const tree = htmlToExportTree("<p>hello <b>bold</b> world</p>")
    expect(tree).toHaveLength(1)
    const p = tree[0]
    if (p.kind !== "paragraph") throw new Error("expected paragraph")
    // Expect text runs: "hello ", "bold" (bold mark), " world"
    const texts = p.children.filter((c) => c.kind === "text")
    expect(texts.length).toBeGreaterThanOrEqual(3)
    const values = texts.map((t) => (t.kind === "text" ? t.value : ""))
    expect(values).toContain("hello ")
    expect(values).toContain("bold")
    expect(values).toContain(" world")
  })

  it("heading preserves whitespace between inline elements", () => {
    const tree = htmlToExportTree("<h2>  Title  <em>note</em></h2>")
    expect(tree).toHaveLength(1)
    const h = tree[0]
    if (h.kind !== "heading") throw new Error("expected heading")
    const values = h.children
      .filter((c) => c.kind === "text")
      .map((t) => (t.kind === "text" ? t.value : ""))
    // Leading/trailing whitespace around text inside a heading is kept.
    expect(values.some((v) => v.startsWith("  Title"))).toBe(true)
  })

  it("hyperlink keeps surrounding whitespace children", () => {
    const tree = htmlToExportTree('<p>see <a href="x">here</a> please</p>')
    const p = tree[0]
    if (p.kind !== "paragraph") throw new Error("expected paragraph")
    const values = p.children
      .filter((c) => c.kind === "text")
      .map((t) => (t.kind === "text" ? t.value : ""))
    expect(values).toContain("see ")
    expect(values).toContain(" please")
  })
})

describe("walker whitespace — block containers drop pure-whitespace text", () => {
  it("list item drops pure-whitespace text between block children", () => {
    const tree = htmlToExportTree("<ul><li>  <p>inner</p>  </li></ul>")
    const list = tree[0]
    if (list.kind !== "list") throw new Error("expected list")
    const li = list.items[0]
    const textRuns = li.children.filter((c) => c.kind === "text")
    // Pure-whitespace text nodes around the inner <p> must be dropped.
    expect(textRuns).toHaveLength(0)
  })

  it("list item keeps non-whitespace text directly inside", () => {
    const tree = htmlToExportTree("<ul><li>hello</li></ul>")
    const list = tree[0]
    if (list.kind !== "list") throw new Error("expected list")
    const li = list.items[0]
    const textRuns = li.children.filter((c) => c.kind === "text")
    expect(textRuns).toHaveLength(1)
    if (textRuns[0].kind === "text") expect(textRuns[0].value).toBe("hello")
  })

  it("blockquote drops pure-whitespace between nested paragraphs", () => {
    const tree = htmlToExportTree("<blockquote>  <p>one</p>  <p>two</p>  </blockquote>")
    const bq = tree[0]
    if (bq.kind !== "blockquote") throw new Error("expected blockquote")
    const paragraphs = bq.children.filter((c) => c.kind === "paragraph")
    const textBetween = bq.children.filter((c) => c.kind === "text")
    expect(paragraphs).toHaveLength(2)
    expect(textBetween).toHaveLength(0)
  })

  it("table cell drops pure-whitespace text between block children", () => {
    const html =
      '<figure class="table"><table><tr><td>  <p>cell</p>  </td></tr></table></figure>'
    const tree = htmlToExportTree(html)
    const table = tree[0]
    if (table.kind !== "table") throw new Error("expected table")
    const cell = table.rows[0].cells[0]
    const texts = cell.children.filter((c) => c.kind === "text")
    expect(texts).toHaveLength(0)
    const paragraphs = cell.children.filter((c) => c.kind === "paragraph")
    expect(paragraphs).toHaveLength(1)
  })

  it("repeatable item drops pure-whitespace between nested blocks", () => {
    const html =
      '<ol class="mddm-repeatable"><li>  <p>one</p>  </li></ol>'
    const tree = htmlToExportTree(html)
    const rep = tree[0]
    if (rep.kind !== "repeatable") throw new Error("expected repeatable")
    const item = rep.items[0]
    const texts = item.children.filter((c) => c.kind === "text")
    expect(texts).toHaveLength(0)
  })

  it("section body drops pure-whitespace between nested blocks", () => {
    const html =
      '<section class="mddm-section">' +
      '<div class="mddm-section-body">  <p>body</p>  </div>' +
      "</section>"
    const tree = htmlToExportTree(html)
    const s = tree[0]
    if (s.kind !== "section") throw new Error("expected section")
    const texts = s.body.filter((c) => c.kind === "text")
    expect(texts).toHaveLength(0)
  })
})

describe("walker whitespace — context-preserving wrappers", () => {
  it("SPAN.restricted-editing-exception preserves whitespace inside a paragraph (inline context)", () => {
    const tree = htmlToExportTree(
      '<p>before <span class="restricted-editing-exception">mid </span>after</p>',
    )
    const p = tree[0]
    if (p.kind !== "paragraph") throw new Error("expected paragraph")
    const values = p.children
      .filter((c) => c.kind === "text")
      .map((t) => (t.kind === "text" ? t.value : ""))
    // The wrapper is a pure pass-through; "before ", "mid ", and "after"
    // must all be present with their inline whitespace intact.
    expect(values).toContain("before ")
    expect(values).toContain("mid ")
    expect(values).toContain("after")
  })

  it("DIV.mddm-rich-block drops pure-whitespace between nested blocks (block context)", () => {
    const tree = htmlToExportTree(
      '<blockquote>  <div class="mddm-rich-block">  <p>one</p>  <p>two</p>  </div>  </blockquote>',
    )
    const bq = tree[0]
    if (bq.kind !== "blockquote") throw new Error("expected blockquote")
    // The wrapper should be flattened and behave like walkBlocks:
    // two paragraphs, zero whitespace-only text runs.
    const paragraphs = bq.children.filter((c) => c.kind === "paragraph")
    const textRuns = bq.children.filter((c) => c.kind === "text")
    expect(paragraphs).toHaveLength(2)
    expect(textRuns).toHaveLength(0)
  })
})
```

Run before refactoring:
```bash
pnpm -C apps/ck5-export vitest run src/__tests__/walker-whitespace.test.ts
```

Expected: **all pass** against the current walker. If any fail, the test expectation is wrong — fix the test to match current behavior before proceeding. These are the invariants the refactor must preserve.

- [ ] **Step 1: Read the current walker and map call sites**

```bash
cat apps/ck5-export/src/html-to-export-tree.ts
```

Every current call of `walkChildren(..., inlineContext, ...)` maps to exactly one of the two new entrypoints, based on the flag's value:

| Current call (in `html-to-export-tree.ts`) | Flag value today | New call |
|---|---|---|
| `htmlToExportTree` → `walkChildren(document.body, false, [])` | false | `walkBlocks(body, [])` |
| `DIV.mddm-rich-block` → `walkChildren(el, inlineContext, marks)` | pass-through | `inInline ? walkInline : walkBlocks` (in `walkElement`) |
| `SPAN.restricted-editing-exception` → same pass-through | pass-through | same |
| `STRONG/B`, `EM/I`, `U`, `S/STRIKE` → `walkChildren(el, true, ...)` | true | `walkInline(el, ...)` |
| `SECTION.mddm-section` (header + body) → `walkChildren(container, false, [])` | false | `walkBlocks(container, [])` |
| `OL.mddm-repeatable` item → `walkChildren(li, false, [])` | false | `walkBlocks(li, [])` |
| `H1..H6` → `walkChildren(el, true, marks)` | true | `walkInline(el, marks)` |
| `P` → `walkChildren(el, true, marks)` | true | `walkInline(el, marks)` |
| `UL` / non-repeatable `OL` item → `walkChildren(li, false, marks)` | false | `walkBlocks(li, marks)` |
| `A` → `walkChildren(el, true, marks)` | true | `walkInline(el, marks)` |
| `BLOCKQUOTE` → `walkChildren(el, false, marks)` | false | `walkBlocks(el, marks)` |
| fall-through at bottom of `walkNode` → `walkChildren(el, inlineContext, marks)` | pass-through | `inInline ? walkInline : walkBlocks` |
| **`buildTableCell` → `walkChildren(el, false, [])` (line ~240)** | **false** | **`walkBlocks(el, [])`** |

The last row is the easiest to miss because `buildTableCell` lives outside `walkNode` and is reached via `collectTableRows → collectCells → buildTableCell`. Task acceptance requires this change to be visible in the diff.

- [ ] **Step 2: Refactor**

Replace the file body with this structure. Important: `buildTableCell` IS updated to call `walkBlocks(el, [])`. The other helpers (`collectTableRows`, `collectCells`, `buildText`, `addMark`, `isWhitespaceOnly`, `parseOptionalNumber`, `findDirectChildByClass`) do not reference `inlineContext` and are copied verbatim.

```ts
import { parseHTML } from "linkedom"
import type { ExportNode, Field, Paragraph, TableCell, TableRow, Text } from "./export-node"

type TextMark = NonNullable<Text["marks"]>[number]

const ELEMENT_NODE = 1
const TEXT_NODE = 3

export function htmlToExportTree(html: string): ExportNode[] {
  const { document } = parseHTML(`<!DOCTYPE html><html><body>${html}</body></html>`)
  return walkBlocks(document.body, [])
}

// Inline walker: keeps all text nodes (including whitespace-only) so
// "hello <b>world</b>" doesn't collapse to "helloworld".
function walkInline(parent: ParentNode, marks: TextMark[]): ExportNode[] {
  const out: ExportNode[] = []
  for (const child of Array.from(parent.childNodes)) {
    out.push(...walkNodeInline(child, marks))
  }
  return out
}

// Block walker: drops pure-whitespace text nodes between block elements.
function walkBlocks(parent: ParentNode, marks: TextMark[]): ExportNode[] {
  const out: ExportNode[] = []
  for (const child of Array.from(parent.childNodes)) {
    out.push(...walkNodeBlock(child, marks))
  }
  return out
}

function walkNodeInline(node: Node, marks: TextMark[]): ExportNode[] {
  if (node.nodeType === TEXT_NODE) {
    return [buildText(node.textContent ?? "", marks)]
  }
  return walkElement(node, true, marks)
}

function walkNodeBlock(node: Node, marks: TextMark[]): ExportNode[] {
  if (node.nodeType === TEXT_NODE) {
    const value = node.textContent ?? ""
    if (isWhitespaceOnly(value)) {
      return []
    }
    return [buildText(value, marks)]
  }
  return walkElement(node, false, marks)
}

function walkElement(node: Node, inInline: boolean, marks: TextMark[]): ExportNode[] {
  if (node.nodeType !== ELEMENT_NODE) {
    return []
  }

  const el = node as HTMLElement
  const tagName = el.tagName.toUpperCase()

  // Unwrap wrappers — recurse in the same inline/block context the caller was in.
  if (tagName === "DIV" && el.classList.contains("mddm-rich-block")) {
    return inInline ? walkInline(el, marks) : walkBlocks(el, marks)
  }
  if (tagName === "SPAN" && el.classList.contains("restricted-editing-exception")) {
    return inInline ? walkInline(el, marks) : walkBlocks(el, marks)
  }

  // Inline marks: always switch to inline and add mark.
  if (tagName === "STRONG" || tagName === "B") return walkInline(el, addMark(marks, "bold"))
  if (tagName === "EM" || tagName === "I") return walkInline(el, addMark(marks, "italic"))
  if (tagName === "U") return walkInline(el, addMark(marks, "underline"))
  if (tagName === "S" || tagName === "STRIKE") return walkInline(el, addMark(marks, "strike"))

  // Block: section.
  if (tagName === "SECTION" && el.classList.contains("mddm-section")) {
    const headerContainer = findDirectChildByClass(el, "mddm-section-header")
    const bodyContainer = findDirectChildByClass(el, "mddm-section-body")
    const header = headerContainer ? walkBlocks(headerContainer, []) : undefined
    const body = bodyContainer ? walkBlocks(bodyContainer, []) : []
    return [
      {
        kind: "section",
        variant: (el.getAttribute("data-variant") || "plain") as "solid" | "bordered" | "plain",
        header: header && header.length > 0 ? header : undefined,
        body,
      },
    ]
  }

  // Block: repeatable (OL with mddm-repeatable).
  if (tagName === "OL" && el.classList.contains("mddm-repeatable")) {
    const items = Array.from(el.children)
      .filter((child) => child.tagName.toUpperCase() === "LI")
      .map((li) => ({ kind: "repeatableItem" as const, children: walkBlocks(li, []) }))
    return [{ kind: "repeatable", items }]
  }

  // Block: table.
  if (tagName === "FIGURE" && el.classList.contains("table")) {
    const tableEl = el.querySelector("table")
    const rows = tableEl ? collectTableRows(tableEl) : []
    return [
      {
        kind: "table",
        variant: (el.getAttribute("data-variant") || "fixed") as "fixed" | "dynamic",
        rows,
      },
    ]
  }

  // Inline: field.
  if (tagName === "SPAN" && el.classList.contains("mddm-field")) {
    const label = el.getAttribute("data-field-label") ?? undefined
    return [
      {
        kind: "field",
        id: el.getAttribute("data-field-id") ?? "",
        label: label && label.length > 0 ? label : undefined,
        fieldType: (el.getAttribute("data-field-type") ?? "text") as Field["fieldType"],
        value: el.textContent ?? "",
      },
    ]
  }

  // Block: heading.
  if (/^H[1-6]$/.test(tagName)) {
    return [
      {
        kind: "heading",
        level: Number.parseInt(tagName[1] ?? "1", 10) as 1 | 2 | 3 | 4 | 5 | 6,
        children: walkInline(el, marks),
      },
    ]
  }

  // Block: paragraph.
  if (tagName === "P") {
    const align = (el.style.textAlign || undefined) as Paragraph["align"]
    return [{ kind: "paragraph", align, children: walkInline(el, marks) }]
  }

  // Block: list.
  if (tagName === "UL" || (tagName === "OL" && !el.classList.contains("mddm-repeatable"))) {
    const items = Array.from(el.children)
      .filter((child) => child.tagName.toUpperCase() === "LI")
      .map((li) => ({ kind: "listItem" as const, children: walkBlocks(li, marks) }))
    return [{ kind: "list", ordered: tagName === "OL", items }]
  }

  // Inline: image.
  if (tagName === "IMG") {
    const imageEl = el as HTMLImageElement
    const width = imageEl.width || parseOptionalNumber(el.getAttribute("width"))
    const height = imageEl.height || parseOptionalNumber(el.getAttribute("height"))
    return [
      {
        kind: "image",
        src: el.getAttribute("src") ?? "",
        alt: el.getAttribute("alt") || undefined,
        width,
        height,
      },
    ]
  }

  // Inline: hyperlink.
  if (tagName === "A") {
    return [
      {
        kind: "hyperlink",
        href: (el as HTMLAnchorElement).href,
        children: walkInline(el, marks),
      },
    ]
  }

  // Inline: line break.
  if (tagName === "BR") return [{ kind: "lineBreak" }]

  // Block: blockquote.
  if (tagName === "BLOCKQUOTE") {
    return [{ kind: "blockquote", children: walkBlocks(el, marks) }]
  }

  // Fall through: recurse preserving context.
  return inInline ? walkInline(el, marks) : walkBlocks(el, marks)
}

// --- helpers ---
// collectTableRows, collectCells, buildText, addMark,
// isWhitespaceOnly, parseOptionalNumber, findDirectChildByClass
// are copied verbatim (they never referenced the old inlineContext flag).
//
// buildTableCell IS updated: its body called walkChildren(el, false, [])
// in the old code, which threaded the boolean. Replace that call with
// walkBlocks(el, []) so cells use the block entrypoint (matches the
// previous `false` value and drops whitespace between <p> children).
function buildTableCell(el: Element): TableCell {
  const colspan = parseOptionalNumber(el.getAttribute("colspan"))
  const rowspan = parseOptionalNumber(el.getAttribute("rowspan"))
  return {
    kind: "tableCell",
    isHeader: el.tagName.toUpperCase() === "TH",
    children: walkBlocks(el, []),
    colspan,
    rowspan,
  }
}
```

- [ ] **Step 3: typecheck + test**

```bash
pnpm -C apps/ck5-export typecheck
pnpm -C apps/ck5-export test
```

Both expected: exit 0. Behavior is preserved: `walkBlocks` drops whitespace, `walkInline` keeps it, marks thread only through inline.

- [ ] **Step 4: Commit**

```bash
git add apps/ck5-export/src/html-to-export-tree.ts
git commit -m "refactor(ck5-export): drop inlineContext parameter from walker

Replace the boolean flag with two entrypoints (walkInline / walkBlocks)
dispatched by the element itself. Makes whitespace rules local to each
block-handling branch instead of threaded top-down."
```

---

## Phase 4 — Housekeeping + Final Verification

### Task 11: Remove stale docgen comment in Go

**Goal:** `service_template_lifecycle.go:677` has a comment mentioning "the docgen service expects" despite Docgen having been deleted in the Plan C destructive PR. Remove the sentence.

**Files:**
- Modify: `internal/modules/documents/application/service_template_lifecycle.go`

**Acceptance Criteria:**
- [ ] The phrase "docgen service" no longer appears in the file.
- [ ] `go build ./...` clean.

**Verify:**
```
grep -n "docgen" internal/modules/documents/application/service_template_lifecycle.go
```
Expected: empty.

**Steps:**

- [ ] **Step 1: Locate context**

```bash
sed -n '670,685p' internal/modules/documents/application/service_template_lifecycle.go
```

- [ ] **Step 2: Edit**

Open the file in your editor, remove or rephrase the comment line at 677. If it was explaining the envelope shape, replace with a neutral phrasing:

```
Before:  // The Envelope the docgen service expects is exactly the BlocksJSON stored
After:   // The Envelope is exactly the BlocksJSON stored (see template_drafts.blocks_json).
```

- [ ] **Step 3: Verify + commit**

```bash
go build ./...
grep -n "docgen" internal/modules/documents/application/service_template_lifecycle.go
```

Expected: `go build` exit 0, grep empty.

```bash
git add internal/modules/documents/application/service_template_lifecycle.go
git commit -m "chore(docs): remove stale docgen reference in template lifecycle service

The docgen service was deleted in the Plan C destructive PR; the
comment referencing 'the docgen service expects' was misleading."
```

---

### Task 12: Append amendments to Plan C spec

**Goal:** Record three intentional deviations from the approved spec so future readers don't chase ghosts: (a) migration column `draft_status` instead of `status`, (b) IR-goldens retirement skipped / deferred, (c) post-audit dead-code sweep.

**Files:**
- Modify (append section): `docs/superpowers/specs/2026-04-16-ck5-plan-c-production-readiness.md`

**Acceptance Criteria:**
- [ ] New `## Amendments (post-ship)` section at end of spec file.
- [ ] Contains three dated entries with rationale.

**Verify:**
```
grep -n "## Amendments" docs/superpowers/specs/2026-04-16-ck5-plan-c-production-readiness.md
```
Expected: one match.

**Steps:**

- [ ] **Step 1: Append the amendments block**

Append this block to the end of `docs/superpowers/specs/2026-04-16-ck5-plan-c-production-readiness.md`:

```markdown

---

## Amendments (post-ship)

### 2026-04-16 — Migration column name: `draft_status`

Spec §PR2 specified column name `status`. Shipped migration `0077_add_template_publish_state.sql` uses `draft_status` to avoid a naming collision with the existing `template_versions.status` column, which tracks a different state machine (draft / published / deprecated lifecycle vs the review workflow). Functional behavior identical.

### 2026-04-16 — IR golden retirement deferred, then obsoleted

Spec §Testing step 3 planned a dual-path golden-parity migration: run both the IR → docx and the HTML → docx paths against equivalent goldens, confirm parity, then retire the IR-only goldens. The dual-path phase was skipped during implementation; both paths shipped together without a parity check. The post-audit sweep (see below) deleted the IR path entirely, making the retirement moot. Any IR-only golden fixtures were removed as part of the `docx-emitter/` directory deletion and are not recoverable without restoring the legacy pipeline.

### 2026-04-16 — Post-audit dead-code sweep

An outsider audit found `apps/ck5-export/src/{docx-emitter,codecs,layout-interpreter}/` shipped but unreachable from any route. The `layout-ir/` directory framed design tokens as a persistent IR in violation of the "IR is ephemeral" principle. All three directories + `layout-ir/` were deleted in plan `docs/superpowers/plans/2026-04-16-ck5-plan-c-post-audit-fixes.md`. The surviving `ck5-docx-emitter.ts` god file was split into `docx-emitter/` (block-per-file) and `layout-ir/` was collapsed to a single `layout-tokens.ts`. Correctness bugs (missing block-exception button, field label not rendered, hyperlink color hardcoded) were fixed at the same time.
```

- [ ] **Step 2: Commit**

```bash
git add docs/superpowers/specs/2026-04-16-ck5-plan-c-production-readiness.md
git commit -m "docs(ck5): append Plan C post-ship amendments

Record three deviations from the approved spec: draft_status column
rename, deferred-then-obsolete IR golden retirement, and the post-audit
dead-code sweep."
```

---

### Task 13: Run every verification gate end-to-end

**Goal:** Close the gates that Plan C spec §PR2 required. Run the full matrix, fix any regression introduced by this plan's refactor, and record command outputs in the final commit.

**Commands to run (all from worktree root):**

| Gate | Command | Expected |
|---|---|---|
| ck5-export typecheck | `pnpm -C apps/ck5-export typecheck` | exit 0 |
| ck5-export build | `pnpm -C apps/ck5-export build` | exit 0 |
| ck5-export tests | `pnpm -C apps/ck5-export test` | all pass |
| Frontend CK5 tests | `pnpm -C frontend/apps/web vitest run src/features/documents/ck5` | all pass |
| Frontend full tests | `pnpm -C frontend/apps/web vitest run` | all pass |
| Frontend build | `pnpm -C frontend/apps/web build` | exit 0 |
| Bundle @blocknote check | `grep -ri "@blocknote" frontend/apps/web/dist \|\| true` | empty |
| Bundle mddm-editor check | `grep -ri "mddm-editor" frontend/apps/web/dist \|\| true` | empty |
| Go build | `go build ./...` | exit 0 |
| Go tests | `go test ./internal/...` | all pass |
| No `layout-ir` refs | `grep -R "layout-ir" apps/ck5-export/src` | empty |
| No dead dir refs | `grep -RE "codecs/\|layout-interpreter/" apps/ck5-export/src --include='*.ts'` | empty |

**Acceptance Criteria:**
- [ ] Every row in the table above passes its expected result.
- [ ] If any row fails, the failure is diagnosed and fixed inline (this task owns regression repair); re-run the entire table until green.

**Verify:** Each command in the table.

**Steps:**

- [ ] **Step 1: Run every gate, record output**

```bash
pnpm -C apps/ck5-export typecheck
pnpm -C apps/ck5-export build
pnpm -C apps/ck5-export test
pnpm -C frontend/apps/web vitest run src/features/documents/ck5
pnpm -C frontend/apps/web vitest run
pnpm -C frontend/apps/web build
grep -ri "@blocknote" frontend/apps/web/dist || echo "clean: no @blocknote in bundle"
grep -ri "mddm-editor" frontend/apps/web/dist || echo "clean: no mddm-editor in bundle"
go build ./...
go test ./internal/...
grep -R "layout-ir" apps/ck5-export/src || echo "clean: no layout-ir refs"
grep -RE "codecs/|layout-interpreter/" apps/ck5-export/src --include='*.ts' || echo "clean: no dead dir refs"
```

- [ ] **Step 2: Fix any regression**

If any test fails, build errors, or typecheck fails: diagnose, repair, re-run the FULL matrix (not just the failing gate). Repeat until every row is green. Do NOT commit until the entire matrix is green — partial greens are not acceptable because `typecheck` passing while `build` fails means the package can't emit.

- [ ] **Step 3: Commit final verification marker**

```bash
git commit --allow-empty -m "test(ck5): verify post-audit sweep — all gates green

- apps/ck5-export typecheck + build + tests
- frontend/apps/web vitest (CK5 + full) + build
- Go build + tests
- bundle clean of @blocknote + mddm-editor
- zero refs to deleted dirs (layout-ir, codecs, layout-interpreter)"
```

---

## Self-Review

### Spec coverage

| Audit finding | Covered by |
|---|---|
| Toolbar missing `restrictedEditingExceptionBlock` | Task 1 |
| Field label rendered as raw id | Task 2 |
| Hardcoded hyperlink color `0563C1` | Task 3 |
| Dead `docx-emitter/` dir shipped | Task 4 |
| Dead `codecs/` + `layout-interpreter/` dirs shipped | Task 5 |
| Orphaned `shared/adapter.ts` | Task 6 |
| `layout-ir/` framed as persistent IR (spec violation) | Task 7 |
| Legacy emitter tests deleted in Task 4 → god file has zero direct unit coverage before the split | Task 8 |
| `ck5-docx-emitter.ts` = 565-LOC god file | Task 9 |
| Walker `inlineContext` boolean threading | Task 10 |
| Stale `docgen` comment in Go | Task 11 |
| Undocumented `draft_status` rename + skipped IR-golden retirement | Task 12 |
| Spec gates §PR2 pre-delete (`rtk pnpm build` bundle check, full vitest) never run | Task 13 |

Every audit item maps to a task.

### Placeholder scan

No "TBD", "similar to Task N", or bare "add appropriate X" entries. Every code block is full and self-contained. Every command has expected output.

### Type consistency

- `Field.label` introduced in Task 2, used in Tasks 2, 8, 9 — consistent signature.
- `LayoutTokens.theme.hyperlink` introduced in Task 3, carried through Task 7 (flatten), Task 9 (split), Task 10 (no impact).
- `walkInline` / `walkBlocks` / `walkElement` signatures in Task 10 match their call sites.
- `DocxBlock` type alias from Task 9 Step 1 used in Steps 8 (table), 9 (repeatable), 10 (section), 11 (dispatch).

### Commit sequence invariant

After every commit in every task, `pnpm -C apps/ck5-export typecheck && pnpm -C apps/ck5-export test && go build ./... && go test ./internal/...` must pass. Each task ends with a green check. Task 13 is the final belt-and-braces gate.

---

## Task Graph / Dependencies

```
1 ──┐
2 ──┼─ (independent bug fixes, any order)
3 ──┘

4 (depends on: 2, 3 — edits to ck5-docx-emitter.ts land before delete of parallel dir)
5 (depends on: 4 — proves consumers already gone)
6 (depends on: 5 — last adapter consumer gone)
7 (depends on: 5 — ComponentRules consumers gone)

8 (depends on: 2, 3, 7 — seed direct emitter coverage against the surviving ck5-docx-emitter.ts after bugs fixed and layout-tokens flattened)
9 (depends on: 4, 7, 8 — docx-emitter dir-name free, layout-tokens import target exists, behavioral tests in place to catch regressions)

10 (depends on: 2 — walker already touched by Task 2; 10 rewrites it cleanly)

11 (independent, can run any time)
12 (depends on: 9, 10 for accuracy of amendment text)
13 (depends on: all above)
```

Recommended linear order: 1 → 2 → 3 → 4 → 5 → 6 → 7 → 8 → 9 → 10 → 11 → 12 → 13.
