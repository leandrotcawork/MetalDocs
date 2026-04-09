# MDDM Phase 3 Blockers Remediation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix all validated Phase 3 blockers so the frontend build is green and the MDDM BlockNote editor/adapter are spec-compliant for Tasks 18-24.

**Architecture:** Keep a strict MDDM-only BlockNote schema and make the adapter the single boundary between editor JSON and MDDM envelope JSON. Remove CKEditor runtime coupling from the web app path so dependency graph and build stay coherent after Task 18. Enforce schema-safe export by explicit prop normalization and unknown-type fail-closed behavior.

**Tech Stack:** TypeScript, React 18, BlockNote (`@blocknote/core`, `@blocknote/react`, `@blocknote/mantine`), Vite, Vitest

---

## File Structure (locked for this remediation)

- `frontend/apps/web/src/features/documents/mddm-editor/schema.ts`
  - Owns MDDM BlockNote schema composition and allowed block set.
- `frontend/apps/web/src/features/documents/mddm-editor/blocks/Field.tsx`
  - Owns Field block spec (`valueMode` + content contract).
- `frontend/apps/web/src/features/documents/mddm-editor/blocks/DataTableRow.tsx`
  - Owns table row block rendering semantics.
- `frontend/apps/web/src/features/documents/mddm-editor/blocks/DataTableCell.tsx`
  - Owns table cell inline-content rendering semantics.
- `frontend/apps/web/src/features/documents/mddm-editor/blocks/Repeatable.tsx`
  - Owns repeatable block defaults (`maxItems` alignment).
- `frontend/apps/web/src/features/documents/mddm-editor/adapter.ts`
  - Owns round-trip mapping, fail-closed type handling, prop normalization.
- `frontend/apps/web/src/features/documents/mddm-editor/__tests__/adapter.test.ts`
  - Owns adapter regressions and schema/canonicalization parity assertions.
- `frontend/apps/web/src/features/documents/browser-editor/BrowserDocumentEditorView.tsx`
  - Must stop importing CKEditor modules.
- `frontend/apps/web/src/features/documents/browser-editor/ckeditorConfig.ts`
  - Remove or de-reference if no longer used.
- `frontend/apps/web/src/lib.types.ts`
- `frontend/apps/web/src/api/documents.ts`
  - Update editor discriminator references if still hardcoded to `ckeditor5`.

---

### Task 1: Lock failing tests for known adapter regressions

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/__tests__/adapter.test.ts`
- Read: `shared/schemas/canonicalize.ts`

- [ ] **Step 1: Add failing quote-edit regression test**

```ts
it("uses live quote content instead of stale metadata on export", () => {
  const blocks = mddmToBlockNote({
    mddm_version: 1,
    template_ref: null,
    blocks: [
      {
        id: "11111111-1111-1111-1111-111111111111",
        type: "quote",
        props: {},
        children: [
          {
            id: "22222222-2222-2222-2222-222222222222",
            type: "paragraph",
            props: {},
            children: [{ text: "OLD" }],
          },
        ],
      },
    ],
  });

  blocks[0].content = [{ type: "text", text: "NEW" }];

  const out = blockNoteToMDDM(blocks);
  const paragraph = out.blocks[0].children?.[0] as any;
  expect(paragraph.children[0].text).toBe("NEW");
});
```

- [ ] **Step 2: Add failing unknown-block fail-closed test**

```ts
it("throws on unknown BlockNote block type", () => {
  expect(() =>
    blockNoteToMDDM([
      {
        id: "33333333-3333-3333-3333-333333333333",
        type: "audio",
        props: {},
      } as any,
    ]),
  ).toThrow(/unsupported block type/i);
});
```

- [ ] **Step 3: Add canonicalized round-trip assertion**

```ts
import { canonicalizeMDDM } from "../../../../../../shared/schemas/canonicalize";

const canonicalIn = canonicalizeMDDM(input as any);
const canonicalOut = canonicalizeMDDM(mddmForm as any);
expect(canonicalOut).toEqual(canonicalIn);
```

- [ ] **Step 4: Run targeted test file and confirm failures**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/__tests__/adapter.test.ts`

Expected: FAIL for quote metadata precedence and unknown-type behavior before implementation fixes.

- [ ] **Step 5: Commit test lock**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/__tests__/adapter.test.ts
git commit -m "test(mddm): lock adapter regressions for quote and unknown block types"
```

---

### Task 2: Enforce strict MDDM schema registration and block semantics

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/schema.ts`
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/blocks/Field.tsx`
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/blocks/DataTableRow.tsx`
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/blocks/DataTableCell.tsx`
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/blocks/Repeatable.tsx`

- [ ] **Step 1: Register custom specs directly (no factory call) and remove permissive defaults**

