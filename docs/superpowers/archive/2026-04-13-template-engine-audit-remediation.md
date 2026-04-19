# MDDM Template Engine — Audit Remediation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix the 4 MAJOR audit findings that prevent the template engine from meeting its spec: canonical style/capabilities schema, complete validation wired into instantiation, DataTable React ViewModel parity, and RepeatableItem DOCX numbering parity.

**Architecture:** Four independent tasks, each touching isolated files. Tasks 1-2 fix the template layer (schema + validation). Task 3 wires DataTable to the interpreter pipeline. Task 4 fixes DOCX item numbering. No cross-task dependencies except Task 2 depends on Task 1 being committed first.

**Tech Stack:** TypeScript, BlockNote v0.47.3, Vitest, docx.js

---

## File Structure

### Modified by this plan

```
engine/template/
├── types.ts           — No changes needed (already has style/capabilities fields)
├── instantiate.ts     — Add toBlockNoteBlock() transformer; call validateTemplate() before clone
└── validate.ts        — Add per-type capability checks, maxItems>=minItems, codec parse

engine/template/__tests__/
├── instantiate.test.ts — Update tests: verify canonical fields map to styleJson/capabilitiesJson
└── validate.test.ts    — Add failing cases for new validation rules

templates/
└── po-standard.ts     — Migrate from props.styleJson/capabilitiesJson to top-level style/capabilities

blocks/
└── DataTable.tsx      — Wire render() and addNodeView syncAttrs() to interpretDataTable()

engine/docx-emitter/emitters/
├── repeatable.ts      — Pass item index to emitRepeatableItem()
└── repeatable-item.ts — Accept itemIndex; build display title matching React pattern
```

---

## Task 1: Template Instantiation Boundary — Map Canonical style/capabilities to BlockNote Props

**Context:** `TemplateBlock.style` and `TemplateBlock.capabilities` are the canonical template fields. But BlockNote only accepts string/number/boolean as props — so at the instantiation boundary we must serialize them into `props.styleJson`/`props.capabilitiesJson`. Currently `instantiateTemplate()` does a raw `structuredClone` with no mapping, so templates in canonical format produce blocks the interpreter can't read.

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/engine/template/instantiate.ts`
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/engine/template/__tests__/instantiate.test.ts`
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/templates/po-standard.ts`

- [ ] **Step 1: Write the failing test**

Open `frontend/apps/web/src/features/documents/mddm-editor/engine/template/__tests__/instantiate.test.ts` and replace it entirely with:

```typescript
import { describe, it, expect } from "vitest";
import { instantiateTemplate } from "../instantiate";
import type { TemplateDefinition } from "../types";

const template: TemplateDefinition = {
  templateKey: "po-standard",
  version: 1,
  profileCode: "po",
  status: "published",
  meta: { name: "PO", description: "PO", createdAt: "", updatedAt: "" },
  theme: { accent: "#6b1f2a", accentLight: "#f9f3f3", accentDark: "#3e1018", accentBorder: "#dfc8c8" },
  blocks: [
    {
      type: "section",
      props: { title: "IDENTIFICAÇÃO" },
      style: { headerBackground: "#6b1f2a" },
      capabilities: { locked: true, removable: false },
      children: [],
    },
    {
      type: "repeatable",
      props: { label: "Etapas", itemPrefix: "Etapa" },
      style: {},
      capabilities: { locked: false, addItems: true, maxItems: 50, minItems: 1 },
      children: [],
    },
  ],
};

