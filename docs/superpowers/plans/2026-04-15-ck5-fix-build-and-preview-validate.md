# CK5 Fix Build + Preview Validate Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers-extended-cc:subagent-driven-development (recommended) or superpowers-extended-cc:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking. **All task prompts to subagents use `/caveman` format (drop articles/filler) for token efficiency. And Use CODEX GPT5.3-CODEX Model Medium Reasoning for code implementation**

**Goal:** Fix all CK5 regressions introduced by Plan A so `pnpm tsc --noEmit` and `pnpm build` pass, runtime warnings are resolved, and Author + Fill editors load cleanly in the browser preview for a full frontend/backend smoke check.

**Architecture:** Tree-shake-safe fixes across 3 layers — (1) rename CK5 v48 renamed types (`Schema→ModelSchema`, `Writer→ModelWriter`, `Element→ModelElement`) at every import site in `frontend/apps/web/src/features/documents/ck5/`; (2) scrub invalid type casts and deprecated matcher APIs from plugin code; (3) reconcile toolbar config so every toolbar string has a registered command in the editor mode that loads it. Pre-existing Tiptap type errors revealed by lockfile regen are triaged separately (out of scope unless blocking build). Final verification: Vite preview on `http://localhost:5173` with `AuthorEditor` + `FillEditor` rendered in a test harness route, exercised via `preview_click` + `preview_snapshot`.

**Tech Stack:** `ckeditor5@48.0.0`, `@ckeditor/ckeditor5-react@^11.1.2`, TypeScript 5.x, Vite, Vitest, Playwright, React 18.

**Working directory:** `C:/Users/leandro.theodoro.MN-NTB-LEANDROT/Documents/MetalDocs-ck5` (worktree on branch `migrate/ck5-frontend-plan-a`).

---

## File Structure

All edits inside the existing `frontend/apps/web/src/features/documents/ck5/` tree. No new files except optional preview harness route.

```
frontend/apps/web/src/features/documents/ck5/
├── plugins/
│   ├── MddmFieldPlugin/
│   │   ├── schema.ts                      — rename Schema → ModelSchema
│   │   └── converters.ts                  — unchanged (imports already correct)
│   ├── MddmSectionPlugin/
│   │   ├── schema.ts                      — rename Schema → ModelSchema
│   │   ├── postFixer.ts                   — rename Element/Writer → ModelElement/ModelWriter
│   │   └── __tests__/postFixer.test.ts    — fix unsafe ModelNode[] cast
│   ├── MddmRepeatablePlugin/
│   │   ├── schema.ts                      — rename Schema → ModelSchema
│   │   ├── converters.ts                  — rename Element as ModelElement (already aliased, verify)
│   │   └── commands/AddRepeatableItemCommand.ts — verify ModelElement usage
│   ├── MddmDataTablePlugin/
│   │   ├── MddmTableLockPlugin.ts         — verify Model* imports
│   │   ├── MddmTableVariantPlugin.ts      — verify Model* imports
│   │   ├── perCellExceptionWalker.ts      — verify Model* imports
│   │   └── __tests__/lock.test.ts         — fix any stale imports
│   └── MddmRichBlockPlugin/
│       ├── schema.ts                      — rename Schema → ModelSchema
│       └── __tests__/plugin.test.ts       — fix `trim: false as const` → `trim: 'none'`
├── config/
│   └── toolbars.ts                        — split into AUTHOR/FILL with correct command names or gate
└── plugins/*/converters.ts                — audit + replace deprecated regex class matcher with object form
```

File separation rationale: each fix is co-located with the code it touches. No new abstractions. TDD: run failing `tsc` → patch → rerun → green. No speculative work.

---

## Conventions

- **Commits:** Conventional commits, scope `ck5`. Example: `fix(ck5): rename Schema to ModelSchema for v48`.
- **Verification:** Every task ends with `rtk pnpm tsc --noEmit` scoped to CK5 files + `rtk pnpm vitest run src/features/documents/ck5` passing.
- **Caveman execution:** All subagent task prompts use terse caveman phrasing. Code/commits stay normal.
- **No scope creep:** Fix only what blocks build or warns at runtime. Do not refactor working code. BlockNote/Tiptap errors out of scope unless confirmed regression.

---

## Phase 0 — Confirm baseline

### Task 0: Snapshot current failure set

**Goal:** Record exact TS error count and file list so we can prove each fix reduces the count.

**Files:** None. Observation only.

**Acceptance Criteria:**
- [ ] Full `tsc --noEmit` output saved to `.tmp/baseline-tsc-errors.txt` in worktree
- [ ] CK5-only error count extracted to `.tmp/baseline-ck5-count.txt`
- [ ] Non-CK5 (Tiptap/BlockNote) error count recorded separately

**Verify:** `wc -l .tmp/baseline-ck5-count.txt` → non-zero

**Steps:**

- [ ] **Step 1: Capture full baseline**

Run (from worktree root):
```bash
cd "C:/Users/leandro.theodoro.MN-NTB-LEANDROT/Documents/MetalDocs-ck5/frontend/apps/web"
mkdir -p ../../.tmp
rtk pnpm tsc --noEmit > ../../.tmp/baseline-tsc-errors.txt 2>&1 || true
grep "error TS" ../../.tmp/baseline-tsc-errors.txt | wc -l > ../../.tmp/baseline-total-count.txt
grep "error TS" ../../.tmp/baseline-tsc-errors.txt | grep "src/features/documents/ck5" | wc -l > ../../.tmp/baseline-ck5-count.txt
```
Expected: Both count files non-empty. CK5 count ≈ 29, total ≈ 59.

- [ ] **Step 2: No commit** — `.tmp/` gitignored.

---

## Phase 1 — Fix CK5 v48 type rename regressions

### Task 1: Rename `Schema` → `ModelSchema` in all plugin schema files

**Goal:** Every `import type { Schema }` becomes `import type { ModelSchema }` with matching parameter type.

**Files:**
- Modify: `frontend/apps/web/src/features/documents/ck5/plugins/MddmFieldPlugin/schema.ts`
- Modify: `frontend/apps/web/src/features/documents/ck5/plugins/MddmSectionPlugin/schema.ts`
- Modify: `frontend/apps/web/src/features/documents/ck5/plugins/MddmRepeatablePlugin/schema.ts`
- Modify: `frontend/apps/web/src/features/documents/ck5/plugins/MddmRichBlockPlugin/schema.ts`

**Acceptance Criteria:**
- [ ] Every `Schema` type reference (import + param) replaced with `ModelSchema`
- [ ] `rtk grep "type { Schema }" frontend/apps/web/src/features/documents/ck5` returns zero matches
- [ ] CK5 `Schema` TS errors eliminated

**Verify:** `cd frontend/apps/web && rtk pnpm tsc --noEmit 2>&1 | grep "no exported member 'Schema'"` → zero lines

**Steps:**

- [ ] **Step 1: Patch `MddmFieldPlugin/schema.ts`**

