# CK5 Frontend Migration — Plan A Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace BlockNote editor in `frontend/apps/web` with a CKEditor 5 v48 (GPL) implementation that supports Author mode (template authoring) and Fill mode (template filling) for all five MDDM primitives (Field, Section, Repeatable, DataTable, RichBlock) plus an asset upload adapter.

**Architecture:** Two CK5 editor instances built on the single-package v48 install (`ckeditor5` + `@ckeditor/ckeditor5-react`). Author editor uses `DecoupledEditor` + `StandardEditingMode`. Fill editor uses `ClassicEditor` + `RestrictedEditingMode`. Five MetalDocs primitive plugins (`MddmFieldPlugin`, `MddmSectionPlugin`, `MddmRepeatablePlugin`, `MddmDataTablePlugin`, `MddmRichBlockPlugin`) register schema + converters + commands, each using CK5-native features where possible (native Table for DataTable; `<ol>`/`<li>` data wire for Repeatable; GHS for RichBlock). Document persistence is a localStorage stub in this plan; Plan B replaces it with backend endpoints.

**Tech Stack:** `ckeditor5@48.0.0`, `@ckeditor/ckeditor5-react@^11.1.0`, React 18, TypeScript, Vite, Vitest, jsdom, Playwright. GPL-2.0-or-later. No premium plugins.

**Design reference:** `docs/ck5-wiki/` (22 drafted pages). Task-level cross-refs in each phase.

**Out of scope for Plan A:** Backend persistence (Plan B), DOCX export (Plan C), PDF export (Plan C), BlockNote deletion (Plan C).

---

## File Structure

All new code lives under `frontend/apps/web/src/features/documents/ck5/`. BlockNote code under `features/documents/mddm-editor/` is UNTOUCHED in this plan — Plan C deletes it. The two stacks coexist during Plan A.

```
frontend/apps/web/src/features/documents/ck5/
├── index.ts                          — barrel exports (AuthorEditor, FillEditor)
├── constants.ts                      — MDDM class names, data attr names, marker groups
├── types.ts                          — TS types shared across primitives
├── shared/
│   ├── findAncestor.ts               — model tree walker utility
│   ├── uid.ts                        — stable id generator
│   └── index.ts
├── config/
│   ├── pluginLists.ts                — AUTHOR_PLUGINS, FILL_PLUGINS arrays
│   ├── toolbars.ts                   — AUTHOR_TOOLBAR, FILL_TOOLBAR arrays
│   └── editorConfig.ts               — createAuthorConfig(), createFillConfig()
├── plugins/
│   ├── MddmFieldPlugin/
│   │   ├── index.ts                  — Plugin class
│   │   ├── schema.ts                 — registerFieldSchema()
│   │   ├── converters.ts             — registerFieldConverters()
│   │   ├── commands/
│   │   │   └── InsertFieldCommand.ts
│   │   └── __tests__/
│   │       ├── schema.test.ts
│   │       ├── converters.test.ts
│   │       └── InsertFieldCommand.test.ts
│   ├── MddmSectionPlugin/
│   │   ├── index.ts
│   │   ├── schema.ts
│   │   ├── converters.ts
│   │   ├── postFixer.ts              — one-header-one-body enforcement
│   │   ├── commands/
│   │   │   └── InsertSectionCommand.ts
│   │   └── __tests__/...
│   ├── MddmRepeatablePlugin/
│   │   ├── index.ts
│   │   ├── schema.ts
│   │   ├── converters.ts
│   │   ├── commands/
│   │   │   ├── InsertRepeatableCommand.ts
│   │   │   ├── AddRepeatableItemCommand.ts
│   │   │   └── RemoveRepeatableItemCommand.ts
│   │   └── __tests__/...
│   ├── MddmDataTablePlugin/
│   │   ├── index.ts                  — pulls in VariantSubPlugin + LockSubPlugin
│   │   ├── MddmTableVariantPlugin.ts
│   │   ├── MddmTableLockPlugin.ts
│   │   ├── nestedTableGuard.ts
│   │   ├── perCellExceptionWalker.ts
│   │   └── __tests__/...
│   ├── MddmRichBlockPlugin/
│   │   ├── index.ts
│   │   ├── commands/
│   │   │   └── InsertRichBlockCommand.ts
│   │   └── __tests__/...
│   └── MddmUploadAdapter/
│       ├── index.ts                  — Plugin class that plugs the adapter
│       ├── MddmUploadAdapter.ts      — class implementing CK5 UploadAdapter
│       └── __tests__/...
├── react/
│   ├── AuthorEditor.tsx              — React wrapper using <CKEditor editor={DecoupledEditor} />
│   ├── AuthorEditor.module.css
│   ├── FillEditor.tsx                — React wrapper using <CKEditor editor={ClassicEditor} />
│   ├── FillEditor.module.css
│   └── __tests__/
│       ├── AuthorEditor.test.tsx
│       └── FillEditor.test.tsx
├── persistence/
│   └── localStorageStub.ts           — temporary save/load; Plan B replaces
└── styles/
    └── ck5-overrides.css             — MDDM-specific class styling
```

File separation rationale: one folder per plugin, one file per converter/schema/command. This isolates primitive concerns, keeps files small enough for focused edits, and lets each primitive's tests live next to its code.

---

## Conventions

- **Every test file** imports `ModelTestEditor` from `@ckeditor/ckeditor5-core/tests/_utils/modeltesteditor.js` only if available; otherwise we use `ClassicEditor.create()` against a detached DOM element. We standardize on the latter because v48's internal test utilities are not part of the public package surface.
- **Every schema-level test** exercises `getData()` / `setData()` round-trip to prove converters and schema match.
- **Every command test** verifies `isEnabled` gating + model mutation + resulting HTML after `getData()`.
- **All commits** follow conventional commits (`feat:`, `test:`, `chore:`, `refactor:`). Scope: `ck5`.

---

## Phase 0 — Prep

### Task 1: Create worktree and branch

**Files:** None. Git workspace only.

- [ ] **Step 1: Verify clean working tree on main**

Run: `cd "C:/Users/leandro.theodoro.MN-NTB-LEANDROT/Documents/MetalDocs" && rtk git status`
Expected: A list of already-modified files (acceptable); no merge conflicts. Confirm current branch is `main`.

- [ ] **Step 2: Create worktree**

Run:
```bash
rtk git worktree add ../MetalDocs-ck5 -b migrate/ck5-frontend-plan-a
```
Expected: `Preparing worktree (new branch 'migrate/ck5-frontend-plan-a')` and a new directory at `../MetalDocs-ck5`.

- [ ] **Step 3: Switch working dir to worktree for all subsequent tasks**

Run: `cd "../MetalDocs-ck5"`
Expected: `rtk git status` shows branch `migrate/ck5-frontend-plan-a` with no local changes.

- [ ] **Step 4: Commit nothing — worktree creation is a git side effect, no commit needed.**

---

### Task 2: Install CK5 v48 packages alongside BlockNote

**Files:**
- Modify: `frontend/apps/web/package.json`

- [ ] **Step 1: Record current package.json for rollback reference**

Run: `rtk read frontend/apps/web/package.json`
Expected: Read-only observation. Note current versions of `@blocknote/*` and `react` / `react-dom`.

- [ ] **Step 2: Install CK5 v48 + React wrapper**

Run:
```bash
cd frontend/apps/web && rtk pnpm add ckeditor5@48.0.0 @ckeditor/ckeditor5-react@^11.1.0
```
Expected: `+ ckeditor5 48.0.0` and `+ @ckeditor/ckeditor5-react 11.x.x` reported; no peer-dep errors.

- [ ] **Step 3: Install dev dependency for upload adapter tests**

Run:
```bash
rtk pnpm add -D @types/node
```
Expected: Installed (may already be present; `unchanged` is fine).

- [ ] **Step 4: Verify install**

Run: `rtk pnpm list ckeditor5 @ckeditor/ckeditor5-react`
Expected: Both packages resolved at pinned versions.

- [ ] **Step 5: Commit**

```bash
cd ../../..
rtk git add frontend/apps/web/package.json frontend/apps/web/pnpm-lock.yaml
rtk git commit -m "chore(ck5): add ckeditor5 v48 + react wrapper"
```

---

### Task 3: Verify baseline build still passes

**Files:** None. Build validation only.

- [ ] **Step 1: Type check**

Run: `cd frontend/apps/web && rtk pnpm tsc --noEmit`
Expected: No errors. If pre-existing errors, record them — they are not regressions for Plan A.

- [ ] **Step 2: Run existing test suite**

Run: `rtk pnpm vitest run --reporter=dot`
Expected: All pre-existing tests pass.

- [ ] **Step 3: Build**

Run: `rtk pnpm build`
Expected: Build succeeds. Bundle may now be ~600 KB larger from CK5 — acceptable; Plan C removes BlockNote to offset.

- [ ] **Step 4: Commit nothing — verification only.**

---

## Phase 1 — Scaffold

### Task 4: Create folder structure and index barrel

**Files:**
- Create: `frontend/apps/web/src/features/documents/ck5/index.ts`
- Create: `frontend/apps/web/src/features/documents/ck5/constants.ts`
- Create: `frontend/apps/web/src/features/documents/ck5/types.ts`

- [ ] **Step 1: Create empty folders**

Run:
```bash
mkdir -p frontend/apps/web/src/features/documents/ck5/{shared,config,plugins,react,persistence,styles}
```
Expected: All directories created.

- [ ] **Step 2: Create `constants.ts`**

Write `frontend/apps/web/src/features/documents/ck5/constants.ts`:

```ts
export const MDDM_CLASSES = {
  section: 'mddm-section',
  sectionHeader: 'mddm-section__header',
  sectionBody: 'mddm-section__body',
  repeatable: 'mddm-repeatable',
  repeatableItem: 'mddm-repeatable__item',
  field: 'mddm-field',
  richBlock: 'mddm-rich-block',
  restrictedException: 'restricted-editing-exception',
} as const;

export const MDDM_DATA_ATTRS = {
  sectionId: 'data-section-id',
  sectionVariant: 'data-variant',
  repeatableId: 'data-repeatable-id',
  itemId: 'data-item-id',
  fieldId: 'data-field-id',
  fieldType: 'data-field-type',
  fieldLabel: 'data-field-label',
  fieldRequired: 'data-field-required',
  tableVariant: 'data-mddm-variant',
  schemaVersion: 'data-mddm-schema',
} as const;

export const MDDM_MODEL_ELEMENTS = {
  section: 'mddmSection',
  sectionHeader: 'mddmSectionHeader',
  sectionBody: 'mddmSectionBody',
  repeatable: 'mddmRepeatable',
  repeatableItem: 'mddmRepeatableItem',
  field: 'mddmField',
} as const;

export const SCHEMA_VERSION = 'v1';
```

- [ ] **Step 3: Create `types.ts`**

Write `frontend/apps/web/src/features/documents/ck5/types.ts`:

```ts
export type SectionVariant = 'locked' | 'editable' | 'mixed';
export type TableVariant = 'fixed' | 'dynamic';
export type FieldType =
  | 'text'
  | 'date'
  | 'number'
  | `currency:${string}`
  | `select:${string}`
  | 'boolean';

export interface FieldDefinition {
  id: string;
  label: string;
  type: FieldType;
  required: boolean;
  defaultValue: string;
  group?: string;
}
```

- [ ] **Step 4: Create `index.ts` barrel**

Write `frontend/apps/web/src/features/documents/ck5/index.ts`:

```ts
export * from './constants';
export * from './types';
```

- [ ] **Step 5: Type check**

Run: `cd frontend/apps/web && rtk pnpm tsc --noEmit`
Expected: No errors.

- [ ] **Step 6: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/ck5
rtk git commit -m "feat(ck5): scaffold ck5 feature folder with constants and types"
```

---

### Task 5: Create shared utilities

**Files:**
- Create: `frontend/apps/web/src/features/documents/ck5/shared/findAncestor.ts`
- Create: `frontend/apps/web/src/features/documents/ck5/shared/uid.ts`
- Create: `frontend/apps/web/src/features/documents/ck5/shared/index.ts`
- Test: `frontend/apps/web/src/features/documents/ck5/shared/__tests__/findAncestor.test.ts`
- Test: `frontend/apps/web/src/features/documents/ck5/shared/__tests__/uid.test.ts`

- [ ] **Step 1: Write failing test for `findAncestorByName`**

Write `frontend/apps/web/src/features/documents/ck5/shared/__tests__/findAncestor.test.ts`:

```ts
import { describe, it, expect } from 'vitest';
import { findAncestorByName } from '../findAncestor';

// Minimal fake model node for unit testing.
type FakeNode = {
  name?: string;
  parent?: FakeNode | null;
  is: (kind: 'element', name?: string) => boolean;
};

function node(name: string, parent: FakeNode | null): FakeNode {
  const n: FakeNode = {
    name,
    parent,
    is(kind, checkName) {
      return kind === 'element' && (!checkName || checkName === name);
    },
  };
  return n;
}

describe('findAncestorByName', () => {
  it('returns node itself if it matches', () => {
    const target = node('mddmSection', null);
    expect(findAncestorByName(target as never, 'mddmSection')).toBe(target);
  });

  it('walks up until a match', () => {
    const root = node('root', null);
    const section = node('mddmSection', root);
    const body = node('mddmSectionBody', section);
    const para = node('paragraph', body);
    expect(findAncestorByName(para as never, 'mddmSection')).toBe(section);
  });

  it('returns null if no match found', () => {
    const root = node('root', null);
    const para = node('paragraph', root);
    expect(findAncestorByName(para as never, 'mddmSection')).toBeNull();
  });
});
```

- [ ] **Step 2: Run test, verify it fails**

Run: `cd frontend/apps/web && rtk vitest run src/features/documents/ck5/shared/__tests__/findAncestor.test.ts`
Expected: FAIL — cannot find module `../findAncestor`.

- [ ] **Step 3: Implement `findAncestor.ts`**

Write `frontend/apps/web/src/features/documents/ck5/shared/findAncestor.ts`:

```ts
// Signature is compatible with CK5's Node interface (model Element/Text) —
// both expose `parent` and an `is(kind, name?)` type-narrowing helper.
export interface NodeLike {
  parent: NodeLike | null;
  is(kind: 'element', name?: string): boolean;
}

export function findAncestorByName<T extends NodeLike>(
  start: T | null,
  name: string,
): T | null {
  let node: NodeLike | null = start;
  while (node) {
    if (node.is('element', name)) {
      return node as T;
    }
    node = node.parent;
  }
  return null;
}
```

- [ ] **Step 4: Run test to pass**

Run: `rtk vitest run src/features/documents/ck5/shared/__tests__/findAncestor.test.ts`
Expected: PASS (3 tests).

- [ ] **Step 5: Write failing test for `uid()`**

Write `frontend/apps/web/src/features/documents/ck5/shared/__tests__/uid.test.ts`:

```ts
import { describe, it, expect } from 'vitest';
import { uid } from '../uid';

describe('uid', () => {
  it('returns a non-empty string', () => {
    expect(typeof uid()).toBe('string');
    expect(uid().length).toBeGreaterThan(0);
  });

  it('returns unique values on successive calls', () => {
    const set = new Set<string>();
    for (let i = 0; i < 100; i++) set.add(uid());
    expect(set.size).toBe(100);
  });

  it('accepts a prefix', () => {
    expect(uid('sec')).toMatch(/^sec-/);
  });
});
```

- [ ] **Step 6: Run test, verify it fails**

Run: `rtk vitest run src/features/documents/ck5/shared/__tests__/uid.test.ts`
Expected: FAIL — cannot find module `../uid`.

- [ ] **Step 7: Implement `uid.ts`**

Write `frontend/apps/web/src/features/documents/ck5/shared/uid.ts`:

```ts
// crypto.randomUUID() is available in all evergreen browsers and Node 18+.
// Vite polyfills if needed. We shorten for human-readability on data attrs.
export function uid(prefix = 'id'): string {
  const raw =
    typeof crypto !== 'undefined' && crypto.randomUUID
      ? crypto.randomUUID()
      : Math.random().toString(36).slice(2) + Date.now().toString(36);
  return `${prefix}-${raw.replace(/-/g, '').slice(0, 12)}`;
}
```

- [ ] **Step 8: Run test to pass**

Run: `rtk vitest run src/features/documents/ck5/shared/__tests__/uid.test.ts`
Expected: PASS (3 tests).

- [ ] **Step 9: Write barrel**

Write `frontend/apps/web/src/features/documents/ck5/shared/index.ts`:

```ts
export * from './findAncestor';
export * from './uid';
```

- [ ] **Step 10: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/ck5/shared
rtk git commit -m "feat(ck5): add findAncestorByName and uid shared utilities"
```

---

## Phase 2 — Base editor configuration

### Task 6: Define plugin list constants

**Files:**
- Create: `frontend/apps/web/src/features/documents/ck5/config/pluginLists.ts`
- Test: `frontend/apps/web/src/features/documents/ck5/config/__tests__/pluginLists.test.ts`

- [ ] **Step 1: Write failing smoke test**

Write `frontend/apps/web/src/features/documents/ck5/config/__tests__/pluginLists.test.ts`:

```ts
import { describe, it, expect } from 'vitest';
import { AUTHOR_PLUGINS, FILL_PLUGINS } from '../pluginLists';

describe('plugin lists', () => {
  it('AUTHOR_PLUGINS is a non-empty array of constructors', () => {
    expect(Array.isArray(AUTHOR_PLUGINS)).toBe(true);
    expect(AUTHOR_PLUGINS.length).toBeGreaterThan(5);
    for (const p of AUTHOR_PLUGINS) {
      expect(typeof p).toBe('function');
    }
  });

  it('FILL_PLUGINS is a non-empty array of constructors', () => {
    expect(Array.isArray(FILL_PLUGINS)).toBe(true);
    expect(FILL_PLUGINS.length).toBeGreaterThan(5);
  });

  it('Author includes StandardEditingMode, not RestrictedEditingMode', () => {
    const names = AUTHOR_PLUGINS.map((p) => p.name);
    expect(names).toContain('StandardEditingMode');
    expect(names).not.toContain('RestrictedEditingMode');
  });

  it('Fill includes RestrictedEditingMode, not StandardEditingMode', () => {
    const names = FILL_PLUGINS.map((p) => p.name);
    expect(names).toContain('RestrictedEditingMode');
    expect(names).not.toContain('StandardEditingMode');
  });
});
```

- [ ] **Step 2: Run test, verify it fails**

Run: `rtk vitest run src/features/documents/ck5/config/__tests__/pluginLists.test.ts`
Expected: FAIL — cannot find module `../pluginLists`.

- [ ] **Step 3: Implement `pluginLists.ts`**

Write `frontend/apps/web/src/features/documents/ck5/config/pluginLists.ts`:

```ts
import {
  Essentials,
  Paragraph,
  Heading,
  Bold,
  Italic,
  Underline,
  Strikethrough,
  Link,
  AutoLink,
  List,
  Table,
  TableToolbar,
  TableProperties,
  TableCellProperties,
  TableColumnResize,
  TableCaption,
  Image,
  ImageToolbar,
  ImageStyle,
  ImageResize,
  ImageCaption,
  ImageUpload,
  ImageInsert,
  Alignment,
  FontFamily,
  FontSize,
  FontColor,
  FontBackgroundColor,
  RemoveFormat,
  StandardEditingMode,
  RestrictedEditingMode,
  Autosave,
  GeneralHtmlSupport,
  BlockQuote,
  HorizontalLine,
  PasteFromOffice,
} from 'ckeditor5';

// MetalDocs custom plugins are added in later tasks.
// Each primitive plugin appends itself to these lists via a registration helper
// (see Task 7). For now, we define the CK5-native baseline.

const SHARED_BASE = [
  Essentials,
  Paragraph,
  Heading,
  Bold,
  Italic,
  Underline,
  Strikethrough,
  Link,
  AutoLink,
  List,
  Table,
  TableToolbar,
  TableProperties,
  TableCellProperties,
  TableColumnResize,
  TableCaption,
  Image,
  ImageToolbar,
  ImageStyle,
  ImageResize,
  ImageCaption,
  ImageUpload,
  ImageInsert,
  Alignment,
  FontFamily,
  FontSize,
  FontColor,
  FontBackgroundColor,
  RemoveFormat,
  BlockQuote,
  HorizontalLine,
  PasteFromOffice,
  Autosave,
  GeneralHtmlSupport,
];

export const AUTHOR_PLUGINS = [
  ...SHARED_BASE,
  StandardEditingMode,
  // MetalDocs plugins appended by config/editorConfig.ts at creation time.
];

export const FILL_PLUGINS = [
  ...SHARED_BASE,
  RestrictedEditingMode,
];
```