```ts
import { BlockNoteSchema, defaultBlockSpecs } from "@blocknote/core";

const {
  paragraph,
  heading,
  bulletListItem,
  numberedListItem,
  image,
  quote,
  codeBlock,
} = defaultBlockSpecs;

export const mddmSchema = BlockNoteSchema.create({
  blockSpecs: {
    paragraph,
    heading,
    bulletListItem,
    numberedListItem,
    image,
    quote,
    codeBlock,
    section: Section,
    fieldGroup: FieldGroup,
    field: Field,
    repeatable: Repeatable,
    repeatableItem: RepeatableItem,
    dataTable: DataTable,
    dataTableRow: DataTableRow,
    dataTableCell: DataTableCell,
    richBlock: RichBlock,
  },
});
```

- [ ] **Step 2: Fix Field block content contract (`inline`)**

```tsx
export const Field = createReactBlockSpec(
  {
    type: "field",
    propSchema: {
      label: { default: "" },
      valueMode: { default: "inline", values: ["inline", "multiParagraph"] as const },
      locked: { default: true },
      __template_block_id: { default: undefined, type: "string" },
    },
    content: "inline",
  },
  {
    render: (props) => (
      <div data-mddm-block="field">
        <strong>{props.block.props.label || "Field"}</strong>
        <div ref={props.contentRef} />
      </div>
    ),
  },
);
```

- [ ] **Step 3: Restore table semantic render structure + repeatable bounds**

```tsx
// DataTableRow.tsx
render: () => <tr data-mddm-block="dataTableRow" />;

// DataTableCell.tsx
render: (props) => (
  <td data-mddm-block="dataTableCell">
    <div ref={props.contentRef} />
  </td>
);

// Repeatable.tsx
maxItems: { default: 200 },
```

- [ ] **Step 4: Run build and targeted test**

Run:
- `cd frontend/apps/web && npm run build`
- `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/__tests__/adapter.test.ts`

Expected: build may still fail due to adapter + CKEditor blockers; no new type errors from schema/blocks changes.

- [ ] **Step 5: Commit schema/blocks hardening**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/schema.ts frontend/apps/web/src/features/documents/mddm-editor/blocks/Field.tsx frontend/apps/web/src/features/documents/mddm-editor/blocks/DataTableRow.tsx frontend/apps/web/src/features/documents/mddm-editor/blocks/DataTableCell.tsx frontend/apps/web/src/features/documents/mddm-editor/blocks/Repeatable.tsx
git commit -m "fix(mddm): enforce strict BlockNote schema and block contracts"
```

---

### Task 3: Make adapter fail-closed and schema-safe

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/adapter.ts`

- [ ] **Step 1: Include `field` in inline mapping and hard-fail unknown types**

```ts
const INLINE_BLOCK_TYPES = new Set<string>([
  "paragraph",
  "heading",
  "bulletListItem",
  "numberedListItem",
  "dataTableCell",
  "code",
  "field",
]);

const ALLOWED_MDDM_TYPES = new Set<string>([
  "section",
  "fieldGroup",
  "field",
  "repeatable",
  "repeatableItem",
  "dataTable",
  "dataTableRow",
  "dataTableCell",
  "richBlock",
  "paragraph",
  "heading",
  "bulletListItem",
  "numberedListItem",
  "image",
  "quote",
  "code",
  "divider",
]);

function toMDDMType(blockNoteType: string): string {
  const mapped = blockNoteType === "codeBlock" ? "code" : blockNoteType;
  if (!ALLOWED_MDDM_TYPES.has(mapped)) {
    throw new Error(`unsupported block type: ${blockNoteType}`);
  }
  return mapped;
}
```

- [ ] **Step 2: Use live quote content as source of truth and remove stale metadata precedence**

```ts
function quoteFromBlockNote(content: unknown): MDDMBlock[] {
  return [
    {
      id: crypto.randomUUID(),
      type: "paragraph",
      props: {},
      children: fromBlockNoteInline(content),
    },
  ];
}

// call-site
if (mddmType === "quote") {
  output.children = quoteFromBlockNote(block.content);
  return output;
}
```

- [ ] **Step 3: Add explicit per-type prop normalization**

```ts
function toMDDMProps(type: string, props: UnknownRecord): UnknownRecord {
  const next = cloneRecord(props);

  switch (type) {
    case "paragraph":
    case "quote":
    case "divider":
    case "dataTableRow":
      return {};
    case "heading":
      return { level: Number(next.level) || 1 };
    case "bulletListItem":
    case "numberedListItem":
      return { level: Number(next.level) || 0 };
    case "field":
      return {
        label: asString(next.label),
        valueMode: asString(next.valueMode) || "inline",
        locked: Boolean(next.locked),
      };
    default:
      break;
  }

  // keep existing image/dataTable handling with whitelist-only returns
  return normalized;
}
```