```ts
import type { ModelSchema } from 'ckeditor5';

export function registerFieldSchema(schema: ModelSchema): void {
  schema.register('mddmField', {
    inheritAllFrom: '$inlineObject',
    allowAttributes: ['fieldId', 'fieldType', 'fieldLabel', 'fieldRequired', 'fieldValue'],
  });
}
```

- [ ] **Step 2: Patch `MddmSectionPlugin/schema.ts`**

```ts
import type { ModelSchema } from 'ckeditor5';

export function registerSectionSchema(schema: ModelSchema): void {
  schema.register('mddmSection', {
    inheritAllFrom: '$blockObject',
    allowChildren: ['mddmSectionHeader', 'mddmSectionBody'],
    allowAttributes: ['sectionId', 'variant'],
  });

  schema.register('mddmSectionHeader', {
    isLimit: true,
    allowIn: 'mddmSection',
    allowContentOf: '$block',
  });

  schema.register('mddmSectionBody', {
    isLimit: true,
    allowIn: 'mddmSection',
    allowContentOf: '$root',
  });
}
```

- [ ] **Step 3: Patch `MddmRepeatablePlugin/schema.ts`**

```ts
import type { ModelSchema } from 'ckeditor5';

export function registerRepeatableSchema(schema: ModelSchema): void {
  schema.register('mddmRepeatable', {
    inheritAllFrom: '$blockObject',
    allowChildren: ['mddmRepeatableItem'],
    allowAttributes: ['repeatableId', 'label', 'min', 'max', 'numberingStyle'],
  });

  schema.register('mddmRepeatableItem', {
    inheritAllFrom: '$container',
    allowIn: 'mddmRepeatable',
    isLimit: true,
  });

  schema.addChildCheck((context, def) => {
    if (context.endsWith('mddmRepeatableItem') && def.name === 'mddmRepeatable') {
      return false;
    }
    return undefined;
  });
}
```

- [ ] **Step 4: Patch `MddmRichBlockPlugin/schema.ts`**

```ts
import type { ModelSchema } from 'ckeditor5';

export function registerRichBlockSchema(schema: ModelSchema): void {
  schema.register('mddmRichBlock', {
    inheritAllFrom: '$container',
    allowIn: ['$root'],
  });
}
```

- [ ] **Step 5: Verify `Schema` gone from CK5**

Run:
```bash
cd frontend/apps/web
rtk grep "type { Schema }" src/features/documents/ck5
rtk pnpm tsc --noEmit 2>&1 | grep "no exported member 'Schema'"
```
Expected: Both commands zero output.

- [ ] **Step 6: Run CK5 tests**

```bash
rtk pnpm vitest run src/features/documents/ck5 --reporter=dot
```
Expected: 78/78 pass (same as baseline).

- [ ] **Step 7: Commit**

```bash
cd ../..
rtk git add frontend/apps/web/src/features/documents/ck5/plugins/*/schema.ts
rtk git commit -m "fix(ck5): rename Schema to ModelSchema for v48 type compatibility"
```

---

### Task 2: Rename `Writer` → `ModelWriter`, `Element` → `ModelElement` in postFixer + converters

**Goal:** All model-layer type imports use v48 `Model*` prefix. Fix unsafe cast in postFixer test.

**Files:**
- Modify: `frontend/apps/web/src/features/documents/ck5/plugins/MddmSectionPlugin/postFixer.ts`
- Modify: `frontend/apps/web/src/features/documents/ck5/plugins/MddmSectionPlugin/__tests__/postFixer.test.ts`
- Audit + modify (only if errors reported by tsc): `plugins/MddmRepeatablePlugin/converters.ts`, `commands/AddRepeatableItemCommand.ts`, `plugins/MddmDataTablePlugin/*.ts`

**Acceptance Criteria:**
- [ ] No `no exported member 'Writer'` or `'Element'` errors from `ckeditor5` module
- [ ] `postFixer.test.ts` cast replaced with typed `ModelElement[]` reference
- [ ] All CK5 tests still pass

**Verify:** `cd frontend/apps/web && rtk pnpm tsc --noEmit 2>&1 | grep -E "no exported member '(Writer|Element)'"` → zero lines

**Steps:**

- [ ] **Step 1: Patch `postFixer.ts`**

```ts
import type { Editor, ModelElement, ModelWriter } from 'ckeditor5';

export function registerSectionPostFixer(editor: Editor): void {
  editor.model.document.registerPostFixer((writer: ModelWriter) => {
    let changed = false;
    const root = editor.model.document.getRoot();
    if (!root) return false;

    const walker = root.getChildren();
    for (const node of walker) {
      if (!(node as ModelElement).is('element', 'mddmSection')) continue;
      const section = node as ModelElement;
      const children = Array.from(section.getChildren()) as ModelElement[];
      const headers = children.filter((c) => c.is('element', 'mddmSectionHeader'));
      const bodies = children.filter((c) => c.is('element', 'mddmSectionBody'));

      for (const extra of headers.slice(1)) {
        writer.remove(extra);
        changed = true;
      }
      for (const extra of bodies.slice(1)) {
        writer.remove(extra);
        changed = true;
      }

      if (headers.length === 0) {
        writer.insertElement('mddmSectionHeader', section, 0);
        changed = true;
      }
      if (bodies.length === 0) {
        const body = writer.createElement('mddmSectionBody');
        writer.append(body, section);
        writer.appendElement('paragraph', body);
        changed = true;
      }
    }
    return changed;
  });
}
```

- [ ] **Step 2: Inspect postFixer.test.ts offending line**

Run: `rtk read frontend/apps/web/src/features/documents/ck5/plugins/MddmSectionPlugin/__tests__/postFixer.test.ts`
Locate line 36 — currently `(root.getChildren() as { name: string }[])...`. Replace the cast with a proper `ModelElement[]` typing.

- [ ] **Step 3: Apply test fix**

Edit the offending cast in `postFixer.test.ts`. Replace:
```ts
const sections = Array.from(root.getChildren()) as { name: string }[];
```
with:
```ts
import type { ModelElement } from 'ckeditor5';
// ...
const sections = Array.from(root.getChildren()) as ModelElement[];
// access via sections[0].name (ModelElement exposes `name`)
```
If the test uses `.name`, `ModelElement` already has it. If it uses a different property not on ModelElement, use `ModelNode` instead.

- [ ] **Step 4: Re-run tsc filtered**

```bash
cd frontend/apps/web
rtk pnpm tsc --noEmit 2>&1 | grep "src/features/documents/ck5" | wc -l
```
Expected: Count drops below the Task 0 baseline CK5 count.

- [ ] **Step 5: Audit remaining Model* gaps**

```bash
rtk pnpm tsc --noEmit 2>&1 | grep -E "src/features/documents/ck5.*error TS(2305|2322|2352)"
```
Expected: Any remaining errors name the exact file. For each, apply the same rename principle (`Writer → ModelWriter`, `Element → ModelElement`). Do NOT rename `ViewElement` — already correct.

- [ ] **Step 6: Run CK5 tests**

```bash
rtk pnpm vitest run src/features/documents/ck5 --reporter=dot
```
Expected: 78/78 pass.

- [ ] **Step 7: Commit**