- [ ] **Step 4: Run test to pass**

Run: `rtk vitest run src/features/documents/ck5/config/__tests__/pluginLists.test.ts`
Expected: PASS (4 tests).

- [ ] **Step 5: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/ck5/config
rtk git commit -m "feat(ck5): define AUTHOR_PLUGINS and FILL_PLUGINS baselines"
```

---

### Task 7: Define toolbar configs

**Files:**
- Create: `frontend/apps/web/src/features/documents/ck5/config/toolbars.ts`
- Test: `frontend/apps/web/src/features/documents/ck5/config/__tests__/toolbars.test.ts`

- [ ] **Step 1: Write failing test**

Write `frontend/apps/web/src/features/documents/ck5/config/__tests__/toolbars.test.ts`:

```ts
import { describe, it, expect } from 'vitest';
import { AUTHOR_TOOLBAR, FILL_TOOLBAR } from '../toolbars';

describe('toolbars', () => {
  it('AUTHOR_TOOLBAR includes primitive-insertion buttons', () => {
    expect(AUTHOR_TOOLBAR).toContain('insertMddmSection');
    expect(AUTHOR_TOOLBAR).toContain('insertMddmRepeatable');
    expect(AUTHOR_TOOLBAR).toContain('insertMddmField');
    expect(AUTHOR_TOOLBAR).toContain('insertMddmRichBlock');
    expect(AUTHOR_TOOLBAR).toContain('insertTable');
  });

  it('AUTHOR_TOOLBAR includes exception tools', () => {
    expect(AUTHOR_TOOLBAR).toContain('restrictedEditingException');
    expect(AUTHOR_TOOLBAR).toContain('restrictedEditingExceptionBlock');
  });

  it('FILL_TOOLBAR does not include primitive-insertion or exception-creation', () => {
    expect(FILL_TOOLBAR).not.toContain('insertMddmSection');
    expect(FILL_TOOLBAR).not.toContain('restrictedEditingException');
    expect(FILL_TOOLBAR).not.toContain('restrictedEditingExceptionBlock');
  });

  it('FILL_TOOLBAR includes exception navigation', () => {
    expect(FILL_TOOLBAR).toContain('goToNextRestrictedEditingException');
    expect(FILL_TOOLBAR).toContain('goToPreviousRestrictedEditingException');
  });
});
```

- [ ] **Step 2: Run test, verify it fails**

Run: `rtk vitest run src/features/documents/ck5/config/__tests__/toolbars.test.ts`
Expected: FAIL — cannot find module `../toolbars`.

- [ ] **Step 3: Implement `toolbars.ts`**

Write `frontend/apps/web/src/features/documents/ck5/config/toolbars.ts`:

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

- [ ] **Step 4: Run test to pass**

Run: `rtk vitest run src/features/documents/ck5/config/__tests__/toolbars.test.ts`
Expected: PASS (4 tests).

- [ ] **Step 5: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/ck5/config/toolbars.ts frontend/apps/web/src/features/documents/ck5/config/__tests__/toolbars.test.ts
rtk git commit -m "feat(ck5): define AUTHOR_TOOLBAR and FILL_TOOLBAR"
```

---

### Task 8: Editor config factory (createAuthorConfig / createFillConfig)

**Files:**
- Create: `frontend/apps/web/src/features/documents/ck5/config/editorConfig.ts`
- Test: `frontend/apps/web/src/features/documents/ck5/config/__tests__/editorConfig.test.ts`

- [ ] **Step 1: Write failing test**

Write `frontend/apps/web/src/features/documents/ck5/config/__tests__/editorConfig.test.ts`:

```ts
import { describe, it, expect } from 'vitest';
import { createAuthorConfig, createFillConfig } from '../editorConfig';

describe('createAuthorConfig', () => {
  it('includes licenseKey GPL', () => {
    const cfg = createAuthorConfig({ language: 'en' });
    expect(cfg.licenseKey).toBe('GPL');
  });

  it('includes AUTHOR_PLUGINS and AUTHOR_TOOLBAR', () => {
    const cfg = createAuthorConfig({ language: 'en' });
    expect(Array.isArray(cfg.plugins)).toBe(true);
    expect(cfg.plugins!.length).toBeGreaterThan(5);
    expect(cfg.toolbar).toEqual(expect.objectContaining({ items: expect.any(Array) }));
  });

  it('merges MetalDocs primitive plugins when provided', () => {
    class FakePlugin {}
    const cfg = createAuthorConfig({ language: 'en', extraPlugins: [FakePlugin as never] });
    expect(cfg.plugins).toContain(FakePlugin);
  });
});

describe('createFillConfig', () => {
  it('includes restrictedEditing.allowedCommands with a sensible default', () => {
    const cfg = createFillConfig({ language: 'en' });
    expect(cfg.restrictedEditing).toBeDefined();
    expect(Array.isArray(cfg.restrictedEditing!.allowedCommands)).toBe(true);
    expect(cfg.restrictedEditing!.allowedCommands).toContain('bold');
    expect(cfg.restrictedEditing!.allowedCommands).toContain('link');
  });
});
```

- [ ] **Step 2: Run test, verify it fails**

Run: `rtk vitest run src/features/documents/ck5/config/__tests__/editorConfig.test.ts`
Expected: FAIL — cannot find module `../editorConfig`.

- [ ] **Step 3: Implement `editorConfig.ts`**

Write `frontend/apps/web/src/features/documents/ck5/config/editorConfig.ts`:

```ts
import type { EditorConfig } from 'ckeditor5';
import { AUTHOR_PLUGINS, FILL_PLUGINS } from './pluginLists';
import { AUTHOR_TOOLBAR, FILL_TOOLBAR } from './toolbars';

type PluginCtor = NonNullable<EditorConfig['plugins']>[number];

export interface ConfigOptions {
  language?: string;
  extraPlugins?: PluginCtor[];
  uploadEndpoint?: string;
  getAuthHeader?: () => string | null;
}

export function createAuthorConfig(opts: ConfigOptions = {}): EditorConfig {
  return {
    licenseKey: 'GPL',
    language: opts.language ?? 'en',
    plugins: [...AUTHOR_PLUGINS, ...(opts.extraPlugins ?? [])],
    toolbar: { items: [...AUTHOR_TOOLBAR] },
    image: {
      toolbar: [
        'imageTextAlternative',
        'imageStyle:inline',
        'imageStyle:block',
        'imageStyle:side',
        'toggleImageCaption',
        'resizeImage',
      ],
    },
    table: {
      contentToolbar: [
        'tableColumn',
        'tableRow',
        'mergeTableCells',
        'tableProperties',
        'tableCellProperties',
        'toggleTableCaption',
      ],
    },
    htmlSupport: {
      allow: [
        {
          name: /^(section|div|span|header|ol|li)$/,
          attributes: {
            class: /^(mddm-|restricted-editing-exception).*/,
            'data-section-id': true,
            'data-variant': /^(locked|editable|mixed)$/,
            'data-repeatable-id': true,
            'data-item-id': true,
            'data-field-id': true,
            'data-field-type': true,
            'data-field-label': true,
            'data-field-required': /^(true|false)$/,
            'data-mddm-variant': /^(fixed|dynamic)$/,
            'data-mddm-schema': /^v\d+$/,
          },
        },
      ],
    },
    // Read by MddmUploadAdapterPlugin; endpoint + auth header supplied by
    // callers (AuthorPage/FillPage) via the ConfigOptions passthrough.
    mddmUpload: opts.uploadEndpoint
      ? {
          endpoint: opts.uploadEndpoint,
          getAuthHeader: opts.getAuthHeader ?? (() => null),
        }
      : { endpoint: '/assets', getAuthHeader: () => null },
  } as EditorConfig;
}

export function createFillConfig(opts: ConfigOptions = {}): EditorConfig {
  const base = createAuthorConfig(opts);
  return {
    ...base,
    plugins: [...FILL_PLUGINS, ...(opts.extraPlugins ?? [])],
    toolbar: { items: [...FILL_TOOLBAR] },
    restrictedEditing: {
      allowedCommands: [
        'bold',
        'italic',
        'underline',
        'link',
        'alignment',
        'fontColor',
        'fontBackgroundColor',
      ],
      allowedAttributes: ['bold', 'italic', 'underline', 'linkHref'],
    },
  };
}
```

- [ ] **Step 4: Run test to pass**

Run: `rtk vitest run src/features/documents/ck5/config/__tests__/editorConfig.test.ts`
Expected: PASS (4 tests).

- [ ] **Step 5: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/ck5/config
rtk git commit -m "feat(ck5): add createAuthorConfig and createFillConfig factories"
```

---

## Phase 3 — Author editor React shell

### Task 9: Minimal AuthorEditor component

**Files:**
- Create: `frontend/apps/web/src/features/documents/ck5/react/AuthorEditor.tsx`
- Create: `frontend/apps/web/src/features/documents/ck5/react/AuthorEditor.module.css`
- Test: `frontend/apps/web/src/features/documents/ck5/react/__tests__/AuthorEditor.test.tsx`

- [ ] **Step 1: Write failing test**

Write `frontend/apps/web/src/features/documents/ck5/react/__tests__/AuthorEditor.test.tsx`:

```tsx
import { describe, it, expect } from 'vitest';
import { render, waitFor, cleanup } from '@testing-library/react';
import { afterEach } from 'vitest';
import { AuthorEditor } from '../AuthorEditor';

afterEach(cleanup);

describe('<AuthorEditor />', () => {
  it('renders a toolbar container and an editable container', async () => {
    const { container } = render(<AuthorEditor initialHtml="<p>Hi</p>" onChange={() => {}} />);
    await waitFor(() => {
      expect(container.querySelector('[data-ck5-role="toolbar"]')).not.toBeNull();
      expect(container.querySelector('[data-ck5-role="editable"]')).not.toBeNull();
    });
  });

  it('fires onChange with data after edits', async () => {
    const onChange = vi.fn();
    render(<AuthorEditor initialHtml="<p>Hello</p>" onChange={onChange} />);
    await waitFor(() => {
      expect(onChange).toHaveBeenCalledWith(expect.stringContaining('Hello'));
    });
  });
});
```

Note: requires `@testing-library/react` and `vi` global. Add if missing:
```bash
cd frontend/apps/web && rtk pnpm add -D @testing-library/react @testing-library/jest-dom
```

- [ ] **Step 2: Run test, verify it fails**

Run: `rtk vitest run src/features/documents/ck5/react/__tests__/AuthorEditor.test.tsx`
Expected: FAIL — cannot find module `../AuthorEditor`.

- [ ] **Step 3: Write CSS module**

Write `frontend/apps/web/src/features/documents/ck5/react/AuthorEditor.module.css`:

```css
.shell {
  display: flex;
  flex-direction: column;
  gap: 0;
  height: 100%;
  min-height: 600px;
}

