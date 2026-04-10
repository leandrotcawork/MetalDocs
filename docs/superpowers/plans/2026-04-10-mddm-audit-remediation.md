# MDDM Audit Remediation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Close all findings from the 2026-04-10 implementation audit (MAJOR/MINOR), then produce explicit verification evidence that the MDDM styling scope is complete.

**Architecture:** Keep the existing MDDM structure, apply minimal surgical fixes, and add lightweight contract tests to prevent regression on structural UI chrome rules. Preserve current boundaries (frontend/editor, docgen renderer, Go export pipeline) and avoid new abstractions.

**Tech Stack:** React + TypeScript + CSS Modules + Vitest + Playwright + Go.

---

## File Structure and Responsibilities

- Create: `frontend/apps/web/src/features/documents/mddm-editor/__tests__/styling-contract.test.ts`
  - Guards two hard requirements:
    - FieldGroup block must remain structural-only (no visible helper text).
    - Bridge CSS must keep explicit side-menu hide selectors for structural blocks.
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/blocks/FieldGroup.tsx`
  - Remove structural helper text; keep only data attributes required by bridge CSS.
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/mddm-editor-global.css`
  - Add missing side-menu hide selectors required by the original plan.
- Create: `frontend/apps/web/src/features/documents/mddm-editor/blocks/__tests__/data-table-columns.test.ts`
  - Covers deterministic parse behavior for DataTable columns helper.
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/blocks/DataTable.tsx`
  - Parse `columnsJson` once per render path and reuse value.
- Create: `docs/superpowers/reports/2026-04-10-mddm-verification-evidence.md`
  - Stores concrete execution evidence for Task 16 visual/export checks.
- Modify: `tasks/todo.md`
  - Add a short completed entry for this remediation and verification closure.

---

### Task 1: Remove FieldGroup Structural Text and Restore Side-Menu Bridge Rules

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/__tests__/styling-contract.test.ts`
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/blocks/FieldGroup.tsx`
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/mddm-editor-global.css`

- [ ] **Step 1: Write the failing contract test**

```ts
import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { describe, expect, it } from "vitest";

const repoRoot = process.cwd();

function readRepoFile(relativePath: string): string {
  return readFileSync(resolve(repoRoot, relativePath), "utf8");
}

describe("MDDM styling contracts", () => {
  it("keeps FieldGroup structural-only (no helper text rendered)", () => {
    const fieldGroupTsx = readRepoFile(
      "frontend/apps/web/src/features/documents/mddm-editor/blocks/FieldGroup.tsx",
    );

    expect(fieldGroupTsx).not.toContain("Field Group");
    expect(fieldGroupTsx).not.toContain("coluna(s)");
  });

  it("keeps explicit side-menu hide selectors in the global bridge CSS", () => {
    const css = readRepoFile(
      "frontend/apps/web/src/features/documents/mddm-editor/mddm-editor-global.css",
    );

    expect(css).toContain('[data-content-type="section"] > .bn-block-outer > .bn-block > .bn-side-menu');
    expect(css).toContain('[data-content-type="fieldGroup"] > .bn-block-outer > .bn-block > .bn-side-menu');
    expect(css).toContain('[data-content-type="repeatable"] > .bn-block-outer > .bn-block > .bn-side-menu');
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run:
```bash
npx vitest run frontend/apps/web/src/features/documents/mddm-editor/__tests__/styling-contract.test.ts
```

Expected:
- FAIL on `"Field Group"` / `"coluna(s)"` still present.
- FAIL on missing side-menu selectors in `mddm-editor-global.css`.

- [ ] **Step 3: Write minimal implementation**

`FieldGroup.tsx`:
```tsx
import { createReactBlockSpec } from "@blocknote/react";
import styles from "./FieldGroup.module.css";