```bash
cd ../..
rtk git add frontend/apps/web/src/features/documents/ck5
rtk git commit -m "fix(ck5): rename Writer/Element to ModelWriter/ModelElement for v48"
```

---

### Task 3: Fix `trim: false` type error in RichBlock test

**Goal:** `getData({ trim })` expects `'none' | 'empty'`. Replace boolean.

**Files:**
- Modify: `frontend/apps/web/src/features/documents/ck5/plugins/MddmRichBlockPlugin/__tests__/plugin.test.ts`

**Acceptance Criteria:**
- [ ] `trim: false as const` replaced with `trim: 'none'`
- [ ] Test still passes semantically (still returns un-trimmed output)
- [ ] TS2322 error on line 52 gone

**Verify:** `cd frontend/apps/web && rtk pnpm tsc --noEmit 2>&1 | grep "MddmRichBlockPlugin/__tests__/plugin.test.ts"` → zero lines

**Steps:**

- [ ] **Step 1: Patch line 52**

Edit `plugin.test.ts`. Replace:
```ts
const html = editor.getData({ trim: false as const });
```
with:
```ts
const html = editor.getData({ trim: 'none' });
```

- [ ] **Step 2: Rerun test**

```bash
cd frontend/apps/web
rtk pnpm vitest run src/features/documents/ck5/plugins/MddmRichBlockPlugin
```
Expected: 4/4 pass.

- [ ] **Step 3: Commit**

```bash
cd ../..
rtk git add frontend/apps/web/src/features/documents/ck5/plugins/MddmRichBlockPlugin/__tests__/plugin.test.ts
rtk git commit -m "fix(ck5): use 'none' instead of false for getData trim option"
```

---

## Phase 2 — Fix toolbar + runtime warnings

### Task 4: Reconcile toolbar items with registered commands

**Goal:** Eliminate `toolbarview-item-unavailable` console warnings. Every toolbar string must resolve to a loaded command in the editor mode using it.

**Files:**
- Modify: `frontend/apps/web/src/features/documents/ck5/config/toolbars.ts`
- Modify: `frontend/apps/web/src/features/documents/ck5/config/editorConfig.ts` (if toolbar arrays need per-mode selection)

**Acceptance Criteria:**
- [ ] `AUTHOR_TOOLBAR` contains only items available when `StandardEditingMode` + MDDM plugins are loaded
- [ ] `FILL_TOOLBAR` contains only items available when `RestrictedEditingMode` is loaded (per CK5 docs, restricted mode registers `goToNext/PreviousRestrictedEditingException` but NOT `restrictedEditingException` / `restrictedEditingExceptionBlock` — the latter two are StandardEditingMode-only)
- [ ] Running CK5 tests produces zero `toolbarview-item-unavailable` stderr lines

**Verify:** `cd frontend/apps/web && rtk pnpm vitest run src/features/documents/ck5 2>&1 | grep "toolbarview-item-unavailable"` → zero lines

**Steps:**

- [ ] **Step 1: Verify CK5 restricted-editing command registration**

Run:
```bash
cd frontend/apps/web
rtk grep -r "goToNextRestrictedEditingException\|restrictedEditingException\b\|restrictedEditingExceptionBlock" node_modules/.pnpm/@ckeditor+ckeditor5-restricted-editing@48.0.0/node_modules/@ckeditor/ckeditor5-restricted-editing/dist --include="*.d.ts"
```
Expected: Confirms which mode (StandardEditingMode vs RestrictedEditingMode) registers each command. Note the mapping.

- [ ] **Step 2: Split toolbar exports by mode**

Edit `toolbars.ts`:
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
  'restrictedEditingException',       // StandardEditingMode registers this
  'restrictedEditingExceptionBlock',  // StandardEditingMode registers this
  '|',
  'undo',
  'redo',
];