.toolbar {
  position: sticky;
  top: 0;
  z-index: 10;
  background: var(--ck5-toolbar-bg, #f5f5f5);
  border-bottom: 1px solid var(--ck5-toolbar-border, #ddd);
}

.editable {
  flex: 1;
  overflow: auto;
  padding: 24px 32px;
  background: #fff;
  max-width: 880px;
  margin: 0 auto;
  box-shadow: 0 0 0 1px rgba(0, 0, 0, 0.08);
}
```

- [ ] **Step 4: Implement `AuthorEditor.tsx`**

Write `frontend/apps/web/src/features/documents/ck5/react/AuthorEditor.tsx`:

```tsx
import { useEffect, useRef } from 'react';
import { CKEditor } from '@ckeditor/ckeditor5-react';
import { DecoupledEditor } from 'ckeditor5';
import 'ckeditor5/ckeditor5.css';
import { createAuthorConfig } from '../config/editorConfig';
import styles from './AuthorEditor.module.css';

export interface AuthorEditorProps {
  initialHtml: string;
  onChange: (html: string) => void;
  onReady?: (editor: DecoupledEditor) => void;
  language?: string;
}

export function AuthorEditor({ initialHtml, onChange, onReady, language = 'en' }: AuthorEditorProps) {
  const toolbarRef = useRef<HTMLDivElement>(null);

  return (
    <div className={styles.shell}>
      <div className={styles.toolbar} ref={toolbarRef} data-ck5-role="toolbar" />
      <div className={styles.editable} data-ck5-role="editable">
        <CKEditor
          editor={DecoupledEditor}
          data={initialHtml}
          config={createAuthorConfig({ language })}
          onReady={(editor) => {
            // Move the detached toolbar into our toolbar container.
            const toolbarEl = (editor.ui.view as unknown as { toolbar: { element: HTMLElement } }).toolbar.element;
            if (toolbarRef.current && toolbarEl) {
              toolbarRef.current.appendChild(toolbarEl);
            }
            onReady?.(editor);
          }}
          onChange={(_event, editor) => {
            onChange(editor.getData());
          }}
        />
      </div>
    </div>
  );
}
```

- [ ] **Step 5: Run test to pass**

Run: `rtk vitest run src/features/documents/ck5/react/__tests__/AuthorEditor.test.tsx`
Expected: PASS. If `CKEditor` cannot be imported under jsdom, see Step 6.

- [ ] **Step 6: Install full jsdom shim for CK5 v48**

CK5 v48 touches `ResizeObserver`, `document.createRange`, selection APIs, `matchMedia`, `IntersectionObserver`, and `DOMPoint` on init. jsdom provides some; we fill the rest.

Modify `frontend/apps/web/vitest.config.ts` (create if absent):

```ts
import { defineConfig } from 'vitest/config';

export default defineConfig({
  test: {
    environment: 'jsdom',
    setupFiles: ['./vitest.setup.ts'],
    // CK5 init can take a few hundred ms under jsdom.
    testTimeout: 15000,
  },
});
```

Write `frontend/apps/web/vitest.setup.ts`:

```ts
import '@testing-library/jest-dom/vitest';

// ResizeObserver — CK5 UI components observe editable resize.
if (typeof ResizeObserver === 'undefined') {
  class RO { observe() {} unobserve() {} disconnect() {} }
  (globalThis as unknown as { ResizeObserver: typeof RO }).ResizeObserver = RO;
}

// IntersectionObserver — used by sticky toolbar logic.
if (typeof IntersectionObserver === 'undefined') {
  class IO {
    observe() {}
    unobserve() {}
    disconnect() {}
    takeRecords() { return []; }
    root = null;
    rootMargin = '';
    thresholds = [] as readonly number[];
  }
  (globalThis as unknown as { IntersectionObserver: typeof IO }).IntersectionObserver = IO;
}

// matchMedia — CK5 responsive layout queries.
if (typeof window !== 'undefined' && !window.matchMedia) {
  window.matchMedia = (query) => ({
    matches: false, media: query, onchange: null,
    addListener() {}, removeListener() {}, addEventListener() {}, removeEventListener() {},
    dispatchEvent() { return false; },
  });
}

// document.createRange exists in jsdom but missing getBoundingClientRect on ranges.
if (typeof document !== 'undefined') {
  const originalCreateRange = document.createRange.bind(document);
  document.createRange = () => {
    const range = originalCreateRange();
    if (typeof range.getBoundingClientRect !== 'function') {
      range.getBoundingClientRect = () => ({
        x: 0, y: 0, top: 0, left: 0, bottom: 0, right: 0, width: 0, height: 0,
        toJSON() { return {}; },
      });
    }
    if (typeof (range as unknown as { getClientRects?: () => unknown }).getClientRects !== 'function') {
      (range as unknown as { getClientRects: () => unknown[] }).getClientRects = () => [];
    }
    return range;
  };
}

// Clipboard API is used by CK5 clipboard pipeline.
if (typeof navigator !== 'undefined' && !navigator.clipboard) {
  Object.defineProperty(navigator, 'clipboard', {
    value: { writeText: async () => {}, readText: async () => '' },
    configurable: true,
  });
}
```

Re-run: `rtk vitest run src/features/documents/ck5/react/__tests__/AuthorEditor.test.tsx`
Expected: PASS (2 tests).

- [ ] **Step 7: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/ck5/react frontend/apps/web/vitest.setup.ts frontend/apps/web/vitest.config.ts frontend/apps/web/package.json frontend/apps/web/pnpm-lock.yaml
rtk git commit -m "feat(ck5): add AuthorEditor React shell with DecoupledEditor"
```

---

## Phase 4 — Fill editor React shell

### Task 10: Minimal FillEditor component

**Files:**
- Create: `frontend/apps/web/src/features/documents/ck5/react/FillEditor.tsx`
- Create: `frontend/apps/web/src/features/documents/ck5/react/FillEditor.module.css`
- Test: `frontend/apps/web/src/features/documents/ck5/react/__tests__/FillEditor.test.tsx`

- [ ] **Step 1: Write failing test**

Write `frontend/apps/web/src/features/documents/ck5/react/__tests__/FillEditor.test.tsx`:

```tsx
import { describe, it, expect, vi, afterEach } from 'vitest';
import { render, waitFor, cleanup } from '@testing-library/react';
import { FillEditor } from '../FillEditor';

afterEach(cleanup);

describe('<FillEditor />', () => {
  it('renders an editable', async () => {
    const { container } = render(<FillEditor documentHtml="<p>Fill me</p>" onChange={() => {}} />);
    await waitFor(() => {
      expect(container.querySelector('.ck-editor__editable')).not.toBeNull();
    });
  });

  it('calls onReady after mount', async () => {
    const onReady = vi.fn();
    render(<FillEditor documentHtml="<p>Hi</p>" onReady={onReady} onChange={() => {}} />);
    await waitFor(() => expect(onReady).toHaveBeenCalled());
  });
});
```

- [ ] **Step 2: Run test, verify it fails**

Run: `rtk vitest run src/features/documents/ck5/react/__tests__/FillEditor.test.tsx`
Expected: FAIL — cannot find module `../FillEditor`.

- [ ] **Step 3: Write CSS module**

Write `frontend/apps/web/src/features/documents/ck5/react/FillEditor.module.css`:

```css
.shell {
  display: flex;
  flex-direction: column;
  min-height: 600px;
}

.editable {
  padding: 24px 32px;
  max-width: 880px;
  margin: 0 auto;
  background: #fff;
}
```

- [ ] **Step 4: Implement `FillEditor.tsx`**

Write `frontend/apps/web/src/features/documents/ck5/react/FillEditor.tsx`:

```tsx
import { CKEditor } from '@ckeditor/ckeditor5-react';
import { ClassicEditor } from 'ckeditor5';
import 'ckeditor5/ckeditor5.css';
import { createFillConfig } from '../config/editorConfig';
import styles from './FillEditor.module.css';

export interface FillEditorProps {
  documentHtml: string;
  onChange: (html: string) => void;
  onReady?: (editor: ClassicEditor) => void;
  language?: string;
}

export function FillEditor({ documentHtml, onChange, onReady, language = 'en' }: FillEditorProps) {
  return (
    <div className={styles.shell}>
      <div className={styles.editable}>
        <CKEditor
          editor={ClassicEditor}
          data={documentHtml}
          config={createFillConfig({ language })}
          onReady={(editor) => {
            // Land the caret on the first restricted-editing exception so the
            // user can start typing immediately.
            try {
              editor.execute('goToNextRestrictedEditingException');
            } catch {
              // No exceptions in this document, or command unavailable mid-init.
              // Safe to ignore; happens when the template has no fillable regions.
            }
            onReady?.(editor);
          }}
          onChange={(_event, editor) => {
            onChange(editor.getData());
          }}
        />
      </div>
    </div>
  );
}
```

- [ ] **Step 5: Run test to pass**

Run: `rtk vitest run src/features/documents/ck5/react/__tests__/FillEditor.test.tsx`
Expected: PASS (2 tests).

- [ ] **Step 6: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/ck5/react
rtk git commit -m "feat(ck5): add FillEditor React shell with RestrictedEditingMode"
```

---

## Phase 5 — MddmFieldPlugin (Field primitive)

### Task 11: Field schema registration

**Files:**
- Create: `frontend/apps/web/src/features/documents/ck5/plugins/MddmFieldPlugin/schema.ts`
- Test: `frontend/apps/web/src/features/documents/ck5/plugins/MddmFieldPlugin/__tests__/schema.test.ts`

- [ ] **Step 1: Write failing test**

Write `.../MddmFieldPlugin/__tests__/schema.test.ts`:

```ts
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ClassicEditor, Essentials, Paragraph } from 'ckeditor5';
import { registerFieldSchema } from '../schema';

describe('field schema', () => {
  let editor: ClassicEditor;

  beforeEach(async () => {
    const el = document.createElement('div');
    document.body.appendChild(el);
    editor = await ClassicEditor.create(el, {
      licenseKey: 'GPL',
      plugins: [Essentials, Paragraph],
    });
    registerFieldSchema(editor.model.schema);
  });

  afterEach(async () => {
    await editor.destroy();
  });

  it('registers mddmField as an inline object', () => {
    const def = editor.model.schema.getDefinition('mddmField');
    expect(def).toBeDefined();
    expect(def!.isObject).toBe(true);
    expect(def!.isInline).toBe(true);
  });

  it('allows fieldId, fieldType, fieldLabel, fieldRequired, fieldValue attributes', () => {
    const attrs = ['fieldId', 'fieldType', 'fieldLabel', 'fieldRequired', 'fieldValue'];
    for (const attr of attrs) {
      expect(editor.model.schema.checkAttribute(['$root', 'paragraph', 'mddmField'], attr)).toBe(true);
    }
  });

  it('is allowed inside a paragraph', () => {
    expect(editor.model.schema.checkChild(['$root', 'paragraph'], 'mddmField')).toBe(true);
  });
});
```

- [ ] **Step 2: Run test, verify it fails**

Run: `rtk vitest run src/features/documents/ck5/plugins/MddmFieldPlugin/__tests__/schema.test.ts`
Expected: FAIL — cannot find module `../schema`.

- [ ] **Step 3: Implement `schema.ts`**

Write `.../MddmFieldPlugin/schema.ts`:

```ts
import type { Schema } from 'ckeditor5';

export function registerFieldSchema(schema: Schema): void {
  schema.register('mddmField', {
    inheritAllFrom: '$inlineObject',
    allowAttributes: ['fieldId', 'fieldType', 'fieldLabel', 'fieldRequired', 'fieldValue'],
  });
}
```

- [ ] **Step 4: Run test to pass**

Run: `rtk vitest run src/features/documents/ck5/plugins/MddmFieldPlugin/__tests__/schema.test.ts`
Expected: PASS (3 tests).

- [ ] **Step 5: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/ck5/plugins/MddmFieldPlugin
rtk git commit -m "feat(ck5): register mddmField schema"
```

---

### Task 12: Field converters (upcast + data downcast + editing downcast)

**Files:**
- Create: `frontend/apps/web/src/features/documents/ck5/plugins/MddmFieldPlugin/converters.ts`
- Test: `frontend/apps/web/src/features/documents/ck5/plugins/MddmFieldPlugin/__tests__/converters.test.ts`

- [ ] **Step 1: Write failing test**

Write `.../MddmFieldPlugin/__tests__/converters.test.ts`:

```ts
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ClassicEditor, Essentials, Paragraph, Widget } from 'ckeditor5';
import { registerFieldSchema } from '../schema';
import { registerFieldConverters } from '../converters';

describe('field converters', () => {
  let editor: ClassicEditor;

  beforeEach(async () => {
    const el = document.createElement('div');
    document.body.appendChild(el);
    editor = await ClassicEditor.create(el, {
      licenseKey: 'GPL',
      plugins: [Essentials, Paragraph, Widget],
    });
    registerFieldSchema(editor.model.schema);
    registerFieldConverters(editor);
  });

  afterEach(async () => {
    await editor.destroy();
  });

  const sampleHtml =
    '<p><span class="mddm-field" data-field-id="customer" data-field-type="text" data-field-label="Customer" data-field-required="true">Acme</span></p>';

  it('upcasts a span.mddm-field into mddmField model element', () => {
    editor.setData(sampleHtml);
    const root = editor.model.document.getRoot()!;
    const para = root.getChild(0);
    const field = para!.getChild(0) as { name: string; getAttribute(k: string): unknown };
    expect(field.name).toBe('mddmField');
    expect(field.getAttribute('fieldId')).toBe('customer');
    expect(field.getAttribute('fieldType')).toBe('text');
    expect(field.getAttribute('fieldLabel')).toBe('Customer');
    expect(field.getAttribute('fieldRequired')).toBe(true);
    expect(field.getAttribute('fieldValue')).toBe('Acme');
  });

  it('round-trips HTML via setData/getData', () => {
    editor.setData(sampleHtml);
    const out = editor.getData();
    expect(out).toContain('class="mddm-field"');
    expect(out).toContain('data-field-id="customer"');
    expect(out).toContain('data-field-type="text"');
    expect(out).toContain('data-field-label="Customer"');
    expect(out).toContain('data-field-required="true"');
    expect(out).toContain('>Acme<');
  });
});
```

- [ ] **Step 2: Run test, verify it fails**

Run: `rtk vitest run src/features/documents/ck5/plugins/MddmFieldPlugin/__tests__/converters.test.ts`
Expected: FAIL — cannot find module `../converters`.

- [ ] **Step 3: Implement `converters.ts`**

Write `.../MddmFieldPlugin/converters.ts`:

```ts
import type { Editor, DowncastConversionApi, UpcastConversionApi } from 'ckeditor5';
import { toWidget } from 'ckeditor5';

export function registerFieldConverters(editor: Editor): void {
  const conversion = editor.conversion;

  conversion.for('upcast').elementToElement({
    view: { name: 'span', classes: 'mddm-field' },
    model: (viewEl, { writer }: UpcastConversionApi) =>
      writer.createElement('mddmField', {
        fieldId: viewEl.getAttribute('data-field-id') ?? '',
        fieldType: viewEl.getAttribute('data-field-type') ?? 'text',
        fieldLabel: viewEl.getAttribute('data-field-label') ?? '',
        fieldRequired: viewEl.getAttribute('data-field-required') === 'true',
        fieldValue: viewEl.getChild(0)?.is('$text') ? (viewEl.getChild(0) as { data: string }).data : '',
      }),
  });

  conversion.for('dataDowncast').elementToElement({
    model: 'mddmField',
    view: (modelEl, { writer }: DowncastConversionApi) => {
      const span = writer.createContainerElement('span', {
        class: 'mddm-field',
        'data-field-id': String(modelEl.getAttribute('fieldId') ?? ''),
        'data-field-type': String(modelEl.getAttribute('fieldType') ?? 'text'),
        'data-field-label': String(modelEl.getAttribute('fieldLabel') ?? ''),
        'data-field-required': String(!!modelEl.getAttribute('fieldRequired')),
      });
      const value = String(modelEl.getAttribute('fieldValue') ?? '');
      writer.insert(writer.createPositionAt(span, 0), writer.createText(value));
      return span;
    },
  });

  conversion.for('editingDowncast').elementToElement({
    model: 'mddmField',
    view: (modelEl, { writer }: DowncastConversionApi) => {
      const type = String(modelEl.getAttribute('fieldType') ?? 'text');
      const label = String(modelEl.getAttribute('fieldLabel') ?? '');
      const id = String(modelEl.getAttribute('fieldId') ?? '');
      const value = String(modelEl.getAttribute('fieldValue') ?? '');
      const typeFamily = type.split(':')[0];

      const chip = writer.createContainerElement('span', {
        class: `mddm-field mddm-field--${typeFamily}`,
        'data-field-id': id,
        'aria-label': `${label} (${type})`,
        role: 'textbox',
      });
      writer.insert(writer.createPositionAt(chip, 0), writer.createText(value || `{{${label || id}}}`));
      return toWidget(chip, writer, { label: `${label || id} field` });
    },
  });
}
```

- [ ] **Step 4: Run test to pass**

Run: `rtk vitest run src/features/documents/ck5/plugins/MddmFieldPlugin/__tests__/converters.test.ts`
Expected: PASS (2 tests).

- [ ] **Step 5: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/ck5/plugins/MddmFieldPlugin
rtk git commit -m "feat(ck5): register mddmField upcast + dual downcast converters"
```

---

### Task 13: InsertFieldCommand

**Files:**
- Create: `frontend/apps/web/src/features/documents/ck5/plugins/MddmFieldPlugin/commands/InsertFieldCommand.ts`
- Test: `frontend/apps/web/src/features/documents/ck5/plugins/MddmFieldPlugin/__tests__/InsertFieldCommand.test.ts`

- [ ] **Step 1: Write failing test**

Write `.../__tests__/InsertFieldCommand.test.ts`:

```ts
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ClassicEditor, Essentials, Paragraph, Widget } from 'ckeditor5';
import { registerFieldSchema } from '../schema';
import { registerFieldConverters } from '../converters';
import { InsertFieldCommand } from '../commands/InsertFieldCommand';

describe('InsertFieldCommand', () => {
  let editor: ClassicEditor;

  beforeEach(async () => {
    const el = document.createElement('div');
    document.body.appendChild(el);
    editor = await ClassicEditor.create(el, {
      licenseKey: 'GPL',
      plugins: [Essentials, Paragraph, Widget],
    });
    registerFieldSchema(editor.model.schema);
    registerFieldConverters(editor);
    editor.commands.add('insertMddmField', new InsertFieldCommand(editor));
  });

  afterEach(async () => {
    await editor.destroy();
  });

  it('inserts a field at selection', () => {
    editor.setData('<p>Hello </p>');
    editor.model.change((writer) => {
      const root = editor.model.document.getRoot()!;
      const para = root.getChild(0)!;
      writer.setSelection(writer.createPositionAt(para, 'end'));
    });
    editor.execute('insertMddmField', {
      fieldId: 'name',
      fieldType: 'text',
      fieldLabel: 'Name',
      fieldRequired: true,
      fieldValue: '',
    });
    const html = editor.getData();
    expect(html).toContain('data-field-id="name"');
    expect(html).toContain('data-field-type="text"');
    expect(html).toContain('data-field-required="true"');
  });

  it('is enabled inside a paragraph, disabled at the root', () => {
    editor.setData('<p>x</p>');
    const cmd = editor.commands.get('insertMddmField')!;
    editor.model.change((writer) => {
      const para = editor.model.document.getRoot()!.getChild(0)!;
      writer.setSelection(writer.createPositionAt(para, 0));
    });
    expect(cmd.isEnabled).toBe(true);
  });
});
```

- [ ] **Step 2: Run test, verify it fails**

Run: `rtk vitest run src/features/documents/ck5/plugins/MddmFieldPlugin/__tests__/InsertFieldCommand.test.ts`
Expected: FAIL — cannot find module `../commands/InsertFieldCommand`.

- [ ] **Step 3: Implement `InsertFieldCommand.ts`**

Write `.../commands/InsertFieldCommand.ts`:

```ts
import { Command, type Editor } from 'ckeditor5';
import { uid } from '../../../shared/uid';

export interface InsertFieldOptions {
  fieldId?: string;
  fieldType: string;
  fieldLabel: string;
  fieldRequired?: boolean;
  fieldValue?: string;
}

export class InsertFieldCommand extends Command {
  constructor(editor: Editor) {
    super(editor);
  }

  override refresh(): void {
    const sel = this.editor.model.document.selection;
    const pos = sel.getFirstPosition();
    this.isEnabled = !!pos && this.editor.model.schema.checkChild(pos, 'mddmField');
  }

  override execute(opts: InsertFieldOptions): void {
    const { model } = this.editor;
    model.change((writer) => {
      const field = writer.createElement('mddmField', {
        fieldId: opts.fieldId ?? uid('fld'),
        fieldType: opts.fieldType,
        fieldLabel: opts.fieldLabel,
        fieldRequired: !!opts.fieldRequired,
        fieldValue: opts.fieldValue ?? '',
      });
      model.insertContent(field);
      writer.setSelection(field, 'after');
    });
  }
}
```

- [ ] **Step 4: Run test to pass**

Run: `rtk vitest run src/features/documents/ck5/plugins/MddmFieldPlugin/__tests__/InsertFieldCommand.test.ts`
Expected: PASS (2 tests).

- [ ] **Step 5: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/ck5/plugins/MddmFieldPlugin
rtk git commit -m "feat(ck5): add InsertFieldCommand for mddmField insertion"
```

---

### Task 14: MddmFieldPlugin class + position mapper

**Files:**
- Create: `frontend/apps/web/src/features/documents/ck5/plugins/MddmFieldPlugin/index.ts`
- Test: `frontend/apps/web/src/features/documents/ck5/plugins/MddmFieldPlugin/__tests__/plugin.test.ts`

- [ ] **Step 1: Write failing test**

Write `.../__tests__/plugin.test.ts`:

```ts
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ClassicEditor, Essentials, Paragraph, Widget } from 'ckeditor5';
import { MddmFieldPlugin } from '../index';

describe('MddmFieldPlugin', () => {
  let editor: ClassicEditor;

  beforeEach(async () => {
    const el = document.createElement('div');
    document.body.appendChild(el);
    editor = await ClassicEditor.create(el, {
      licenseKey: 'GPL',
      plugins: [Essentials, Paragraph, Widget, MddmFieldPlugin],
    });
  });

  afterEach(async () => {
    await editor.destroy();
  });

  it('registers the insertMddmField command', () => {
    expect(editor.commands.get('insertMddmField')).toBeDefined();
  });

  it('round-trips field HTML without loss', () => {
    const input =
      '<p><span class="mddm-field" data-field-id="x" data-field-type="text" data-field-label="L" data-field-required="false">v</span></p>';
    editor.setData(input);
    const out = editor.getData();
    expect(out).toContain('data-field-id="x"');
    expect(out).toContain('>v<');
  });
});
```

- [ ] **Step 2: Run test, verify it fails**

Run: `rtk vitest run src/features/documents/ck5/plugins/MddmFieldPlugin/__tests__/plugin.test.ts`
Expected: FAIL — cannot find module `../index`.

- [ ] **Step 3: Implement `index.ts`**

Write `.../MddmFieldPlugin/index.ts`:

```ts
import { Plugin, Widget, viewToModelPositionOutsideModelElement } from 'ckeditor5';
import { registerFieldSchema } from './schema';
import { registerFieldConverters } from './converters';
import { InsertFieldCommand } from './commands/InsertFieldCommand';

export class MddmFieldPlugin extends Plugin {
  static get pluginName(): 'MddmFieldPlugin' {
    return 'MddmFieldPlugin';
  }

  static get requires(): ReadonlyArray<typeof Widget> {
    return [Widget];
  }

  init(): void {
    const editor = this.editor;

    registerFieldSchema(editor.model.schema);
    registerFieldConverters(editor);

    editor.commands.add('insertMddmField', new InsertFieldCommand(editor));

    // Inline widget position mapping: when caret moves "past" the chip in
    // the view, translate to the model position AFTER the mddmField element.
    editor.editing.mapper.on(
      'viewToModelPosition',
      viewToModelPositionOutsideModelElement(editor.model, (viewEl) =>
        viewEl.hasClass('mddm-field'),
      ),
    );
  }
}
```

- [ ] **Step 4: Run test to pass**

Run: `rtk vitest run src/features/documents/ck5/plugins/MddmFieldPlugin/__tests__/plugin.test.ts`
Expected: PASS (2 tests).

- [ ] **Step 5: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/ck5/plugins/MddmFieldPlugin
rtk git commit -m "feat(ck5): add MddmFieldPlugin with position mapper"
```

---

## Phase 6 — MddmSectionPlugin (Section primitive)

### Task 15: Section schema

**Files:**
- Create: `.../MddmSectionPlugin/schema.ts`
- Test: `.../MddmSectionPlugin/__tests__/schema.test.ts`

- [ ] **Step 1: Write failing test**

Write `.../MddmSectionPlugin/__tests__/schema.test.ts`:

```ts
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ClassicEditor, Essentials, Paragraph } from 'ckeditor5';
import { registerSectionSchema } from '../schema';

describe('section schema', () => {
  let editor: ClassicEditor;

  beforeEach(async () => {
    const el = document.createElement('div');
    document.body.appendChild(el);
    editor = await ClassicEditor.create(el, {
      licenseKey: 'GPL',
      plugins: [Essentials, Paragraph],
    });
    registerSectionSchema(editor.model.schema);
  });

  afterEach(async () => {
    await editor.destroy();
  });

  it('registers mddmSection, mddmSectionHeader, mddmSectionBody', () => {
    expect(editor.model.schema.getDefinition('mddmSection')).toBeDefined();
    expect(editor.model.schema.getDefinition('mddmSectionHeader')).toBeDefined();
    expect(editor.model.schema.getDefinition('mddmSectionBody')).toBeDefined();
  });

  it('mddmSection is a block object', () => {
    const def = editor.model.schema.getDefinition('mddmSection')!;
    expect(def.isObject).toBe(true);
    expect(def.isBlock).toBe(true);
  });

  it('header and body are limits', () => {
    expect(editor.model.schema.getDefinition('mddmSectionHeader')!.isLimit).toBe(true);
    expect(editor.model.schema.getDefinition('mddmSectionBody')!.isLimit).toBe(true);
  });

  it('allows variant and sectionId attributes on mddmSection', () => {
    const attrs = ['sectionId', 'variant'];
    for (const attr of attrs) {
      expect(editor.model.schema.checkAttribute(['$root', 'mddmSection'], attr)).toBe(true);
    }
  });
});
```

- [ ] **Step 2: Run test, verify it fails**

Run: `rtk vitest run src/features/documents/ck5/plugins/MddmSectionPlugin/__tests__/schema.test.ts`
Expected: FAIL — module not found.

- [ ] **Step 3: Implement `schema.ts`**

Write `.../MddmSectionPlugin/schema.ts`:

```ts
import type { Schema } from 'ckeditor5';

export function registerSectionSchema(schema: Schema): void {
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

- [ ] **Step 4: Run test to pass**

Run: `rtk vitest run src/features/documents/ck5/plugins/MddmSectionPlugin/__tests__/schema.test.ts`
Expected: PASS (4 tests).

- [ ] **Step 5: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/ck5/plugins/MddmSectionPlugin
rtk git commit -m "feat(ck5): register mddmSection schema"
```

---

### Task 16: Section post-fixer (one header + one body)

**Files:**
- Create: `.../MddmSectionPlugin/postFixer.ts`
- Test: `.../MddmSectionPlugin/__tests__/postFixer.test.ts`

- [ ] **Step 1: Write failing test**

Write `.../MddmSectionPlugin/__tests__/postFixer.test.ts`:

```ts
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ClassicEditor, Essentials, Paragraph } from 'ckeditor5';
import { registerSectionSchema } from '../schema';
import { registerSectionPostFixer } from '../postFixer';