- [ ] **Step 4: Run adapter tests and confirm pass**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/__tests__/adapter.test.ts`

Expected: PASS including quote-edit and unknown-type tests.

- [ ] **Step 5: Commit adapter remediation**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/adapter.ts frontend/apps/web/src/features/documents/mddm-editor/__tests__/adapter.test.ts
git commit -m "fix(mddm): harden adapter for quote edits and schema-safe export"
```

---

### Task 4: Remove CKEditor coupling from browser editor path

**Files:**
- Modify: `frontend/apps/web/src/features/documents/browser-editor/BrowserDocumentEditorView.tsx`
- Modify: `frontend/apps/web/src/api/documents.ts`
- Modify: `frontend/apps/web/src/lib.types.ts`
- Delete or de-reference: `frontend/apps/web/src/features/documents/browser-editor/ckeditorConfig.ts`

- [ ] **Step 1: Replace CKEditor imports/usages with `MDDMEditor` bridge**

```tsx
import { MDDMEditor } from "../mddm-editor/MDDMEditor";
import { blockNoteToMDDM, mddmToBlockNote } from "../mddm-editor/adapter";

const [blockNoteDocument, setBlockNoteDocument] = useState<any[]>([]);

// on load
setBlockNoteDocument(mddmToBlockNote(bundle.mddmEnvelope));

// editor mount
<MDDMEditor
  initialContent={blockNoteDocument as any}
  onChange={(blocks) => {
    setBlockNoteDocument(blocks as any[]);
    const envelope = blockNoteToMDDM(blocks as any[]);
    setEditorData(JSON.stringify(envelope));
  }}
/>
```

- [ ] **Step 2: Update editor discriminator typing/checks away from CKEditor literal**

```ts
// lib.types.ts
editor: "mddm-blocknote";

// api/documents.ts
if (value?.editor !== "mddm-blocknote") {
  throw new Error("DOCUMENT_EDITOR_UNSUPPORTED");
}
```

- [ ] **Step 3: Remove unused CKEditor config/module references**

```bash
rg --line-number "ckeditor|CKEditor" frontend/apps/web/src
```

Expected: no matches in `src/` after migration (or only historical comments intentionally retained).

- [ ] **Step 4: Run frontend build and smoke test command**

Run:
- `cd frontend/apps/web && npm run build`
- `cd frontend/apps/web && npm run e2e:smoke`

Expected: build PASS; smoke either PASS or documented infra-dependent failure with exact reason.

- [ ] **Step 5: Commit CKEditor removal completion**

```bash
git add frontend/apps/web/src/features/documents/browser-editor/BrowserDocumentEditorView.tsx frontend/apps/web/src/api/documents.ts frontend/apps/web/src/lib.types.ts frontend/apps/web/src/features/documents/browser-editor/ckeditorConfig.ts
git commit -m "refactor(mddm): replace browser CKEditor path with BlockNote editor flow"
```

---

### Task 5: Final verification and review handoff

**Files:**
- No functional code changes unless verification reveals defects

- [ ] **Step 1: Run complete Phase 3 verification matrix**

Run:
- `git -C .worktrees/mddm-foundational status --short`
- `cd frontend/apps/web && npm run build`
- `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/__tests__/adapter.test.ts`
- `cd frontend/apps/web && npm run e2e:smoke`

Expected: all pass, or e2e failure documented with reproducible external dependency reason.

- [ ] **Step 2: Run Phase 3 review pass with findings-first output**

Run:
- `git -C .worktrees/mddm-foundational diff --name-only 2d9efa4..HEAD`

Expected: only intended remediation files.

- [ ] **Step 3: Commit verification evidence note**

```bash
git add tasks/lessons.md
git commit -m "docs(process): record Phase 3 blocker remediation verification"
```

---

## Self-Review

### 1. Spec coverage
- Task 18 blocker (CKEditor removal incomplete) is addressed by Task 4.
- Task 21 Field contract mismatch is addressed by Task 2 + Task 3.
- Task 22 row/cell semantic mismatch is addressed by Task 2.
- Task 23 schema registration + allowed block constraints is addressed by Task 2.
- Task 24 adapter round-trip/canonicalization and fail-closed behavior is addressed by Task 1 + Task 3.

No uncovered Phase 3 blocker remains in this plan.

### 2. Placeholder scan
- No `TODO/TBD/similar to` placeholders present.
- Every implementation step includes concrete file-level code snippets and runnable commands.

### 3. Type consistency
- Uses one editor discriminator target (`mddm-blocknote`) consistently in API/types task.
- Keeps `codeBlock <-> code` mapping explicit in adapter.
- Keeps `field` content and adapter inline mapping aligned.