export const FILL_TOOLBAR: readonly string[] = [
  'undo',
  'redo',
  '|',
  'bold',
  'italic',
  'underline',
  'link',
  '|',
  'bulletedList',
  'numberedList',
  'alignment',
  'fontColor',
  '|',
  'goToPreviousRestrictedEditingException',
  'goToNextRestrictedEditingException',
];
```
If Step 1 reveals a different registration map, adjust accordingly — remove items not registered in the loading mode, keep items that are.

- [ ] **Step 3: If tests still warn, inspect test editor plugin list**

Check which tests trigger the warning. Run:
```bash
rtk pnpm vitest run src/features/documents/ck5 2>&1 | grep -B1 "toolbarview-item-unavailable"
```
For each flagged test, the test editor config may be using `AUTHOR_TOOLBAR` without loading `StandardEditingMode` or using `FILL_TOOLBAR` without `RestrictedEditingMode`. Either:
- Add the correct mode plugin to the test editor config, OR
- Use a minimal test-specific toolbar instead of the full constant.

Prefer the first — tests should exercise the real toolbar.

- [ ] **Step 4: Re-run CK5 tests — count warnings**

```bash
rtk pnpm vitest run src/features/documents/ck5 2>&1 | grep -c "toolbarview-item-unavailable"
```
Expected: `0`.

- [ ] **Step 5: Commit**

```bash
cd ../..
rtk git add frontend/apps/web/src/features/documents/ck5/config
rtk git commit -m "fix(ck5): align toolbar items with registered commands per editor mode"
```

---

### Task 5: Migrate deprecated regex matchers in `editorConfig.ts` htmlSupport

**Goal:** Eliminate `matcher-pattern-deprecated-attributes-class-key` warning. The offending matcher is NOT in converters — it's in `config/editorConfig.ts` inside `htmlSupport.allow[0].attributes.class` plus several other regex attribute values in the same object.

**Files:**
- Modify: `frontend/apps/web/src/features/documents/ck5/config/editorConfig.ts` (lines 55–73)
- Audit (defensive): `rtk grep -rE "classes:\s*/\^" src/features/documents/ck5` — report zero matches, but still scan

**Acceptance Criteria:**
- [ ] The `attributes.class` regex at `editorConfig.ts:60` is replaced with the v48-preferred form (array literal, function, or explicit object)
- [ ] All sibling regex attribute values (`data-variant`, `data-field-required`, `data-mddm-variant`, `data-mddm-schema`) migrated to the same preferred form
- [ ] The regex `name: /^(section|div|span|header|ol|li)$/` stays as-is (element `name` matcher still supports regex in v48)
- [ ] CK5 vitest runs yield zero `matcher-pattern-deprecated-attributes-class-key` stderr lines
- [ ] Existing converter round-trip tests (`exceptionRoundTrip.integration.test.ts`, section/repeatable/richblock converter tests) still pass — proof that GHS still matches real HTML

**Verify:**
```bash
cd frontend/apps/web
rtk pnpm vitest run src/features/documents/ck5 2>&1 | grep -c "matcher-pattern-deprecated"
```
Expected: `0`.

**Steps:**

- [ ] **Step 1: Confirm the real source**

Run:
```bash
cd frontend/apps/web
rtk grep -rnE "class:\s*/\^|classes:\s*/\^" src/features/documents/ck5
```
Expected: One hit at `src/features/documents/ck5/config/editorConfig.ts:60`. If additional hits appear in converters, add them to the patch list.

- [ ] **Step 2: Patch `editorConfig.ts` htmlSupport.allow[0]**

Replace the single `allow` entry with class + attribute matchers that use the v48-preferred forms. The `class` value accepts: `true`, a string, a string array, or a predicate function `(value) => boolean`. For attribute values the same forms apply — prefer enumerated string arrays when the set is small, prefer a predicate when open-ended.

Edit lines 55–74 of `config/editorConfig.ts` to:

```ts
htmlSupport: {
  allow: [
    {
      name: /^(section|div|span|header|ol|li)$/,
      classes: (className: string) =>
        className.startsWith('mddm-') ||
        className.startsWith('restricted-editing-exception'),
      attributes: {
        'data-section-id': true,
        'data-variant': ['locked', 'editable', 'mixed'],
        'data-repeatable-id': true,
        'data-item-id': true,
        'data-field-id': true,
        'data-field-type': true,
        'data-field-label': true,
        'data-field-required': ['true', 'false'],
        'data-mddm-variant': ['fixed', 'dynamic'],
        'data-mddm-schema': (value: string) => /^v\d+$/.test(value),
      },
    },
  ],
},
```

Notes:
- The key renamed from `class` inside `attributes` to a sibling `classes` key because v48 treats classes as a first-class matcher, not an attribute. Confirm by reading CK5 v48 GHS typings at `node_modules/.pnpm/@ckeditor+ckeditor5-html-support@48.0.0/node_modules/@ckeditor/ckeditor5-html-support/dist/**/*.d.ts` if uncertain before writing.
- `data-mddm-schema` kept as predicate because the regex `/^v\d+$/` has no bounded enumeration.

- [ ] **Step 3: Cross-check v48 GHS typings**

Run:
```bash
cd frontend/apps/web
rtk grep -rE "^.*interface (MatcherPattern|DocumentListPluginConfig|GeneralHtmlSupportConfig)" node_modules/.pnpm/@ckeditor+ckeditor5-html-support@48.0.0/node_modules/@ckeditor/ckeditor5-html-support/dist 2>&1 | head -10
```
Expected: Locate the accepted matcher shape. If the CK5 v48 API uses a different key name (e.g. `classes` top-level vs inside `attributes`), adjust Step 2 exactly to match the typing.

- [ ] **Step 4: Rerun CK5 tests + count warnings**

```bash
rtk pnpm vitest run src/features/documents/ck5 --reporter=dot 2>&1 | tail -5
WARN=$(rtk pnpm vitest run src/features/documents/ck5 2>&1 | grep -c "matcher-pattern-deprecated")
echo "deprecated_matcher_warnings=$WARN"
```
Expected: `78/78 pass` and `deprecated_matcher_warnings=0`.

- [ ] **Step 5: Type check scoped**

```bash
rtk pnpm tsc --noEmit 2>&1 | grep "src/features/documents/ck5/config/editorConfig.ts" | wc -l
```
Expected: `0`.

- [ ] **Step 6: Commit**

```bash
cd ../..
rtk git add frontend/apps/web/src/features/documents/ck5/config/editorConfig.ts
rtk git commit -m "fix(ck5): migrate GHS htmlSupport matchers off deprecated regex attr form"
```

---

## Phase 3 — Triage pre-existing non-CK5 TS errors

### Task 6: Triage pre-existing Tiptap/BlockNote TS errors with evidence

**Goal:** Produce *real evidence* (not lockfile text diff) of which of the 30 non-CK5 TS errors are pre-existing and which are regressions introduced on the branch, then fix or isolate accordingly so `pnpm build` exits 0.

**Files:**
- Possibly modify: `frontend/apps/web/package.json`, `frontend/apps/web/pnpm-lock.yaml`, `frontend/apps/web/tsconfig.json` (or a new `tsconfig.build.json`)
- Evidence artifacts: `.tmp/main-tsc.txt`, `.tmp/branch-tsc.txt`, `.tmp/reachability.txt`

**Acceptance Criteria:**
- [ ] Actual `pnpm tsc --noEmit` output from the `main` branch captured at `.tmp/main-tsc.txt` (not a lockfile diff)
- [ ] Diff between `.tmp/main-tsc.txt` and `.tmp/branch-tsc.txt` scoped to non-CK5 files is computed; any file with errors on main is classified "pre-existing", any file clean on main is classified "regression"
- [ ] For every file classified "regression", the concrete dep version or API change causing it is identified and fixed (not excluded)
- [ ] For every file classified "pre-existing", reachability from the Vite entry graph is proven via `tsc --explainFiles` or a recursive grep of imports — if unreachable, isolate via a dedicated `tsconfig.build.json` whose `include` starts from `src/main.tsx` and whose `exclude` covers only those unreachable files; if reachable, it must be fixed, not excluded
- [ ] `rtk pnpm build` exits 0 with zero `error TS` lines

**Verify:**
```bash
cd frontend/apps/web
rtk pnpm tsc --noEmit 2>&1 | grep -c "error TS"
rtk pnpm build 2>&1 | tail -5
```
Expected: first command prints `0`; second ends `✓ built in …`.

**Steps:**

- [ ] **Step 1: Capture real `main` tsc baseline**

The verification pass already did this in a stash pop — rebuild fresh from a clean main checkout to guarantee the numbers stand up.

```bash
cd "C:/Users/leandro.theodoro.MN-NTB-LEANDROT/Documents/MetalDocs"
rtk git worktree add ../MetalDocs-main-audit main 2>/dev/null || true
cd "../MetalDocs-main-audit/frontend/apps/web"
rtk pnpm install --frozen-lockfile
mkdir -p ../../../MetalDocs-ck5/.tmp
rtk pnpm tsc --noEmit > ../../../MetalDocs-ck5/.tmp/main-tsc.txt 2>&1 || true
```
Expected: File written. Capture error count: `grep -c "error TS" ../../../MetalDocs-ck5/.tmp/main-tsc.txt`.

- [ ] **Step 2: Capture branch tsc baseline**

```bash
cd "C:/Users/leandro.theodoro.MN-NTB-LEANDROT/Documents/MetalDocs-ck5/frontend/apps/web"
rtk pnpm tsc --noEmit > ../../.tmp/branch-tsc.txt 2>&1 || true
grep -c "error TS" ../../.tmp/branch-tsc.txt
```
Expected: non-zero count.

- [ ] **Step 3: Classify each non-CK5 file**

```bash
cd "C:/Users/leandro.theodoro.MN-NTB-LEANDROT/Documents/MetalDocs-ck5"
grep "error TS" .tmp/main-tsc.txt   | awk -F'(' '{print $1}' | sort -u > .tmp/main-files.txt
grep "error TS" .tmp/branch-tsc.txt | awk -F'(' '{print $1}' | sort -u > .tmp/branch-files.txt
comm -13 .tmp/main-files.txt .tmp/branch-files.txt > .tmp/regression-files.txt
comm -12 .tmp/main-files.txt .tmp/branch-files.txt > .tmp/preexisting-files.txt
wc -l .tmp/regression-files.txt .tmp/preexisting-files.txt
```
Expected: Two files listing the two categories. Every file ends up in exactly one bucket.

- [ ] **Step 4: Fix each regression file**

For every file in `.tmp/regression-files.txt`:
- Inspect the first error from `.tmp/branch-tsc.txt` filtered to that file.
- Identify the API that broke. If it's a Tiptap chain command like `toggleBold`, check whether the extension in `RichField.tsx` was registered; if a dep dropped an API, pin the dep back via `package.json` with the exact version from main's `pnpm-lock.yaml`.
- Run `rtk pnpm install` and re-check with `rtk pnpm tsc --noEmit 2>&1 | grep <file>`. Iterate until zero errors for that file.

No shortcuts: excluding a regression from tsc is **not allowed** in this task.

- [ ] **Step 5: Prove reachability for pre-existing files**

For every file in `.tmp/preexisting-files.txt`:

```bash
cd frontend/apps/web
rtk pnpm tsc --noEmit --explainFiles 2>&1 | grep -A1 "$(cat ../../.tmp/preexisting-files.txt | head -1)" | head -20
```
Or use a recursive import grep as corroboration:
```bash
FILE=$(cat ../../.tmp/preexisting-files.txt | head -1)
rtk grep -rE "from.*${FILE%%.tsx}|from.*${FILE%%.ts}" src --exclude-dir=node_modules
```
Report per file: `reachable` or `unreachable`. Save results to `.tmp/reachability.txt`.

- [ ] **Step 6a — Reachable pre-existing file: fix it**

Same approach as Step 4. Treat as a regression blocker.

- [ ] **Step 6b — Unreachable pre-existing file: isolate via entry-rooted build tsconfig**

The build tsconfig MUST be entry-rooted, not folder-rooted. A `include: ["src"]` pattern re-includes every source file and voids the reachability proof.

Create `frontend/apps/web/tsconfig.build.json`:
```json
{
  "extends": "./tsconfig.json",
  "compilerOptions": { "noEmit": true },
  "files": ["src/main.tsx"],
  "exclude": ["<PATHS_FROM_UNREACHABLE_LIST>"]
}
```
Replace `<PATHS_FROM_UNREACHABLE_LIST>` with the actual file globs from `.tmp/reachability.txt`. The `files` entry tells TS to start from `main.tsx` and only compile transitively-referenced modules; the explicit `exclude` is a belt-and-suspenders backstop that catches any file the reachability scan missed. If `main.tsx` is not the real entry in this app, substitute it with the true entry path shown in `vite.config.ts`.

Point `package.json` `scripts.build` to use it: `vite build && tsc --noEmit -p tsconfig.build.json` (or the inverse order if the existing script runs tsc first).

Add a comment at the top of `tsconfig.build.json`:
```
// TEMPORARY: Plan C deletes BlockNote and removes these excludes.
```

- [ ] **Step 7: Gate**

```bash
cd frontend/apps/web
rtk pnpm tsc --noEmit 2>&1 | grep -c "error TS"
rtk pnpm build 2>&1 | tail -5
```
Expected: `0` and `✓ built in …` respectively.

- [ ] **Step 8: Cleanup audit worktree**

```bash
cd "C:/Users/leandro.theodoro.MN-NTB-LEANDROT/Documents/MetalDocs"
rtk git worktree remove ../MetalDocs-main-audit --force 2>/dev/null || true
```

- [ ] **Step 9: Commit**

```bash
cd "C:/Users/leandro.theodoro.MN-NTB-LEANDROT/Documents/MetalDocs-ck5"
rtk git add frontend/apps/web
rtk git commit -m "fix(ck5): unblock build — <regression-fixes|isolate-unreachable>"
```
Fill the commit scope with the actual path taken (Step 6a or Step 6b). If both paths were needed, list both.

---

## Phase 4 — Runtime validation via Vite preview

### Task 7: Expose editor refs + backend-save hook on the existing hash-route harness

**Goal:** The harness already lives at `#/test-harness/ck5?mode=author|fill&tpl=...&doc=...` (single mode per URL). Do NOT invent a `/?ck5=1` route. Instead, extend `AuthorPage` and `FillPage` to expose their editor instances and a save function on `window` so automated preview tooling can introspect and click-through the real flow.