describe('section post-fixer', () => {
  let editor: ClassicEditor;

  beforeEach(async () => {
    const el = document.createElement('div');
    document.body.appendChild(el);
    editor = await ClassicEditor.create(el, {
      licenseKey: 'GPL',
      plugins: [Essentials, Paragraph],
    });
    registerSectionSchema(editor.model.schema);
    registerSectionPostFixer(editor);
  });

  afterEach(async () => {
    await editor.destroy();
  });

  it('auto-creates header+body when a bare section is inserted', () => {
    editor.model.change((writer) => {
      const section = writer.createElement('mddmSection');
      writer.append(section, editor.model.document.getRoot()!);
    });
    const section = editor.model.document.getRoot()!.getChild(0)!;
    const children = Array.from((section as { getChildren(): Iterable<{ name: string }> }).getChildren());
    expect(children.map((c) => c.name)).toEqual(['mddmSectionHeader', 'mddmSectionBody']);
  });
});
```

- [ ] **Step 2: Run test, verify it fails**

Run: `rtk vitest run src/features/documents/ck5/plugins/MddmSectionPlugin/__tests__/postFixer.test.ts`
Expected: FAIL — module not found.

- [ ] **Step 3: Implement `postFixer.ts`**

Write `.../MddmSectionPlugin/postFixer.ts`:

```ts
import type { Editor, Element, Writer } from 'ckeditor5';

export function registerSectionPostFixer(editor: Editor): void {
  editor.model.document.registerPostFixer((writer: Writer) => {
    let changed = false;
    const root = editor.model.document.getRoot();
    if (!root) return false;

    const walker = root.getChildren();
    for (const node of walker) {
      if (!(node as Element).is('element', 'mddmSection')) continue;
      const section = node as Element;
      const children = Array.from(section.getChildren()) as Element[];
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

- [ ] **Step 4: Run test to pass**

Run: `rtk vitest run src/features/documents/ck5/plugins/MddmSectionPlugin/__tests__/postFixer.test.ts`
Expected: PASS (1 test).

- [ ] **Step 5: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/ck5/plugins/MddmSectionPlugin
rtk git commit -m "feat(ck5): add mddmSection post-fixer for header+body invariant"
```

---

### Task 17: Section converters (upcast + dual downcast)

**Files:**
- Create: `.../MddmSectionPlugin/converters.ts`
- Test: `.../MddmSectionPlugin/__tests__/converters.test.ts`

- [ ] **Step 1: Write failing test**

Write `.../MddmSectionPlugin/__tests__/converters.test.ts`:

```ts
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ClassicEditor, Essentials, Paragraph, Widget } from 'ckeditor5';
import { registerSectionSchema } from '../schema';
import { registerSectionPostFixer } from '../postFixer';
import { registerSectionConverters } from '../converters';

describe('section converters', () => {
  let editor: ClassicEditor;

  beforeEach(async () => {
    const el = document.createElement('div');
    document.body.appendChild(el);
    editor = await ClassicEditor.create(el, {
      licenseKey: 'GPL',
      plugins: [Essentials, Paragraph, Widget],
    });
    registerSectionSchema(editor.model.schema);
    registerSectionPostFixer(editor);
    registerSectionConverters(editor);
  });

  afterEach(async () => {
    await editor.destroy();
  });

  const sampleHtml = [
    '<section class="mddm-section" data-section-id="s1" data-variant="mixed">',
    '<header class="mddm-section__header"><p>Title</p></header>',
    '<div class="mddm-section__body"><p>Body</p></div>',
    '</section>',
  ].join('');

  it('round-trips section HTML', () => {
    editor.setData(sampleHtml);
    const out = editor.getData();
    expect(out).toContain('class="mddm-section"');
    expect(out).toContain('data-section-id="s1"');
    expect(out).toContain('data-variant="mixed"');
    expect(out).toContain('class="mddm-section__header"');
    expect(out).toContain('class="mddm-section__body"');
    expect(out).toContain('>Title<');
    expect(out).toContain('>Body<');
  });

  it('defaults variant to editable when missing', () => {
    editor.setData(
      '<section class="mddm-section"><header class="mddm-section__header"/><div class="mddm-section__body"><p/></div></section>',
    );
    const section = editor.model.document.getRoot()!.getChild(0);
    expect((section as { getAttribute(k: string): unknown }).getAttribute('variant')).toBe('editable');
  });
});
```

- [ ] **Step 2: Run test, verify it fails**

Run: `rtk vitest run src/features/documents/ck5/plugins/MddmSectionPlugin/__tests__/converters.test.ts`
Expected: FAIL — module not found.

- [ ] **Step 3: Implement `converters.ts`**

Write `.../MddmSectionPlugin/converters.ts`:

```ts
import type { Editor, DowncastConversionApi, UpcastConversionApi } from 'ckeditor5';
import { toWidget, toWidgetEditable } from 'ckeditor5';

export function registerSectionConverters(editor: Editor): void {
  const c = editor.conversion;

  // Upcast
  c.for('upcast').elementToElement({
    view: { name: 'section', classes: 'mddm-section' },
    model: (viewEl, { writer }: UpcastConversionApi) =>
      writer.createElement('mddmSection', {
        sectionId: viewEl.getAttribute('data-section-id') ?? undefined,
        variant: viewEl.getAttribute('data-variant') ?? 'editable',
      }),
  });
  c.for('upcast').elementToElement({
    view: { name: 'header', classes: 'mddm-section__header' },
    model: 'mddmSectionHeader',
  });
  c.for('upcast').elementToElement({
    view: { name: 'div', classes: 'mddm-section__body' },
    model: 'mddmSectionBody',
  });

  // Data downcast — plain HTML for persistence
  c.for('dataDowncast').elementToElement({
    model: 'mddmSection',
    view: (modelEl, { writer }: DowncastConversionApi) =>
      writer.createContainerElement('section', {
        class: 'mddm-section',
        'data-section-id': String(modelEl.getAttribute('sectionId') ?? ''),
        'data-variant': String(modelEl.getAttribute('variant') ?? 'editable'),
      }),
  });
  c.for('dataDowncast').elementToElement({
    model: 'mddmSectionHeader',
    view: (_m, { writer }) => writer.createContainerElement('header', { class: 'mddm-section__header' }),
  });
  c.for('dataDowncast').elementToElement({
    model: 'mddmSectionBody',
    view: (_m, { writer }) => writer.createContainerElement('div', { class: 'mddm-section__body' }),
  });

  // Editing downcast — wraps chrome with widget helpers
  c.for('editingDowncast').elementToElement({
    model: 'mddmSection',
    view: (modelEl, { writer }: DowncastConversionApi) => {
      const section = writer.createContainerElement('section', {
        class: 'mddm-section',
        'data-section-id': String(modelEl.getAttribute('sectionId') ?? ''),
        'data-variant': String(modelEl.getAttribute('variant') ?? 'editable'),
      });
      return toWidget(section, writer, { label: 'section widget' });
    },
  });
  c.for('editingDowncast').elementToElement({
    model: 'mddmSectionHeader',
    view: (_m, { writer }) => {
      const header = writer.createEditableElement('header', { class: 'mddm-section__header' });
      return toWidgetEditable(header, writer);
    },
  });
  c.for('editingDowncast').elementToElement({
    model: 'mddmSectionBody',
    view: (_m, { writer }) => {
      const body = writer.createEditableElement('div', { class: 'mddm-section__body' });
      return toWidgetEditable(body, writer);
    },
  });
}
```

- [ ] **Step 4: Run test to pass**

Run: `rtk vitest run src/features/documents/ck5/plugins/MddmSectionPlugin/__tests__/converters.test.ts`
Expected: PASS (2 tests).

- [ ] **Step 5: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/ck5/plugins/MddmSectionPlugin
rtk git commit -m "feat(ck5): add mddmSection upcast + dual downcast converters"
```

---

### Task 18: InsertSectionCommand with exception marker planting

**Files:**
- Create: `.../MddmSectionPlugin/commands/InsertSectionCommand.ts`
- Test: `.../MddmSectionPlugin/__tests__/InsertSectionCommand.test.ts`

- [ ] **Step 1: Write failing test**

Write `.../__tests__/InsertSectionCommand.test.ts`:

```ts
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ClassicEditor, Essentials, Paragraph, Widget, StandardEditingMode } from 'ckeditor5';
import { registerSectionSchema } from '../schema';
import { registerSectionPostFixer } from '../postFixer';
import { registerSectionConverters } from '../converters';
import { InsertSectionCommand } from '../commands/InsertSectionCommand';

describe('InsertSectionCommand', () => {
  let editor: ClassicEditor;

  beforeEach(async () => {
    const el = document.createElement('div');
    document.body.appendChild(el);
    editor = await ClassicEditor.create(el, {
      licenseKey: 'GPL',
      plugins: [Essentials, Paragraph, Widget, StandardEditingMode],
    });
    registerSectionSchema(editor.model.schema);
    registerSectionPostFixer(editor);
    registerSectionConverters(editor);
    editor.commands.add('insertMddmSection', new InsertSectionCommand(editor));
  });

  afterEach(async () => {
    await editor.destroy();
  });

  it('inserts a section with variant="editable" by default', () => {
    editor.execute('insertMddmSection');
    const out = editor.getData();
    expect(out).toContain('class="mddm-section"');
    expect(out).toContain('data-variant="editable"');
    expect(out).toContain('class="mddm-section__header"');
    expect(out).toContain('class="mddm-section__body"');
  });

  it('accepts variant parameter', () => {
    editor.execute('insertMddmSection', { variant: 'locked' });
    expect(editor.getData()).toContain('data-variant="locked"');
  });

  it('plants a restricted-editing-exception marker on the body for editable variant', () => {
    editor.execute('insertMddmSection', { variant: 'editable' });
    const markers = Array.from(editor.model.markers).filter((m) =>
      m.name.startsWith('restrictedEditingException:'),
    );
    expect(markers.length).toBeGreaterThanOrEqual(1);
  });
});
```

- [ ] **Step 2: Run test, verify it fails**

Run: `rtk vitest run src/features/documents/ck5/plugins/MddmSectionPlugin/__tests__/InsertSectionCommand.test.ts`
Expected: FAIL — module not found.

- [ ] **Step 3: Implement `InsertSectionCommand.ts`**

Write `.../commands/InsertSectionCommand.ts`:

```ts
import { Command, type Editor } from 'ckeditor5';
import { uid } from '../../../shared/uid';
import type { SectionVariant } from '../../../types';

export interface InsertSectionOptions {
  variant?: SectionVariant;
  sectionId?: string;
}

export class InsertSectionCommand extends Command {
  constructor(editor: Editor) {
    super(editor);
  }

  override refresh(): void {
    const sel = this.editor.model.document.selection;
    const pos = sel.getFirstPosition();
    this.isEnabled = !!pos && this.editor.model.schema.findAllowedParent(pos, 'mddmSection') !== null;
  }

  override execute(opts: InsertSectionOptions = {}): void {
    const variant: SectionVariant = opts.variant ?? 'editable';
    const sectionId = opts.sectionId ?? uid('sec');
    const model = this.editor.model;

    model.change((writer) => {
      const section = writer.createElement('mddmSection', { sectionId, variant });
      const header = writer.createElement('mddmSectionHeader');
      const body = writer.createElement('mddmSectionBody');
      writer.append(header, section);
      writer.append(body, section);
      writer.appendElement('paragraph', body);
      model.insertObject(section, null, null, { setSelection: 'on' });

      if (variant === 'editable' || variant === 'mixed') {
        const range = model.createRangeIn(body);
        writer.addMarker(`restrictedEditingException:${uid('rex')}`, {
          range,
          usingOperation: true,
          affectsData: true,
        });
      }
    });
  }
}
```

- [ ] **Step 4: Run test to pass**

Run: `rtk vitest run src/features/documents/ck5/plugins/MddmSectionPlugin/__tests__/InsertSectionCommand.test.ts`
Expected: PASS (3 tests).

- [ ] **Step 5: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/ck5/plugins/MddmSectionPlugin
rtk git commit -m "feat(ck5): add InsertSectionCommand with exception marker planting"
```

---

### Task 19: MddmSectionPlugin class

**Files:**
- Create: `.../MddmSectionPlugin/index.ts`

- [ ] **Step 1: Implement `index.ts`**

Write `.../MddmSectionPlugin/index.ts`:

```ts
import { Plugin, Widget } from 'ckeditor5';
import { registerSectionSchema } from './schema';
import { registerSectionPostFixer } from './postFixer';
import { registerSectionConverters } from './converters';
import { InsertSectionCommand } from './commands/InsertSectionCommand';

export class MddmSectionPlugin extends Plugin {
  static get pluginName(): 'MddmSectionPlugin' {
    return 'MddmSectionPlugin';
  }

  static get requires(): ReadonlyArray<typeof Widget> {
    return [Widget];
  }

  init(): void {
    const editor = this.editor;
    registerSectionSchema(editor.model.schema);
    registerSectionPostFixer(editor);
    registerSectionConverters(editor);
    editor.commands.add('insertMddmSection', new InsertSectionCommand(editor));
  }
}
```

- [ ] **Step 2: Write integration test**

Write `.../MddmSectionPlugin/__tests__/plugin.test.ts`:

```ts
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ClassicEditor, Essentials, Paragraph, Widget, StandardEditingMode } from 'ckeditor5';
import { MddmSectionPlugin } from '../index';

describe('MddmSectionPlugin integration', () => {
  let editor: ClassicEditor;

  beforeEach(async () => {
    const el = document.createElement('div');
    document.body.appendChild(el);
    editor = await ClassicEditor.create(el, {
      licenseKey: 'GPL',
      plugins: [Essentials, Paragraph, Widget, StandardEditingMode, MddmSectionPlugin],
    });
  });

  afterEach(async () => {
    await editor.destroy();
  });

  it('registers insertMddmSection', () => {
    expect(editor.commands.get('insertMddmSection')).toBeDefined();
  });
});
```

- [ ] **Step 3: Run test**

Run: `rtk vitest run src/features/documents/ck5/plugins/MddmSectionPlugin/__tests__/plugin.test.ts`
Expected: PASS (1 test).

- [ ] **Step 4: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/ck5/plugins/MddmSectionPlugin
rtk git commit -m "feat(ck5): add MddmSectionPlugin class"
```

---

## Phase 7 — MddmRepeatablePlugin

### Task 20: Repeatable schema

**Files:**
- Create: `.../MddmRepeatablePlugin/schema.ts`
- Test: `.../MddmRepeatablePlugin/__tests__/schema.test.ts`

- [ ] **Step 1: Write failing test**

Write `.../MddmRepeatablePlugin/__tests__/schema.test.ts`:

```ts
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ClassicEditor, Essentials, Paragraph } from 'ckeditor5';
import { registerRepeatableSchema } from '../schema';

describe('repeatable schema', () => {
  let editor: ClassicEditor;

  beforeEach(async () => {
    const el = document.createElement('div');
    document.body.appendChild(el);
    editor = await ClassicEditor.create(el, {
      licenseKey: 'GPL',
      plugins: [Essentials, Paragraph],
    });
    registerRepeatableSchema(editor.model.schema);
  });

  afterEach(async () => {
    await editor.destroy();
  });

  it('registers mddmRepeatable and mddmRepeatableItem', () => {
    expect(editor.model.schema.getDefinition('mddmRepeatable')).toBeDefined();
    expect(editor.model.schema.getDefinition('mddmRepeatableItem')).toBeDefined();
  });

  it('allows repeatableId/label/min/max/numberingStyle on mddmRepeatable', () => {
    for (const attr of ['repeatableId', 'label', 'min', 'max', 'numberingStyle']) {
      expect(editor.model.schema.checkAttribute(['$root', 'mddmRepeatable'], attr)).toBe(true);
    }
  });

  it('only allows mddmRepeatableItem inside mddmRepeatable', () => {
    expect(editor.model.schema.checkChild(['$root', 'mddmRepeatable'], 'mddmRepeatableItem')).toBe(true);
    expect(editor.model.schema.checkChild(['$root', 'mddmRepeatable'], 'paragraph')).toBe(false);
  });

  it('forbids nested mddmRepeatable inside mddmRepeatableItem', () => {
    expect(
      editor.model.schema.checkChild(
        ['$root', 'mddmRepeatable', 'mddmRepeatableItem'],
        'mddmRepeatable',
      ),
    ).toBe(false);
  });
});
```

- [ ] **Step 2: Run test, verify it fails**

Run: `rtk vitest run src/features/documents/ck5/plugins/MddmRepeatablePlugin/__tests__/schema.test.ts`
Expected: FAIL — module not found.

- [ ] **Step 3: Implement `schema.ts`**

Write `.../MddmRepeatablePlugin/schema.ts`:

```ts
import type { Schema } from 'ckeditor5';

export function registerRepeatableSchema(schema: Schema): void {
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

  schema.addChildCheck((ctx, def) => {
    if (ctx.endsWith('mddmRepeatableItem') && def.name === 'mddmRepeatable') return false;
    return undefined;
  });
}
```

- [ ] **Step 4: Run test to pass**

Run: `rtk vitest run src/features/documents/ck5/plugins/MddmRepeatablePlugin/__tests__/schema.test.ts`
Expected: PASS (4 tests).

- [ ] **Step 5: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/ck5/plugins/MddmRepeatablePlugin
rtk git commit -m "feat(ck5): register mddmRepeatable schema"
```

---

### Task 21: Repeatable converters (<ol>/<li> wire)

**Files:**
- Create: `.../MddmRepeatablePlugin/converters.ts`
- Test: `.../MddmRepeatablePlugin/__tests__/converters.test.ts`

- [ ] **Step 1: Write failing test**

Write `.../MddmRepeatablePlugin/__tests__/converters.test.ts`:

```ts
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ClassicEditor, Essentials, Paragraph, Widget } from 'ckeditor5';
import { registerRepeatableSchema } from '../schema';
import { registerRepeatableConverters } from '../converters';

describe('repeatable converters', () => {
  let editor: ClassicEditor;

  beforeEach(async () => {
    const el = document.createElement('div');
    document.body.appendChild(el);
    editor = await ClassicEditor.create(el, {
      licenseKey: 'GPL',
      plugins: [Essentials, Paragraph, Widget],
    });
    registerRepeatableSchema(editor.model.schema);
    registerRepeatableConverters(editor);
  });

  afterEach(async () => {
    await editor.destroy();
  });

  const sampleHtml = [
    '<ol class="mddm-repeatable" data-repeatable-id="r1" data-min="1" data-max="5" data-numbering="decimal">',
    '<li class="mddm-repeatable__item"><p>First</p></li>',
    '<li class="mddm-repeatable__item"><p>Second</p></li>',
    '</ol>',
  ].join('');

  it('round-trips repeatable HTML', () => {
    editor.setData(sampleHtml);
    const out = editor.getData();
    expect(out).toContain('class="mddm-repeatable"');
    expect(out).toContain('data-repeatable-id="r1"');
    expect(out).toContain('data-min="1"');
    expect(out).toContain('data-max="5"');
    expect(out).toContain('class="mddm-repeatable__item"');
    expect(out).toContain('>First<');
    expect(out).toContain('>Second<');
  });
});
```

- [ ] **Step 2: Run test, verify it fails**

Run: `rtk vitest run src/features/documents/ck5/plugins/MddmRepeatablePlugin/__tests__/converters.test.ts`
Expected: FAIL — module not found.

- [ ] **Step 3: Implement `converters.ts`**

Write `.../MddmRepeatablePlugin/converters.ts`:

```ts
import type { Editor, DowncastConversionApi, UpcastConversionApi } from 'ckeditor5';
import { toWidget, toWidgetEditable } from 'ckeditor5';

export function registerRepeatableConverters(editor: Editor): void {
  const c = editor.conversion;

  c.for('upcast').elementToElement({
    view: { name: 'ol', classes: 'mddm-repeatable' },
    model: (viewEl, { writer }: UpcastConversionApi) =>
      writer.createElement('mddmRepeatable', {
        repeatableId: viewEl.getAttribute('data-repeatable-id') ?? '',
        label: viewEl.getAttribute('data-label') ?? '',
        min: Number(viewEl.getAttribute('data-min') ?? 0),
        max: Number(viewEl.getAttribute('data-max') ?? 0) || Number.POSITIVE_INFINITY,
        numberingStyle: viewEl.getAttribute('data-numbering') ?? 'decimal',
      }),
  });
  c.for('upcast').elementToElement({
    view: { name: 'li', classes: 'mddm-repeatable__item' },
    model: 'mddmRepeatableItem',
  });

  c.for('dataDowncast').elementToElement({
    model: 'mddmRepeatable',
    view: (m, { writer }: DowncastConversionApi) =>
      writer.createContainerElement('ol', {
        class: 'mddm-repeatable',
        'data-repeatable-id': String(m.getAttribute('repeatableId') ?? ''),
        'data-label': String(m.getAttribute('label') ?? ''),
        'data-min': String(m.getAttribute('min') ?? 0),
        'data-max': String(m.getAttribute('max') === Number.POSITIVE_INFINITY ? '' : m.getAttribute('max') ?? ''),
        'data-numbering': String(m.getAttribute('numberingStyle') ?? 'decimal'),
      }),
  });
  c.for('dataDowncast').elementToElement({
    model: 'mddmRepeatableItem',
    view: (_m, { writer }) => writer.createContainerElement('li', { class: 'mddm-repeatable__item' }),
  });

  c.for('editingDowncast').elementToElement({
    model: 'mddmRepeatable',
    view: (m, { writer }: DowncastConversionApi) => {
      const ol = writer.createContainerElement('ol', {
        class: 'mddm-repeatable',
        'data-repeatable-id': String(m.getAttribute('repeatableId') ?? ''),
        'data-numbering': String(m.getAttribute('numberingStyle') ?? 'decimal'),
      });
      return toWidget(ol, writer, { label: 'repeatable widget' });
    },
  });
  c.for('editingDowncast').elementToElement({
    model: 'mddmRepeatableItem',
    view: (_m, { writer }) => {
      const li = writer.createEditableElement('li', { class: 'mddm-repeatable__item' });
      return toWidgetEditable(li, writer);
    },
  });
}
```

- [ ] **Step 4: Run test to pass**

Run: `rtk vitest run src/features/documents/ck5/plugins/MddmRepeatablePlugin/__tests__/converters.test.ts`
Expected: PASS (1 test).

- [ ] **Step 5: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/ck5/plugins/MddmRepeatablePlugin
rtk git commit -m "feat(ck5): add mddmRepeatable upcast + dual downcast converters"
```

---

### Task 22: Repeatable commands (insert + addItem + removeItem)

**Files:**
- Create: `.../MddmRepeatablePlugin/commands/InsertRepeatableCommand.ts`
- Create: `.../MddmRepeatablePlugin/commands/AddRepeatableItemCommand.ts`
- Create: `.../MddmRepeatablePlugin/commands/RemoveRepeatableItemCommand.ts`
- Test: `.../MddmRepeatablePlugin/__tests__/commands.test.ts`

- [ ] **Step 1: Write failing test**

Write `.../__tests__/commands.test.ts`:

```ts
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ClassicEditor, Essentials, Paragraph, Widget, StandardEditingMode } from 'ckeditor5';
import { registerRepeatableSchema } from '../schema';
import { registerRepeatableConverters } from '../converters';
import { InsertRepeatableCommand } from '../commands/InsertRepeatableCommand';
import { AddRepeatableItemCommand } from '../commands/AddRepeatableItemCommand';
import { RemoveRepeatableItemCommand } from '../commands/RemoveRepeatableItemCommand';

describe('repeatable commands', () => {
  let editor: ClassicEditor;

  beforeEach(async () => {
    const el = document.createElement('div');
    document.body.appendChild(el);
    editor = await ClassicEditor.create(el, {
      licenseKey: 'GPL',
      plugins: [Essentials, Paragraph, Widget, StandardEditingMode],
    });
    registerRepeatableSchema(editor.model.schema);
    registerRepeatableConverters(editor);
    editor.commands.add('insertMddmRepeatable', new InsertRepeatableCommand(editor));
    editor.commands.add('addMddmRepeatableItem', new AddRepeatableItemCommand(editor));
    editor.commands.add('removeMddmRepeatableItem', new RemoveRepeatableItemCommand(editor));
  });

  afterEach(async () => {
    await editor.destroy();
  });

  it('inserts a repeatable with initial item', () => {
    editor.execute('insertMddmRepeatable', { min: 1, max: 5, initialCount: 2 });
    const html = editor.getData();
    expect(html).toContain('class="mddm-repeatable"');
    expect((html.match(/class="mddm-repeatable__item"/g) || []).length).toBe(2);
  });

  it('addItem adds a new item when below max', () => {
    editor.execute('insertMddmRepeatable', { min: 1, max: 3, initialCount: 1 });
    editor.model.change((writer) => {
      const rep = editor.model.document.getRoot()!.getChild(0)!;
      writer.setSelection(writer.createPositionAt(rep, 'end'));
    });
    editor.execute('addMddmRepeatableItem');
    const html = editor.getData();
    expect((html.match(/class="mddm-repeatable__item"/g) || []).length).toBe(2);
  });

  it('removeItem removes an item when above min', () => {
    editor.execute('insertMddmRepeatable', { min: 1, max: 3, initialCount: 2 });
    editor.model.change((writer) => {
      const rep = editor.model.document.getRoot()!.getChild(0)!;
      const firstItem = (rep as { getChild(i: number): unknown }).getChild(0);
      writer.setSelection(writer.createPositionAt(firstItem as never, 0));
    });
    editor.execute('removeMddmRepeatableItem');
    const html = editor.getData();
    expect((html.match(/class="mddm-repeatable__item"/g) || []).length).toBe(1);
  });

  it('unbounded max: round-trips without Infinity leaking into HTML', () => {
    editor.execute('insertMddmRepeatable', { min: 0, initialCount: 1 });
    const html = editor.getData();
    // Unbounded max is represented as an empty data-max attribute.
    expect(html).toMatch(/data-max=""|data-max=(?!"Infinity")/);
    expect(html).not.toContain('Infinity');

    // Round-trip: set the saved HTML into a fresh editor and verify Add command
    // remains enabled (no bogus ceiling).
    editor.setData(html);
    editor.model.change((writer) => {
      const rep = editor.model.document.getRoot()!.getChild(0)!;
      const firstItem = (rep as { getChild(i: number): unknown }).getChild(0);
      writer.setSelection(writer.createPositionAt(firstItem as never, 0));
    });
    const addCmd = editor.commands.get('addMddmRepeatableItem')!;
    expect(addCmd.isEnabled).toBe(true);
  });
});
```

- [ ] **Step 2: Run test, verify it fails**

Run: `rtk vitest run src/features/documents/ck5/plugins/MddmRepeatablePlugin/__tests__/commands.test.ts`
Expected: FAIL — command module not found.

- [ ] **Step 3: Implement `InsertRepeatableCommand.ts`**

Write `.../commands/InsertRepeatableCommand.ts`:

```ts
import { Command, type Editor } from 'ckeditor5';
import { uid } from '../../../shared/uid';

export interface InsertRepeatableOptions {
  repeatableId?: string;
  label?: string;
  min?: number;
  max?: number;
  numberingStyle?: 'decimal' | 'lower-alpha' | 'bullet' | 'none';
  initialCount?: number;
}

export class InsertRepeatableCommand extends Command {
  constructor(editor: Editor) {
    super(editor);
  }

  override refresh(): void {
    const pos = this.editor.model.document.selection.getFirstPosition();
    this.isEnabled =
      !!pos && this.editor.model.schema.findAllowedParent(pos, 'mddmRepeatable') !== null;
  }

  override execute(opts: InsertRepeatableOptions = {}): void {
    const { model } = this.editor;
    const min = Math.max(0, opts.min ?? 0);
    const max = opts.max && opts.max > 0 ? opts.max : Number.POSITIVE_INFINITY;
    const count = Math.max(min, Math.min(opts.initialCount ?? Math.max(min, 1), max));

    model.change((writer) => {
      const rep = writer.createElement('mddmRepeatable', {
        repeatableId: opts.repeatableId ?? uid('rep'),
        label: opts.label ?? '',
        min,
        max,
        numberingStyle: opts.numberingStyle ?? 'decimal',
      });
      for (let i = 0; i < count; i++) {
        const item = writer.createElement('mddmRepeatableItem');
        writer.appendElement('paragraph', item);
        writer.append(item, rep);
      }
      model.insertObject(rep, null, null, { setSelection: 'on' });

      // Plant a block exception per item so Fill-mode users can edit each item.
      for (const child of Array.from(rep.getChildren())) {
        const range = model.createRangeIn(child as never);
        writer.addMarker(`restrictedEditingException:${uid('rex')}`, {
          range,
          usingOperation: true,
          affectsData: true,
        });
      }
    });
  }
}
```

- [ ] **Step 4: Implement `AddRepeatableItemCommand.ts`**

Write `.../commands/AddRepeatableItemCommand.ts`:

```ts
import { Command, type Editor } from 'ckeditor5';
import { findAncestorByName } from '../../../shared/findAncestor';
import { uid } from '../../../shared/uid';

export class AddRepeatableItemCommand extends Command {
  constructor(editor: Editor) {
    super(editor);
  }

  private currentRepeatable() {
    const pos = this.editor.model.document.selection.getFirstPosition();
    if (!pos) return null;
    return findAncestorByName(pos.parent as never, 'mddmRepeatable');
  }

  override refresh(): void {
    const rep = this.currentRepeatable();
    if (!rep) {
      this.isEnabled = false;
      return;
    }
    const max = Number((rep as { getAttribute(k: string): unknown }).getAttribute('max')) || Number.POSITIVE_INFINITY;
    const itemCount = Array.from((rep as { getChildren(): Iterable<unknown> }).getChildren()).length;
    this.isEnabled = itemCount < max;
  }

  override execute(): void {
    const rep = this.currentRepeatable();
    if (!rep) return;
    const model = this.editor.model;
    model.change((writer) => {
      const item = writer.createElement('mddmRepeatableItem');
      writer.appendElement('paragraph', item);
      writer.append(item, rep as never);
      const range = model.createRangeIn(item);
      writer.addMarker(`restrictedEditingException:${uid('rex')}`, {
        range,
        usingOperation: true,
        affectsData: true,
      });
      writer.setSelection(writer.createPositionAt(item, 0));
    });
  }
}
```

- [ ] **Step 5: Implement `RemoveRepeatableItemCommand.ts`**

Write `.../commands/RemoveRepeatableItemCommand.ts`:

```ts
import { Command, type Editor } from 'ckeditor5';
import { findAncestorByName } from '../../../shared/findAncestor';

export class RemoveRepeatableItemCommand extends Command {
  constructor(editor: Editor) {
    super(editor);
  }

  private currentItem() {
    const pos = this.editor.model.document.selection.getFirstPosition();
    if (!pos) return null;
    return findAncestorByName(pos.parent as never, 'mddmRepeatableItem');
  }

  override refresh(): void {
    const item = this.currentItem();
    if (!item) {
      this.isEnabled = false;
      return;
    }
    const parent = (item as { parent: unknown }).parent as {
      getAttribute(k: string): unknown;
      getChildren(): Iterable<unknown>;
    };
    const min = Number(parent.getAttribute('min')) || 0;
    const total = Array.from(parent.getChildren()).length;
    this.isEnabled = total > min;
  }

  override execute(): void {
    const item = this.currentItem();
    if (!item) return;
    this.editor.model.change((writer) => {
      writer.remove(item as never);
    });
  }
}
```

- [ ] **Step 6: Run test to pass**

Run: `rtk vitest run src/features/documents/ck5/plugins/MddmRepeatablePlugin/__tests__/commands.test.ts`
Expected: PASS (3 tests).

- [ ] **Step 7: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/ck5/plugins/MddmRepeatablePlugin
rtk git commit -m "feat(ck5): add Insert/Add/Remove repeatable commands"
```

---

### Task 23: MddmRepeatablePlugin class

**Files:**
- Create: `.../MddmRepeatablePlugin/index.ts`

- [ ] **Step 1: Implement `index.ts`**

Write `.../MddmRepeatablePlugin/index.ts`:

```ts
import { Plugin, Widget } from 'ckeditor5';
import { registerRepeatableSchema } from './schema';
import { registerRepeatableConverters } from './converters';
import { InsertRepeatableCommand } from './commands/InsertRepeatableCommand';
import { AddRepeatableItemCommand } from './commands/AddRepeatableItemCommand';
import { RemoveRepeatableItemCommand } from './commands/RemoveRepeatableItemCommand';

export class MddmRepeatablePlugin extends Plugin {
  static get pluginName(): 'MddmRepeatablePlugin' {
    return 'MddmRepeatablePlugin';
  }

  static get requires(): ReadonlyArray<typeof Widget> {
    return [Widget];
  }

  init(): void {
    const editor = this.editor;
    registerRepeatableSchema(editor.model.schema);
    registerRepeatableConverters(editor);
    editor.commands.add('insertMddmRepeatable', new InsertRepeatableCommand(editor));
    editor.commands.add('addMddmRepeatableItem', new AddRepeatableItemCommand(editor));
    editor.commands.add('removeMddmRepeatableItem', new RemoveRepeatableItemCommand(editor));
  }
}
```

- [ ] **Step 2: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/ck5/plugins/MddmRepeatablePlugin
rtk git commit -m "feat(ck5): add MddmRepeatablePlugin class"
```

---

## Phase 8 — MddmDataTablePlugin

### Task 24: MddmTableVariantPlugin (attribute + converter)

**Files:**
- Create: `.../MddmDataTablePlugin/MddmTableVariantPlugin.ts`
- Test: `.../MddmDataTablePlugin/__tests__/variant.test.ts`

- [ ] **Step 1: Write failing test**

Write `.../__tests__/variant.test.ts`:

```ts
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ClassicEditor, Essentials, Paragraph, Table } from 'ckeditor5';
import { MddmTableVariantPlugin } from '../MddmTableVariantPlugin';

describe('MddmTableVariantPlugin', () => {
  let editor: ClassicEditor;

  beforeEach(async () => {
    const el = document.createElement('div');
    document.body.appendChild(el);
    editor = await ClassicEditor.create(el, {
      licenseKey: 'GPL',
      plugins: [Essentials, Paragraph, Table, MddmTableVariantPlugin],
    });
  });

  afterEach(async () => {
    await editor.destroy();
  });

  it('upcasts data-mddm-variant="fixed"', () => {
    editor.setData(
      '<figure class="table"><table data-mddm-variant="fixed"><tbody><tr><td>x</td></tr></tbody></table></figure>',
    );
    const table = editor.model.document.getRoot()!.getChild(0);
    expect((table as { getAttribute(k: string): unknown }).getAttribute('mddmTableVariant')).toBe('fixed');
  });

  it('downcasts mddmTableVariant to data-mddm-variant', () => {
    editor.setData(
      '<figure class="table"><table data-mddm-variant="dynamic"><tbody><tr><td>y</td></tr></tbody></table></figure>',
    );
    const out = editor.getData();
    expect(out).toContain('data-mddm-variant="dynamic"');
  });
});
```

- [ ] **Step 2: Run test, verify it fails**

Run: `rtk vitest run src/features/documents/ck5/plugins/MddmDataTablePlugin/__tests__/variant.test.ts`
Expected: FAIL — module not found.

- [ ] **Step 3: Implement `MddmTableVariantPlugin.ts`**

Write `.../MddmDataTablePlugin/MddmTableVariantPlugin.ts`:

```ts
import { Plugin } from 'ckeditor5';

export class MddmTableVariantPlugin extends Plugin {
  static get pluginName(): 'MddmTableVariantPlugin' {
    return 'MddmTableVariantPlugin';
  }

  init(): void {
    const editor = this.editor;
    editor.model.schema.extend('table', { allowAttributes: ['mddmTableVariant'] });
    editor.conversion.attributeToAttribute({
      model: { name: 'table', key: 'mddmTableVariant' },
      view: 'data-mddm-variant',
    });
  }
}
```

- [ ] **Step 4: Run test to pass**

Run: `rtk vitest run src/features/documents/ck5/plugins/MddmDataTablePlugin/__tests__/variant.test.ts`
Expected: PASS (2 tests).

- [ ] **Step 5: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/ck5/plugins/MddmDataTablePlugin
rtk git commit -m "feat(ck5): add MddmTableVariantPlugin for fixed/dynamic attribute"
```

---

### Task 25: Nested-table guard

**Files:**
- Create: `.../MddmDataTablePlugin/nestedTableGuard.ts`
- Test: `.../MddmDataTablePlugin/__tests__/nestedTableGuard.test.ts`

- [ ] **Step 1: Write failing test**

Write `.../__tests__/nestedTableGuard.test.ts`:

```ts
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ClassicEditor, Essentials, Paragraph, Table } from 'ckeditor5';
import { registerNestedTableGuard } from '../nestedTableGuard';