export const FieldGroup = createReactBlockSpec(
  {
    type: "fieldGroup",
    propSchema: {
      columns: { default: 1, values: [1, 2] as const },
      locked: { default: true },
      __template_block_id: { default: "" },
    },
    content: "none",
  },
  {
    render: (props) => (
      <div
        className={styles.fieldGroup}
        data-mddm-block="fieldGroup"
        data-columns={props.block.props.columns}
        data-locked={props.block.props.locked}
      />
    ),
  },
);
```

Add to `mddm-editor-global.css`:
```css
.bn-container [data-content-type="section"] > .bn-block-outer > .bn-block > .bn-side-menu,
.bn-container [data-content-type="fieldGroup"] > .bn-block-outer > .bn-block > .bn-side-menu,
.bn-container [data-content-type="repeatable"] > .bn-block-outer > .bn-block > .bn-side-menu {
  display: none;
}
```

- [ ] **Step 4: Run tests and typecheck**

Run:
```bash
npx vitest run frontend/apps/web/src/features/documents/mddm-editor/__tests__/styling-contract.test.ts
frontend/apps/web/node_modules/.bin/tsc.cmd --noEmit -p frontend/apps/web/tsconfig.json
```

Expected:
- Vitest PASS
- TSC PASS (exit code 0, no diagnostics)

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/__tests__/styling-contract.test.ts frontend/apps/web/src/features/documents/mddm-editor/blocks/FieldGroup.tsx frontend/apps/web/src/features/documents/mddm-editor/mddm-editor-global.css
git commit -m "fix(mddm): remove fieldgroup helper text and restore side-menu bridge selectors"
```

---

### Task 2: Remove Repeated DataTable Column Parsing

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/blocks/__tests__/data-table-columns.test.ts`
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/blocks/DataTable.tsx`

- [ ] **Step 1: Write failing unit test for parse helper export**

```ts
import { describe, expect, it } from "vitest";
import { parseDataTableColumns } from "../DataTable";

describe("parseDataTableColumns", () => {
  it("returns only valid column objects", () => {
    const result = parseDataTableColumns(
      JSON.stringify([
        { key: "c1", label: "Nome" },
        { key: "c2", label: "Cargo" },
        { key: "bad" },
      ]),
    );

    expect(result).toEqual([
      { key: "c1", label: "Nome" },
      { key: "c2", label: "Cargo" },
    ]);
  });

  it("returns empty array for invalid json", () => {
    expect(parseDataTableColumns("{")).toEqual([]);
  });
});
```

- [ ] **Step 2: Run test to verify it fails**

Run:
```bash
npx vitest run frontend/apps/web/src/features/documents/mddm-editor/blocks/__tests__/data-table-columns.test.ts
```

Expected:
- FAIL because `parseDataTableColumns` is not exported yet.

- [ ] **Step 3: Implement minimal refactor**

`DataTable.tsx`:
```tsx
type Column = { key: string; label: string };

export function parseDataTableColumns(columnsJson: string): Column[] {
  try {
    const parsed = JSON.parse(columnsJson);
    return Array.isArray(parsed)
      ? parsed.filter((column): column is Column => (
          Boolean(column)
          && typeof column === "object"
          && typeof (column as { key?: unknown }).key === "string"
          && typeof (column as { label?: unknown }).label === "string"
        ))
      : [];
  } catch {
    return [];
  }
}
```

In render, parse once:
```tsx
render: (props) => {
  const columns = parseDataTableColumns(props.block.props.columnsJson);
  return (
    <div className={styles.dataTable} data-mddm-block="dataTable" data-density={props.block.props.density || "normal"}>
      <div className={styles.dataTableHeader}>
        <strong className={styles.tableLabel}>{props.block.props.label || "Data Table"}</strong>
        <span className={styles.tableMeta}>{columns.length} colunas</span>
      </div>
      {columns.length > 0 ? (
        <div className={styles.tableGrid} style={{ gridTemplateColumns: `repeat(${columns.length}, minmax(0, 1fr))` }}>
          {columns.map((column) => (
            <div key={column.key} className={styles.tableHeaderCell}>
              {column.label}
            </div>
          ))}
        </div>
      ) : null}
      <button type="button" className={styles.addRowButton}>+ Adicionar linha</button>
    </div>
  );
}
```

- [ ] **Step 4: Run tests and typecheck**

Run:
```bash
npx vitest run frontend/apps/web/src/features/documents/mddm-editor/blocks/__tests__/data-table-columns.test.ts
frontend/apps/web/node_modules/.bin/tsc.cmd --noEmit -p frontend/apps/web/tsconfig.json
```