**Files:**
- Modify: `frontend/apps/web/src/features/documents/ck5/react/AuthorPage.tsx` — expose editor + save on window (dev-only)
- Modify: `frontend/apps/web/src/features/documents/ck5/react/FillPage.tsx` — expose editor + save on window (dev-only)
- No change to: `frontend/apps/web/src/main.tsx` (harness routing already correct)
- No change to: `frontend/apps/web/src/test-harness/CK5TestHarness.tsx` (single-mode render is correct — dual-pane is NOT a goal)

**Acceptance Criteria:**
- [ ] `http://localhost:5173/#/test-harness/ck5?mode=author&tpl=smoke` renders Author and sets `window.__ck5.authorEditor` to the `DecoupledEditor` instance
- [ ] `http://localhost:5173/#/test-harness/ck5?mode=fill&tpl=smoke&doc=smoke-1` renders Fill and sets `window.__ck5.fillEditor`
- [ ] In both modes `window.__ck5.save(html)` force-persists current editor HTML via the active persistence layer
- [ ] Hooks only attach when `import.meta.env.DEV === true`
- [ ] No console errors other than the CKEditor GPL banner

**Verify:**
```
preview_eval  code: "typeof window.__ck5?.authorEditor?.getData"
```
Expected: `'function'` in author mode, `undefined` in fill mode (and vice versa for fillEditor).

**Steps:**

- [ ] **Step 1: Define a shared window-types file**

Create `frontend/apps/web/src/features/documents/ck5/react/windowHooks.ts`:
```ts
import type { DecoupledEditor, ClassicEditor } from 'ckeditor5';

declare global {
  interface Window {
    __ck5?: {
      authorEditor?: DecoupledEditor;
      fillEditor?: ClassicEditor;
      save?: (html?: string) => Promise<void> | void;
    };
  }
}

export function installAuthorHook(editor: DecoupledEditor, save: (html?: string) => void) {
  if (!import.meta.env.DEV) return;
  window.__ck5 = { ...(window.__ck5 ?? {}), authorEditor: editor, save };
}

export function installFillHook(editor: ClassicEditor, save: (html?: string) => void) {
  if (!import.meta.env.DEV) return;
  window.__ck5 = { ...(window.__ck5 ?? {}), fillEditor: editor, save };
}

export function clearHooks() {
  if (!import.meta.env.DEV) return;
  delete window.__ck5;
}
```

- [ ] **Step 2: Wire `AuthorPage.tsx` onReady to install the hook**

In `AuthorPage.tsx`, update `onReady` + add a cleanup effect:
```tsx
import { useEffect } from 'react';
import { installAuthorHook, clearHooks } from './windowHooks';
// ...
const onReady = useCallback((editor: DecoupledEditor) => {
  editorRef.current = editor;
  applyPerCellExceptions(editor);
  installAuthorHook(editor, (html) => {
    const finalHtml = html ?? editor.getData();
    saveTemplate(tplId, finalHtml, existing?.manifest ?? { fields: [] });
  });
}, [tplId, existing]);

useEffect(() => clearHooks, []);
```