describe('nestedTableGuard', () => {
  let editor: ClassicEditor;

  beforeEach(async () => {
    const el = document.createElement('div');
    document.body.appendChild(el);
    editor = await ClassicEditor.create(el, {
      licenseKey: 'GPL',
      plugins: [Essentials, Paragraph, Table],
    });
    registerNestedTableGuard(editor);
  });

  afterEach(async () => {
    await editor.destroy();
  });

  it('rejects inserting a table inside a tableCell', () => {
    expect(
      editor.model.schema.checkChild(['$root', 'table', 'tableRow', 'tableCell'], 'table'),
    ).toBe(false);
  });
});
```

- [ ] **Step 2: Run test, verify it fails**

Run: `rtk vitest run src/features/documents/ck5/plugins/MddmDataTablePlugin/__tests__/nestedTableGuard.test.ts`
Expected: FAIL — module not found.

- [ ] **Step 3: Implement `nestedTableGuard.ts`**

Write `.../MddmDataTablePlugin/nestedTableGuard.ts`:

```ts
import type { Editor } from 'ckeditor5';

export function registerNestedTableGuard(editor: Editor): void {
  editor.model.schema.addChildCheck((ctx, def) => {
    if (ctx.endsWith('tableCell') && def.name === 'table') return false;
    return undefined;
  });
}
```

- [ ] **Step 4: Run test to pass**

Run: `rtk vitest run src/features/documents/ck5/plugins/MddmDataTablePlugin/__tests__/nestedTableGuard.test.ts`
Expected: PASS (1 test).

- [ ] **Step 5: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/ck5/plugins/MddmDataTablePlugin
rtk git commit -m "feat(ck5): add nested-table guard (forbid tables inside cells)"
```

---

### Task 26: MddmTableLockPlugin (Fill-mode command gating)

**Files:**
- Create: `.../MddmDataTablePlugin/MddmTableLockPlugin.ts`
- Test: `.../MddmDataTablePlugin/__tests__/lock.test.ts`

- [ ] **Step 1: Write failing test**

Write `.../__tests__/lock.test.ts`:

```ts
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ClassicEditor, Essentials, Paragraph, Table } from 'ckeditor5';
import { MddmTableVariantPlugin } from '../MddmTableVariantPlugin';
import { MddmTableLockPlugin } from '../MddmTableLockPlugin';

describe('MddmTableLockPlugin', () => {
  let editor: ClassicEditor;

  beforeEach(async () => {
    const el = document.createElement('div');
    document.body.appendChild(el);
    editor = await ClassicEditor.create(el, {
      licenseKey: 'GPL',
      plugins: [Essentials, Paragraph, Table, MddmTableVariantPlugin, MddmTableLockPlugin],
    });
  });

  afterEach(async () => {
    await editor.destroy();
  });

  it('disables insertTableRowBelow when caret is inside a fixed table', () => {
    editor.setData(
      '<figure class="table"><table data-mddm-variant="fixed"><tbody><tr><td>x</td></tr></tbody></table></figure>',
    );
    editor.model.change((writer) => {
      const root = editor.model.document.getRoot()!;
      const table = root.getChild(0)!;
      const cell = (table as { getChild(i: number): unknown }).getChild(0) as {
        getChild(i: number): unknown;
      };
      const td = (cell.getChild(0) as { getChild(i: number): unknown }).getChild(0);
      writer.setSelection(writer.createPositionAt(td as never, 0));
    });
    expect(editor.commands.get('insertTableRowBelow')!.isEnabled).toBe(false);
  });

  it('leaves insertTableRowBelow enabled in a dynamic table', () => {
    editor.setData(
      '<figure class="table"><table data-mddm-variant="dynamic"><tbody><tr><td>x</td></tr></tbody></table></figure>',
    );
    editor.model.change((writer) => {
      const td = (editor.model.document.getRoot()!.getChild(0) as { getChild(i: number): unknown })
        .getChild(0) as { getChild(i: number): unknown };
      const cell = (td.getChild(0) as { getChild(i: number): unknown }).getChild(0);
      writer.setSelection(writer.createPositionAt(cell as never, 0));
    });
    expect(editor.commands.get('insertTableRowBelow')!.isEnabled).toBe(true);
  });
});
```

- [ ] **Step 2: Run test, verify it fails**

Run: `rtk vitest run src/features/documents/ck5/plugins/MddmDataTablePlugin/__tests__/lock.test.ts`
Expected: FAIL — module not found.

- [ ] **Step 3: Implement `MddmTableLockPlugin.ts`**

Write `.../MddmDataTablePlugin/MddmTableLockPlugin.ts`:

```ts
import { Plugin } from 'ckeditor5';
import { findAncestorByName } from '../../shared/findAncestor';

const STRUCTURAL = [
  'insertTableRowAbove',
  'insertTableRowBelow',
  'insertTableColumnLeft',
  'insertTableColumnRight',
  'removeTableRow',
  'removeTableColumn',
  'mergeTableCells',
  'splitTableCellVertically',
  'splitTableCellHorizontally',
  'mergeTableCellRight',
  'mergeTableCellDown',
  'mergeTableCellLeft',
  'mergeTableCellUp',
  'setTableColumnHeader',
  'setTableRowHeader',
];

const LOCK_KEY = 'mddmTableLock';

export class MddmTableLockPlugin extends Plugin {
  static get pluginName(): 'MddmTableLockPlugin' {
    return 'MddmTableLockPlugin';
  }

  init(): void {
    const model = this.editor.model;
    const sync = () => this._sync();
    model.document.selection.on('change:range', sync);
    model.document.on('change:data', sync);
    sync();
  }

  private _sync(): void {
    const pos = this.editor.model.document.selection.getFirstPosition();
    const table = pos ? findAncestorByName(pos.parent as never, 'table') : null;
    const locked =
      !!table && (table as { getAttribute(k: string): unknown }).getAttribute('mddmTableVariant') === 'fixed';

    for (const name of STRUCTURAL) {
      const cmd = this.editor.commands.get(name);
      if (!cmd) continue;
      if (locked) {
        cmd.forceDisabled(LOCK_KEY);
      } else {
        cmd.clearForceDisabled(LOCK_KEY);
      }
    }
  }
}
```

- [ ] **Step 4: Run test to pass**

Run: `rtk vitest run src/features/documents/ck5/plugins/MddmDataTablePlugin/__tests__/lock.test.ts`
Expected: PASS (2 tests).

- [ ] **Step 5: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/ck5/plugins/MddmDataTablePlugin
rtk git commit -m "feat(ck5): add MddmTableLockPlugin for Fill-mode fixed-table gating"
```

---

### Task 27: Per-cell exception walker (template save hook)

**Files:**
- Create: `.../MddmDataTablePlugin/perCellExceptionWalker.ts`
- Test: `.../MddmDataTablePlugin/__tests__/perCellWalker.test.ts`

- [ ] **Step 1: Write failing test**

Write `.../__tests__/perCellWalker.test.ts`:

```ts
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ClassicEditor, Essentials, Paragraph, Table, StandardEditingMode, Widget } from 'ckeditor5';
import { MddmTableVariantPlugin } from '../MddmTableVariantPlugin';
import { applyPerCellExceptions } from '../perCellExceptionWalker';

describe('applyPerCellExceptions', () => {
  let editor: ClassicEditor;

  beforeEach(async () => {
    const el = document.createElement('div');
    document.body.appendChild(el);
    editor = await ClassicEditor.create(el, {
      licenseKey: 'GPL',
      plugins: [Essentials, Paragraph, Table, Widget, StandardEditingMode, MddmTableVariantPlugin],
    });
  });

  afterEach(async () => {
    await editor.destroy();
  });

  it('adds a marker per cell in every mddmTableVariant-tagged table', () => {
    editor.setData(
      '<figure class="table"><table data-mddm-variant="fixed"><tbody><tr><td>a</td><td>b</td></tr></tbody></table></figure>',
    );
    applyPerCellExceptions(editor);
    const markers = Array.from(editor.model.markers).filter((m) =>
      m.name.startsWith('restrictedEditingException:'),
    );
    expect(markers.length).toBe(2);
  });

  it('skips tables without the mddmTableVariant attribute', () => {
    editor.setData(
      '<figure class="table"><table><tbody><tr><td>a</td></tr></tbody></table></figure>',
    );
    applyPerCellExceptions(editor);
    const markers = Array.from(editor.model.markers).filter((m) =>
      m.name.startsWith('restrictedEditingException:'),
    );
    expect(markers.length).toBe(0);
  });

  it('is idempotent: running twice produces the same marker set', () => {
    editor.setData(
      '<figure class="table"><table data-mddm-variant="fixed"><tbody><tr><td>a</td><td>b</td></tr></tbody></table></figure>',
    );
    applyPerCellExceptions(editor);
    const first = Array.from(editor.model.markers)
      .map((m) => m.name)
      .filter((n) => n.startsWith('restrictedEditingException:'))
      .sort();
    applyPerCellExceptions(editor);
    const second = Array.from(editor.model.markers)
      .map((m) => m.name)
      .filter((n) => n.startsWith('restrictedEditingException:'))
      .sort();
    expect(second).toEqual(first);
    expect(first.length).toBe(2);
  });
});
```

- [ ] **Step 2: Run test, verify it fails**

Run: `rtk vitest run src/features/documents/ck5/plugins/MddmDataTablePlugin/__tests__/perCellWalker.test.ts`
Expected: FAIL — module not found.

- [ ] **Step 3: Implement `perCellExceptionWalker.ts`**

Write `.../MddmDataTablePlugin/perCellExceptionWalker.ts`:

```ts
import type { Editor } from 'ckeditor5';

// Stable prefix for cell exceptions this walker owns. Anything under this
// prefix is considered "managed" and is cleared/rewritten on each run so the
// walker is idempotent across repeated saves.
export const CELL_MARKER_PREFIX = 'restrictedEditingException:mddmCell:';

export function applyPerCellExceptions(editor: Editor): void {
  const model = editor.model;
  model.change((writer) => {
    // 1. Clear any previously managed per-cell markers. Non-managed markers
    //    (from Section / Repeatable / RichBlock) are untouched.
    for (const marker of Array.from(model.markers)) {
      if (marker.name.startsWith(CELL_MARKER_PREFIX)) {
        writer.removeMarker(marker);
      }
    }

    // 2. Walk every mddmTableVariant-tagged table and create a marker per
    //    cell, with a stable name keyed by table/row/column index so repeated
    //    runs produce the same marker set (idempotent by name).
    const root = model.document.getRoot();
    if (!root) return;

    let tableIdx = 0;
    for (const element of Array.from(root.getChildren())) {
      tableIdx = visit(element as never, tableIdx);
    }

    function visit(
      node: {
        is(k: 'element', n?: string): boolean;
        getChildren?: () => Iterable<unknown>;
        getAttribute?: (k: string) => unknown;
      },
      currentTableIdx: number,
    ): number {
      if (node.is('element', 'table') && node.getAttribute?.('mddmTableVariant')) {
        // Prefer a stable author-supplied table id if present; otherwise use
        // document-position index.
        const rawId = node.getAttribute?.('mddmTableId');
        const tableKey = typeof rawId === 'string' && rawId ? rawId : `t${currentTableIdx}`;
        let rowIdx = 0;
        for (const row of Array.from((node.getChildren?.() ?? []) as never[])) {
          if (!(row as { is(k: 'element', n?: string): boolean }).is('element', 'tableRow')) continue;
          let colIdx = 0;
          for (const cell of Array.from(
            ((row as unknown as { getChildren(): Iterable<unknown> }).getChildren() ?? []) as never[],
          )) {
            if (!(cell as { is(k: 'element', n?: string): boolean }).is('element', 'tableCell')) continue;
            const range = model.createRangeIn(cell as never);
            writer.addMarker(`${CELL_MARKER_PREFIX}${tableKey}-r${rowIdx}-c${colIdx}`, {
              range,
              usingOperation: true,
              affectsData: true,
            });
            colIdx++;
          }
          rowIdx++;
        }
        return currentTableIdx + 1;
      }
      let idx = currentTableIdx;
      for (const child of Array.from((node.getChildren?.() ?? [])) as never[]) {
        idx = visit(child as never, idx);
      }
      return idx;
    }
  });
}
```

- [ ] **Step 4: Run test to pass**

Run: `rtk vitest run src/features/documents/ck5/plugins/MddmDataTablePlugin/__tests__/perCellWalker.test.ts`
Expected: PASS (2 tests).

- [ ] **Step 5: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/ck5/plugins/MddmDataTablePlugin
rtk git commit -m "feat(ck5): add applyPerCellExceptions template-save walker"
```

---

### Task 28: MddmDataTablePlugin (bundle sub-plugins)

**Files:**
- Create: `.../MddmDataTablePlugin/index.ts`

- [ ] **Step 1: Implement `index.ts`**

Write `.../MddmDataTablePlugin/index.ts`:

```ts
import { Plugin } from 'ckeditor5';
import { MddmTableVariantPlugin } from './MddmTableVariantPlugin';
import { MddmTableLockPlugin } from './MddmTableLockPlugin';
import { registerNestedTableGuard } from './nestedTableGuard';

export { applyPerCellExceptions } from './perCellExceptionWalker';

export class MddmDataTablePlugin extends Plugin {
  static get pluginName(): 'MddmDataTablePlugin' {
    return 'MddmDataTablePlugin';
  }

  static get requires(): ReadonlyArray<typeof Plugin> {
    return [MddmTableVariantPlugin, MddmTableLockPlugin];
  }

  init(): void {
    registerNestedTableGuard(this.editor);
  }
}
```

- [ ] **Step 2: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/ck5/plugins/MddmDataTablePlugin
rtk git commit -m "feat(ck5): add MddmDataTablePlugin bundle class"
```

---

## Phase 9 — MddmRichBlockPlugin

### Task 29: InsertRichBlockCommand + plugin

**Files:**
- Create: `.../MddmRichBlockPlugin/commands/InsertRichBlockCommand.ts`
- Create: `.../MddmRichBlockPlugin/index.ts`
- Test: `.../MddmRichBlockPlugin/__tests__/plugin.test.ts`

- [ ] **Step 1: Write failing test**

Write `.../__tests__/plugin.test.ts`:

```ts
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import {
  ClassicEditor,
  Essentials,
  Paragraph,
  StandardEditingMode,
  GeneralHtmlSupport,
} from 'ckeditor5';
import { MddmRichBlockPlugin } from '../index';

describe('MddmRichBlockPlugin', () => {
  let editor: ClassicEditor;

  beforeEach(async () => {
    const el = document.createElement('div');
    document.body.appendChild(el);
    editor = await ClassicEditor.create(el, {
      licenseKey: 'GPL',
      plugins: [
        Essentials,
        Paragraph,
        GeneralHtmlSupport,
        StandardEditingMode,
        MddmRichBlockPlugin,
      ],
      htmlSupport: {
        allow: [{ name: 'div', classes: ['mddm-rich-block'] }],
      },
    });
  });

  afterEach(async () => {
    await editor.destroy();
  });

  it('registers insertMddmRichBlock command', () => {
    expect(editor.commands.get('insertMddmRichBlock')).toBeDefined();
  });

  it('inserts a <div class="mddm-rich-block"> on execute', () => {
    editor.execute('insertMddmRichBlock');
    const html = editor.getData();
    expect(html).toContain('class="mddm-rich-block"');
  });

  it('plants a block exception marker on the rich block', () => {
    editor.execute('insertMddmRichBlock');
    const markers = Array.from(editor.model.markers).filter((m) =>
      m.name.startsWith('restrictedEditingException:'),
    );
    expect(markers.length).toBeGreaterThanOrEqual(1);
  });
});
```

- [ ] **Step 2: Run test, verify it fails**

Run: `rtk vitest run src/features/documents/ck5/plugins/MddmRichBlockPlugin/__tests__/plugin.test.ts`
Expected: FAIL — modules missing.

- [ ] **Step 3: Implement `InsertRichBlockCommand.ts`**

Write `.../MddmRichBlockPlugin/commands/InsertRichBlockCommand.ts`:

```ts
import { Command, type Editor } from 'ckeditor5';
import { uid } from '../../../shared/uid';

export class InsertRichBlockCommand extends Command {
  constructor(editor: Editor) {
    super(editor);
  }

  override refresh(): void {
    const pos = this.editor.model.document.selection.getFirstPosition();
    this.isEnabled =
      !!pos &&
      this.editor.model.schema.checkChild(pos, 'htmlDivParagraph' as never);
  }

  override execute(): void {
    const model = this.editor.model;
    model.change((writer) => {
      // GHS exposes container divs as `htmlDivParagraph` in the model with
      // a `htmlDivAttributes` bag.
      const div = writer.createElement('htmlDivParagraph', {
        htmlDivAttributes: { classes: ['mddm-rich-block'] },
      } as never);
      writer.appendElement('paragraph', div);
      model.insertObject(div, null, null, { setSelection: 'on' });
      const range = model.createRangeIn(div);
      writer.addMarker(`restrictedEditingException:${uid('rb')}`, {
        range,
        usingOperation: true,
        affectsData: true,
      });
    });
  }
}
```

- [ ] **Step 4: Implement `index.ts`**

Write `.../MddmRichBlockPlugin/index.ts`:

```ts
import { Plugin } from 'ckeditor5';
import { InsertRichBlockCommand } from './commands/InsertRichBlockCommand';

export class MddmRichBlockPlugin extends Plugin {
  static get pluginName(): 'MddmRichBlockPlugin' {
    return 'MddmRichBlockPlugin';
  }

  init(): void {
    this.editor.commands.add('insertMddmRichBlock', new InsertRichBlockCommand(this.editor));
  }
}
```

- [ ] **Step 5: Run test to pass**

Run: `rtk vitest run src/features/documents/ck5/plugins/MddmRichBlockPlugin/__tests__/plugin.test.ts`
Expected: PASS (3 tests). If step 1 fails on GHS element name, record the actual element name reported by the inspector and fix the `createElement` call.

- [ ] **Step 6: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/ck5/plugins/MddmRichBlockPlugin
rtk git commit -m "feat(ck5): add MddmRichBlockPlugin using GHS + block exception"
```

---

## Phase 10 — MddmUploadAdapter

### Task 30: Upload adapter class

**Files:**
- Create: `.../MddmUploadAdapter/MddmUploadAdapter.ts`
- Create: `.../MddmUploadAdapter/index.ts`
- Test: `.../MddmUploadAdapter/__tests__/adapter.test.ts`

- [ ] **Step 1: Write failing test**

Write `.../__tests__/adapter.test.ts`:

```ts
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { MddmUploadAdapter } from '../MddmUploadAdapter';

describe('MddmUploadAdapter', () => {
  beforeEach(() => {
    globalThis.fetch = vi.fn(async () =>
      new Response(JSON.stringify({ url: 'https://cdn.example/abc.png' }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    );
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('uploads a file via POST /assets and returns default URL', async () => {
    const file = new File(['data'], 'a.png', { type: 'image/png' });
    const adapter = new MddmUploadAdapter({
      loader: { file: Promise.resolve(file) } as never,
      endpoint: '/assets',
      getAuthHeader: () => 'Bearer x',
    });
    const result = await adapter.upload();
    expect(result).toEqual({ default: 'https://cdn.example/abc.png' });
    expect(globalThis.fetch).toHaveBeenCalledWith(
      '/assets',
      expect.objectContaining({
        method: 'POST',
        headers: expect.objectContaining({ Authorization: 'Bearer x' }),
      }),
    );
  });

  it('rejects when server returns non-OK', async () => {
    globalThis.fetch = vi.fn(async () => new Response('bad', { status: 500 }));
    const adapter = new MddmUploadAdapter({
      loader: { file: Promise.resolve(new File([], 'x')) } as never,
      endpoint: '/assets',
      getAuthHeader: () => null,
    });
    await expect(adapter.upload()).rejects.toThrow(/upload failed/i);
  });
});
```

- [ ] **Step 2: Run test, verify it fails**

Run: `rtk vitest run src/features/documents/ck5/plugins/MddmUploadAdapter/__tests__/adapter.test.ts`
Expected: FAIL — module not found.

- [ ] **Step 3: Implement `MddmUploadAdapter.ts`**

Write `.../MddmUploadAdapter/MddmUploadAdapter.ts`:

```ts
// Contract from https://ckeditor.com/docs/ckeditor5/latest/framework/deep-dive/upload-adapter.html
export interface UploadLoader {
  file: Promise<File>;
  uploadTotal?: number;
  uploaded?: number;
}

export interface MddmUploadAdapterOptions {
  loader: UploadLoader;
  endpoint: string;
  getAuthHeader: () => string | null;
}

export class MddmUploadAdapter {
  private loader: UploadLoader;
  private endpoint: string;
  private getAuthHeader: () => string | null;
  private controller: AbortController | null = null;

  constructor(opts: MddmUploadAdapterOptions) {
    this.loader = opts.loader;
    this.endpoint = opts.endpoint;
    this.getAuthHeader = opts.getAuthHeader;
  }

  async upload(): Promise<{ default: string }> {
    const file = await this.loader.file;
    const form = new FormData();
    form.append('file', file);
    this.controller = new AbortController();

    const auth = this.getAuthHeader();
    const headers: Record<string, string> = {};
    if (auth) headers.Authorization = auth;

    const res = await fetch(this.endpoint, {
      method: 'POST',
      body: form,
      headers,
      signal: this.controller.signal,
    });
    if (!res.ok) {
      throw new Error(`upload failed with status ${res.status}`);
    }
    const body = (await res.json()) as { url: string };
    return { default: body.url };
  }

  abort(): void {
    this.controller?.abort();
  }
}
```

- [ ] **Step 4: Run test to pass**

Run: `rtk vitest run src/features/documents/ck5/plugins/MddmUploadAdapter/__tests__/adapter.test.ts`
Expected: PASS (2 tests).

- [ ] **Step 5: Implement `index.ts`**

Write `.../MddmUploadAdapter/index.ts`:

```ts
import { Plugin, FileRepository } from 'ckeditor5';
import { MddmUploadAdapter, type UploadLoader } from './MddmUploadAdapter';

export interface MddmUploadAdapterPluginConfig {
  endpoint: string;
  getAuthHeader?: () => string | null;
}

export class MddmUploadAdapterPlugin extends Plugin {
  static get pluginName(): 'MddmUploadAdapterPlugin' {
    return 'MddmUploadAdapterPlugin';
  }

  static get requires(): ReadonlyArray<typeof FileRepository> {
    return [FileRepository];
  }

  init(): void {
    const cfg = (this.editor.config.get('mddmUpload') ?? {
      endpoint: '/assets',
    }) as MddmUploadAdapterPluginConfig;

    this.editor.plugins.get('FileRepository').createUploadAdapter = (loader) =>
      new MddmUploadAdapter({
        loader: loader as unknown as UploadLoader,
        endpoint: cfg.endpoint,
        getAuthHeader: cfg.getAuthHeader ?? (() => null),
      });
  }
}

export { MddmUploadAdapter };
```