Expected:
- Vitest PASS
- TSC PASS

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/blocks/DataTable.tsx frontend/apps/web/src/features/documents/mddm-editor/blocks/__tests__/data-table-columns.test.ts
git commit -m "refactor(mddm): parse data table columns once per render path"
```

---

### Task 3: Close Verification and Process Findings with Explicit Evidence

**Files:**
- Create: `docs/superpowers/reports/2026-04-10-mddm-verification-evidence.md`
- Modify: `tasks/todo.md`

- [ ] **Step 1: Write verification evidence document (failing expectation first)**

Create report skeleton:
```md
# MDDM Verification Evidence — 2026-04-10

## Automated Checks
- [ ] `frontend/apps/web/node_modules/.bin/tsc.cmd --noEmit -p frontend/apps/web/tsconfig.json`
- [ ] `cd apps/docgen && npm.cmd run typecheck`
- [ ] `go build ./...`

## Manual Browser Checks
- [ ] New PO document shows section bars, numbering, optional badge, field grids, repeatable accent, data-table header, add-row button.
- [ ] No structural helper text (FieldGroup).
- [ ] No unexpected BlockNote side-menu chrome on structural blocks.

## DOCX Checks
- [ ] Exported DOCX keeps section header shading.
- [ ] Field table labels use themed shading.
- [ ] Data table header uses themed shading.
- [ ] Color parity with editor theme.
```

- [ ] **Step 2: Execute checks and fill actual evidence**

Run:
```bash
frontend/apps/web/node_modules/.bin/tsc.cmd --noEmit -p frontend/apps/web/tsconfig.json
cd apps/docgen && npm.cmd run typecheck
go build ./...
```

Expected:
- All PASS.

Then replace unchecked boxes with checked boxes and append timestamps plus concise observed result lines.

- [ ] **Step 3: Register remediation completion in todo**

Append in `tasks/todo.md`:
```md
---

## Feature: MDDM audit remediation (2026-04-10)
Area: `frontend/apps/web/src/features/documents/mddm-editor/` + `docs/superpowers/reports/` | Risk: low | Goal: close implementation-audit findings

## Tasks
- [x] T1: Remove FieldGroup helper text and restore side-menu bridge selectors
- [x] T2: Remove repeated DataTable column parsing path
- [x] T3: Record objective verification evidence (build + visual + DOCX parity)

## Acceptance tests
- [x] `frontend/apps/web/node_modules/.bin/tsc.cmd --noEmit -p frontend/apps/web/tsconfig.json`
- [x] `cd apps/docgen && npm.cmd run typecheck`
- [x] `go build ./...`
- [x] Report saved in `docs/superpowers/reports/2026-04-10-mddm-verification-evidence.md`
```

- [ ] **Step 4: Commit**

```bash
git add docs/superpowers/reports/2026-04-10-mddm-verification-evidence.md tasks/todo.md
git commit -m "docs(mddm): record audit remediation verification evidence"
```

---

### Task 4: Final Gate (No Open MAJOR/CRITICAL)

**Files:**
- Modify: none (validation-only gate)

- [ ] **Step 1: Re-run targeted audit checklist**

Run:
```bash
git diff --name-only HEAD~3..HEAD
```

Expected changed scope includes only:
- `FieldGroup.tsx`
- `mddm-editor-global.css`
- `DataTable.tsx`
- new tests
- verification docs

- [ ] **Step 2: Re-run complete compile gates**

Run:
```bash
frontend/apps/web/node_modules/.bin/tsc.cmd --noEmit -p frontend/apps/web/tsconfig.json
cd apps/docgen && npm.cmd run typecheck
go build ./...
```

Expected:
- all commands pass with exit code 0.

- [ ] **Step 3: Optional e2e smoke (if environment up)**

Run:
```bash
cd frontend/apps/web && npm.cmd run e2e:smoke
```

Expected:
- smoke suite passes; if infra is unavailable, record reason in report and keep status transparent.

- [ ] **Step 4: Commit status note**

```bash
git commit --allow-empty -m "chore(mddm): close audit remediation gate"
```