- [ ] **Step 3: Wire `FillPage.tsx` with the same pattern**

Update `FillPage.tsx` to thread an `onReady` prop through `FillEditor` if not already there (check first via `rtk read`). In `onReady`, call `installFillHook(editor, (html) => saveDocument(docId, html ?? editor.getData()))`. Add `useEffect(() => clearHooks, []);`.

If `FillEditor` does not expose `onReady`, extend its prop surface minimally — one callback, invoked after `ClassicEditor.create(...)` resolves.

- [ ] **Step 4: Type-check the new hook file**

```bash
cd frontend/apps/web
rtk pnpm tsc --noEmit 2>&1 | grep -E "AuthorPage|FillPage|windowHooks"
```
Expected: zero lines.

- [ ] **Step 5: Start preview + verify hooks**

```
preview_start
```
Then:
```
preview_eval  code: "window.location.hash = '#/test-harness/ck5?mode=author&tpl=smoke'"
preview_eval  code: "Boolean(window.__ck5?.authorEditor) && typeof window.__ck5.save"
```
Expected second eval returns `'function'`.

Then:
```
preview_eval  code: "window.location.hash = '#/test-harness/ck5?mode=fill&tpl=smoke&doc=smoke-1'"
preview_eval  code: "Boolean(window.__ck5?.fillEditor) && typeof window.__ck5.save"
```
Expected: `'function'`.

- [ ] **Step 6: Console check**

```
preview_console_logs
```
Expected: No `error`-level entries. Warnings acceptable only for GPL banner.

- [ ] **Step 7: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/ck5/react
rtk git commit -m "feat(ck5): expose dev-only window hooks on AuthorPage and FillPage for preview automation"
```

---

### Task 8: Exercise Author insertion then Fill load, driving the hash-route harness

**Goal:** Prove the editors are interactive and data round-trips from Author → persistence → Fill via the real harness URLs. No dual-pane assumptions.

**Files:** None. Preview-driven validation only.

**Acceptance Criteria:**
- [ ] In Author mode (`#/test-harness/ck5?mode=author&tpl=smoke`), clicking the Insert Section toolbar button inserts a `<section class="mddm-section">...</section>` into the editor data
- [ ] `window.__ck5.save()` persists the Author HTML (localStorage first; Task 10 switches this to backend)
- [ ] Navigating the preview to `#/test-harness/ck5?mode=fill&tpl=smoke&doc=smoke-1` renders the same HTML inside `FillEditor` with restricted-editing exceptions visible
- [ ] At least one Fill exception-navigation toolbar click moves selection into the exception
- [ ] `.tmp/ck5-preview-author.png` and `.tmp/ck5-preview-fill.png` both captured
- [ ] `preview_console_logs` returns zero `error` entries across both modes (GPL banner warnings acceptable)

**Verify:**
```
preview_eval  code: "window.__ck5.fillEditor.getData().includes('class=\"mddm-section\"')"
```
Expected: `true`.

**Steps:**

- [ ] **Step 1: Navigate to Author**

```
preview_eval  code: "window.location.hash = '#/test-harness/ck5?mode=author&tpl=smoke'"
preview_snapshot
```
Expected: Author page rendered. MDDM toolbar buttons visible.

- [ ] **Step 2: Click Insert Section**

```
preview_click  ref="button[aria-label*='section' i]"
preview_snapshot
```
If no matching button, inspect the snapshot for the real selector (toolbar icon titles vary). Adjust the click ref.

Expected: Post-click snapshot contains `class="mddm-section"` markup in the Author editable.

- [ ] **Step 3: Read + persist Author data**

```
preview_eval  code: "window.__ck5.authorEditor.getData()"
```
Capture the returned HTML. It must contain `class="mddm-section"`.

```
preview_eval  code: "window.__ck5.save(); 'saved'"
```
Expected: returns `'saved'`. Also confirm via:
```
preview_eval  code: "localStorage.getItem('ck5.tpl.smoke')"
```
Expected: Non-null; stringified JSON containing the section markup. (In Task 10 this expectation flips to the backend endpoint.)

- [ ] **Step 4: Capture Author screenshot**

```
preview_screenshot  path=".tmp/ck5-preview-author.png"
```

- [ ] **Step 5: Navigate to Fill**

```
preview_eval  code: "window.location.hash = '#/test-harness/ck5?mode=fill&tpl=smoke&doc=smoke-1'"
preview_snapshot
```
Expected: Fill page rendered; the restricted-editing exception markup from the template is visible in the editor.

- [ ] **Step 6: Exercise exception navigation**

```
preview_click  ref="button[aria-label*='exception' i][aria-label*='next' i]"
preview_snapshot
```
If no matching button, inspect the snapshot. Expected: Post-click snapshot shows the selection has moved to within `class="restricted-editing-exception"`.

- [ ] **Step 7: Verify Fill has the Author data**

```
preview_eval  code: "window.__ck5.fillEditor.getData().includes('class=\"mddm-section\"')"
```
Expected: `true`.

- [ ] **Step 8: Capture Fill screenshot**

```
preview_screenshot  path=".tmp/ck5-preview-fill.png"
```

- [ ] **Step 9: Final console + network check across both modes**

```
preview_console_logs
preview_network
```
Expected: No `error` entries; no failed network requests.

- [ ] **Step 10: Commit nothing — verification only. Record pass/fail per acceptance criterion in session notes.**

---

### Task 9: Full vitest + tsc + build gate

**Goal:** Hard gate before sign-off. All three pipelines green.

**Files:** None.

**Acceptance Criteria:**
- [ ] `rtk pnpm vitest run` → 0 failures across entire repo
- [ ] `rtk pnpm tsc --noEmit` → 0 errors
- [ ] `rtk pnpm build` → exit 0
- [ ] Playwright smoke `rtk pnpm playwright test tests/e2e/ck5-smoke.spec.ts` → passes

**Verify:** All four commands exit 0.

**Steps:**

- [ ] **Step 1: Full vitest**

```bash
cd frontend/apps/web
rtk pnpm vitest run --reporter=dot 2>&1 | tail -10
```
Expected: `Test Files N passed (N)` / `Tests M passed (M)` / exit 0.

- [ ] **Step 2: Full tsc**

```bash
rtk pnpm tsc --noEmit 2>&1 | grep -c "error TS"
```
Expected: `0`.

- [ ] **Step 3: Full build**

```bash
rtk pnpm build 2>&1 | tail -10
```
Expected: `✓ built in …`.

- [ ] **Step 4: Playwright smoke (optional — only if Playwright infra is ready)**

```bash
rtk pnpm playwright test tests/e2e/ck5-smoke.spec.ts
```
Expected: `1 passed`.

- [ ] **Step 5: Commit nothing — gate only.** If any step fails, loop back to the relevant phase.

---

## Phase 5 — Preview-assisted backend validation

### Task 10: Wire backend persistence + prove save/load round-trip