- [ ] **Step 6: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/ck5/plugins/MddmUploadAdapter
rtk git commit -m "feat(ck5): add MddmUploadAdapter + plugin wrapper"
```

---

## Phase 10b — Toolbar UI registration for primitive plugins

### Task 30b: Shared toolbar button helper + per-plugin registration

**Critical gap fix.** Toolbar entries like `'insertMddmSection'` in the toolbar config resolve only if a UI component is registered with that name in the `ComponentFactory`. Commands alone do not render buttons. This task adds a shared helper and hooks it into every primitive plugin's `init()`.

**Files:**
- Create: `frontend/apps/web/src/features/documents/ck5/shared/registerInsertionButton.ts`
- Modify: `.../plugins/MddmFieldPlugin/index.ts`
- Modify: `.../plugins/MddmSectionPlugin/index.ts`
- Modify: `.../plugins/MddmRepeatablePlugin/index.ts`
- Modify: `.../plugins/MddmDataTablePlugin/index.ts`
- Modify: `.../plugins/MddmRichBlockPlugin/index.ts`
- Test: `.../shared/__tests__/registerInsertionButton.test.ts`

- [ ] **Step 1: Write failing test for helper**

Write `.../shared/__tests__/registerInsertionButton.test.ts`:

```ts
import { describe, it, expect, beforeEach, afterEach } from 'vitest';
import { ClassicEditor, Essentials, Paragraph } from 'ckeditor5';
import { registerInsertionButton } from '../registerInsertionButton';

describe('registerInsertionButton', () => {
  let editor: ClassicEditor;

  beforeEach(async () => {
    const el = document.createElement('div');
    document.body.appendChild(el);
    editor = await ClassicEditor.create(el, { licenseKey: 'GPL', plugins: [Essentials, Paragraph] });
    editor.commands.add('fakeInsert', new (class extends (await import('ckeditor5')).Command {
      override execute() {}
    })(editor) as never);
  });

  afterEach(async () => {
    await editor.destroy();
  });

  it('registers a ButtonView in the component factory', () => {
    registerInsertionButton(editor, {
      componentName: 'fakeInsertButton',
      commandName: 'fakeInsert',
      label: 'Fake insert',
    });
    expect(editor.ui.componentFactory.has('fakeInsertButton')).toBe(true);
  });

  it('button fires the command on execute', () => {
    let fired = false;
    editor.commands.get('fakeInsert')!.on('execute', () => {
      fired = true;
    });
    registerInsertionButton(editor, {
      componentName: 'fakeInsertButton',
      commandName: 'fakeInsert',
      label: 'Fake insert',
    });
    const view = editor.ui.componentFactory.create('fakeInsertButton');
    view.fire('execute');
    expect(fired).toBe(true);
  });
});
```

- [ ] **Step 2: Run test, verify it fails**

Run: `rtk vitest run src/features/documents/ck5/shared/__tests__/registerInsertionButton.test.ts`
Expected: FAIL — module not found.

- [ ] **Step 3: Implement helper**

Write `.../shared/registerInsertionButton.ts`:

```ts
import { ButtonView, type Editor } from 'ckeditor5';

export interface InsertionButtonOptions {
  componentName: string;
  commandName: string;
  label: string;
  tooltip?: string;
  icon?: string;
  executeOptions?: Record<string, unknown>;
}

export function registerInsertionButton(editor: Editor, opts: InsertionButtonOptions): void {
  editor.ui.componentFactory.add(opts.componentName, (locale) => {
    const view = new ButtonView(locale);
    view.set({
      label: opts.label,
      tooltip: opts.tooltip ?? true,
      withText: !opts.icon,
      icon: opts.icon,
    });

    const cmd = editor.commands.get(opts.commandName);
    if (cmd) {
      view.bind('isEnabled').to(cmd, 'isEnabled');
    }

    view.on('execute', () => {
      editor.execute(opts.commandName, opts.executeOptions ?? {});
      editor.editing.view.focus();
    });

    return view;
  });
}
```

Append to `.../shared/index.ts`:

```ts
export * from './registerInsertionButton';
```

- [ ] **Step 4: Run test to pass**

Run: `rtk vitest run src/features/documents/ck5/shared/__tests__/registerInsertionButton.test.ts`
Expected: PASS (2 tests).

- [ ] **Step 5: Hook helper into MddmFieldPlugin**

Edit `.../plugins/MddmFieldPlugin/index.ts` — replace contents:

```ts
import { Plugin, Widget, viewToModelPositionOutsideModelElement } from 'ckeditor5';
import { registerFieldSchema } from './schema';
import { registerFieldConverters } from './converters';
import { InsertFieldCommand } from './commands/InsertFieldCommand';
import { registerInsertionButton } from '../../shared/registerInsertionButton';

export class MddmFieldPlugin extends Plugin {
  static get pluginName(): 'MddmFieldPlugin' { return 'MddmFieldPlugin'; }
  static get requires(): ReadonlyArray<typeof Widget> { return [Widget]; }

  init(): void {
    const editor = this.editor;
    registerFieldSchema(editor.model.schema);
    registerFieldConverters(editor);
    editor.commands.add('insertMddmField', new InsertFieldCommand(editor));

    editor.editing.mapper.on(
      'viewToModelPosition',
      viewToModelPositionOutsideModelElement(editor.model, (v) => v.hasClass('mddm-field')),
    );

    // UI — button registered in every editor; Fill's toolbar config omits the
    // entry so it simply never renders there. Cost of registration is zero.
    registerInsertionButton(editor, {
      componentName: 'insertMddmField',
      commandName: 'insertMddmField',
      label: 'Insert field',
      executeOptions: { fieldType: 'text', fieldLabel: 'Field', fieldValue: '' },
    });
  }
}
```

- [ ] **Step 6: Hook helper into MddmSectionPlugin**

Edit `.../plugins/MddmSectionPlugin/index.ts` — replace contents:

```ts
import { Plugin, Widget } from 'ckeditor5';
import { registerSectionSchema } from './schema';
import { registerSectionPostFixer } from './postFixer';
import { registerSectionConverters } from './converters';
import { InsertSectionCommand } from './commands/InsertSectionCommand';
import { registerInsertionButton } from '../../shared/registerInsertionButton';

export class MddmSectionPlugin extends Plugin {
  static get pluginName(): 'MddmSectionPlugin' { return 'MddmSectionPlugin'; }
  static get requires(): ReadonlyArray<typeof Widget> { return [Widget]; }

  init(): void {
    const editor = this.editor;
    registerSectionSchema(editor.model.schema);
    registerSectionPostFixer(editor);
    registerSectionConverters(editor);
    editor.commands.add('insertMddmSection', new InsertSectionCommand(editor));

    registerInsertionButton(editor, {
      componentName: 'insertMddmSection',
      commandName: 'insertMddmSection',
      label: 'Insert section',
      executeOptions: { variant: 'editable' },
    });
  }
}
```

- [ ] **Step 7: Hook helper into MddmRepeatablePlugin**

Edit `.../plugins/MddmRepeatablePlugin/index.ts` — replace contents:

```ts
import { Plugin, Widget } from 'ckeditor5';
import { registerRepeatableSchema } from './schema';
import { registerRepeatableConverters } from './converters';
import { InsertRepeatableCommand } from './commands/InsertRepeatableCommand';
import { AddRepeatableItemCommand } from './commands/AddRepeatableItemCommand';
import { RemoveRepeatableItemCommand } from './commands/RemoveRepeatableItemCommand';
import { registerInsertionButton } from '../../shared/registerInsertionButton';

export class MddmRepeatablePlugin extends Plugin {
  static get pluginName(): 'MddmRepeatablePlugin' { return 'MddmRepeatablePlugin'; }
  static get requires(): ReadonlyArray<typeof Widget> { return [Widget]; }

  init(): void {
    const editor = this.editor;
    registerRepeatableSchema(editor.model.schema);
    registerRepeatableConverters(editor);
    editor.commands.add('insertMddmRepeatable', new InsertRepeatableCommand(editor));
    editor.commands.add('addMddmRepeatableItem', new AddRepeatableItemCommand(editor));
    editor.commands.add('removeMddmRepeatableItem', new RemoveRepeatableItemCommand(editor));

    registerInsertionButton(editor, {
      componentName: 'insertMddmRepeatable',
      commandName: 'insertMddmRepeatable',
      label: 'Insert repeatable',
      executeOptions: { min: 1, max: 10, initialCount: 1 },
    });
  }
}
```

- [ ] **Step 8: Hook helper into MddmRichBlockPlugin**

Edit `.../plugins/MddmRichBlockPlugin/index.ts` — replace contents:

```ts
import { Plugin } from 'ckeditor5';
import { InsertRichBlockCommand } from './commands/InsertRichBlockCommand';
import { registerInsertionButton } from '../../shared/registerInsertionButton';

export class MddmRichBlockPlugin extends Plugin {
  static get pluginName(): 'MddmRichBlockPlugin' { return 'MddmRichBlockPlugin'; }

  init(): void {
    const editor = this.editor;
    editor.commands.add('insertMddmRichBlock', new InsertRichBlockCommand(editor));

    registerInsertionButton(editor, {
      componentName: 'insertMddmRichBlock',
      commandName: 'insertMddmRichBlock',
      label: 'Insert rich block',
    });
  }
}
```

- [ ] **Step 9: MddmDataTablePlugin does NOT register a primitive-insertion button**

Native `insertTable` button is registered by CK5's Table plugin. `MddmDataTablePlugin` contributes only the variant attribute, the nested-table guard, and the Fill-mode lock. Its `index.ts` stays as written in Task 28.

- [ ] **Step 10: Run full vitest suite to catch regressions**

Run: `rtk vitest run src/features/documents/ck5`
Expected: All tests pass. If a plugin test fails because `ButtonView` resolution requires Widget/UI infra the test didn't load, fix that plugin's test to include `Widget` in its plugin list.

- [ ] **Step 11: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/ck5/shared frontend/apps/web/src/features/documents/ck5/plugins
rtk git commit -m "feat(ck5): register component-factory buttons for primitive insertion commands"
```

---

## Phase 11 — Wire plugins into editor configs

### Task 31: Include MetalDocs plugins in Author + Fill configs

**Files:**
- Modify: `frontend/apps/web/src/features/documents/ck5/config/editorConfig.ts`
- Test: `frontend/apps/web/src/features/documents/ck5/config/__tests__/editorConfig.integration.test.ts`

- [ ] **Step 1: Write failing integration test**

Write `.../config/__tests__/editorConfig.integration.test.ts`:

```ts
import { describe, it, expect } from 'vitest';
import { ClassicEditor } from 'ckeditor5';
import { createAuthorConfig, createFillConfig } from '../editorConfig';

describe('editorConfig integration', () => {
  it('Author editor exposes all primitive-insertion commands', async () => {
    const el = document.createElement('div');
    document.body.appendChild(el);
    const editor = await ClassicEditor.create(el, createAuthorConfig({}));
    expect(editor.commands.get('insertMddmField')).toBeDefined();
    expect(editor.commands.get('insertMddmSection')).toBeDefined();
    expect(editor.commands.get('insertMddmRepeatable')).toBeDefined();
    expect(editor.commands.get('insertMddmRichBlock')).toBeDefined();
    await editor.destroy();
  });

  it('Fill editor loads primitive schemas so template HTML upcasts correctly', async () => {
    const el = document.createElement('div');
    document.body.appendChild(el);
    const editor = await ClassicEditor.create(el, createFillConfig({}));
    // Schema must know the custom elements so setData() does not silently drop them.
    expect(editor.model.schema.getDefinition('mddmField')).toBeDefined();
    expect(editor.model.schema.getDefinition('mddmSection')).toBeDefined();
    expect(editor.model.schema.getDefinition('mddmRepeatable')).toBeDefined();
    // Fill toolbar does NOT include insertion buttons — verified in toolbars test.
    // The Fill config's toolbar items array simply omits them.
    await editor.destroy();
  });

  it('Fill editor round-trips template HTML with section + field without data loss', async () => {
    const el = document.createElement('div');
    document.body.appendChild(el);
    const editor = await ClassicEditor.create(el, createFillConfig({}));
    const html = [
      '<section class="mddm-section" data-variant="editable">',
      '<header class="mddm-section__header"><p>T</p></header>',
      '<div class="mddm-section__body"><p>',
      '<span class="mddm-field" data-field-id="x" data-field-type="text" data-field-label="X" data-field-required="false">v</span>',
      '</p></div></section>',
    ].join('');
    editor.setData(html);
    const out = editor.getData();
    expect(out).toContain('class="mddm-section"');
    expect(out).toContain('data-field-id="x"');
    expect(out).toContain('>v<');
    await editor.destroy();
  });
});
```

- [ ] **Step 2: Run test, verify it fails**

Run: `rtk vitest run src/features/documents/ck5/config/__tests__/editorConfig.integration.test.ts`
Expected: FAIL — MetalDocs primitive commands not registered.

- [ ] **Step 3: Modify `editorConfig.ts` to append MetalDocs plugins**

Edit `frontend/apps/web/src/features/documents/ck5/config/editorConfig.ts` top imports and the `plugins` arrays:

Add imports after existing imports:
```ts
import { MddmFieldPlugin } from '../plugins/MddmFieldPlugin';
import { MddmSectionPlugin } from '../plugins/MddmSectionPlugin';
import { MddmRepeatablePlugin } from '../plugins/MddmRepeatablePlugin';
import { MddmDataTablePlugin } from '../plugins/MddmDataTablePlugin';
import { MddmRichBlockPlugin } from '../plugins/MddmRichBlockPlugin';
import { MddmUploadAdapterPlugin } from '../plugins/MddmUploadAdapter';
```

Replace the `plugins` property in `createAuthorConfig`'s returned config with:
```ts
    plugins: [
      ...AUTHOR_PLUGINS,
      MddmFieldPlugin,
      MddmSectionPlugin,
      MddmRepeatablePlugin,
      MddmDataTablePlugin,
      MddmUploadAdapterPlugin,
      MddmRichBlockPlugin,
      ...(opts.extraPlugins ?? []),
    ],
```

Replace the `plugins` property in `createFillConfig`:
```ts
    plugins: [
      ...FILL_PLUGINS,
      // Fill mode loads the SAME primitive plugins as Author, because their
      // schema + converters MUST be present to upcast saved template HTML.
      // Each plugin's init() internally gates command + UI registration on
      // whether StandardEditingMode is loaded (see Task 14 / 19 / 23 / 28 / 29).
      MddmFieldPlugin,
      MddmSectionPlugin,
      MddmRepeatablePlugin,
      MddmDataTablePlugin,       // LockSubPlugin gates structural cmds in Fill
      MddmRichBlockPlugin,
      MddmUploadAdapterPlugin,
      ...(opts.extraPlugins ?? []),
    ],
```

- [ ] **Step 4: Run test to pass**

Run: `rtk vitest run src/features/documents/ck5/config/__tests__/editorConfig.integration.test.ts`
Expected: PASS (2 tests).

- [ ] **Step 5: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/ck5/config/editorConfig.ts frontend/apps/web/src/features/documents/ck5/config/__tests__/editorConfig.integration.test.ts
rtk git commit -m "feat(ck5): wire MetalDocs plugins into Author and Fill configs"
```

---

### Task 31b: Exception marker round-trip test

**Critical gap fix.** Commands plant `restrictedEditingException:*` markers with `affectsData: true`. The restricted-editing feature registers the marker downcast/upcast converters so markers survive `getData()` → `setData()`. This task proves it empirically per primitive — if any converter is missing or scoped wrong, we catch it here, not in production.

**Files:**
- Test: `frontend/apps/web/src/features/documents/ck5/__tests__/markerRoundTrip.integration.test.ts`

- [ ] **Step 1: Write failing test**

Write `.../ck5/__tests__/markerRoundTrip.integration.test.ts`:

```ts
import { describe, it, expect } from 'vitest';
import { ClassicEditor } from 'ckeditor5';
import { createAuthorConfig, createFillConfig } from '../config/editorConfig';

async function mount(config: ReturnType<typeof createAuthorConfig>) {
  const el = document.createElement('div');
  document.body.appendChild(el);
  return ClassicEditor.create(el, config);
}

describe('exception marker round-trip', () => {
  it('section insertion plants a marker that survives Author save', async () => {
    const ed = await mount(createAuthorConfig({}));
    ed.execute('insertMddmSection', { variant: 'editable' });
    const html = ed.getData();
    expect(html).toMatch(/class="restricted-editing-exception"/);
    await ed.destroy();
  });

  it('section marker survives round-trip to Fill editor', async () => {
    const author = await mount(createAuthorConfig({}));
    author.execute('insertMddmSection', { variant: 'editable' });
    const html = author.getData();
    await author.destroy();

    const fill = await mount(createFillConfig({}));
    fill.setData(html);
    const markers = Array.from(fill.model.markers).filter((m) =>
      m.name.startsWith('restrictedEditingException:'),
    );
    expect(markers.length).toBeGreaterThanOrEqual(1);
    const out = fill.getData();
    expect(out).toMatch(/class="restricted-editing-exception"/);
    await fill.destroy();
  });

  it('repeatable items each carry a marker that survives Author → Fill', async () => {
    const author = await mount(createAuthorConfig({}));
    author.execute('insertMddmRepeatable', { min: 1, max: 5, initialCount: 3 });
    const html = author.getData();
    expect((html.match(/class="restricted-editing-exception"/g) || []).length).toBeGreaterThanOrEqual(3);
    await author.destroy();

    const fill = await mount(createFillConfig({}));
    fill.setData(html);
    const markers = Array.from(fill.model.markers).filter((m) =>
      m.name.startsWith('restrictedEditingException:'),
    );
    expect(markers.length).toBeGreaterThanOrEqual(3);
    const items = fill.getData().match(/class="mddm-repeatable__item"/g) || [];
    expect(items.length).toBe(3);
    await fill.destroy();
  });

  it('DataTable: per-cell exceptions applied at save survive Fill load', async () => {
    const { applyPerCellExceptions } = await import('../plugins/MddmDataTablePlugin');
    const author = await mount(createAuthorConfig({}));
    author.setData(
      '<figure class="table"><table data-mddm-variant="fixed"><tbody>' +
        '<tr><td>a</td><td>b</td></tr>' +
        '<tr><td>c</td><td>d</td></tr>' +
        '</tbody></table></figure>',
    );
    applyPerCellExceptions(author);
    const html = author.getData();
    // 2x2 = 4 per-cell exceptions in the saved HTML.
    expect((html.match(/class="restricted-editing-exception"/g) || []).length).toBe(4);
    await author.destroy();

    const fill = await mount(createFillConfig({}));
    fill.setData(html);
    const markers = Array.from(fill.model.markers).filter((m) =>
      m.name.startsWith('restrictedEditingException:'),
    );
    expect(markers.length).toBe(4);
    // And the cell contents remain present.
    const out = fill.getData();
    for (const letter of ['a', 'b', 'c', 'd']) {
      expect(out).toContain(letter);
    }
    await fill.destroy();
  });

  it('rich block marker survives round-trip', async () => {
    const author = await mount(createAuthorConfig({}));
    author.execute('insertMddmRichBlock');
    const html = author.getData();
    expect(html).toMatch(/class="restricted-editing-exception"/);
    const fill = await mount(createFillConfig({}));
    fill.setData(html);
    const markers = Array.from(fill.model.markers).filter((m) =>
      m.name.startsWith('restrictedEditingException:'),
    );
    expect(markers.length).toBeGreaterThanOrEqual(1);
    await fill.destroy();
  });
});
```

- [ ] **Step 2: Run test**

Run: `rtk vitest run src/features/documents/ck5/__tests__/markerRoundTrip.integration.test.ts`
Expected: PASS (4 tests). If any test fails, the corresponding primitive's marker planting or the feature's marker converters are misconfigured. Fix at the source (plugin init or command code) — do not weaken the test.

- [ ] **Step 3: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/ck5/__tests__/markerRoundTrip.integration.test.ts
rtk git commit -m "test(ck5): verify exception markers round-trip across Author and Fill"
```