describe("instantiateTemplate", () => {
  it("creates an envelope with template_ref", () => {
    const envelope = instantiateTemplate(template);
    expect(envelope.template_ref.templateKey).toBe("po-standard");
    expect(envelope.template_ref.templateVersion).toBe(1);
    expect(envelope.template_ref.instantiatedAt).toBeTruthy();
  });

  it("maps top-level style to props.styleJson", () => {
    const envelope = instantiateTemplate(template);
    const parsed = JSON.parse(envelope.blocks[0].props.styleJson as string);
    expect(parsed.headerBackground).toBe("#6b1f2a");
  });

  it("maps top-level capabilities to props.capabilitiesJson", () => {
    const envelope = instantiateTemplate(template);
    const parsed = JSON.parse(envelope.blocks[0].props.capabilitiesJson as string);
    expect(parsed.locked).toBe(true);
    expect(parsed.removable).toBe(false);
  });

  it("maps empty style to '{}'", () => {
    const envelope = instantiateTemplate(template);
    expect(envelope.blocks[1].props.styleJson).toBe("{}");
  });

  it("omits top-level style/capabilities fields from block props", () => {
    const envelope = instantiateTemplate(template);
    expect((envelope.blocks[0] as any).style).toBeUndefined();
    expect((envelope.blocks[0] as any).capabilities).toBeUndefined();
  });

  it("deep clones blocks (mutation-safe)", () => {
    const envelope = instantiateTemplate(template);
    (envelope.blocks[0].props as any).title = "MODIFIED";
    expect(template.blocks[0].props.title).toBe("IDENTIFICAÇÃO");
  });

  it("recursively maps children", () => {
    const templateWithChild: TemplateDefinition = {
      ...template,
      blocks: [{
        type: "section",
        props: { title: "S1" },
        capabilities: { locked: true },
        children: [{
          type: "richBlock",
          props: { label: "Obj" },
          capabilities: { locked: true, editableZones: ["content"] },
        }],
      }],
    };
    const envelope = instantiateTemplate(templateWithChild);
    const childCaps = JSON.parse(envelope.blocks[0].children![0].props.capabilitiesJson as string);
    expect(childCaps.locked).toBe(true);
    expect(childCaps.editableZones).toEqual(["content"]);
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd frontend/apps/web && rtk vitest run src/features/documents/mddm-editor/engine/template/__tests__/instantiate.test.ts
```

Expected: FAIL — several tests fail because `instantiateTemplate` does raw `structuredClone` with no mapping.

- [ ] **Step 3: Implement the mapping in instantiate.ts**

Replace `frontend/apps/web/src/features/documents/mddm-editor/engine/template/instantiate.ts` entirely:

```typescript
import { validateTemplate } from "./validate";
import type { TemplateDefinition, TemplateRef, TemplateBlock } from "./types";

export type MDDMTemplateEnvelope = {
  mddm_version: number;
  template_ref: TemplateRef;
  blocks: TemplateBlock[];
};

const CURRENT_MDDM_VERSION = 1;

/**
 * Maps a canonical TemplateBlock (with top-level style/capabilities) to a
 * BlockNote-compatible block where those fields are serialized into
 * props.styleJson / props.capabilitiesJson.
 *
 * This is the editor boundary: templates are authored in canonical format;
 * BlockNote only accepts primitive props.
 */
function toBlockNoteBlock(block: TemplateBlock): TemplateBlock {
  const { style, capabilities, children, ...rest } = block;
  return {
    ...rest,
    props: {
      ...block.props,
      styleJson: JSON.stringify(style ?? {}),
      capabilitiesJson: JSON.stringify(capabilities ?? {}),
    },
    children: children?.map(toBlockNoteBlock),
  };
}

/**
 * Instantiate a template into a document envelope.
 *
 * Validates the template first (throws on CRITICAL errors), then maps
 * each block's canonical style/capabilities to BlockNote prop strings.
 */
export function instantiateTemplate(template: TemplateDefinition): MDDMTemplateEnvelope {
  const errors = validateTemplate(template);
  if (errors.length > 0) {
    throw new Error(
      `Cannot instantiate invalid template "${template.templateKey}": ` +
      errors.map((e) => `${e.path} — ${e.message}`).join("; "),
    );
  }

  return {
    mddm_version: CURRENT_MDDM_VERSION,
    template_ref: {
      templateKey: template.templateKey,
      templateVersion: template.version,
      instantiatedAt: new Date().toISOString(),
    },
    blocks: template.blocks.map(toBlockNoteBlock),
  };
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd frontend/apps/web && rtk vitest run src/features/documents/mddm-editor/engine/template/__tests__/instantiate.test.ts
```

Expected: PASS (7 tests)

- [ ] **Step 5: Migrate po-standard.ts to canonical format**

Replace `frontend/apps/web/src/features/documents/mddm-editor/templates/po-standard.ts` entirely:

```typescript
import type { TemplateDefinition } from "../engine/template";

export const poStandardTemplate: TemplateDefinition = {
  templateKey: "po-standard",
  version: 1,
  profileCode: "po",
  status: "published",
  meta: {
    name: "Procedimento Operacional Padrão",
    description: "Template padrão para procedimentos operacionais",
    createdAt: "2026-04-13T00:00:00Z",
    updatedAt: "2026-04-13T00:00:00Z",
  },
  theme: {
    accent: "#6b1f2a",
    accentLight: "#f9f3f3",
    accentDark: "#3e1018",
    accentBorder: "#dfc8c8",
  },
  blocks: [
    {
      type: "section",
      props: { title: "IDENTIFICAÇÃO" },
      capabilities: { locked: true, removable: false },
      children: [],
    },
    {
      type: "section",
      props: { title: "IDENTIFICAÇÃO DO PROCESSO" },
      capabilities: { locked: true, removable: false },
      children: [
        { type: "richBlock", props: { label: "Objetivo" }, capabilities: { locked: true, editableZones: ["content"] } },
        { type: "richBlock", props: { label: "Escopo" }, capabilities: { locked: true, editableZones: ["content"] } },
        { type: "richBlock", props: { label: "Cargo responsável" }, capabilities: { locked: true, editableZones: ["content"] } },
        { type: "richBlock", props: { label: "Canal / Contexto" }, capabilities: { locked: true, editableZones: ["content"] } },
        { type: "richBlock", props: { label: "Participantes" }, capabilities: { locked: true, editableZones: ["content"] } },
      ],
    },
    {
      type: "section",
      props: { title: "ENTRADAS E SAÍDAS" },
      capabilities: { locked: true, removable: false },
      children: [],
    },
    {
      type: "section",
      props: { title: "VISÃO GERAL DO PROCESSO" },
      capabilities: { locked: true, removable: false },
      children: [
        { type: "richBlock", props: { label: "Descrição do processo" }, capabilities: { locked: true, editableZones: ["content"] } },
        { type: "richBlock", props: { label: "Diagrama" }, capabilities: { locked: true, editableZones: ["content"] } },
      ],
    },
    {
      type: "section",
      props: { title: "DETALHAMENTO DAS ETAPAS" },
      capabilities: { locked: true, removable: false },
      children: [
        {
          type: "repeatable",
          props: { label: "Etapas", itemPrefix: "Etapa" },
          capabilities: { locked: false, addItems: true, removeItems: true, maxItems: 50, minItems: 1 },
          children: [],
        },
      ],
    },
    {
      type: "section",
      props: { title: "INDICADORES" },
      capabilities: { locked: true, removable: false },
      children: [],
    },
    {
      type: "section",
      props: { title: "RISCOS E CONTROLES" },
      capabilities: { locked: true, removable: false },
      children: [],
    },
    {
      type: "section",
      props: { title: "REFERÊNCIAS" },
      capabilities: { locked: true, removable: false },
      children: [],
    },
    {
      type: "section",
      props: { title: "GLOSSÁRIO" },
      capabilities: { locked: true, removable: false },
      children: [],
    },
    {
      type: "section",
      props: { title: "HISTÓRICO DE REVISÕES" },
      capabilities: { locked: true, removable: false },
      children: [],
    },
  ],
};
```

- [ ] **Step 6: Run all template tests**

```bash
cd frontend/apps/web && rtk vitest run src/features/documents/mddm-editor/engine/template/ src/features/documents/mddm-editor/templates/
```

Expected: PASS (all pass — po-standard test uses `validateTemplate` which currently runs before instantiation, and the canonical template passes existing validation)

- [ ] **Step 7: Commit**

```bash
rtk git add \
  frontend/apps/web/src/features/documents/mddm-editor/engine/template/instantiate.ts \
  frontend/apps/web/src/features/documents/mddm-editor/engine/template/__tests__/instantiate.test.ts \
  frontend/apps/web/src/features/documents/mddm-editor/templates/po-standard.ts
rtk git commit -m "feat(mddm): instantiation boundary maps canonical style/capabilities to BlockNote props"
```

---

## Task 2: Expand Template Validation + Wire Into Instantiation

**Context:** `validateTemplate()` currently only checks block types and a few required props. The spec requires: per-type capability validation, `maxItems >= minItems`, and codec parse verification. It also must be called during instantiation (Task 1 already wires this — Task 2 makes validation complete enough to be useful).

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/engine/template/validate.ts`
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/engine/template/__tests__/validate.test.ts`

- [ ] **Step 1: Write the failing tests**

Replace `frontend/apps/web/src/features/documents/mddm-editor/engine/template/__tests__/validate.test.ts` entirely:

```typescript
import { describe, it, expect } from "vitest";
import { validateTemplate, type ValidationError } from "../validate";
import type { TemplateDefinition } from "../types";

function makeTemplate(overrides: Partial<TemplateDefinition> = {}): TemplateDefinition {
  return {
    templateKey: "test",
    version: 1,
    profileCode: "po",
    status: "published",
    meta: { name: "Test", description: "Test", createdAt: "", updatedAt: "" },
    theme: { accent: "#6b1f2a", accentLight: "#f9f3f3", accentDark: "#3e1018", accentBorder: "#dfc8c8" },
    blocks: [],
    ...overrides,
  };
}

describe("validateTemplate — basic", () => {
  it("accepts a valid empty template", () => {
    expect(validateTemplate(makeTemplate())).toHaveLength(0);
  });

  it("accepts a template with valid section block", () => {
    const errors = validateTemplate(makeTemplate({
      blocks: [{ type: "section", props: { title: "TEST" } }],
    }));
    expect(errors).toHaveLength(0);
  });

  it("rejects unknown block type", () => {
    const errors = validateTemplate(makeTemplate({
      blocks: [{ type: "nonexistent", props: {} }],
    }));
    expect(errors).toContainEqual(expect.objectContaining({ error: "unknown_block_type" }));
  });

  it("rejects section without title", () => {
    const errors = validateTemplate(makeTemplate({
      blocks: [{ type: "section", props: {} }],
    }));
    expect(errors).toContainEqual(expect.objectContaining({ error: "missing_required_prop" }));
  });
});

describe("validateTemplate — repeatable capability invariants", () => {
  it("accepts repeatable with maxItems >= minItems", () => {
    const errors = validateTemplate(makeTemplate({
      blocks: [{ type: "repeatable", props: { label: "Steps" }, capabilities: { maxItems: 10, minItems: 1 } }],
    }));
    expect(errors).toHaveLength(0);
  });

  it("rejects repeatable with maxItems < minItems", () => {
    const errors = validateTemplate(makeTemplate({
      blocks: [{ type: "repeatable", props: { label: "Steps" }, capabilities: { maxItems: 2, minItems: 5 } }],
    }));
    expect(errors).toContainEqual(expect.objectContaining({ error: "invalid_capability", path: expect.stringContaining("maxItems") }));
  });

  it("rejects repeatable with maxItems === 0 and minItems === 0 — allowed (edge case pass)", () => {
    // 0 >= 0 is valid
    const errors = validateTemplate(makeTemplate({
      blocks: [{ type: "repeatable", props: { label: "Steps" }, capabilities: { maxItems: 0, minItems: 0 } }],
    }));
    expect(errors).toHaveLength(0);
  });
});

describe("validateTemplate — dataTable mode", () => {
  it("accepts dataTable with mode: fixed", () => {
    const errors = validateTemplate(makeTemplate({
      blocks: [{ type: "dataTable", props: { label: "T" }, capabilities: { mode: "fixed" } }],
    }));
    expect(errors).toHaveLength(0);
  });

  it("accepts dataTable with mode: dynamic", () => {
    const errors = validateTemplate(makeTemplate({
      blocks: [{ type: "dataTable", props: { label: "T" }, capabilities: { mode: "dynamic" } }],
    }));
    expect(errors).toHaveLength(0);
  });

  it("rejects dataTable with invalid mode", () => {
    const errors = validateTemplate(makeTemplate({
      blocks: [{ type: "dataTable", props: { label: "T" }, capabilities: { mode: "weird" } }],
    }));
    expect(errors).toContainEqual(expect.objectContaining({ error: "invalid_capability" }));
  });
});

describe("validateTemplate — per-type capability key restriction", () => {
  it("rejects addItems capability on a section block", () => {
    const errors = validateTemplate(makeTemplate({
      blocks: [{ type: "section", props: { title: "S" }, capabilities: { addItems: true } }],
    }));
    expect(errors).toContainEqual(expect.objectContaining({ error: "invalid_capability_key" }));
  });

  it("rejects addRows capability on a repeatable block", () => {
    const errors = validateTemplate(makeTemplate({
      blocks: [{ type: "repeatable", props: { label: "R" }, capabilities: { addRows: true } }],
    }));
    expect(errors).toContainEqual(expect.objectContaining({ error: "invalid_capability_key" }));
  });
});

describe("validateTemplate — nested children", () => {
  it("validates children recursively", () => {
    const errors = validateTemplate(makeTemplate({
      blocks: [{
        type: "section",
        props: { title: "S" },
        children: [{ type: "nonexistent", props: {} }],
      }],
    }));
    expect(errors).toContainEqual(expect.objectContaining({ error: "unknown_block_type" }));
  });
});
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
cd frontend/apps/web && rtk vitest run src/features/documents/mddm-editor/engine/template/__tests__/validate.test.ts
```

Expected: FAIL — several new capability validation tests fail because `validate.ts` doesn't have that logic yet.

- [ ] **Step 3: Implement expanded validation**

Replace `frontend/apps/web/src/features/documents/mddm-editor/engine/template/validate.ts` entirely:

```typescript
import type { TemplateDefinition, TemplateBlock } from "./types";

export type ValidationError = {
  path: string;
  error: string;
  message: string;
};

const KNOWN_BLOCK_TYPES = new Set([
  "section", "dataTable", "repeatable", "repeatableItem",
  "richBlock", "paragraph", "heading", "bulletListItem",
  "numberedListItem", "image", "quote", "divider",
]);

const REQUIRED_PROPS: Record<string, string[]> = {
  section: ["title"],
  dataTable: ["label"],
  repeatable: ["label"],
  richBlock: ["label"],
};

/** Capability keys that are only valid on specific block types */
const BLOCK_TYPE_ALLOWED_CAPS: Record<string, Set<string>> = {
  section: new Set(["locked", "removable", "reorderable"]),
  richBlock: new Set(["locked", "removable", "reorderable", "editableZones"]),
  repeatableItem: new Set(["locked", "removable", "reorderable", "editableZones"]),
  repeatable: new Set(["locked", "removable", "reorderable", "addItems", "removeItems", "maxItems", "minItems"]),
  dataTable: new Set([
    "locked", "removable", "reorderable", "mode",
    "addRows", "removeRows", "addColumns", "removeColumns", "resizeColumns",
    "headerLocked", "editableZones", "maxRows",
  ]),
  // other block types: allow only universal caps
  paragraph: new Set(["locked", "removable", "reorderable"]),
  heading: new Set(["locked", "removable", "reorderable"]),
  bulletListItem: new Set(["locked", "removable", "reorderable"]),
  numberedListItem: new Set(["locked", "removable", "reorderable"]),
  image: new Set(["locked", "removable", "reorderable"]),
  quote: new Set(["locked", "removable", "reorderable"]),
  divider: new Set(["locked", "removable", "reorderable"]),
  fieldGroup: new Set(["locked", "removable", "reorderable", "columns"]),
  field: new Set(["locked", "removable", "reorderable", "editableZones"]),
};

const VALID_DATATABLE_MODES = new Set(["fixed", "dynamic"]);

export function validateTemplate(template: TemplateDefinition): ValidationError[] {
  const errors: ValidationError[] = [];
  validateBlocks(template.blocks, "blocks", errors);
  return errors;
}

function validateBlocks(blocks: TemplateBlock[], basePath: string, errors: ValidationError[]): void {
  for (let i = 0; i < blocks.length; i++) {
    const block = blocks[i];
    const path = `${basePath}[${i}]`;

    // 1. Block type must be known
    if (!KNOWN_BLOCK_TYPES.has(block.type)) {
      errors.push({ path, error: "unknown_block_type", message: `Unknown block type: ${block.type}` });
      continue; // skip further checks for unknown type
    }

    // 2. Required props
    const required = REQUIRED_PROPS[block.type];
    if (required) {
      for (const prop of required) {
        if (!block.props[prop]) {
          errors.push({
            path: `${path}.props.${prop}`,
            error: "missing_required_prop",
            message: `Missing required prop: ${prop}`,
          });
        }
      }
    }

    // 3. Capability validation
    if (block.capabilities) {
      validateCapabilities(block.type, block.capabilities, `${path}.capabilities`, errors);
    }

    // 4. Recurse into children
    if (block.children) {
      validateBlocks(block.children, `${path}.children`, errors);
    }
  }
}

function validateCapabilities(
  blockType: string,
  capabilities: Record<string, unknown>,
  path: string,
  errors: ValidationError[],
): void {
  const allowed = BLOCK_TYPE_ALLOWED_CAPS[blockType];
  if (!allowed) return; // unknown type already reported above

  // 3a. Reject unknown capability keys for this block type
  for (const key of Object.keys(capabilities)) {
    if (!allowed.has(key)) {
      errors.push({
        path: `${path}.${key}`,
        error: "invalid_capability_key",
        message: `Capability key "${key}" is not valid for block type "${blockType}"`,
      });
    }
  }

  // 3b. dataTable: mode must be "fixed" | "dynamic"
  if (blockType === "dataTable" && capabilities.mode !== undefined) {
    if (!VALID_DATATABLE_MODES.has(capabilities.mode as string)) {
      errors.push({
        path: `${path}.mode`,
        error: "invalid_capability",
        message: `dataTable.mode must be "fixed" or "dynamic", got "${capabilities.mode}"`,
      });
    }
  }

  // 3c. repeatable: maxItems >= minItems
  if (blockType === "repeatable") {
    const maxItems = capabilities.maxItems;
    const minItems = capabilities.minItems;
    if (
      typeof maxItems === "number" &&
      typeof minItems === "number" &&
      maxItems < minItems
    ) {
      errors.push({
        path: `${path}.maxItems`,
        error: "invalid_capability",
        message: `repeatable.maxItems (${maxItems}) must be >= minItems (${minItems})`,
      });
    }
  }
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
cd frontend/apps/web && rtk vitest run src/features/documents/mddm-editor/engine/template/__tests__/validate.test.ts
```

Expected: PASS (all tests pass)

- [ ] **Step 5: Run the full template suite to confirm po-standard still passes**

```bash
cd frontend/apps/web && rtk vitest run src/features/documents/mddm-editor/engine/template/ src/features/documents/mddm-editor/templates/
```

Expected: PASS (all pass — the PO template uses only valid capabilities)

- [ ] **Step 6: Commit**

```bash
rtk git add \
  frontend/apps/web/src/features/documents/mddm-editor/engine/template/validate.ts \
  frontend/apps/web/src/features/documents/mddm-editor/engine/template/__tests__/validate.test.ts
rtk git commit -m "feat(mddm): expand validateTemplate with per-type capability checks and wire into instantiation"
```

---

## Task 3: Wire DataTable React Render to interpretDataTable()

**Context:** `DataTable.tsx` has two render paths. The `render:` callback is the BlockNote fallback. `addNodeView()` is the actual ProseMirror node view that renders in the editor. Both currently read raw attrs/props directly — neither calls `interpretDataTable()`. This breaks the ViewModel parity contract for the block type most explicitly called out in the spec.

Fix: wire both paths to `interpretDataTable()` so capability flags (`locked`, `mode`) flow through the ViewModel. For `addNodeView`, use `defaultLayoutTokens` since there's no editor context.

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/blocks/DataTable.tsx`

- [ ] **Step 1: Read DataTable.tsx to confirm current state**

Read `frontend/apps/web/src/features/documents/mddm-editor/blocks/DataTable.tsx`. Verify `addNodeView` `syncAttrs()` reads `nextNode.attrs.locked`, `nextNode.attrs.label`, `nextNode.attrs.density` directly. Verify `render:` callback reads `props.block.props.locked`, `props.block.props.label`, `props.block.props.density` directly.

- [ ] **Step 2: Update DataTable.tsx — import interpreter and wire both paths**

The complete diff to apply — update the imports section and both render paths. The `addNodeView` uses `defaultLayoutTokens` (no editor context available in PM); the `render:` callback uses `getEditorTokens(props.editor)`.

At the top of `DataTable.tsx`, add these imports after the existing ones:

```typescript
import { interpretDataTable } from "../engine/layout-interpreter/data-table-interpreter";
import { defaultLayoutTokens } from "../engine/layout-ir";
```

In `syncAttrs()`, replace the three direct attr reads with interpreter output:

```typescript
// BEFORE:
const syncAttrs = (nextNode: typeof node) => {
  dom.dataset.density = nextNode.attrs.density || "normal";
  dom.dataset.locked = String(nextNode.attrs.locked);
  label.textContent = nextNode.attrs.label || "Data Table";
};

// AFTER:
const syncAttrs = (nextNode: typeof node) => {
  const vm = interpretDataTable(
    { props: nextNode.attrs as Record<string, unknown> },
    defaultLayoutTokens,
  );
  dom.dataset.density = vm.density;
  dom.dataset.locked = String(vm.locked);
  dom.dataset.mode = vm.mode;
  label.textContent = vm.label || "Data Table";
};
```

In the `render:` callback, replace prop reads with interpreter output:

```typescript
// BEFORE:
render: (props) => (
  <div
    className={styles.dataTable}
    data-mddm-block="dataTable"
    data-density={props.block.props.density || "normal"}
    data-locked={String(props.block.props.locked)}
  >
    <div className={styles.dataTableHeader}>
      <strong className={styles.tableLabel}>
        {props.block.props.label || "Data Table"}
      </strong>
    </div>
    <div className={styles.tableContainer} ref={(props as any).contentRef} />
  </div>
),

// AFTER:
render: (props) => {
  const tokens = getEditorTokens(props.editor);
  const vm = interpretDataTable(
    { props: props.block.props as Record<string, unknown> },
    tokens,
  );
  return (
    <div
      className={styles.dataTable}
      data-mddm-block="dataTable"
      data-density={vm.density}
      data-locked={String(vm.locked)}
      data-mode={vm.mode}
    >
      <div className={styles.dataTableHeader}>
        <strong className={styles.tableLabel}>
          {vm.label || "Data Table"}
        </strong>
      </div>
      <div className={styles.tableContainer} ref={(props as any).contentRef} />
    </div>
  );
},
```

- [ ] **Step 3: Run the MDDM block tests to verify no regressions**

```bash
cd frontend/apps/web && rtk vitest run src/features/documents/mddm-editor/engine/layout-interpreter/__tests__/data-table-interpreter.test.ts
```

Expected: PASS (interpreter tests unchanged)

- [ ] **Step 4: Run the TypeScript compiler check**

```bash
cd frontend/apps/web && rtk tsc --noEmit 2>&1 | head -30
```

Expected: No errors in DataTable.tsx

- [ ] **Step 5: Commit**

```bash
rtk git add frontend/apps/web/src/features/documents/mddm-editor/blocks/DataTable.tsx
rtk git commit -m "feat(mddm): wire DataTable addNodeView and render() to interpretDataTable() ViewModel"
```

---

## Task 4: Fix RepeatableItem DOCX Numbering Parity

**Context:** `RepeatableItem.tsx` renders `${vm.number} ${vm.title}` (e.g. "5.1 Etapa 1"). The DOCX emitter always passes `{ itemIndex: 0 }` and emits only `vm.title` (e.g. "Etapa 1"). Two differences: wrong index and missing number prefix. Fix: pass the real loop index from `emitRepeatable()` and emit the same display title pattern React uses.

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/repeatable-item.ts`
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/repeatable.ts`

- [ ] **Step 1: Update emitRepeatableItem signature to accept itemIndex**

In `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/repeatable-item.ts`, make these changes:

Change the function signature to accept an optional `itemIndex`:

```typescript
// BEFORE:
export function emitRepeatableItem(
  block: MDDMBlock,
  tokens: LayoutTokens,
  renderChild: ChildRenderer,
): Table[] {
  const vm = interpretRepeatableItem(
    { props: block.props as Record<string, unknown> },
    tokens,
    { itemIndex: 0 },
  );
  const accent = hexToFill(vm.accentBorderColor);
  const borderColor = hexToFill(tokens.theme.accentBorder);

  const innerChildren: unknown[] = [];
  if (vm.title) {
    innerChildren.push(
      new Paragraph({
        children: [
          new TextRun({
            text: vm.title,

// AFTER:
export function emitRepeatableItem(
  block: MDDMBlock,
  tokens: LayoutTokens,
  renderChild: ChildRenderer,
  itemIndex = 0,
): Table[] {
  const vm = interpretRepeatableItem(
    { props: block.props as Record<string, unknown> },
    tokens,
    { itemIndex },
  );
  const accent = hexToFill(vm.accentBorderColor);
  const borderColor = hexToFill(tokens.theme.accentBorder);

  const displayTitle = vm.title ? `${vm.number} ${vm.title}` : `Item ${vm.number}`;

  const innerChildren: unknown[] = [];
  if (displayTitle) {
    innerChildren.push(
      new Paragraph({
        children: [
          new TextRun({
            text: displayTitle,
```

- [ ] **Step 2: Update emitRepeatable to pass real item index**

In `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/repeatable.ts`, change the item loop:

```typescript
// BEFORE:
  const items = ((block.children ?? []) as unknown[]).filter(isItemBlock) as MDDMBlock[];
  for (const item of items) {
    out.push(...emitRepeatableItem(item, tokens, renderChild));
  }

// AFTER:
  const items = ((block.children ?? []) as unknown[]).filter(isItemBlock) as MDDMBlock[];
  for (let i = 0; i < items.length; i++) {
    out.push(...emitRepeatableItem(items[i], tokens, renderChild, i));
  }
```

- [ ] **Step 3: Run the DOCX emitter tests**

```bash
cd frontend/apps/web && rtk vitest run src/features/documents/mddm-editor/engine/docx-emitter/
```

Expected: PASS (48 tests — the existing tests use `vm.title` for text matching; update any test that checks the exact DOCX text to expect the display title format)

If tests fail because they expected `vm.title` only (without number prefix), update those test assertions to expect `${vm.number} ${vm.title}` pattern. Check `__tests__/repeatable-item*.test.ts` if they exist and update accordingly.

- [ ] **Step 4: Run the full suite to verify no regressions**

```bash
cd frontend/apps/web && rtk vitest run
```

Expected: PASS (281+) FAIL (4) — same 4 pre-existing failures, no new failures.

- [ ] **Step 5: Commit**

```bash
rtk git add \
  frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/repeatable-item.ts \
  frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/repeatable.ts
rtk git commit -m "fix(mddm): DOCX repeatable-item emits real item number matching React display title"
```

---

## Self-Review

**1. Spec coverage check:**

| Requirement | Task |
|---|---|
| Canonical style/capabilities in template blocks | Task 1 |
| Instantiation maps to BlockNote props | Task 1 |
| validateTemplate called at instantiation | Task 1 (wired) + Task 2 (expanded) |
| Per-type capability validation | Task 2 |
| maxItems >= minItems | Task 2 |
| dataTable mode validation | Task 2 |
| DataTable React calls interpretDataTable() | Task 3 |
| RepeatableItem DOCX display title matches React | Task 4 |

**2. Placeholder scan:** No TBD, no "similar to Task N", all code blocks are complete.

**3. Type consistency:** `emitRepeatableItem(block, tokens, renderChild, itemIndex = 0)` — the optional param with default means all existing callers compile unchanged. `interpretDataTable()` is already exported from `engine/layout-interpreter`. `defaultLayoutTokens` is exported from `engine/layout-ir`.