**Goal:** Plan goal demands "full frontend backend validation". Plan A persistence is localStorage-only. This task REQUIRES extending persistence so Author save and Fill load hit the `apps/api` backend; the acceptance path is a real round-trip through the HTTP stack, not localStorage. No optional fallback.

**Repo context confirmed (pre-plan research):**
- Backend start script: `scripts/dev-api.ps1` (Windows) — runs `go run ./apps/api/cmd/metaldocs-api`. The `start-api.ps1` referenced earlier in this plan tree does NOT exist; it was a stash-only file on main.
- Default API port: `APP_PORT` from `.env`, falls back to `8080`.
- Vite proxy: `vite.config.ts` already proxies `/api/v1` from Vite port `4173` to `http://127.0.0.1:$APP_PORT`. Use **same-origin relative paths** (`/api/v1/...`) in the client — do NOT hit the backend cross-origin.
- Auth: session cookie via `/api/v1/auth/login`. Most endpoints are guarded (see `permissions_test.go`). `fetch` MUST use `credentials: 'include'`. A test user must log in before the preview round-trip.
- Real endpoints discovered:
  - `POST /api/v1/templates` — create template
  - `GET /api/v1/templates` — list
  - `GET /api/v1/templates/:key/draft` — fetch draft
  - `POST /api/v1/templates/:key/publish` — publish
  - `POST /api/v1/documents` — create document
  - `GET /api/v1/documents/:id` — fetch document
  - `POST /api/v1/documents/:id/content/browser` — save browser content (this is the CK5 save target)

**Files:**
- Create: `frontend/apps/web/src/features/documents/ck5/persistence/apiPersistence.ts` — fetch client against `/api/v1/...` with `credentials: 'include'`
- Create: `frontend/apps/web/src/features/documents/ck5/persistence/__tests__/apiPersistence.test.ts`
- Create: `frontend/apps/web/src/features/documents/ck5/persistence/index.ts` — barrel picking backend vs localStorage via `VITE_CK5_PERSISTENCE`
- Modify: `frontend/apps/web/src/features/documents/ck5/react/AuthorPage.tsx` — import from the barrel; async-safe load
- Modify: `frontend/apps/web/src/features/documents/ck5/react/FillPage.tsx` — same
- Modify (create if missing): `frontend/apps/web/.env.development` — `VITE_CK5_PERSISTENCE=api` only (no `VITE_CK5_API_BASE` needed; Vite proxies)
- Verify-only: `scripts/dev-api.ps1`, `vite.config.ts`, `apps/api/cmd/metaldocs-api/*` for endpoint shapes and auth flow

**Acceptance Criteria:**
- [ ] Backend API starts via `pwsh -File scripts/dev-api.ps1` (Windows) or `go run ./apps/api/cmd/metaldocs-api` (bash with env loaded); listens on `APP_PORT` (default 8080)
- [ ] `/api/v1/health/live` returns 200 via the Vite-proxied URL `http://localhost:4173/api/v1/health/live`
- [ ] `apiPersistence.ts` uses **same-origin relative paths** (`/api/v1/templates/:key/draft`, `/api/v1/documents/:id`, `/api/v1/documents/:id/content/browser`) and sets `credentials: 'include'` on every request
- [ ] Preview first authenticates via a login helper (see Step 8), then Author save produces a 2xx `POST /api/v1/documents/:id/content/browser` seen in `preview_network`; Fill reload loads the saved HTML via `GET /api/v1/documents/:id`
- [ ] Vitest unit test covers happy + 4xx + 5xx paths using `vi.fn()` over `global.fetch`
- [ ] No console `error` entries across the round-trip (GPL license banner warning is the only allowed warning)
- [ ] `.env.development` committed only with non-secret defaults (`VITE_CK5_PERSISTENCE=api`); secret envs go in `.env.development.local` which stays gitignored

**Verify:**
```
preview_network  (filter: method=POST or PUT, path contains 'template' or 'document')
```
Expected: 2xx status for the save request. Then:
```
preview_eval  code: "window.location.hash = '#/test-harness/ck5?mode=author&tpl=smoke'; 'reloading'"
preview_eval  code: "new Promise(r => setTimeout(() => r(window.__ck5.authorEditor.getData()), 1000))"
```
Expected: Returned HTML contains `class=\"mddm-section\"` — proves the data survived across the page load via the backend.

**Steps:**

- [ ] **Step 1: Discover backend endpoint shapes**

```bash
cd "C:/Users/leandro.theodoro.MN-NTB-LEANDROT/Documents/MetalDocs"
rtk grep -rE "router\.(get|post|put|delete)|app\.(get|post|put|delete)" apps/api --include="*.ts" --include="*.js" | head -40
rtk read scripts/start-api.ps1
rtk read scripts/start-api.sh
```
Capture: port, base path, endpoint methods + paths for template (CRUD) + document (CRUD). If no endpoints exist, STOP and escalate to the user — this plan's goal cannot be satisfied without them.

- [ ] **Step 2: Write failing test for `apiPersistence`**

Create `frontend/apps/web/src/features/documents/ck5/persistence/__tests__/apiPersistence.test.ts`:
```ts
import { describe, it, expect, vi, beforeEach } from 'vitest';
import * as api from '../apiPersistence';

describe('apiPersistence', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
  });

  it('saveTemplate POSTs the template payload', async () => {
    (globalThis.fetch as unknown as ReturnType<typeof vi.fn>).mockResolvedValue(
      new Response(JSON.stringify({ id: 'tpl-1' }), { status: 200 }),
    );
    await api.saveTemplate('tpl-1', '<p>x</p>', { fields: [] });
    expect(globalThis.fetch).toHaveBeenCalledWith(
      expect.stringContaining('/templates/tpl-1'),
      expect.objectContaining({ method: 'PUT' }),
    );
  });

  it('loadTemplate throws on non-2xx', async () => {
    (globalThis.fetch as unknown as ReturnType<typeof vi.fn>).mockResolvedValue(
      new Response('nope', { status: 500 }),
    );
    await expect(api.loadTemplate('tpl-1')).rejects.toThrow();
  });
});
```

Run:
```bash
cd frontend/apps/web
rtk pnpm vitest run src/features/documents/ck5/persistence/__tests__/apiPersistence.test.ts
```
Expected: FAIL — `apiPersistence` not found.

- [ ] **Step 3: Implement `apiPersistence.ts`**