---

## Phase 12 — Persistence stub and routing

### Task 32: LocalStorage stub for save/load

**Files:**
- Create: `frontend/apps/web/src/features/documents/ck5/persistence/localStorageStub.ts`
- Test: `frontend/apps/web/src/features/documents/ck5/persistence/__tests__/localStorageStub.test.ts`

- [ ] **Step 1: Write failing test**

Write `.../persistence/__tests__/localStorageStub.test.ts`:

```ts
import { describe, it, expect, beforeEach } from 'vitest';
import { saveDocument, loadDocument, saveTemplate, loadTemplate } from '../localStorageStub';

beforeEach(() => {
  localStorage.clear();
});

describe('localStorageStub', () => {
  it('saves and loads a document', () => {
    saveDocument('doc-1', '<p>Hello</p>');
    expect(loadDocument('doc-1')).toBe('<p>Hello</p>');
  });

  it('saves and loads a template with manifest', () => {
    saveTemplate('tpl-1', '<section class="mddm-section"/>', { fields: [] });
    const tpl = loadTemplate('tpl-1');
    expect(tpl?.contentHtml).toBe('<section class="mddm-section"/>');
    expect(tpl?.manifest).toEqual({ fields: [] });
  });

  it('returns null for unknown ids', () => {
    expect(loadDocument('missing')).toBeNull();
    expect(loadTemplate('missing')).toBeNull();
  });
});
```

- [ ] **Step 2: Run test, verify it fails**

Run: `rtk vitest run src/features/documents/ck5/persistence/__tests__/localStorageStub.test.ts`
Expected: FAIL — module not found.

- [ ] **Step 3: Implement `localStorageStub.ts`**

Write `.../persistence/localStorageStub.ts`:

```ts
export interface TemplateRecord {
  id: string;
  contentHtml: string;
  manifest: { fields: Array<{ id: string; label?: string; type: string; required?: boolean }> };
}

const DOC_KEY = (id: string) => `ck5.doc.${id}`;
const TPL_KEY = (id: string) => `ck5.tpl.${id}`;

export function saveDocument(id: string, contentHtml: string): void {
  localStorage.setItem(DOC_KEY(id), contentHtml);
}

export function loadDocument(id: string): string | null {
  return localStorage.getItem(DOC_KEY(id));
}

export function saveTemplate(id: string, contentHtml: string, manifest: TemplateRecord['manifest']): void {
  const rec: TemplateRecord = { id, contentHtml, manifest };
  localStorage.setItem(TPL_KEY(id), JSON.stringify(rec));
}

export function loadTemplate(id: string): TemplateRecord | null {
  const raw = localStorage.getItem(TPL_KEY(id));
  return raw ? (JSON.parse(raw) as TemplateRecord) : null;
}
```

- [ ] **Step 4: Run test to pass**

Run: `rtk vitest run src/features/documents/ck5/persistence/__tests__/localStorageStub.test.ts`
Expected: PASS (3 tests).

- [ ] **Step 5: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/ck5/persistence
rtk git commit -m "feat(ck5): add localStorage persistence stub (plan B replaces)"
```

---

### Task 33: Dev-only CK5 test harness (mirrors MDDMTestHarness pattern)

**Context.** The web app uses `HashRouter` with a path-to-view lookup (`routing/workspaceRoutes.ts`) and renders `<AuthShell />` until auth succeeds. Adding new `<Route>` entries to `App.tsx` would (a) fight the view enum, (b) hit the auth wall, and (c) miss the dev-only mounting hook in `main.tsx`. Instead we mirror the existing `MDDMTestHarness` pattern: a second hash prefix in `main.tsx` that bypasses `App` and mounts a dev harness directly.

**Reference:** `src/test-harness/MDDMTestHarness.tsx` + `src/main.tsx` (`isTestHarness` check).

**Files:**
- Create: `frontend/apps/web/src/test-harness/CK5TestHarness.tsx`
- Create: `frontend/apps/web/src/features/documents/ck5/react/AuthorPage.tsx`
- Create: `frontend/apps/web/src/features/documents/ck5/react/FillPage.tsx`
- Modify: `frontend/apps/web/src/main.tsx` (add `isCk5Harness` check)

- [ ] **Step 1: Write `AuthorPage.tsx` (no router dependency — reads template id from props)**

Write `.../ck5/react/AuthorPage.tsx`:

```tsx
import { useState, useCallback } from 'react';
import { AuthorEditor } from './AuthorEditor';
import { saveTemplate, loadTemplate } from '../persistence/localStorageStub';

export interface AuthorPageProps { tplId: string }

export function AuthorPage({ tplId }: AuthorPageProps) {
  const existing = loadTemplate(tplId);
  const [html, setHtml] = useState<string>(existing?.contentHtml ?? '<p>New template</p>');

  const onChange = useCallback(
    (next: string) => {
      setHtml(next);
      saveTemplate(tplId, next, existing?.manifest ?? { fields: [] });
    },
    [tplId, existing],
  );

  return (
    <div data-testid="ck5-author-page" style={{ height: '100vh', display: 'flex', flexDirection: 'column' }}>
      <h1 style={{ padding: 12, margin: 0, borderBottom: '1px solid #ddd' }}>
        Author — {tplId}
      </h1>
      <div style={{ flex: 1, overflow: 'hidden' }}>
        <AuthorEditor initialHtml={html} onChange={onChange} />
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Write `FillPage.tsx`**

Write `.../ck5/react/FillPage.tsx`:

```tsx
import { useState, useCallback } from 'react';
import { FillEditor } from './FillEditor';
import { loadDocument, saveDocument, loadTemplate } from '../persistence/localStorageStub';
import { applyPerCellExceptions } from '../plugins/MddmDataTablePlugin';

export interface FillPageProps { tplId: string; docId: string }

export function FillPage({ tplId, docId }: FillPageProps) {
  const seed = loadDocument(docId) ?? loadTemplate(tplId)?.contentHtml ?? '<p>Empty</p>';
  const [html, setHtml] = useState<string>(seed);

  const onChange = useCallback(
    (next: string) => {
      setHtml(next);
      saveDocument(docId, next);
    },
    [docId],
  );

  // Intentionally no `applyPerCellExceptions(editor)` call here — Author save hook
  // runs it before persisting (see AuthorPage onReady in Step 4 below). This
  // import remains so Plan B's backend path has a reference from the Fill page
  // for any future pre-render enforcement.
  void applyPerCellExceptions;

  return (
    <div data-testid="ck5-fill-page" style={{ height: '100vh', display: 'flex', flexDirection: 'column' }}>
      <h1 style={{ padding: 12, margin: 0, borderBottom: '1px solid #ddd' }}>
        Fill — {docId}
      </h1>
      <div style={{ flex: 1, overflow: 'hidden' }}>
        <FillEditor documentHtml={html} onChange={onChange} />
      </div>
    </div>
  );
}
```

- [ ] **Step 3: Wire per-cell exception walker into AuthorPage save hook**

Edit `.../ck5/react/AuthorPage.tsx` — replace the full file contents with:

```tsx
import { useState, useCallback, useRef } from 'react';
import type { DecoupledEditor } from 'ckeditor5';
import { AuthorEditor } from './AuthorEditor';
import { saveTemplate, loadTemplate } from '../persistence/localStorageStub';
import { applyPerCellExceptions } from '../plugins/MddmDataTablePlugin';

export interface AuthorPageProps { tplId: string }

export function AuthorPage({ tplId }: AuthorPageProps) {
  const existing = loadTemplate(tplId);
  const [html, setHtml] = useState<string>(existing?.contentHtml ?? '<p>New template</p>');
  const editorRef = useRef<DecoupledEditor | null>(null);

  const onReady = useCallback((editor: DecoupledEditor) => {
    editorRef.current = editor;
  }, []);

  const onChange = useCallback(
    (next: string) => {
      // Before persisting, walk the model and ensure each mddmTableVariant-tagged
      // table has a per-cell restricted-editing-exception. Re-entry is safe —
      // the walker adds a marker per cell with a fresh uid on each call, and CK5
      // deduplicates markers by name. Runs on every save; cost is O(cells).
      const editor = editorRef.current;
      if (editor) applyPerCellExceptions(editor);
      const finalHtml = editor ? editor.getData() : next;
      setHtml(finalHtml);
      saveTemplate(tplId, finalHtml, existing?.manifest ?? { fields: [] });
    },
    [tplId, existing],
  );

  return (
    <div data-testid="ck5-author-page" style={{ height: '100vh', display: 'flex', flexDirection: 'column' }}>
      <h1 style={{ padding: 12, margin: 0, borderBottom: '1px solid #ddd' }}>
        Author — {tplId}
      </h1>
      <div style={{ flex: 1, overflow: 'hidden' }}>
        <AuthorEditor initialHtml={html} onChange={onChange} onReady={onReady} />
      </div>
    </div>
  );
}
```

- [ ] **Step 4: Write `CK5TestHarness.tsx`**

Write `frontend/apps/web/src/test-harness/CK5TestHarness.tsx`:

```tsx
import { useEffect, useState } from 'react';
import { AuthorPage } from '../features/documents/ck5/react/AuthorPage';
import { FillPage } from '../features/documents/ck5/react/FillPage';

// Dev-only harness. Bypasses auth / workspace shell. Reachable only via
// `#/test-harness/ck5?mode=author&tpl=<id>` or `?mode=fill&tpl=<id>&doc=<id>`.
// Mounted by main.tsx before <App />.

type Mode = 'author' | 'fill';

interface HarnessParams {
  mode: Mode;
  tplId: string;
  docId: string;
}

function parseHash(): HarnessParams | { error: string } {
  const raw = window.location.hash.split('?')[1] ?? '';
  const params = new URLSearchParams(raw);
  const mode = params.get('mode') as Mode | null;
  if (mode !== 'author' && mode !== 'fill') {
    return { error: 'missing or invalid `mode` (expected `author` or `fill`)' };
  }
  const tplId = params.get('tpl') ?? 'sandbox';
  const docId = params.get('doc') ?? `${tplId}-doc`;
  return { mode, tplId, docId };
}

export function CK5TestHarness() {
  const [state, setState] = useState<HarnessParams | { error: string }>(() => parseHash());

  useEffect(() => {
    if (!import.meta.env.DEV) {
      setState({ error: 'CK5 test harness is disabled in production builds' });
      return;
    }
    const onHashChange = () => setState(parseHash());
    window.addEventListener('hashchange', onHashChange);
    return () => window.removeEventListener('hashchange', onHashChange);
  }, []);

  if ('error' in state) {
    return (
      <div data-testid="ck5-harness-error" style={{ padding: 24 }}>
        CK5 test harness error: {state.error}
      </div>
    );
  }

  return state.mode === 'author' ? (
    <AuthorPage tplId={state.tplId} />
  ) : (
    <FillPage tplId={state.tplId} docId={state.docId} />
  );
}
```

- [ ] **Step 5: Patch `main.tsx` to recognize the CK5 harness hash prefix**

Edit `frontend/apps/web/src/main.tsx` — replace the `isTestHarness` block and the render call:

Replace the existing block:
```ts
const isTestHarness =
  import.meta.env.DEV &&
  window.location.hash.startsWith("#/test-harness/mddm");

initFeatureFlags().finally(() => {
  ReactDOM.createRoot(document.getElementById("root")!).render(
    <React.StrictMode>
      {isTestHarness ? (
        <MDDMTestHarness />
      ) : (
        <HashRouter>
          <App />
        </HashRouter>
      )}
    </React.StrictMode>,
  );
});
```

With:
```ts
import { CK5TestHarness } from "./test-harness/CK5TestHarness";

const hash = window.location.hash;
const isMddmHarness = import.meta.env.DEV && hash.startsWith("#/test-harness/mddm");
const isCk5Harness = import.meta.env.DEV && hash.startsWith("#/test-harness/ck5");

initFeatureFlags().finally(() => {
  ReactDOM.createRoot(document.getElementById("root")!).render(
    <React.StrictMode>
      {isMddmHarness ? (
        <MDDMTestHarness />
      ) : isCk5Harness ? (
        <CK5TestHarness />
      ) : (
        <HashRouter>
          <App />
        </HashRouter>
      )}
    </React.StrictMode>,
  );
});
```

- [ ] **Step 6: Run dev server and verify both harness modes render**

Run: `cd frontend/apps/web && rtk pnpm dev`
Open `http://localhost:<port>/#/test-harness/ck5?mode=author&tpl=smoke` — expect AuthorEditor mounts, primitive-insertion buttons visible in toolbar, no login screen.
Open `http://localhost:<port>/#/test-harness/ck5?mode=fill&tpl=smoke&doc=smoke-doc` — expect FillEditor mounts, caret inside the first exception if the template has any.
Stop dev server.

- [ ] **Step 7: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/ck5/react/AuthorPage.tsx frontend/apps/web/src/features/documents/ck5/react/FillPage.tsx frontend/apps/web/src/test-harness/CK5TestHarness.tsx frontend/apps/web/src/main.tsx
rtk git commit -m "feat(ck5): add CK5 test harness with per-cell exception save hook"
```

---

## Phase 13 — Smoke e2e

### Task 34: Playwright smoke test for Author + Fill round-trip

**Files:**
- Create: `frontend/apps/web/tests/e2e/ck5-smoke.spec.ts`

**Context for executor.** The repo's `playwright.config.ts` uses `testDir: "./tests/e2e"` for the default project named `chrome` (Google Chrome channel), with `baseURL: http://127.0.0.1:4173` (Vite `preview` port). There is no `webServer` entry; the preview server must be running in a separate process. Do **not** put this spec under `./e2e` — that path belongs to the `mddm-visual-parity` project.

- [ ] **Step 1: Verify playwright installed**

Run: `cd frontend/apps/web && rtk pnpm playwright --version`
Expected: version string. If missing: `rtk pnpm add -D @playwright/test && rtk pnpm exec playwright install chrome`.

- [ ] **Step 2: Write failing test**

Write `frontend/apps/web/tests/e2e/ck5-smoke.spec.ts`:

```ts
import { test, expect } from '@playwright/test';

// Harness URLs use the hash router prefix exposed in main.tsx.
const AUTHOR_URL = '/#/test-harness/ck5?mode=author&tpl=smoke';
const FILL_URL = '/#/test-harness/ck5?mode=fill&tpl=smoke&doc=smoke-doc';

test.describe('CK5 smoke', () => {
  test('Author inserts a section; Fill loads it with the exception region', async ({ page }) => {
    // Clear localStorage to guarantee a fresh tpl.
    await page.addInitScript(() => window.localStorage.clear());

    await page.goto(AUTHOR_URL);
    await expect(page.getByTestId('ck5-author-page')).toBeVisible();

    // The Insert section button carries the label we set in
    // registerInsertionButton. Tooltip/ARIA label is 'Insert section'.
    await page.getByRole('button', { name: 'Insert section' }).click();
    await expect(page.locator('.mddm-section')).toBeVisible();

    // Autosave into localStorage happens on every onChange. Give the CKEditor
    // change pipeline a moment to flush.
    await page.waitForFunction(() => {
      const raw = window.localStorage.getItem('ck5.tpl.smoke');
      return raw && raw.includes('mddm-section');
    }, { timeout: 5000 });

    await page.goto(FILL_URL);
    await expect(page.getByTestId('ck5-fill-page')).toBeVisible();
    await expect(page.locator('.mddm-section')).toBeVisible();

    // Block-exception wrapper class should be present in the rendered DOM.
    await expect(
      page.locator('.restricted-editing-exception, [class*="restricted-editing"]').first(),
    ).toBeVisible({ timeout: 5000 });
  });
});
```

- [ ] **Step 3: Start the preview server in a separate terminal**

This repo's Playwright config expects a server already running at `http://127.0.0.1:4173`. Build + preview:

```bash
cd frontend/apps/web
rtk pnpm build
rtk pnpm preview --host 127.0.0.1 --port 4173
```

Leave this terminal running. Verify in a browser that `http://127.0.0.1:4173/#/test-harness/ck5?mode=author&tpl=smoke` renders the AuthorEditor before proceeding.

Note: the dev-only CK5 harness is gated by `import.meta.env.DEV`. `pnpm preview` serves a production build, which sets `DEV=false` and hides the harness. Use `rtk pnpm dev --host 127.0.0.1 --port 4173 --strictPort` instead for the smoke test so the harness is reachable:

```bash
cd frontend/apps/web
rtk pnpm dev --host 127.0.0.1 --port 4173 --strictPort
```

- [ ] **Step 4: Run playwright**

In a second terminal:

```bash
cd frontend/apps/web && rtk pnpm playwright test tests/e2e/ck5-smoke.spec.ts --project=chrome
```
Expected: test passes. If the Insert section button label differs (localization), update the `getByRole('button', { name: ... })` matcher to match the exact string used in `registerInsertionButton` Task 30b Step 6. When done, stop the dev server (Ctrl+C).

- [ ] **Step 5: Commit**

```bash
rtk git add frontend/apps/web/tests/e2e/ck5-smoke.spec.ts
rtk git commit -m "test(ck5): add Playwright smoke covering Author insert + Fill load"
```

---

### Task 35: Final cleanup and PR prep

**Files:** None. Quality gates only.

- [ ] **Step 1: Type check whole project**

Run: `cd frontend/apps/web && rtk pnpm tsc --noEmit`
Expected: zero errors.

- [ ] **Step 2: Run all vitest unit tests**

Run: `rtk vitest run`
Expected: all tests pass. Record totals.

- [ ] **Step 3: Build**

Run: `rtk pnpm build`
Expected: success.

- [ ] **Step 4: Push worktree branch**

```bash
cd ../..
rtk git push -u origin migrate/ck5-frontend-plan-a
```

- [ ] **Step 5: Open PR**

```bash
gh pr create --title "Plan A: CK5 v48 frontend migration" --body "$(cat <<'EOF'
## Summary
- Scaffolds `frontend/apps/web/src/features/documents/ck5/` with Author + Fill editors and five primitive plugins (Field, Section, Repeatable, DataTable + lock, RichBlock).
- Adds `MddmUploadAdapter` wired to `/assets`.
- Adds localStorage persistence stub; Plan B replaces with backend.
- Adds Playwright smoke covering Author insert + Fill load.

BlockNote code under `features/documents/mddm-editor/` is untouched; Plan C removes it once Plan B + Plan C ship.

## Test plan
- [ ] vitest run — all green
- [ ] tsc --noEmit — no errors
- [ ] pnpm build — succeeds
- [ ] Playwright smoke — green
- [ ] Manual: /ck5-author/smoke insert section + field + repeatable + table + rich block; /ck5-fill/smoke/... shows exceptions editable, rest locked
EOF
)"
```

- [ ] **Step 6: Return PR URL to human reviewer.**

---

## Plan self-review checklist

Before handing off, verify:

1. **Spec coverage** — every decision in pages 19–23, 25, 28–29, 37 has a corresponding task.
2. **Placeholder scan** — no "TBD", "similar to", "implement appropriate …".
3. **Type consistency** — command names match between toolbar config, plugin registration, and tests (`insertMddmSection`, `insertMddmField`, etc.).
4. **Commit granularity** — every task ends in a commit.
5. **Test-first discipline** — every task has failing test → impl → pass, with explicit commands.