Use the exact endpoint shapes captured in Step 1. Example skeleton (adapt paths):
```ts
import type { TemplateRecord } from './localStorageStub';

const BASE = import.meta.env.VITE_CK5_API_BASE ?? 'http://localhost:3000';

async function json<T>(res: Response): Promise<T> {
  if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
  return res.json() as Promise<T>;
}

export async function saveTemplate(
  id: string,
  contentHtml: string,
  manifest: TemplateRecord['manifest'],
): Promise<void> {
  const res = await fetch(`${BASE}/templates/${encodeURIComponent(id)}`, {
    method: 'PUT',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ id, contentHtml, manifest }),
  });
  await json<unknown>(res);
}

export async function loadTemplate(id: string): Promise<TemplateRecord | null> {
  const res = await fetch(`${BASE}/templates/${encodeURIComponent(id)}`);
  if (res.status === 404) return null;
  return json<TemplateRecord>(res);
}

export async function saveDocument(id: string, contentHtml: string): Promise<void> {
  const res = await fetch(`${BASE}/documents/${encodeURIComponent(id)}`, {
    method: 'PUT',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify({ id, contentHtml }),
  });
  await json<unknown>(res);
}

export async function loadDocument(id: string): Promise<string | null> {
  const res = await fetch(`${BASE}/documents/${encodeURIComponent(id)}`);
  if (res.status === 404) return null;
  const rec = await json<{ contentHtml: string }>(res);
  return rec.contentHtml;
}
```

- [ ] **Step 4: Test green**

```bash
rtk pnpm vitest run src/features/documents/ck5/persistence/__tests__/apiPersistence.test.ts
```
Expected: 2/2 pass.

- [ ] **Step 5: Add barrel with env-driven selection**

Create `frontend/apps/web/src/features/documents/ck5/persistence/index.ts`:
```ts
import * as local from './localStorageStub';
import * as api from './apiPersistence';

const mode = import.meta.env.VITE_CK5_PERSISTENCE ?? 'local';

export const { saveTemplate, loadTemplate, saveDocument, loadDocument } =
  mode === 'api' ? (api as unknown as typeof local) : local;

export type { TemplateRecord } from './localStorageStub';
```

⚠ Because `loadTemplate` is async in `api` but sync in `local`, callers must always `await`. Update `AuthorPage.tsx` + `FillPage.tsx` to `await loadTemplate`/`loadDocument` inside a `useEffect` — seed state from `null` initial, replace once the promise resolves. Do not block render.

- [ ] **Step 6: Update AuthorPage + FillPage to use the barrel**

Change imports:
```ts
// Before
import { saveTemplate, loadTemplate } from '../persistence/localStorageStub';

// After
import { saveTemplate, loadTemplate } from '../persistence';
```
Wrap the initial load in a `useEffect` that awaits the (possibly async) result. Keep types compatible (localStorage still sync — `await` of a non-promise resolves fine).

- [ ] **Step 7: Environment wiring**

Create `frontend/apps/web/.env.development` (or append if present):
```
VITE_CK5_PERSISTENCE=api
VITE_CK5_API_BASE=http://localhost:<api-port>
```
Replace `<api-port>` with the value discovered in Step 1. Do not commit secrets.

Also add `.env.development.local` to `.gitignore` if not already present; the committed `.env.development` contains only non-secret defaults.

- [ ] **Step 8: Start backend**

Open a second shell (do not block the agent):
```bash
pwsh -File scripts/start-api.ps1   # Windows
# or
bash scripts/start-api.sh          # WSL / git-bash
```
Verify:
```bash
rtk curl http://localhost:<api-port>/health
```
Expected: 200.

- [ ] **Step 9: Preview round-trip**

From the preview (already started in Task 8):
```
preview_eval  code: "window.location.hash = '#/test-harness/ck5?mode=author&tpl=smoke'"
preview_click  ref="button[aria-label*='section' i]"
preview_eval  code: "await window.__ck5.save(); 'saved'"
preview_network
```
Expected: `saved` + a 2xx PUT in `preview_network` for `/templates/smoke`.

Then:
```
preview_eval  code: "window.location.hash = '#/test-harness/ck5?mode=fill&tpl=smoke&doc=smoke-1'"
preview_eval  code: "new Promise(r => setTimeout(() => r(window.__ck5.fillEditor.getData().includes('mddm-section')), 1500))"
```
Expected: `true`.

- [ ] **Step 10: Negative path smoke**

Stop the API. Reload Author. Expect a visible error state (toast, banner, or boundary), NOT a white screen. Adjust error-handling in AuthorPage/FillPage if the UI breaks — minimal `try/catch` around the load, render an error banner.

Restart the API to finish the task in a healthy state.

- [ ] **Step 11: Full gate**

```bash
cd frontend/apps/web
rtk pnpm vitest run src/features/documents/ck5 --reporter=dot
rtk pnpm tsc --noEmit 2>&1 | grep -c "error TS"
rtk pnpm build 2>&1 | tail -5
```
Expected: all pass, `0`, `✓ built in …`.

- [ ] **Step 12: Commit**

```bash
cd ../..
rtk git add frontend/apps/web/src/features/documents/ck5/persistence frontend/apps/web/src/features/documents/ck5/react frontend/apps/web/.env.development frontend/apps/web/.gitignore
rtk git commit -m "feat(ck5): add apiPersistence + env-driven persistence barrel wiring to apps/api"
```

---

## Phase 6 — Final sweep

### Task 11: Re-run verification skill + finishing-a-development-branch

**Goal:** Apply the `verification-before-completion` discipline one last time, then enter the `finishing-a-development-branch` flow.

**Files:** None.

**Acceptance Criteria:**
- [ ] Verification-before-completion skill invoked and all evidence-based claims hold
- [ ] No red flags: no "should work" claims; every claim tied to a command output run in this session
- [ ] finishing-a-development-branch skill invoked for merge-or-PR decision

**Verify:** Skill invocation transcripts show green evidence for all completion claims.

**Steps:**

- [ ] **Step 1: Invoke verification-before-completion**

Claude: `Skill superpowers-extended-cc:verification-before-completion`. Walk through each acceptance criterion from Tasks 1–10 and attach the exact command + output that proves it.

- [ ] **Step 2: Invoke finishing-a-development-branch**

Claude: `Skill superpowers-extended-cc:finishing-a-development-branch`. Decide whether to PR or merge.

- [ ] **Step 3: If PR**

Follow the skill's PR workflow. PR title: `fix(ck5): unblock build and validate editors end-to-end`. PR body summarises Phases 1–5 with links to the key commits.

---

## Self-Review Checklist

- [ ] **Spec coverage:** Every regression surfaced in the verification report has a task (type renames T1–T2, trim T3, toolbar T4, deprecated matcher T5, Tiptap triage T6, preview T7, end-to-end T8, gate T9, backend T10, close-out T11).
- [ ] **No placeholders:** Every step has concrete commands and code. `TBD` / `similar to Task N` absent.
- [ ] **Type consistency:** `ModelSchema`, `ModelWriter`, `ModelElement` used uniformly. `ViewElement` intentionally not renamed (already correct in v48).
- [ ] **Caveman directive honored:** Plan narrative is terse; subagent task prompts use `/caveman` format for token efficiency. Code + commits remain normal per caveman rules.
- [ ] **Verifiable acceptance criteria** on every task — each expressed as a command whose output confirms the claim.

---

## Caveman subagent prompt template

When dispatching a subagent per task, prepend the prompt with:

```
/caveman full. Drop articles/filler. Fragments OK. Code + commits normal.
Task: <Task N title>
Goal: <one line>
Files: <exact paths>
Verify: <exact command + expected output>
Steps: follow task N steps exactly.
```

This keeps subagent input tokens minimal while preserving technical precision.
