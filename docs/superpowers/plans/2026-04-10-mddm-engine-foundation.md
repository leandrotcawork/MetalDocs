# MDDM Engine Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the foundational MDDM rendering engine — Layout IR, asset resolver, canonicalize/migrate pipeline, a minimum-viable set of DOCX emitters (paragraph, heading, section, field, fieldGroup), `toExternalHTML` hooks on those custom blocks, print stylesheet, backend HTML→PDF endpoint, and client-side `exportDocx()`/`exportPdf()` functions — producing a working end-to-end DOCX + PDF export for simple documents behind a feature flag.

**Architecture:** The engine replaces the external docgen service with a client-side pipeline. The Layout IR is a shared TypeScript module defining absolute tokens and component layout rules. docx.js (MIT) emits DOCX client-side. PDF uses BlockNote's built-in `blocksToFullHTML()` wrapped in a print stylesheet, sent to Gotenberg's Chromium route via a new backend proxy endpoint. Custom blocks gain `toExternalHTML` hooks for full-fidelity HTML export. All emitters and hooks consume the same Layout IR, enforcing the three-tier render compatibility contract from the design spec.

**Tech Stack:** TypeScript 5.6, React 18, BlockNote 0.47.3 (core only, MIT), docx.js 9.x (MIT), Vitest 4.1, Go 1.22, Gotenberg 8.x (Chromium route), Carlito open-source font.

**Spec:** `docs/superpowers/specs/2026-04-10-mddm-unified-document-engine-design.md`

---

## File Structure

### New files (frontend)

```
frontend/apps/web/
├── vitest.config.ts                                     # NEW: Vitest config for web app
└── src/features/
    ├── featureFlags.ts                                  # NEW: feature flag registry
    └── documents/
        ├── browser-editor/
        │   └── SaveBeforeExportDialog.tsx               # NEW: export state contract dialog
        └── mddm-editor/
            ├── MDDMViewer.tsx                           # NEW: read-only viewer
            └── engine/
                ├── index.ts                             # NEW: barrel export
                ├── layout-ir/
                │   ├── index.ts                         # NEW
                │   ├── tokens.ts                        # NEW: LayoutTokens type + defaults
                │   ├── components.ts                    # NEW: component layout rules
                │   ├── compatibility-contract.ts        # NEW: three-tier contract
                │   └── __tests__/
                │       ├── tokens.test.ts               # NEW
                │       ├── components.test.ts           # NEW
                │       └── compatibility-contract.test.ts # NEW
                ├── helpers/
                │   ├── units.ts                         # NEW: mmToTwip, ptToHalfPt, mmToEmu
                │   └── __tests__/
                │       └── units.test.ts                # NEW
                ├── asset-resolver/
                │   ├── index.ts                         # NEW
                │   ├── allowlist.ts                     # NEW: URL regex allowlist
                │   ├── ceilings.ts                      # NEW: resource ceilings
                │   ├── asset-resolver.ts                # NEW: AssetResolver implementation
                │   └── __tests__/
                │       ├── allowlist.test.ts            # NEW
                │       └── asset-resolver.test.ts       # NEW
                ├── canonicalize-migrate/
                │   ├── index.ts                         # NEW
                │   ├── pipeline.ts                      # NEW: canonicalizeAndMigrate()
                │   └── __tests__/
                │       └── pipeline.test.ts             # NEW
                ├── docx-emitter/
                │   ├── index.ts                         # NEW
                │   ├── emitter.ts                       # NEW: mddmToDocx entry point
                │   ├── inline-content.ts                # NEW: text run mapper
                │   ├── emitters/
                │   │   ├── paragraph.ts                 # NEW
                │   │   ├── heading.ts                   # NEW
                │   │   ├── section.ts                   # NEW
                │   │   ├── field.ts                     # NEW
                │   │   └── field-group.ts               # NEW
                │   └── __tests__/
                │       ├── emitter.test.ts              # NEW
                │       ├── inline-content.test.ts       # NEW
                │       ├── paragraph.test.ts            # NEW
                │       ├── heading.test.ts              # NEW
                │       ├── section.test.ts              # NEW
                │       ├── field.test.ts                # NEW
                │       └── field-group.test.ts          # NEW
                ├── external-html/
                │   ├── index.ts                         # NEW
                │   ├── section-html.tsx                 # NEW: toExternalHTML for Section
                │   ├── field-html.tsx                   # NEW: toExternalHTML for Field
                │   ├── field-group-html.tsx             # NEW: toExternalHTML for FieldGroup
                │   └── __tests__/
                │       ├── section-html.test.tsx        # NEW
                │       ├── field-html.test.tsx          # NEW
                │       └── field-group-html.test.tsx    # NEW
                ├── print-stylesheet/
                │   ├── index.ts                         # NEW
                │   └── print-css.ts                     # NEW: CSS as TS string
                ├── export/
                │   ├── index.ts                         # NEW
                │   ├── wrap-print-document.ts           # NEW
                │   ├── export-docx.ts                   # NEW
                │   ├── export-pdf.ts                    # NEW
                │   └── __tests__/
                │       ├── wrap-print-document.test.ts  # NEW
                │       └── export-docx.test.ts          # NEW
                ├── completeness-gate/
                │   ├── block-registry.ts                # NEW
                │   └── __tests__/
                │       └── completeness.test.ts         # NEW
                └── golden/
                    ├── golden-helpers.ts                # NEW: XML normalization
                    ├── fixtures/
                    │   └── 01-simple-po/
                    │       ├── input.mddm.json          # NEW
                    │       ├── expected.full.html       # NEW
                    │       └── expected.document.xml    # NEW
                    └── __tests__/
                        └── golden-runner.test.ts        # NEW
```

### Modified files (frontend)

```
frontend/apps/web/package.json                          # MODIFY: add docx dep, remove xl-docx-exporter, add test script
frontend/apps/web/src/features/documents/mddm-editor/blocks/Section.tsx       # MODIFY: register toExternalHTML
frontend/apps/web/src/features/documents/mddm-editor/blocks/Field.tsx         # MODIFY: register toExternalHTML
frontend/apps/web/src/features/documents/mddm-editor/blocks/FieldGroup.tsx    # MODIFY: register toExternalHTML
frontend/apps/web/src/features/documents/browser-editor/BrowserDocumentEditorView.tsx # MODIFY: wire new export path behind feature flag
```

### New/modified files (backend)

```
internal/platform/render/gotenberg/client.go                                  # MODIFY: add ConvertHTMLToPDF
internal/platform/render/gotenberg/client_test.go                             # MODIFY or CREATE: tests for new method
internal/modules/documents/delivery/http/handler_render_pdf.go                # NEW: POST /documents/{id}/render/pdf handler
internal/modules/documents/delivery/http/handler_render_pdf_test.go           # NEW: tests
internal/modules/documents/delivery/http/handler.go                           # MODIFY: register new route
```

### Infrastructure

```
docker/gotenberg/Dockerfile                             # NEW: Gotenberg image with Carlito font
docker/gotenberg/verify-carlito.sh                      # NEW: Phase 0 gating verification script
```

---

## Part 1 — Project Setup

### Task 1: Install docx.js and remove xl-docx-exporter

**Files:**
- Modify: `frontend/apps/web/package.json`

- [ ] **Step 1: Remove xl-docx-exporter and install docx.js**

Run:
```bash
cd frontend/apps/web
npm uninstall @blocknote/xl-docx-exporter
npm install docx@^9.0.0
```

- [ ] **Step 2: Verify installation**

Run: `cd frontend/apps/web && npm list docx`
Expected: `docx@9.x.x` listed as a direct dependency.

Run: `cd frontend/apps/web && npm list @blocknote/xl-docx-exporter 2>&1 | grep -c "(empty)"` 
Expected: `1` (package no longer present).

- [ ] **Step 3: Commit**

```bash
git add frontend/apps/web/package.json frontend/apps/web/package-lock.json
git commit -m "chore(mddm-engine): add docx.js, remove xl-docx-exporter (AGPL)"
```

### Task 2: Add vitest config and test script for web app

**Files:**
- Create: `frontend/apps/web/vitest.config.ts`
- Modify: `frontend/apps/web/package.json`

- [ ] **Step 1: Create vitest.config.ts**

Write to `frontend/apps/web/vitest.config.ts`:

```ts
import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  test: {
    include: ["src/**/*.test.{ts,tsx}"],
    environment: "jsdom",
    globals: false,
  },
});
```

- [ ] **Step 2: Add jsdom dev dependency**

Run: `cd frontend/apps/web && npm install -D jsdom@^25.0.0`
Expected: `jsdom` added to `devDependencies`.

- [ ] **Step 3: Add test script to package.json**

In `frontend/apps/web/package.json`, add to `scripts`:

```json
"test": "vitest run",
"test:watch": "vitest"
```

- [ ] **Step 4: Verify vitest runs (no tests yet is fine)**

Run: `cd frontend/apps/web && npm test 2>&1 | tail -5`
Expected: Exit code 0 with "No test files found" or similar. If existing tests in `__tests__/` are found and pass, also acceptable.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/vitest.config.ts frontend/apps/web/package.json frontend/apps/web/package-lock.json
git commit -m "chore(mddm-engine): add vitest config and test script for web app"
```

### Task 3: Scaffold engine directory structure

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/index.ts`

- [ ] **Step 1: Create engine directories**

Run:
```bash
cd frontend/apps/web/src/features/documents/mddm-editor
mkdir -p engine/layout-ir/__tests__
mkdir -p engine/helpers/__tests__
mkdir -p engine/asset-resolver/__tests__
mkdir -p engine/canonicalize-migrate/__tests__
mkdir -p engine/docx-emitter/emitters
mkdir -p engine/docx-emitter/__tests__
mkdir -p engine/external-html/__tests__
mkdir -p engine/print-stylesheet
mkdir -p engine/export/__tests__
mkdir -p engine/completeness-gate/__tests__
mkdir -p engine/golden/fixtures/01-simple-po
mkdir -p engine/golden/__tests__
```

- [ ] **Step 2: Create the root barrel export**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/index.ts`:

```ts
// MDDM Rendering Engine — public surface.
// Consumers import from here, not from internal paths.

export * from "./layout-ir";
export * from "./asset-resolver";
export * from "./canonicalize-migrate";
export * from "./docx-emitter";
export * from "./export";
```

- [ ] **Step 3: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/
git commit -m "chore(mddm-engine): scaffold engine directory structure"
```

---

## Part 2 — Layout IR

### Task 4: Implement Layout IR tokens

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/layout-ir/tokens.ts`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/layout-ir/__tests__/tokens.test.ts`

- [ ] **Step 1: Write the failing test**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/layout-ir/__tests__/tokens.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { defaultLayoutTokens, type LayoutTokens } from "../tokens";

describe("Layout IR tokens", () => {
  it("provides A4 page dimensions in mm", () => {
    expect(defaultLayoutTokens.page.widthMm).toBe(210);
    expect(defaultLayoutTokens.page.heightMm).toBe(297);
  });

  it("computes contentWidthMm from page width minus horizontal margins", () => {
    const { page } = defaultLayoutTokens;
    expect(page.contentWidthMm).toBe(page.widthMm - page.marginLeft - page.marginRight);
  });

  it("uses Carlito as the default exportFont", () => {
    expect(defaultLayoutTokens.typography.exportFont).toBe("Carlito");
  });

  it("uses absolute lineHeightPt (no unitless line-heights)", () => {
    expect(typeof defaultLayoutTokens.typography.lineHeightPt).toBe("number");
    expect(defaultLayoutTokens.typography.lineHeightPt).toBeGreaterThan(0);
  });

  it("has theme accent colors", () => {
    expect(defaultLayoutTokens.theme.accent).toMatch(/^#[0-9a-fA-F]{6}$/);
    expect(defaultLayoutTokens.theme.accentLight).toMatch(/^#[0-9a-fA-F]{6}$/);
    expect(defaultLayoutTokens.theme.accentDark).toMatch(/^#[0-9a-fA-F]{6}$/);
    expect(defaultLayoutTokens.theme.accentBorder).toMatch(/^#[0-9a-fA-F]{6}$/);
  });

  it("is readonly-typed (compile-time check)", () => {
    const tokens: LayoutTokens = defaultLayoutTokens;
    expect(tokens).toBe(defaultLayoutTokens);
  });
});
```

- [ ] **Step 2: Run the test — expect failure**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/layout-ir/__tests__/tokens.test.ts`
Expected: FAIL with "Cannot find module '../tokens'" or similar.

- [ ] **Step 3: Implement tokens.ts**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/layout-ir/tokens.ts`:

```ts
// MDDM Layout IR — design tokens shared between React, DOCX, and PDF renderers.
// All dimensions are absolute (mm/pt). No unitless or relative values.

export type LayoutTokens = Readonly<{
  page: Readonly<{
    widthMm: number;
    heightMm: number;
    marginTop: number;
    marginRight: number;
    marginBottom: number;
    marginLeft: number;
    contentWidthMm: number;
  }>;
  typography: Readonly<{
    editorFont: string;
    exportFont: string;
    exportFontFallbacks: readonly string[];
    baseSizePt: number;
    headingSizePt: Readonly<{ h1: number; h2: number; h3: number }>;
    lineHeightPt: number;
    labelSizePt: number;
  }>;
  spacing: Readonly<{
    sectionGapMm: number;
    fieldGapMm: number;
    blockGapMm: number;
    cellPaddingMm: number;
  }>;
  theme: Readonly<{
    accent: string;
    accentLight: string;
    accentDark: string;
    accentBorder: string;
  }>;
}>;

const PAGE_WIDTH_MM = 210;
const PAGE_HEIGHT_MM = 297;
const MARGIN_TOP_MM = 25;
const MARGIN_RIGHT_MM = 20;
const MARGIN_BOTTOM_MM = 25;
const MARGIN_LEFT_MM = 25;

export const defaultLayoutTokens: LayoutTokens = {
  page: {
    widthMm: PAGE_WIDTH_MM,
    heightMm: PAGE_HEIGHT_MM,
    marginTop: MARGIN_TOP_MM,
    marginRight: MARGIN_RIGHT_MM,
    marginBottom: MARGIN_BOTTOM_MM,
    marginLeft: MARGIN_LEFT_MM,
    contentWidthMm: PAGE_WIDTH_MM - MARGIN_LEFT_MM - MARGIN_RIGHT_MM,
  },
  typography: {
    editorFont: "Inter",
    exportFont: "Carlito",
    exportFontFallbacks: ["Liberation Sans", "Arial", "sans-serif"],
    baseSizePt: 11,
    headingSizePt: { h1: 18, h2: 15, h3: 13 },
    lineHeightPt: 15,
    labelSizePt: 9,
  },
  spacing: {
    sectionGapMm: 6,
    fieldGapMm: 3,
    blockGapMm: 2,
    cellPaddingMm: 2,
  },
  theme: {
    accent: "#6b1f2a",
    accentLight: "#f9f3f3",
    accentDark: "#3e1018",
    accentBorder: "#dfc8c8",
  },
};
```

- [ ] **Step 4: Run the test — expect pass**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/layout-ir/__tests__/tokens.test.ts`
Expected: PASS — 6 tests passing.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/layout-ir/tokens.ts frontend/apps/web/src/features/documents/mddm-editor/engine/layout-ir/__tests__/tokens.test.ts
git commit -m "feat(mddm-engine): add Layout IR tokens with A4 defaults and Carlito font"
```

### Task 5: Implement Layout IR component rules

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/layout-ir/components.ts`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/layout-ir/__tests__/components.test.ts`

- [ ] **Step 1: Write the failing test**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/layout-ir/__tests__/components.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { defaultComponentRules, type ComponentRules } from "../components";

describe("Layout IR component rules", () => {
  it("defines Section with fixed 8mm header height and full width", () => {
    expect(defaultComponentRules.section.headerHeightMm).toBe(8);
    expect(defaultComponentRules.section.fullWidth).toBe(true);
    expect(defaultComponentRules.section.headerFontSizePt).toBeGreaterThan(0);
  });

  it("defines Field with 35/65 label/value split", () => {
    expect(defaultComponentRules.field.labelWidthPercent).toBe(35);
    expect(defaultComponentRules.field.valueWidthPercent).toBe(65);
    expect(defaultComponentRules.field.labelWidthPercent + defaultComponentRules.field.valueWidthPercent).toBe(100);
  });

  it("defines FieldGroup with valid column counts", () => {
    expect([1, 2]).toContain(defaultComponentRules.fieldGroup.defaultColumns);
    expect(defaultComponentRules.fieldGroup.fullWidth).toBe(true);
  });

  it("exports the ComponentRules type", () => {
    const rules: ComponentRules = defaultComponentRules;
    expect(rules).toBe(defaultComponentRules);
  });
});
```

- [ ] **Step 2: Run test — expect failure**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/layout-ir/__tests__/components.test.ts`
Expected: FAIL — cannot find module `../components`.

- [ ] **Step 3: Implement components.ts**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/layout-ir/components.ts`:

```ts
// MDDM component layout rules. Reference absolute dimensions so React and
// docx.js produce equivalent output. See spec section "Layout IR".

export type SectionRule = Readonly<{
  headerHeightMm: number;
  headerFontSizePt: number;
  headerFontWeight: "bold" | "normal";
  headerFontColor: string;
  headerBackgroundToken: "theme.accent";
  fullWidth: true;
}>;

export type FieldRule = Readonly<{
  labelWidthPercent: number;
  valueWidthPercent: number;
  labelBackgroundToken: "theme.accentLight";
  labelFontSizePt: number;
  borderColorToken: "theme.accentBorder";
  borderWidthPt: number;
  minHeightMm: number;
}>;

export type FieldGroupRule = Readonly<{
  defaultColumns: 1 | 2;
  gapMm: number;
  fullWidth: true;
}>;

export type ComponentRules = Readonly<{
  section: SectionRule;
  field: FieldRule;
  fieldGroup: FieldGroupRule;
}>;

export const defaultComponentRules: ComponentRules = {
  section: {
    headerHeightMm: 8,
    headerFontSizePt: 13,
    headerFontWeight: "bold",
    headerFontColor: "#ffffff",
    headerBackgroundToken: "theme.accent",
    fullWidth: true,
  },
  field: {
    labelWidthPercent: 35,
    valueWidthPercent: 65,
    labelBackgroundToken: "theme.accentLight",
    labelFontSizePt: 9,
    borderColorToken: "theme.accentBorder",
    borderWidthPt: 0.5,
    minHeightMm: 7,
  },
  fieldGroup: {
    defaultColumns: 2,
    gapMm: 0,
    fullWidth: true,
  },
};
```

- [ ] **Step 4: Run test — expect pass**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/layout-ir/__tests__/components.test.ts`
Expected: PASS — 4 tests passing.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/layout-ir/components.ts frontend/apps/web/src/features/documents/mddm-editor/engine/layout-ir/__tests__/components.test.ts
git commit -m "feat(mddm-engine): add Layout IR component rules for Section/Field/FieldGroup"
```

### Task 6: Implement compatibility contract

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/layout-ir/compatibility-contract.ts`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/layout-ir/__tests__/compatibility-contract.test.ts`

- [ ] **Step 1: Write the failing test**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/layout-ir/__tests__/compatibility-contract.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { COMPATIBILITY_CONTRACT, isForbiddenConstruct } from "../compatibility-contract";

describe("Compatibility contract", () => {
  it("defines tier2 tolerance < 2% for editor vs PDF", () => {
    expect(COMPATIBILITY_CONTRACT.tier2.pixelDiffEditorToPdf).toBeLessThan(0.02 + Number.EPSILON);
  });

  it("defines tier2 tolerance < 5% for editor vs DOCX", () => {
    expect(COMPATIBILITY_CONTRACT.tier2.pixelDiffEditorToDocx).toBeLessThan(0.05 + Number.EPSILON);
  });

  it("forbids auto-fit columns", () => {
    expect(isForbiddenConstruct("autoFitColumns")).toBe(true);
  });

  it("forbids Flexbox layouts", () => {
    expect(isForbiddenConstruct("flexbox")).toBe(true);
  });

  it("forbids unitless line-heights", () => {
    expect(isForbiddenConstruct("unitlessLineHeight")).toBe(true);
  });

  it("caps nested DataTable depth at 2 levels", () => {
    expect(COMPATIBILITY_CONTRACT.forbidden.nestedDataTableMaxDepth).toBe(2);
  });

  it("allows known-safe constructs", () => {
    expect(isForbiddenConstruct("absoluteLineHeight")).toBe(false);
    expect(isForbiddenConstruct("explicitColumnWidths")).toBe(false);
  });
});
```

- [ ] **Step 2: Run test — expect failure**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/layout-ir/__tests__/compatibility-contract.test.ts`
Expected: FAIL — cannot find module.

- [ ] **Step 3: Implement compatibility-contract.ts**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/layout-ir/compatibility-contract.ts`:

```ts
// Render Compatibility Contract — three tiers governing editor/DOCX/PDF parity.
// See spec section "Render Compatibility Contract".

export type CompatibilityContract = Readonly<{
  tier1: Readonly<{
    blockStructure: "byte-exact";
    blockProps: "byte-exact";
    colors: "byte-exact";
    fontFamily: "byte-exact";
    columnProportions: "byte-exact";
  }>;
  tier2: Readonly<{
    pixelDiffEditorToPdf: number;
    pixelDiffEditorToDocx: number;
    verticalCellDriftPx: number;
    horizontalCharDriftPx: number;
  }>;
  forbidden: Readonly<{
    autoFitColumns: "error";
    unitlessLineHeight: "error";
    emLineHeight: "error";
    negativeMargins: "error";
    flexbox: "error";
    gridFrUnits: "error";
    nestedDataTableMaxDepth: number;
    percentageFontSize: "error";
    transforms: "error";
    filters: "error";
    fixedPositioning: "error";
    viewportUnits: "error";
    externalUrlsDuringPdfExport: "error";
  }>;
  degradation: Readonly<{
    logLevel: "warn";
    telemetry: boolean;
    userNotification: "toast";
  }>;
}>;

export const COMPATIBILITY_CONTRACT: CompatibilityContract = {
  tier1: {
    blockStructure: "byte-exact",
    blockProps: "byte-exact",
    colors: "byte-exact",
    fontFamily: "byte-exact",
    columnProportions: "byte-exact",
  },
  tier2: {
    pixelDiffEditorToPdf: 0.02,
    pixelDiffEditorToDocx: 0.05,
    verticalCellDriftPx: 3,
    horizontalCharDriftPx: 1,
  },
  forbidden: {
    autoFitColumns: "error",
    unitlessLineHeight: "error",
    emLineHeight: "error",
    negativeMargins: "error",
    flexbox: "error",
    gridFrUnits: "error",
    nestedDataTableMaxDepth: 2,
    percentageFontSize: "error",
    transforms: "error",
    filters: "error",
    fixedPositioning: "error",
    viewportUnits: "error",
    externalUrlsDuringPdfExport: "error",
  },
  degradation: {
    logLevel: "warn",
    telemetry: true,
    userNotification: "toast",
  },
};

export type ForbiddenConstruct = keyof CompatibilityContract["forbidden"];

const FORBIDDEN_SET: ReadonlySet<string> = new Set(Object.keys(COMPATIBILITY_CONTRACT.forbidden));

export function isForbiddenConstruct(name: string): boolean {
  return FORBIDDEN_SET.has(name) && name !== "nestedDataTableMaxDepth";
}
```

- [ ] **Step 4: Run test — expect pass**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/layout-ir/__tests__/compatibility-contract.test.ts`
Expected: PASS — 7 tests passing.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/layout-ir/compatibility-contract.ts frontend/apps/web/src/features/documents/mddm-editor/engine/layout-ir/__tests__/compatibility-contract.test.ts
git commit -m "feat(mddm-engine): add three-tier render compatibility contract"
```

### Task 7: Create layout-ir barrel export

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/layout-ir/index.ts`

- [ ] **Step 1: Write the barrel export**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/layout-ir/index.ts`:

```ts
export { defaultLayoutTokens, type LayoutTokens } from "./tokens";
export {
  defaultComponentRules,
  type ComponentRules,
  type SectionRule,
  type FieldRule,
  type FieldGroupRule,
} from "./components";
export {
  COMPATIBILITY_CONTRACT,
  isForbiddenConstruct,
  type CompatibilityContract,
  type ForbiddenConstruct,
} from "./compatibility-contract";
```

- [ ] **Step 2: Verify the barrel compiles**

Run: `cd frontend/apps/web && npx tsc --noEmit --project tsconfig.json 2>&1 | grep -E "layout-ir" | head -5`
Expected: No errors referencing `layout-ir/index.ts` or its imports.

- [ ] **Step 3: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/layout-ir/index.ts
git commit -m "feat(mddm-engine): add layout-ir barrel export"
```

---

## Part 3 — Unit Helpers

### Task 8: Implement unit conversion helpers

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/helpers/units.ts`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/helpers/__tests__/units.test.ts`

- [ ] **Step 1: Write the failing test**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/helpers/__tests__/units.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { mmToTwip, ptToHalfPt, mmToEmu, mmToPt, percentToTablePct } from "../units";

describe("Unit conversions", () => {
  it("mmToTwip: 25mm equals 1417 twips (OOXML twentieths of a point)", () => {
    expect(mmToTwip(25)).toBe(1417);
  });

  it("mmToTwip: 0mm is 0 twips", () => {
    expect(mmToTwip(0)).toBe(0);
  });

  it("ptToHalfPt: 11pt equals 22 half-points", () => {
    expect(ptToHalfPt(11)).toBe(22);
  });

  it("ptToHalfPt: rounds to nearest integer", () => {
    expect(ptToHalfPt(11.25)).toBe(23);
  });

  it("mmToEmu: 10mm equals 360000 EMU", () => {
    expect(mmToEmu(10)).toBe(360000);
  });

  it("mmToPt: 10mm equals 28.35pt approximately", () => {
    expect(mmToPt(10)).toBeCloseTo(28.35, 2);
  });

  it("percentToTablePct: 35 percent equals 1750 fiftieths", () => {
    expect(percentToTablePct(35)).toBe(1750);
  });

  it("percentToTablePct: clamps out-of-range values", () => {
    expect(percentToTablePct(150)).toBe(5000);
    expect(percentToTablePct(-10)).toBe(0);
  });
});
```

- [ ] **Step 2: Run test — expect failure**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/helpers/__tests__/units.test.ts`
Expected: FAIL — cannot find module `../units`.

- [ ] **Step 3: Implement units.ts**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/helpers/units.ts`:

```ts
// Unit conversions for OOXML / docx.js output.
// Reference: ECMA-376 Part 1 §17.18.74 (ST_TwipsMeasure),
// §17.3.1.13 (ST_HalfPoint), §20.1.2.1 (ST_EmuAbsMeasure).

// 1 inch = 25.4 mm = 1440 twips (twentieths of a point)
const TWIPS_PER_MM = 1440 / 25.4;

// 1 inch = 914400 EMU (English Metric Units)
const EMU_PER_MM = 914400 / 25.4;

// 1 inch = 72 points
const POINTS_PER_MM = 72 / 25.4;

export function mmToTwip(mm: number): number {
  return Math.round(mm * TWIPS_PER_MM);
}

export function mmToEmu(mm: number): number {
  return Math.round(mm * EMU_PER_MM);
}

export function mmToPt(mm: number): number {
  return mm * POINTS_PER_MM;
}

// OOXML font sizes are stored in half-points (so size 22 = 11pt)
export function ptToHalfPt(pt: number): number {
  return Math.round(pt * 2);
}

// OOXML table column widths can be expressed in fiftieths of a percent (0-5000)
// when type="pct". 100% = 5000.
export function percentToTablePct(percent: number): number {
  const clamped = Math.max(0, Math.min(100, percent));
  return Math.round(clamped * 50);
}
```

- [ ] **Step 4: Run test — expect pass**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/helpers/__tests__/units.test.ts`
Expected: PASS — 8 tests passing.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/helpers/units.ts frontend/apps/web/src/features/documents/mddm-editor/engine/helpers/__tests__/units.test.ts
git commit -m "feat(mddm-engine): add mm/pt/twip/EMU unit conversion helpers"
```

---

## Part 4 — Asset Resolver

### Task 9: Implement URL allowlist

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/asset-resolver/allowlist.ts`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/asset-resolver/__tests__/allowlist.test.ts`

- [ ] **Step 1: Write the failing test**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/asset-resolver/__tests__/allowlist.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { isAllowlistedAssetUrl } from "../allowlist";

describe("Asset URL allowlist", () => {
  it("allows /api/images/{uuid} URLs", () => {
    expect(isAllowlistedAssetUrl("/api/images/00000000-0000-4000-8000-000000000001")).toBe(true);
    expect(isAllowlistedAssetUrl("/api/images/abcdef12-3456-7890-abcd-ef1234567890")).toBe(true);
  });

  it("allows full URLs pointing at the same origin", () => {
    expect(isAllowlistedAssetUrl("https://metaldocs.example/api/images/00000000-0000-4000-8000-000000000001")).toBe(true);
  });

  it("rejects arbitrary external URLs", () => {
    expect(isAllowlistedAssetUrl("https://evil.example/pwn.png")).toBe(false);
    expect(isAllowlistedAssetUrl("http://attacker.net/image")).toBe(false);
  });

  it("rejects javascript: and data: protocols at the allowlist level", () => {
    expect(isAllowlistedAssetUrl("javascript:alert(1)")).toBe(false);
    expect(isAllowlistedAssetUrl("data:text/html,<script>")).toBe(false);
  });

  it("rejects non-UUID image paths", () => {
    expect(isAllowlistedAssetUrl("/api/images/../etc/passwd")).toBe(false);
    expect(isAllowlistedAssetUrl("/api/images/not-a-uuid")).toBe(false);
  });

  it("rejects paths outside /api/images/", () => {
    expect(isAllowlistedAssetUrl("/api/secrets/token")).toBe(false);
    expect(isAllowlistedAssetUrl("/api/images_v2/foo")).toBe(false);
  });
});
```

- [ ] **Step 2: Run test — expect failure**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/asset-resolver/__tests__/allowlist.test.ts`
Expected: FAIL — cannot find module.

- [ ] **Step 3: Implement allowlist.ts**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/asset-resolver/allowlist.ts`:

```ts
// Asset URL allowlist. Only the MetalDocs image endpoint is permitted,
// keyed by UUID. All external or alternate paths are rejected. This is a
// pre-fetch check; the actual request still carries the browser session
// cookie, so auth enforcement still lives on the backend.

const IMAGE_PATH_REGEX = /^\/api\/images\/[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}$/i;

export function isAllowlistedAssetUrl(url: string): boolean {
  if (typeof url !== "string" || url.length === 0) {
    return false;
  }

  // Reject dangerous pseudo-protocols explicitly.
  const lowered = url.toLowerCase().trim();
  if (lowered.startsWith("javascript:") || lowered.startsWith("data:") || lowered.startsWith("file:")) {
    return false;
  }

  // Extract pathname: either a relative /api/images/... or an absolute URL.
  let pathname: string;
  if (url.startsWith("/")) {
    pathname = url;
  } else {
    try {
      const parsed = new URL(url, "https://placeholder.local");
      if (parsed.protocol !== "http:" && parsed.protocol !== "https:") {
        return false;
      }
      pathname = parsed.pathname;
    } catch {
      return false;
    }
  }

  return IMAGE_PATH_REGEX.test(pathname);
}
```

- [ ] **Step 4: Run test — expect pass**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/asset-resolver/__tests__/allowlist.test.ts`
Expected: PASS — 6 tests passing.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/asset-resolver/allowlist.ts frontend/apps/web/src/features/documents/mddm-editor/engine/asset-resolver/__tests__/allowlist.test.ts
git commit -m "feat(mddm-engine): add asset URL allowlist for export pipelines"
```

### Task 10: Implement resource ceilings

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/asset-resolver/ceilings.ts`

- [ ] **Step 1: Implement ceilings.ts**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/asset-resolver/ceilings.ts`:

```ts
// Resource ceilings for asset resolution and export. Mirrors spec section
// "Global Resource Ceilings". Exceeding any limit aborts the export with a
// specific error message.

export const RESOURCE_CEILINGS = {
  // Per asset
  maxImageSizeBytes: 5 * 1024 * 1024, // 5 MB
  maxImageDimensionPx: 10000,

  // Per document
  maxTotalAssetBytes: 50 * 1024 * 1024, // 50 MB
  maxImagesPerDocument: 200,

  // Content-level
  maxBlockCount: 5000,
  maxNestingDepth: 10,
  maxTextRunLength: 100000,

  // Pipeline timings
  maxDocxGenerationMs: 30_000,
  maxHtmlPayloadBytes: 10 * 1024 * 1024, // 10 MB
  maxGotenbergConversionMs: 60_000,
  maxConcurrentExportsPerUser: 3,
} as const;

export type ResourceCeilings = typeof RESOURCE_CEILINGS;

export class ResourceCeilingExceededError extends Error {
  constructor(
    public readonly limit: keyof ResourceCeilings,
    public readonly observed: number,
    public readonly allowed: number,
  ) {
    super(`Resource ceiling "${String(limit)}" exceeded: observed ${observed}, allowed ${allowed}`);
    this.name = "ResourceCeilingExceededError";
  }
}
```

- [ ] **Step 2: Verify the module imports cleanly**

Run: `cd frontend/apps/web && npx tsc --noEmit 2>&1 | grep ceilings | head -5`
Expected: No errors referencing `ceilings.ts`.

- [ ] **Step 3: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/asset-resolver/ceilings.ts
git commit -m "feat(mddm-engine): add resource ceiling constants for export pipeline"
```

### Task 11: Implement AssetResolver

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/asset-resolver/asset-resolver.ts`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/asset-resolver/__tests__/asset-resolver.test.ts`

- [ ] **Step 1: Write the failing test**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/asset-resolver/__tests__/asset-resolver.test.ts`:

```ts
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { AssetResolver, AssetResolverError } from "../asset-resolver";

// Minimal PNG (1x1 red pixel). Starts with PNG magic bytes 89 50 4E 47.
const TINY_PNG = new Uint8Array([
  0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a,
  0x00, 0x00, 0x00, 0x0d, 0x49, 0x48, 0x44, 0x52,
  0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
  0x08, 0x02, 0x00, 0x00, 0x00, 0x90, 0x77, 0x53,
  0xde, 0x00, 0x00, 0x00, 0x0c, 0x49, 0x44, 0x41,
  0x54, 0x08, 0x99, 0x63, 0xf8, 0xcf, 0xc0, 0x00,
  0x00, 0x00, 0x03, 0x00, 0x01, 0x5a, 0x4d, 0x7f,
  0x5c, 0x00, 0x00, 0x00, 0x00, 0x49, 0x45, 0x4e,
  0x44, 0xae, 0x42, 0x60, 0x82,
]);

const VALID_URL = "/api/images/00000000-0000-4000-8000-000000000001";

function mockFetchOnce(response: Response): void {
  vi.stubGlobal("fetch", vi.fn().mockResolvedValue(response));
}

describe("AssetResolver", () => {
  let resolver: AssetResolver;

  beforeEach(() => {
    resolver = new AssetResolver();
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("rejects URLs that fail the allowlist before fetching", async () => {
    const fetchSpy = vi.fn();
    vi.stubGlobal("fetch", fetchSpy);

    await expect(resolver.resolveAsset("https://evil.example/pwn.png"))
      .rejects.toBeInstanceOf(AssetResolverError);
    expect(fetchSpy).not.toHaveBeenCalled();
  });

  it("rejects assets exceeding maxImageSizeBytes", async () => {
    const huge = new Uint8Array(6 * 1024 * 1024); // 6MB > 5MB limit
    mockFetchOnce(new Response(huge, { status: 200, headers: { "Content-Type": "image/png" } }));

    await expect(resolver.resolveAsset(VALID_URL))
      .rejects.toThrow(/ceiling/i);
  });

  it("rejects content whose magic bytes do not match the Content-Type", async () => {
    const badBytes = new Uint8Array([0x00, 0x00, 0x00, 0x00]);
    mockFetchOnce(new Response(badBytes, { status: 200, headers: { "Content-Type": "image/png" } }));

    await expect(resolver.resolveAsset(VALID_URL))
      .rejects.toThrow(/magic/i);
  });

  it("returns resolved bytes and metadata for a valid PNG", async () => {
    mockFetchOnce(new Response(TINY_PNG, { status: 200, headers: { "Content-Type": "image/png" } }));

    const asset = await resolver.resolveAsset(VALID_URL);
    expect(asset.mimeType).toBe("image/png");
    expect(asset.bytes.byteLength).toBe(TINY_PNG.byteLength);
    expect(asset.sizeBytes).toBe(TINY_PNG.byteLength);
  });

  it("rejects disallowed MIME types like image/svg+xml", async () => {
    mockFetchOnce(new Response(new Uint8Array([0x3c, 0x73, 0x76, 0x67]), {
      status: 200,
      headers: { "Content-Type": "image/svg+xml" },
    }));

    await expect(resolver.resolveAsset(VALID_URL))
      .rejects.toThrow(/mime/i);
  });
});
```

- [ ] **Step 2: Run test — expect failure**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/asset-resolver/__tests__/asset-resolver.test.ts`
Expected: FAIL — cannot find module `../asset-resolver`.

- [ ] **Step 3: Implement asset-resolver.ts**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/asset-resolver/asset-resolver.ts`:

```ts
import { isAllowlistedAssetUrl } from "./allowlist";
import { RESOURCE_CEILINGS, ResourceCeilingExceededError } from "./ceilings";

export type AllowedMimeType = "image/png" | "image/jpeg" | "image/gif" | "image/webp";

export type ResolvedAsset = {
  bytes: Uint8Array;
  mimeType: AllowedMimeType;
  sizeBytes: number;
};

export class AssetResolverError extends Error {
  constructor(message: string, public readonly code: string) {
    super(message);
    this.name = "AssetResolverError";
  }
}

const ALLOWED_MIME: ReadonlySet<AllowedMimeType> = new Set([
  "image/png",
  "image/jpeg",
  "image/gif",
  "image/webp",
]);

// Magic-byte signatures used to validate declared Content-Type.
function detectMimeByMagic(bytes: Uint8Array): AllowedMimeType | null {
  if (bytes.length >= 8 &&
      bytes[0] === 0x89 && bytes[1] === 0x50 && bytes[2] === 0x4e && bytes[3] === 0x47) {
    return "image/png";
  }
  if (bytes.length >= 3 && bytes[0] === 0xff && bytes[1] === 0xd8 && bytes[2] === 0xff) {
    return "image/jpeg";
  }
  if (bytes.length >= 6 && bytes[0] === 0x47 && bytes[1] === 0x49 && bytes[2] === 0x46 &&
      bytes[3] === 0x38 && (bytes[4] === 0x37 || bytes[4] === 0x39) && bytes[5] === 0x61) {
    return "image/gif";
  }
  if (bytes.length >= 12 &&
      bytes[0] === 0x52 && bytes[1] === 0x49 && bytes[2] === 0x46 && bytes[3] === 0x46 &&
      bytes[8] === 0x57 && bytes[9] === 0x45 && bytes[10] === 0x42 && bytes[11] === 0x50) {
    return "image/webp";
  }
  return null;
}

export class AssetResolver {
  async resolveAsset(url: string): Promise<ResolvedAsset> {
    if (!isAllowlistedAssetUrl(url)) {
      throw new AssetResolverError(`Asset URL not allowlisted: ${url}`, "NOT_ALLOWLISTED");
    }

    const response = await fetch(url, { credentials: "same-origin" });
    if (!response.ok) {
      throw new AssetResolverError(
        `Asset fetch failed: ${response.status} ${response.statusText}`,
        "FETCH_FAILED",
      );
    }

    const declaredType = (response.headers.get("Content-Type") ?? "").split(";")[0]!.trim().toLowerCase() as AllowedMimeType;
    if (!ALLOWED_MIME.has(declaredType)) {
      throw new AssetResolverError(`Disallowed MIME type: ${declaredType}`, "MIME_NOT_ALLOWED");
    }

    const buffer = await response.arrayBuffer();
    const bytes = new Uint8Array(buffer);

    if (bytes.byteLength > RESOURCE_CEILINGS.maxImageSizeBytes) {
      throw new ResourceCeilingExceededError(
        "maxImageSizeBytes",
        bytes.byteLength,
        RESOURCE_CEILINGS.maxImageSizeBytes,
      );
    }

    const detected = detectMimeByMagic(bytes);
    if (detected === null || detected !== declaredType) {
      throw new AssetResolverError(
        `Asset magic bytes do not match declared Content-Type: declared=${declaredType}, detected=${detected ?? "unknown"}`,
        "MAGIC_MISMATCH",
      );
    }

    return {
      bytes,
      mimeType: detected,
      sizeBytes: bytes.byteLength,
    };
  }
}
```

- [ ] **Step 4: Run test — expect pass**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/asset-resolver/__tests__/asset-resolver.test.ts`
Expected: PASS — 5 tests passing.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/asset-resolver/asset-resolver.ts frontend/apps/web/src/features/documents/mddm-editor/engine/asset-resolver/__tests__/asset-resolver.test.ts
git commit -m "feat(mddm-engine): add AssetResolver with allowlist + magic-byte validation"
```

### Task 12: Create asset-resolver barrel export

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/asset-resolver/index.ts`

- [ ] **Step 1: Write the barrel export**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/asset-resolver/index.ts`:

```ts
export { isAllowlistedAssetUrl } from "./allowlist";
export { RESOURCE_CEILINGS, ResourceCeilingExceededError, type ResourceCeilings } from "./ceilings";
export {
  AssetResolver,
  AssetResolverError,
  type AllowedMimeType,
  type ResolvedAsset,
} from "./asset-resolver";
```

- [ ] **Step 2: Verify compilation**

Run: `cd frontend/apps/web && npx tsc --noEmit 2>&1 | grep asset-resolver | head -5`
Expected: No errors.

- [ ] **Step 3: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/asset-resolver/index.ts
git commit -m "feat(mddm-engine): add asset-resolver barrel export"
```

---

## Part 5 — Canonicalize + Migrate Pipeline

### Task 13: Implement canonicalizeAndMigrate pipeline

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/canonicalize-migrate/pipeline.ts`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/canonicalize-migrate/__tests__/pipeline.test.ts`

- [ ] **Step 1: Write the failing test**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/canonicalize-migrate/__tests__/pipeline.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import {
  canonicalizeAndMigrate,
  CURRENT_MDDM_VERSION,
  MigrationError,
} from "../pipeline";
import type { MDDMEnvelope } from "../../../adapter";

function makeEnvelope(overrides: Partial<MDDMEnvelope> = {}): MDDMEnvelope {
  return {
    mddm_version: CURRENT_MDDM_VERSION,
    template_ref: null,
    blocks: [],
    ...overrides,
  };
}

describe("canonicalizeAndMigrate", () => {
  it("returns the envelope unchanged when already at current version", async () => {
    const envelope = makeEnvelope({
      blocks: [
        { id: "b1", type: "paragraph", props: {}, content: [{ type: "text", text: "hello" }], children: [] },
      ],
    });
    const result = await canonicalizeAndMigrate(envelope);
    expect(result.mddm_version).toBe(CURRENT_MDDM_VERSION);
    expect(result.blocks).toHaveLength(1);
  });

  it("sorts object keys for deterministic canonicalization", async () => {
    const envelope = makeEnvelope({
      blocks: [{ zkey: "z", id: "b1", type: "paragraph", props: {}, children: [] } as any],
    });
    const result = await canonicalizeAndMigrate(envelope);
    const firstBlockKeys = Object.keys(result.blocks[0] as Record<string, unknown>);
    const sorted = [...firstBlockKeys].sort();
    expect(firstBlockKeys).toEqual(sorted);
  });

  it("throws MigrationError when version is newer than current", async () => {
    const envelope = makeEnvelope({ mddm_version: CURRENT_MDDM_VERSION + 100 });
    await expect(canonicalizeAndMigrate(envelope)).rejects.toBeInstanceOf(MigrationError);
  });

  it("throws MigrationError when version is missing", async () => {
    const envelope = { template_ref: null, blocks: [] } as unknown as MDDMEnvelope;
    await expect(canonicalizeAndMigrate(envelope)).rejects.toBeInstanceOf(MigrationError);
  });
});
```

- [ ] **Step 2: Run test — expect failure**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/canonicalize-migrate/__tests__/pipeline.test.ts`
Expected: FAIL — cannot find module `../pipeline`.

- [ ] **Step 3: Implement pipeline.ts**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/canonicalize-migrate/pipeline.ts`:

```ts
import { canonicalizeMDDM } from "../../../../../../../../shared/schemas/canonicalize";
import type { MDDMEnvelope } from "../../adapter";

// The highest mddm_version this engine knows how to render. Bumped whenever
// a new forward migration is added to MIGRATIONS below.
export const CURRENT_MDDM_VERSION = 1;

export class MigrationError extends Error {
  constructor(message: string, public readonly code: string) {
    super(message);
    this.name = "MigrationError";
  }
}

type Migration = (envelope: MDDMEnvelope) => MDDMEnvelope;

// Registered forward migrations keyed by the source version they upgrade FROM.
// Example: MIGRATIONS[1] upgrades a v1 envelope to v2.
// Plan 1 only supports the current version; future plans register migrations here.
const MIGRATIONS: Record<number, Migration> = {};

export async function canonicalizeAndMigrate(envelope: MDDMEnvelope): Promise<MDDMEnvelope> {
  if (envelope === null || typeof envelope !== "object") {
    throw new MigrationError("Envelope is not an object", "INVALID_ENVELOPE");
  }

  const version = (envelope as { mddm_version?: unknown }).mddm_version;
  if (typeof version !== "number" || !Number.isInteger(version) || version < 1) {
    throw new MigrationError("Envelope missing a valid mddm_version", "MISSING_VERSION");
  }

  if (version > CURRENT_MDDM_VERSION) {
    throw new MigrationError(
      `Envelope version ${version} is newer than current engine version ${CURRENT_MDDM_VERSION}`,
      "VERSION_TOO_NEW",
    );
  }

  let current: MDDMEnvelope = envelope;
  while ((current.mddm_version ?? 0) < CURRENT_MDDM_VERSION) {
    const from = current.mddm_version ?? 0;
    const migrate = MIGRATIONS[from];
    if (!migrate) {
      throw new MigrationError(
        `No migration registered from version ${from} to ${from + 1}`,
        "MIGRATION_MISSING",
      );
    }
    current = migrate(current);
  }

  // Canonicalize last so the returned envelope is stable regardless of key ordering
  // in the input.
  return canonicalizeMDDM(current) as MDDMEnvelope;
}
```

- [ ] **Step 4: Run test — expect pass**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/canonicalize-migrate/__tests__/pipeline.test.ts`
Expected: PASS — 4 tests passing.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/canonicalize-migrate/pipeline.ts frontend/apps/web/src/features/documents/mddm-editor/engine/canonicalize-migrate/__tests__/pipeline.test.ts
git commit -m "feat(mddm-engine): add canonicalize+migrate pipeline stub (current version)"
```

### Task 14: Create canonicalize-migrate barrel export

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/canonicalize-migrate/index.ts`

- [ ] **Step 1: Write barrel**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/canonicalize-migrate/index.ts`:

```ts
export {
  canonicalizeAndMigrate,
  CURRENT_MDDM_VERSION,
  MigrationError,
} from "./pipeline";
```

- [ ] **Step 2: Verify compilation**

Run: `cd frontend/apps/web && npx tsc --noEmit 2>&1 | grep canonicalize-migrate | head -5`
Expected: No errors.

- [ ] **Step 3: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/canonicalize-migrate/index.ts
git commit -m "feat(mddm-engine): add canonicalize-migrate barrel export"
```

---

## Part 6 — Inline Content Mapper

### Task 15: Implement inline content mapper for docx.js

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/inline-content.ts`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/inline-content.test.ts`

- [ ] **Step 1: Write the failing test**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/inline-content.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { TextRun } from "docx";
import { inlineContentToRuns } from "../inline-content";
import { defaultLayoutTokens } from "../../layout-ir";

describe("inlineContentToRuns", () => {
  it("emits a single TextRun for plain text", () => {
    const runs = inlineContentToRuns(
      [{ type: "text", text: "Hello world", styles: {} }],
      defaultLayoutTokens,
    );
    expect(runs).toHaveLength(1);
    expect(runs[0]).toBeInstanceOf(TextRun);
  });

  it("emits bold runs when styles.bold is true", () => {
    const runs = inlineContentToRuns(
      [{ type: "text", text: "Bold", styles: { bold: true } }],
      defaultLayoutTokens,
    );
    expect(runs).toHaveLength(1);
    const xml = (runs[0] as TextRun).options;
    expect(xml).toMatchObject({ bold: true });
  });

  it("emits multiple runs for mixed styling", () => {
    const runs = inlineContentToRuns(
      [
        { type: "text", text: "Normal ", styles: {} },
        { type: "text", text: "bold", styles: { bold: true } },
        { type: "text", text: " and ", styles: {} },
        { type: "text", text: "italic", styles: { italic: true } },
      ],
      defaultLayoutTokens,
    );
    expect(runs).toHaveLength(4);
  });

  it("honors exportFont and baseSizePt from tokens", () => {
    const runs = inlineContentToRuns(
      [{ type: "text", text: "Hi", styles: {} }],
      defaultLayoutTokens,
    );
    const options = (runs[0] as TextRun).options;
    expect(options.font).toBe(defaultLayoutTokens.typography.exportFont);
    expect(options.size).toBe(defaultLayoutTokens.typography.baseSizePt * 2);
  });

  it("returns empty array for empty input", () => {
    expect(inlineContentToRuns([], defaultLayoutTokens)).toEqual([]);
  });
});
```

- [ ] **Step 2: Run test — expect failure**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/docx-emitter/__tests__/inline-content.test.ts`
Expected: FAIL — cannot find module.

- [ ] **Step 3: Implement inline-content.ts**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/inline-content.ts`:

```ts
import { TextRun } from "docx";
import type { LayoutTokens } from "../layout-ir";
import { ptToHalfPt } from "../helpers/units";

// BlockNote inline content style shape — a subset we support today. Mirrors the
// styles recognized by the MDDM adapter (bold, italic, underline, strike, code).
export type InlineStyles = {
  bold?: boolean;
  italic?: boolean;
  underline?: boolean;
  strike?: boolean;
  code?: boolean;
};

export type InlineContent = {
  type: "text";
  text: string;
  styles?: InlineStyles;
};

export function inlineContentToRuns(
  content: readonly InlineContent[],
  tokens: LayoutTokens,
): TextRun[] {
  if (!content || content.length === 0) {
    return [];
  }

  const font = tokens.typography.exportFont;
  const baseHalfPt = ptToHalfPt(tokens.typography.baseSizePt);

  return content.map((node) => {
    const styles = node.styles ?? {};
    return new TextRun({
      text: node.text,
      font: styles.code ? "Consolas" : font,
      size: baseHalfPt,
      bold: styles.bold === true,
      italics: styles.italic === true,
      underline: styles.underline === true ? {} : undefined,
      strike: styles.strike === true,
    });
  });
}
```

- [ ] **Step 4: Run test — expect pass**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/docx-emitter/__tests__/inline-content.test.ts`
Expected: PASS — 5 tests passing.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/inline-content.ts frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/inline-content.test.ts
git commit -m "feat(mddm-engine): add inline content mapper (text runs with styles)"
```

---

## Part 7 — DOCX Emitters (MVP set)

### Task 16: Implement paragraph emitter

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/paragraph.ts`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/paragraph.test.ts`

- [ ] **Step 1: Write the failing test**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/paragraph.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { Paragraph } from "docx";
import { emitParagraph } from "../emitters/paragraph";
import { defaultLayoutTokens } from "../../layout-ir";

describe("emitParagraph", () => {
  it("emits one docx Paragraph for a paragraph block with text", () => {
    const block = {
      id: "p1",
      type: "paragraph",
      props: {},
      content: [{ type: "text", text: "Hello", styles: {} }],
      children: [],
    };
    const out = emitParagraph(block as any, defaultLayoutTokens);
    expect(out).toHaveLength(1);
    expect(out[0]).toBeInstanceOf(Paragraph);
  });

  it("emits an empty Paragraph when content is empty", () => {
    const block = { id: "p2", type: "paragraph", props: {}, content: [], children: [] };
    const out = emitParagraph(block as any, defaultLayoutTokens);
    expect(out).toHaveLength(1);
    expect(out[0]).toBeInstanceOf(Paragraph);
  });
});
```

- [ ] **Step 2: Run test — expect failure**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/docx-emitter/__tests__/paragraph.test.ts`
Expected: FAIL — cannot find module.

- [ ] **Step 3: Implement paragraph.ts**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/paragraph.ts`:

```ts
import { Paragraph } from "docx";
import type { LayoutTokens } from "../../layout-ir";
import { inlineContentToRuns, type InlineContent } from "../inline-content";

export type ParagraphBlock = {
  id: string;
  type: "paragraph";
  props?: Record<string, unknown>;
  content?: InlineContent[];
  children?: unknown[];
};

export function emitParagraph(
  block: ParagraphBlock,
  tokens: LayoutTokens,
): Paragraph[] {
  const runs = inlineContentToRuns(block.content ?? [], tokens);
  return [
    new Paragraph({
      children: runs,
    }),
  ];
}
```

- [ ] **Step 4: Run test — expect pass**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/docx-emitter/__tests__/paragraph.test.ts`
Expected: PASS — 2 tests passing.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/paragraph.ts frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/paragraph.test.ts
git commit -m "feat(mddm-engine): add paragraph DOCX emitter"
```

### Task 17: Implement heading emitter

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/heading.ts`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/heading.test.ts`

- [ ] **Step 1: Write the failing test**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/heading.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { Paragraph, HeadingLevel } from "docx";
import { emitHeading } from "../emitters/heading";
import { defaultLayoutTokens } from "../../layout-ir";

describe("emitHeading", () => {
  it("emits a Paragraph with HEADING_1 for level 1", () => {
    const block = {
      id: "h1",
      type: "heading",
      props: { level: 1 },
      content: [{ type: "text", text: "Title", styles: {} }],
      children: [],
    };
    const out = emitHeading(block as any, defaultLayoutTokens);
    expect(out).toHaveLength(1);
    expect(out[0]).toBeInstanceOf(Paragraph);
    expect((out[0] as any).options.heading).toBe(HeadingLevel.HEADING_1);
  });

  it("emits HEADING_2 for level 2", () => {
    const block = {
      id: "h2",
      type: "heading",
      props: { level: 2 },
      content: [{ type: "text", text: "Sub", styles: {} }],
      children: [],
    };
    const out = emitHeading(block as any, defaultLayoutTokens);
    expect((out[0] as any).options.heading).toBe(HeadingLevel.HEADING_2);
  });

  it("defaults to HEADING_1 when level is missing or invalid", () => {
    const block = {
      id: "h3",
      type: "heading",
      props: {},
      content: [{ type: "text", text: "Default", styles: {} }],
      children: [],
    };
    const out = emitHeading(block as any, defaultLayoutTokens);
    expect((out[0] as any).options.heading).toBe(HeadingLevel.HEADING_1);
  });
});
```

- [ ] **Step 2: Run test — expect failure**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/docx-emitter/__tests__/heading.test.ts`
Expected: FAIL — cannot find module.

- [ ] **Step 3: Implement heading.ts**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/heading.ts`:

```ts
import { Paragraph, HeadingLevel } from "docx";
import type { LayoutTokens } from "../../layout-ir";
import { inlineContentToRuns, type InlineContent } from "../inline-content";

export type HeadingBlock = {
  id: string;
  type: "heading";
  props?: { level?: number };
  content?: InlineContent[];
  children?: unknown[];
};

function levelToHeading(level: number | undefined): typeof HeadingLevel[keyof typeof HeadingLevel] {
  switch (level) {
    case 1: return HeadingLevel.HEADING_1;
    case 2: return HeadingLevel.HEADING_2;
    case 3: return HeadingLevel.HEADING_3;
    default: return HeadingLevel.HEADING_1;
  }
}

export function emitHeading(
  block: HeadingBlock,
  tokens: LayoutTokens,
): Paragraph[] {
  const runs = inlineContentToRuns(block.content ?? [], tokens);
  return [
    new Paragraph({
      heading: levelToHeading(block.props?.level),
      children: runs,
    }),
  ];
}
```

- [ ] **Step 4: Run test — expect pass**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/docx-emitter/__tests__/heading.test.ts`
Expected: PASS — 3 tests passing.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/heading.ts frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/heading.test.ts
git commit -m "feat(mddm-engine): add heading DOCX emitter (h1-h3)"
```

### Task 18: Implement section emitter

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/section.ts`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/section.test.ts`

- [ ] **Step 1: Write the failing test**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/section.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { Table } from "docx";
import { emitSection } from "../emitters/section";
import { defaultLayoutTokens } from "../../layout-ir";

describe("emitSection", () => {
  it("emits a full-width Table wrapping the section header", () => {
    const block = {
      id: "s1",
      type: "section",
      props: { title: "1. Procedimento", color: "red" },
      content: undefined,
      children: [],
    };
    const out = emitSection(block as any, defaultLayoutTokens);
    expect(out).toHaveLength(1);
    expect(out[0]).toBeInstanceOf(Table);
  });

  it("uses the token accent color for header background", () => {
    const block = {
      id: "s2",
      type: "section",
      props: { title: "Header" },
      content: undefined,
      children: [],
    };
    const tokens = {
      ...defaultLayoutTokens,
      theme: { ...defaultLayoutTokens.theme, accent: "#123456" },
    };
    const out = emitSection(block as any, tokens);
    // The underlying shading should match accent (docx.js stores as uppercase hex no hash)
    const tableOptions = (out[0] as any).options;
    const firstRow = tableOptions.rows[0];
    const firstCell = firstRow.options.children[0];
    expect(firstCell.options.shading.fill).toBe("123456");
  });

  it("renders empty title when title prop is missing", () => {
    const block = {
      id: "s3",
      type: "section",
      props: {},
      content: undefined,
      children: [],
    };
    const out = emitSection(block as any, defaultLayoutTokens);
    expect(out).toHaveLength(1);
  });
});
```

- [ ] **Step 2: Run test — expect failure**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/docx-emitter/__tests__/section.test.ts`
Expected: FAIL — cannot find module.

- [ ] **Step 3: Implement section.ts**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/section.ts`:

```ts
import {
  Table,
  TableRow,
  TableCell,
  Paragraph,
  TextRun,
  WidthType,
  HeightRule,
  VerticalAlign,
} from "docx";
import type { LayoutTokens } from "../../layout-ir";
import { defaultComponentRules } from "../../layout-ir";
import { mmToTwip, ptToHalfPt } from "../../helpers/units";

export type SectionBlock = {
  id: string;
  type: "section";
  props?: { title?: string; color?: string };
  children?: unknown[];
};

function hexToFill(hex: string): string {
  // docx.js fill expects hex without leading # and uppercase
  return hex.replace(/^#/, "").toUpperCase();
}

export function emitSection(
  block: SectionBlock,
  tokens: LayoutTokens,
): Table[] {
  const rule = defaultComponentRules.section;
  const title = block.props?.title ?? "";
  const fill = hexToFill(tokens.theme.accent);

  const cell = new TableCell({
    shading: { fill, type: "clear", color: "auto" },
    verticalAlign: VerticalAlign.CENTER,
    children: [
      new Paragraph({
        children: [
          new TextRun({
            text: title,
            bold: rule.headerFontWeight === "bold",
            color: rule.headerFontColor.replace(/^#/, "").toUpperCase(),
            size: ptToHalfPt(rule.headerFontSizePt),
            font: tokens.typography.exportFont,
          }),
        ],
      }),
    ],
  });

  const row = new TableRow({
    height: {
      value: mmToTwip(rule.headerHeightMm),
      rule: HeightRule.EXACT,
    },
    children: [cell],
  });

  const table = new Table({
    width: { size: 100, type: WidthType.PERCENTAGE },
    rows: [row],
  });

  return [table];
}
```

- [ ] **Step 4: Run test — expect pass**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/docx-emitter/__tests__/section.test.ts`
Expected: PASS — 3 tests passing.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/section.ts frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/section.test.ts
git commit -m "feat(mddm-engine): add section DOCX emitter (full-width header with accent)"
```

### Task 19: Implement field emitter

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/field.ts`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/field.test.ts`

- [ ] **Step 1: Write the failing test**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/field.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { Table } from "docx";
import { emitField } from "../emitters/field";
import { defaultLayoutTokens } from "../../layout-ir";

describe("emitField", () => {
  it("emits a Table with two cells using 35/65 split", () => {
    const block = {
      id: "f1",
      type: "field",
      props: { label: "Responsável" },
      content: [{ type: "text", text: "João Silva", styles: {} }],
      children: [],
    };
    const out = emitField(block as any, defaultLayoutTokens);
    expect(out).toHaveLength(1);
    expect(out[0]).toBeInstanceOf(Table);

    const tableOptions = (out[0] as any).options;
    const firstRow = tableOptions.rows[0];
    const cells = firstRow.options.children;
    expect(cells).toHaveLength(2);

    // First cell width should be 35% (1750 in docx.js fiftieths)
    expect(cells[0].options.width.size).toBe(1750);
    expect(cells[1].options.width.size).toBe(3250);
  });

  it("applies the accentLight background to the label cell", () => {
    const block = {
      id: "f2",
      type: "field",
      props: { label: "Label" },
      content: [],
      children: [],
    };
    const tokens = {
      ...defaultLayoutTokens,
      theme: { ...defaultLayoutTokens.theme, accentLight: "#ffeeff" },
    };
    const out = emitField(block as any, tokens);
    const labelCell = (out[0] as any).options.rows[0].options.children[0];
    expect(labelCell.options.shading.fill).toBe("FFEEFF");
  });
});
```

- [ ] **Step 2: Run test — expect failure**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/docx-emitter/__tests__/field.test.ts`
Expected: FAIL — cannot find module.

- [ ] **Step 3: Implement field.ts**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/field.ts`:

```ts
import {
  Table,
  TableRow,
  TableCell,
  Paragraph,
  TextRun,
  WidthType,
  BorderStyle,
} from "docx";
import type { LayoutTokens } from "../../layout-ir";
import { defaultComponentRules } from "../../layout-ir";
import { percentToTablePct, ptToHalfPt } from "../../helpers/units";
import { inlineContentToRuns, type InlineContent } from "../inline-content";

export type FieldBlock = {
  id: string;
  type: "field";
  props?: { label?: string; valueMode?: "inline" | "multiParagraph" };
  content?: InlineContent[];
  children?: unknown[];
};

function hexToFill(hex: string): string {
  return hex.replace(/^#/, "").toUpperCase();
}

export function emitField(
  block: FieldBlock,
  tokens: LayoutTokens,
): Table[] {
  const rule = defaultComponentRules.field;
  const label = block.props?.label ?? "";
  const labelFill = hexToFill(tokens.theme.accentLight);
  const borderColor = hexToFill(tokens.theme.accentBorder);

  const borders = {
    top:    { style: BorderStyle.SINGLE, size: 4, color: borderColor },
    bottom: { style: BorderStyle.SINGLE, size: 4, color: borderColor },
    left:   { style: BorderStyle.SINGLE, size: 4, color: borderColor },
    right:  { style: BorderStyle.SINGLE, size: 4, color: borderColor },
  };

  const labelCell = new TableCell({
    width: { size: percentToTablePct(rule.labelWidthPercent), type: WidthType.PERCENTAGE },
    shading: { fill: labelFill, type: "clear", color: "auto" },
    borders,
    children: [
      new Paragraph({
        children: [
          new TextRun({
            text: label,
            size: ptToHalfPt(rule.labelFontSizePt),
            font: tokens.typography.exportFont,
          }),
        ],
      }),
    ],
  });

  const valueRuns = inlineContentToRuns(block.content ?? [], tokens);
  const valueCell = new TableCell({
    width: { size: percentToTablePct(rule.valueWidthPercent), type: WidthType.PERCENTAGE },
    borders,
    children: [
      new Paragraph({ children: valueRuns.length > 0 ? valueRuns : [new TextRun({ text: "" })] }),
    ],
  });

  const row = new TableRow({ children: [labelCell, valueCell] });

  return [
    new Table({
      width: { size: 100, type: WidthType.PERCENTAGE },
      rows: [row],
    }),
  ];
}
```

- [ ] **Step 4: Run test — expect pass**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/docx-emitter/__tests__/field.test.ts`
Expected: PASS — 2 tests passing.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/field.ts frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/field.test.ts
git commit -m "feat(mddm-engine): add field DOCX emitter (35/65 split with label background)"
```

### Task 20: Implement field-group emitter

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/field-group.ts`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/field-group.test.ts`

- [ ] **Step 1: Write the failing test**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/field-group.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { Table } from "docx";
import { emitFieldGroup } from "../emitters/field-group";
import { defaultLayoutTokens } from "../../layout-ir";

describe("emitFieldGroup", () => {
  it("emits a single outer Table wrapping child fields", () => {
    const block = {
      id: "fg1",
      type: "fieldGroup",
      props: { columns: 2 },
      children: [
        { id: "f1", type: "field", props: { label: "A" }, content: [], children: [] },
        { id: "f2", type: "field", props: { label: "B" }, content: [], children: [] },
      ],
    };
    const out = emitFieldGroup(block as any, defaultLayoutTokens);
    expect(out).toHaveLength(1);
    expect(out[0]).toBeInstanceOf(Table);
  });

  it("arranges two fields side-by-side for columns=2", () => {
    const block = {
      id: "fg2",
      type: "fieldGroup",
      props: { columns: 2 },
      children: [
        { id: "f1", type: "field", props: { label: "A" }, content: [], children: [] },
        { id: "f2", type: "field", props: { label: "B" }, content: [], children: [] },
      ],
    };
    const out = emitFieldGroup(block as any, defaultLayoutTokens);
    const rows = (out[0] as any).options.rows;
    // One row holding two cells (each containing a nested field table)
    expect(rows).toHaveLength(1);
    expect(rows[0].options.children).toHaveLength(2);
  });

  it("stacks fields vertically for columns=1", () => {
    const block = {
      id: "fg3",
      type: "fieldGroup",
      props: { columns: 1 },
      children: [
        { id: "f1", type: "field", props: { label: "A" }, content: [], children: [] },
        { id: "f2", type: "field", props: { label: "B" }, content: [], children: [] },
      ],
    };
    const out = emitFieldGroup(block as any, defaultLayoutTokens);
    const rows = (out[0] as any).options.rows;
    expect(rows).toHaveLength(2);
  });
});
```

- [ ] **Step 2: Run test — expect failure**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/docx-emitter/__tests__/field-group.test.ts`
Expected: FAIL — cannot find module.

- [ ] **Step 3: Implement field-group.ts**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/field-group.ts`:

```ts
import {
  Table,
  TableRow,
  TableCell,
  WidthType,
  BorderStyle,
} from "docx";
import type { LayoutTokens } from "../../layout-ir";
import { emitField, type FieldBlock } from "./field";

export type FieldGroupBlock = {
  id: string;
  type: "fieldGroup";
  props?: { columns?: 1 | 2 };
  children?: FieldBlock[];
};

const NO_BORDER = {
  top:    { style: BorderStyle.NONE, size: 0, color: "auto" },
  bottom: { style: BorderStyle.NONE, size: 0, color: "auto" },
  left:   { style: BorderStyle.NONE, size: 0, color: "auto" },
  right:  { style: BorderStyle.NONE, size: 0, color: "auto" },
} as const;

export function emitFieldGroup(
  block: FieldGroupBlock,
  tokens: LayoutTokens,
): Table[] {
  const columns = block.props?.columns === 1 ? 1 : 2;
  const fields = (block.children ?? []).filter((c): c is FieldBlock => c?.type === "field");

  if (fields.length === 0) {
    return [
      new Table({
        width: { size: 100, type: WidthType.PERCENTAGE },
        rows: [new TableRow({ children: [new TableCell({ borders: NO_BORDER, children: [] })] })],
      }),
    ];
  }

  const cellWidthPct = Math.floor(5000 / columns);
  const rows: TableRow[] = [];

  for (let i = 0; i < fields.length; i += columns) {
    const rowCells: TableCell[] = [];
    for (let c = 0; c < columns; c++) {
      const field = fields[i + c];
      const fieldTable = field ? emitField(field, tokens) : [];
      rowCells.push(
        new TableCell({
          width: { size: cellWidthPct, type: WidthType.PERCENTAGE },
          borders: NO_BORDER,
          children: fieldTable,
        }),
      );
    }
    rows.push(new TableRow({ children: rowCells }));
  }

  return [
    new Table({
      width: { size: 100, type: WidthType.PERCENTAGE },
      rows,
    }),
  ];
}
```

- [ ] **Step 4: Run test — expect pass**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/docx-emitter/__tests__/field-group.test.ts`
Expected: PASS — 3 tests passing.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/field-group.ts frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/field-group.test.ts
git commit -m "feat(mddm-engine): add field-group DOCX emitter with 1/2 column layout"
```

---

## Part 8 — mddmToDocx Entry Point

### Task 21: Implement main mddmToDocx function

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitter.ts`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/emitter.test.ts`

- [ ] **Step 1: Write the failing test**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/emitter.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { mddmToDocx, MissingEmitterError } from "../emitter";
import { defaultLayoutTokens } from "../../layout-ir";
import type { MDDMEnvelope } from "../../../adapter";

describe("mddmToDocx", () => {
  it("returns a Blob for a paragraph-only envelope", async () => {
    const envelope: MDDMEnvelope = {
      mddm_version: 1,
      template_ref: null,
      blocks: [
        { id: "p1", type: "paragraph", props: {}, content: [{ type: "text", text: "Hello" }], children: [] },
      ],
    };
    const blob = await mddmToDocx(envelope, defaultLayoutTokens);
    expect(blob).toBeInstanceOf(Blob);
    expect(blob.size).toBeGreaterThan(0);
    expect(blob.type).toBe("application/vnd.openxmlformats-officedocument.wordprocessingml.document");
  });

  it("returns a Blob for a section+field envelope", async () => {
    const envelope: MDDMEnvelope = {
      mddm_version: 1,
      template_ref: null,
      blocks: [
        { id: "s1", type: "section", props: { title: "1. Procedimento" }, children: [] },
        {
          id: "f1",
          type: "field",
          props: { label: "Responsável" },
          content: [{ type: "text", text: "João" }],
          children: [],
        },
      ],
    };
    const blob = await mddmToDocx(envelope, defaultLayoutTokens);
    expect(blob.size).toBeGreaterThan(0);
  });

  it("throws MissingEmitterError for unknown block types", async () => {
    const envelope: MDDMEnvelope = {
      mddm_version: 1,
      template_ref: null,
      blocks: [{ id: "x", type: "unknownXYZ" as any, props: {}, children: [] }],
    };
    await expect(mddmToDocx(envelope, defaultLayoutTokens))
      .rejects.toBeInstanceOf(MissingEmitterError);
  });
});
```

- [ ] **Step 2: Run test — expect failure**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/docx-emitter/__tests__/emitter.test.ts`
Expected: FAIL — cannot find module `../emitter`.

- [ ] **Step 3: Implement emitter.ts**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitter.ts`:

```ts
import { Document, Packer } from "docx";
import type { MDDMEnvelope } from "../../adapter";
import type { LayoutTokens } from "../layout-ir";
import { mmToTwip } from "../helpers/units";
import { emitParagraph, type ParagraphBlock } from "./emitters/paragraph";
import { emitHeading, type HeadingBlock } from "./emitters/heading";
import { emitSection, type SectionBlock } from "./emitters/section";
import { emitField, type FieldBlock } from "./emitters/field";
import { emitFieldGroup, type FieldGroupBlock } from "./emitters/field-group";

const DOCX_MIME = "application/vnd.openxmlformats-officedocument.wordprocessingml.document";

export class MissingEmitterError extends Error {
  constructor(public readonly blockType: string) {
    super(`No DOCX emitter registered for block type "${blockType}"`);
    this.name = "MissingEmitterError";
  }
}

type AnyBlock = { id: string; type: string; props?: Record<string, unknown>; children?: unknown[]; content?: unknown };

type Emitter = (block: AnyBlock, tokens: LayoutTokens) => unknown[];

const emitters: Record<string, Emitter> = {
  paragraph: (b, t) => emitParagraph(b as ParagraphBlock, t),
  heading: (b, t) => emitHeading(b as HeadingBlock, t),
  section: (b, t) => emitSection(b as SectionBlock, t),
  field: (b, t) => emitField(b as FieldBlock, t),
  fieldGroup: (b, t) => emitFieldGroup(b as FieldGroupBlock, t),
};

export async function mddmToDocx(
  envelope: MDDMEnvelope,
  tokens: LayoutTokens,
): Promise<Blob> {
  const blocks = envelope.blocks ?? [];
  const children: unknown[] = [];

  for (const block of blocks) {
    const typed = block as AnyBlock;
    const emit = emitters[typed.type];
    if (!emit) {
      throw new MissingEmitterError(typed.type);
    }
    const out = emit(typed, tokens);
    children.push(...out);
  }

  const doc = new Document({
    sections: [
      {
        properties: {
          page: {
            size: {
              width: mmToTwip(tokens.page.widthMm),
              height: mmToTwip(tokens.page.heightMm),
            },
            margin: {
              top: mmToTwip(tokens.page.marginTop),
              right: mmToTwip(tokens.page.marginRight),
              bottom: mmToTwip(tokens.page.marginBottom),
              left: mmToTwip(tokens.page.marginLeft),
            },
          },
        },
        children: children as any,
      },
    ],
  });

  const blob = await Packer.toBlob(doc);
  // Packer returns a Blob in browsers; force the MIME type for download handling.
  return new Blob([await blob.arrayBuffer()], { type: DOCX_MIME });
}
```

- [ ] **Step 4: Run test — expect pass**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/docx-emitter/__tests__/emitter.test.ts`
Expected: PASS — 3 tests passing.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitter.ts frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/emitter.test.ts
git commit -m "feat(mddm-engine): add mddmToDocx entry point wiring 5 MVP emitters"
```

### Task 22: Create docx-emitter barrel export

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/index.ts`

- [ ] **Step 1: Write barrel**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/index.ts`:

```ts
export { mddmToDocx, MissingEmitterError } from "./emitter";
export { inlineContentToRuns, type InlineContent, type InlineStyles } from "./inline-content";
export { emitParagraph, type ParagraphBlock } from "./emitters/paragraph";
export { emitHeading, type HeadingBlock } from "./emitters/heading";
export { emitSection, type SectionBlock } from "./emitters/section";
export { emitField, type FieldBlock } from "./emitters/field";
export { emitFieldGroup, type FieldGroupBlock } from "./emitters/field-group";
```

- [ ] **Step 2: Verify compilation**

Run: `cd frontend/apps/web && npx tsc --noEmit 2>&1 | grep -E "docx-emitter" | head -5`
Expected: No errors.

- [ ] **Step 3: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/index.ts
git commit -m "feat(mddm-engine): add docx-emitter barrel export"
```

---

## Part 9 — toExternalHTML Hooks

### Task 23: Section toExternalHTML component

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/external-html/section-html.tsx`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/external-html/__tests__/section-html.test.tsx`

- [ ] **Step 1: Write the failing test**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/external-html/__tests__/section-html.test.tsx`:

```tsx
import { describe, expect, it } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import { SectionExternalHTML } from "../section-html";
import { defaultLayoutTokens } from "../../layout-ir";

describe("SectionExternalHTML", () => {
  it("renders a table-based header with the section title", () => {
    const html = renderToStaticMarkup(
      <SectionExternalHTML title="1. Procedimento" tokens={defaultLayoutTokens} />,
    );
    expect(html).toContain("<table");
    expect(html).toContain("1. Procedimento");
    expect(html).toContain("mddm-section-header");
  });

  it("does NOT use display:flex (flexbox is forbidden)", () => {
    const html = renderToStaticMarkup(
      <SectionExternalHTML title="x" tokens={defaultLayoutTokens} />,
    );
    expect(html).not.toContain("display:flex");
    expect(html).not.toContain("display: flex");
  });

  it("uses the theme accent color for the header background", () => {
    const tokens = {
      ...defaultLayoutTokens,
      theme: { ...defaultLayoutTokens.theme, accent: "#abcdef" },
    };
    const html = renderToStaticMarkup(<SectionExternalHTML title="x" tokens={tokens} />);
    expect(html.toLowerCase()).toContain("#abcdef");
  });

  it("uses absolute pt font size (no em or percent)", () => {
    const html = renderToStaticMarkup(
      <SectionExternalHTML title="x" tokens={defaultLayoutTokens} />,
    );
    expect(html).toMatch(/font-size:\s*\d+pt/);
    expect(html).not.toMatch(/font-size:\s*\d+em/);
  });
});
```

- [ ] **Step 2: Run test — expect failure**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/external-html/__tests__/section-html.test.tsx`
Expected: FAIL — cannot find module.

- [ ] **Step 3: Implement section-html.tsx**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/external-html/section-html.tsx`:

```tsx
import type { LayoutTokens } from "../layout-ir";
import { defaultComponentRules } from "../layout-ir";

export type SectionExternalHTMLProps = {
  title: string;
  tokens: LayoutTokens;
};

export function SectionExternalHTML({ title, tokens }: SectionExternalHTMLProps) {
  const rule = defaultComponentRules.section;

  return (
    <table
      className="mddm-section-header"
      data-mddm-block="section"
      style={{
        width: "100%",
        borderCollapse: "collapse",
        margin: `${tokens.spacing.blockGapMm}mm 0`,
      }}
    >
      <tbody>
        <tr>
          <td
            style={{
              background: tokens.theme.accent,
              height: `${rule.headerHeightMm}mm`,
              color: rule.headerFontColor,
              fontWeight: rule.headerFontWeight,
              fontSize: `${rule.headerFontSizePt}pt`,
              padding: "0 4mm",
              verticalAlign: "middle",
            }}
          >
            {title}
          </td>
        </tr>
      </tbody>
    </table>
  );
}
```

- [ ] **Step 4: Run test — expect pass**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/external-html/__tests__/section-html.test.tsx`
Expected: PASS — 4 tests passing.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/external-html/section-html.tsx frontend/apps/web/src/features/documents/mddm-editor/engine/external-html/__tests__/section-html.test.tsx
git commit -m "feat(mddm-engine): add Section toExternalHTML component (table-based)"
```

### Task 24: Field toExternalHTML component

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/external-html/field-html.tsx`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/external-html/__tests__/field-html.test.tsx`

- [ ] **Step 1: Write the failing test**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/external-html/__tests__/field-html.test.tsx`:

```tsx
import { describe, expect, it } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import { FieldExternalHTML } from "../field-html";
import { defaultLayoutTokens } from "../../layout-ir";

describe("FieldExternalHTML", () => {
  it("renders a two-column table with label and value cells", () => {
    const html = renderToStaticMarkup(
      <FieldExternalHTML label="Responsável" tokens={defaultLayoutTokens}>
        <span>João Silva</span>
      </FieldExternalHTML>,
    );
    expect(html).toContain("<table");
    expect(html).toContain("Responsável");
    expect(html).toContain("João Silva");
    expect(html).toContain("mddm-field");
  });

  it("renders label cell with 35% width and value cell with 65% width", () => {
    const html = renderToStaticMarkup(
      <FieldExternalHTML label="L" tokens={defaultLayoutTokens}>
        V
      </FieldExternalHTML>,
    );
    expect(html).toContain("35%");
    expect(html).toContain("65%");
  });

  it("does not use flexbox", () => {
    const html = renderToStaticMarkup(
      <FieldExternalHTML label="L" tokens={defaultLayoutTokens}>
        V
      </FieldExternalHTML>,
    );
    expect(html).not.toContain("display:flex");
    expect(html).not.toContain("display: flex");
  });
});
```

- [ ] **Step 2: Run test — expect failure**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/external-html/__tests__/field-html.test.tsx`
Expected: FAIL.

- [ ] **Step 3: Implement field-html.tsx**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/external-html/field-html.tsx`:

```tsx
import type { ReactNode } from "react";
import type { LayoutTokens } from "../layout-ir";
import { defaultComponentRules } from "../layout-ir";

export type FieldExternalHTMLProps = {
  label: string;
  tokens: LayoutTokens;
  children?: ReactNode;
};

export function FieldExternalHTML({ label, tokens, children }: FieldExternalHTMLProps) {
  const rule = defaultComponentRules.field;
  const borderStyle = `${rule.borderWidthPt}pt solid ${tokens.theme.accentBorder}`;

  return (
    <table
      className="mddm-field"
      data-mddm-block="field"
      style={{
        width: "100%",
        borderCollapse: "collapse",
        tableLayout: "fixed",
      }}
    >
      <tbody>
        <tr>
          <td
            style={{
              width: `${rule.labelWidthPercent}%`,
              background: tokens.theme.accentLight,
              fontSize: `${rule.labelFontSizePt}pt`,
              padding: `${tokens.spacing.cellPaddingMm}mm`,
              border: borderStyle,
              verticalAlign: "top",
              minHeight: `${rule.minHeightMm}mm`,
            }}
          >
            {label}
          </td>
          <td
            style={{
              width: `${rule.valueWidthPercent}%`,
              padding: `${tokens.spacing.cellPaddingMm}mm`,
              border: borderStyle,
              verticalAlign: "top",
              minHeight: `${rule.minHeightMm}mm`,
            }}
          >
            {children}
          </td>
        </tr>
      </tbody>
    </table>
  );
}
```

- [ ] **Step 4: Run test — expect pass**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/external-html/__tests__/field-html.test.tsx`
Expected: PASS — 3 tests passing.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/external-html/field-html.tsx frontend/apps/web/src/features/documents/mddm-editor/engine/external-html/__tests__/field-html.test.tsx
git commit -m "feat(mddm-engine): add Field toExternalHTML component (35/65 table layout)"
```

### Task 25: FieldGroup toExternalHTML component

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/external-html/field-group-html.tsx`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/external-html/__tests__/field-group-html.test.tsx`

- [ ] **Step 1: Write the failing test**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/external-html/__tests__/field-group-html.test.tsx`:

```tsx
import { describe, expect, it } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import { FieldGroupExternalHTML } from "../field-group-html";
import { defaultLayoutTokens } from "../../layout-ir";

describe("FieldGroupExternalHTML", () => {
  it("renders a wrapping table with data-columns attribute", () => {
    const html = renderToStaticMarkup(
      <FieldGroupExternalHTML columns={2} tokens={defaultLayoutTokens}>
        <span>child</span>
      </FieldGroupExternalHTML>,
    );
    expect(html).toContain("<table");
    expect(html).toContain('data-columns="2"');
    expect(html).toContain("mddm-field-group");
  });

  it("supports columns=1", () => {
    const html = renderToStaticMarkup(
      <FieldGroupExternalHTML columns={1} tokens={defaultLayoutTokens}>
        <span>child</span>
      </FieldGroupExternalHTML>,
    );
    expect(html).toContain('data-columns="1"');
  });

  it("does not use flexbox or CSS grid fr units", () => {
    const html = renderToStaticMarkup(
      <FieldGroupExternalHTML columns={2} tokens={defaultLayoutTokens}>
        <span>x</span>
      </FieldGroupExternalHTML>,
    );
    expect(html).not.toContain("display:flex");
    expect(html).not.toContain("grid-template-columns:1fr");
  });
});
```

- [ ] **Step 2: Run test — expect failure**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/external-html/__tests__/field-group-html.test.tsx`
Expected: FAIL.

- [ ] **Step 3: Implement field-group-html.tsx**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/external-html/field-group-html.tsx`:

```tsx
import type { ReactNode } from "react";
import type { LayoutTokens } from "../layout-ir";

export type FieldGroupExternalHTMLProps = {
  columns: 1 | 2;
  tokens: LayoutTokens;
  children?: ReactNode;
};

export function FieldGroupExternalHTML({ columns, tokens, children }: FieldGroupExternalHTMLProps) {
  return (
    <table
      className="mddm-field-group"
      data-mddm-block="fieldGroup"
      data-columns={String(columns)}
      style={{
        width: "100%",
        borderCollapse: "collapse",
        margin: `${tokens.spacing.blockGapMm}mm 0`,
      }}
    >
      <tbody>
        <tr>
          <td style={{ padding: 0, verticalAlign: "top" }}>{children}</td>
        </tr>
      </tbody>
    </table>
  );
}
```

- [ ] **Step 4: Run test — expect pass**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/external-html/__tests__/field-group-html.test.tsx`
Expected: PASS — 3 tests passing.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/external-html/field-group-html.tsx frontend/apps/web/src/features/documents/mddm-editor/engine/external-html/__tests__/field-group-html.test.tsx
git commit -m "feat(mddm-engine): add FieldGroup toExternalHTML wrapper component"
```

### Task 26: Create external-html barrel export

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/external-html/index.ts`

- [ ] **Step 1: Write barrel**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/external-html/index.ts`:

```ts
export { SectionExternalHTML, type SectionExternalHTMLProps } from "./section-html";
export { FieldExternalHTML, type FieldExternalHTMLProps } from "./field-html";
export { FieldGroupExternalHTML, type FieldGroupExternalHTMLProps } from "./field-group-html";
```

- [ ] **Step 2: Verify compilation**

Run: `cd frontend/apps/web && npx tsc --noEmit 2>&1 | grep external-html | head -5`
Expected: No errors.

- [ ] **Step 3: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/external-html/index.ts
git commit -m "feat(mddm-engine): add external-html barrel export"
```

### Task 27: Register toExternalHTML on the Section block spec

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/blocks/Section.tsx`

- [ ] **Step 1: Inspect current Section.tsx**

Run: `cat frontend/apps/web/src/features/documents/mddm-editor/blocks/Section.tsx | head -40`
Expected: Shows current `createReactBlockSpec` call with a `render` function.

- [ ] **Step 2: Add toExternalHTML to the Section block spec**

In `frontend/apps/web/src/features/documents/mddm-editor/blocks/Section.tsx`, add these imports at the top (after existing imports):

```tsx
import { SectionExternalHTML } from "../engine/external-html";
import { defaultLayoutTokens } from "../engine/layout-ir";
```

Find the `createReactBlockSpec(...)` call. It takes two arguments: `blockConfig` (first object) and `blockImplementation` (second object containing `render`). Add a `toExternalHTML` property to the `blockImplementation` object (second argument) alongside `render`:

```tsx
toExternalHTML: ({ block }) => (
  <SectionExternalHTML
    title={(block.props as { title?: string }).title ?? ""}
    tokens={defaultLayoutTokens}
  />
),
```

- [ ] **Step 3: Verify the file compiles**

Run: `cd frontend/apps/web && npx tsc --noEmit 2>&1 | grep Section.tsx | head -5`
Expected: No errors.

- [ ] **Step 4: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/blocks/Section.tsx
git commit -m "feat(mddm-engine): register toExternalHTML on Section block"
```

### Task 28: Register toExternalHTML on the Field block spec

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/blocks/Field.tsx`

- [ ] **Step 1: Inspect current Field.tsx**

Run: `cat frontend/apps/web/src/features/documents/mddm-editor/blocks/Field.tsx | head -50`
Expected: Shows `createReactBlockSpec` call.

- [ ] **Step 2: Add toExternalHTML to Field block spec**

In `frontend/apps/web/src/features/documents/mddm-editor/blocks/Field.tsx`, add imports:

```tsx
import { FieldExternalHTML } from "../engine/external-html";
import { defaultLayoutTokens } from "../engine/layout-ir";
```

Add to the `blockImplementation` (second argument of `createReactBlockSpec`) next to `render`:

```tsx
toExternalHTML: ({ block, contentRef }) => (
  <FieldExternalHTML
    label={(block.props as { label?: string }).label ?? ""}
    tokens={defaultLayoutTokens}
  >
    <span ref={contentRef as unknown as React.Ref<HTMLSpanElement>} />
  </FieldExternalHTML>
),
```

(The `contentRef` pattern mirrors BlockNote's documented inline-content block API; BlockNote replaces the referenced element with the serialized inline content.)

- [ ] **Step 3: Verify the file compiles**

Run: `cd frontend/apps/web && npx tsc --noEmit 2>&1 | grep Field.tsx | head -5`
Expected: No errors.

- [ ] **Step 4: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/blocks/Field.tsx
git commit -m "feat(mddm-engine): register toExternalHTML on Field block"
```

### Task 29: Register toExternalHTML on the FieldGroup block spec

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/blocks/FieldGroup.tsx`

- [ ] **Step 1: Inspect current FieldGroup.tsx**

Run: `cat frontend/apps/web/src/features/documents/mddm-editor/blocks/FieldGroup.tsx | head -50`
Expected: Shows current block spec.

- [ ] **Step 2: Add toExternalHTML to FieldGroup block spec**

In `frontend/apps/web/src/features/documents/mddm-editor/blocks/FieldGroup.tsx`, add imports:

```tsx
import { FieldGroupExternalHTML } from "../engine/external-html";
import { defaultLayoutTokens } from "../engine/layout-ir";
```

Add to the `blockImplementation`:

```tsx
toExternalHTML: ({ block, contentRef }) => {
  const columns = (block.props as { columns?: 1 | 2 }).columns === 1 ? 1 : 2;
  return (
    <FieldGroupExternalHTML columns={columns} tokens={defaultLayoutTokens}>
      <div ref={contentRef as unknown as React.Ref<HTMLDivElement>} />
    </FieldGroupExternalHTML>
  );
},
```

- [ ] **Step 3: Verify the file compiles**

Run: `cd frontend/apps/web && npx tsc --noEmit 2>&1 | grep FieldGroup.tsx | head -5`
Expected: No errors.

- [ ] **Step 4: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/blocks/FieldGroup.tsx
git commit -m "feat(mddm-engine): register toExternalHTML on FieldGroup block"
```

---

## Part 10 — Print Stylesheet

### Task 30: Define the print stylesheet as a TypeScript string

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/print-stylesheet/print-css.ts`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/print-stylesheet/index.ts`

- [ ] **Step 1: Implement print-css.ts**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/print-stylesheet/print-css.ts`:

```ts
import { defaultLayoutTokens } from "../layout-ir";

// Print stylesheet sent alongside the HTML to Gotenberg's Chromium route.
// Uses the Carlito / Liberation Sans / Arial fallback stack defined in the
// Font Strategy section of the spec. All dimensions are absolute (mm/pt).
export const PRINT_STYLESHEET = `
@page {
  size: A4;
  margin: ${defaultLayoutTokens.page.marginTop}mm ${defaultLayoutTokens.page.marginRight}mm ${defaultLayoutTokens.page.marginBottom}mm ${defaultLayoutTokens.page.marginLeft}mm;
}

html, body {
  margin: 0;
  padding: 0;
  font-family: "Carlito", "Liberation Sans", "Arial", sans-serif;
  font-size: ${defaultLayoutTokens.typography.baseSizePt}pt;
  line-height: ${defaultLayoutTokens.typography.lineHeightPt}pt;
  color: #111111;
  -webkit-print-color-adjust: exact;
  print-color-adjust: exact;
  font-kerning: normal;
  font-feature-settings: "liga" 1, "kern" 1;
  font-synthesis: none;
}

.mddm-section-header,
.mddm-field,
.mddm-field-group {
  page-break-inside: avoid;
}

/* Hide editor-only chrome in case any leaks through. */
.bn-side-menu,
.bn-formatting-toolbar,
.bn-slash-menu,
.bn-drag-handle {
  display: none !important;
}

/* MDDM block base styling used alongside inline styles from toExternalHTML. */
[data-mddm-block] {
  box-sizing: border-box;
}
`;
```

- [ ] **Step 2: Write the index**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/print-stylesheet/index.ts`:

```ts
export { PRINT_STYLESHEET } from "./print-css";
```

- [ ] **Step 3: Verify compilation**

Run: `cd frontend/apps/web && npx tsc --noEmit 2>&1 | grep print-stylesheet | head -5`
Expected: No errors.

- [ ] **Step 4: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/print-stylesheet/
git commit -m "feat(mddm-engine): add print stylesheet with Carlito font stack"
```

---

## Part 11 — Frontend Export Functions

### Task 31: Implement wrapInPrintDocument helper

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/export/wrap-print-document.ts`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/export/__tests__/wrap-print-document.test.ts`

- [ ] **Step 1: Write the failing test**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/export/__tests__/wrap-print-document.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { wrapInPrintDocument } from "../wrap-print-document";

describe("wrapInPrintDocument", () => {
  it("wraps body HTML in a full HTML document with DOCTYPE", () => {
    const result = wrapInPrintDocument("<p>hi</p>");
    expect(result).toContain("<!DOCTYPE html>");
    expect(result).toContain("<html");
    expect(result).toContain("<body");
    expect(result).toContain("<p>hi</p>");
  });

  it("injects the print stylesheet in <style>", () => {
    const result = wrapInPrintDocument("<p>x</p>");
    expect(result).toContain("<style");
    expect(result).toContain("@page");
    expect(result).toContain("Carlito");
  });

  it("sets UTF-8 meta charset", () => {
    const result = wrapInPrintDocument("<p>x</p>");
    expect(result).toContain("charset=\"UTF-8\"");
  });
});
```

- [ ] **Step 2: Run test — expect failure**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/export/__tests__/wrap-print-document.test.ts`
Expected: FAIL — cannot find module.

- [ ] **Step 3: Implement wrap-print-document.ts**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/export/wrap-print-document.ts`:

```ts
import { PRINT_STYLESHEET } from "../print-stylesheet";

export function wrapInPrintDocument(bodyHtml: string): string {
  return `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8" />
<title>MDDM Document</title>
<style>${PRINT_STYLESHEET}</style>
</head>
<body>
${bodyHtml}
</body>
</html>`;
}
```

- [ ] **Step 4: Run test — expect pass**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/export/__tests__/wrap-print-document.test.ts`
Expected: PASS — 3 tests passing.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/export/wrap-print-document.ts frontend/apps/web/src/features/documents/mddm-editor/engine/export/__tests__/wrap-print-document.test.ts
git commit -m "feat(mddm-engine): add wrapInPrintDocument HTML wrapper"
```

### Task 32: Implement exportDocx

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/export/export-docx.ts`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/export/__tests__/export-docx.test.ts`

- [ ] **Step 1: Write the failing test**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/export/__tests__/export-docx.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { exportDocx } from "../export-docx";
import { defaultLayoutTokens } from "../../layout-ir";
import type { MDDMEnvelope } from "../../../adapter";

describe("exportDocx", () => {
  it("generates a DOCX Blob for a simple envelope", async () => {
    const envelope: MDDMEnvelope = {
      mddm_version: 1,
      template_ref: null,
      blocks: [
        {
          id: "p1",
          type: "paragraph",
          props: {},
          content: [{ type: "text", text: "Hello world", styles: {} }],
          children: [],
        },
      ],
    };

    const blob = await exportDocx(envelope, defaultLayoutTokens);
    expect(blob).toBeInstanceOf(Blob);
    expect(blob.size).toBeGreaterThan(100);
    expect(blob.type).toBe("application/vnd.openxmlformats-officedocument.wordprocessingml.document");
  });

  it("runs canonicalize+migrate before emitting", async () => {
    // Envelope with keys intentionally out of sorted order — canonicalize should normalize.
    const envelope = {
      template_ref: null,
      mddm_version: 1,
      blocks: [
        {
          type: "paragraph",
          id: "p1",
          props: {},
          content: [{ type: "text", text: "x", styles: {} }],
          children: [],
        },
      ],
    } as unknown as MDDMEnvelope;

    const blob = await exportDocx(envelope, defaultLayoutTokens);
    expect(blob.size).toBeGreaterThan(0);
  });
});
```

- [ ] **Step 2: Run test — expect failure**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/export/__tests__/export-docx.test.ts`
Expected: FAIL.

- [ ] **Step 3: Implement export-docx.ts**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/export/export-docx.ts`:

```ts
import type { MDDMEnvelope } from "../../adapter";
import type { LayoutTokens } from "../layout-ir";
import { canonicalizeAndMigrate } from "../canonicalize-migrate";
import { mddmToDocx } from "../docx-emitter";

export async function exportDocx(
  envelope: MDDMEnvelope,
  tokens: LayoutTokens,
): Promise<Blob> {
  const canonical = await canonicalizeAndMigrate(envelope);
  return mddmToDocx(canonical, tokens);
}
```

- [ ] **Step 4: Run test — expect pass**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/export/__tests__/export-docx.test.ts`
Expected: PASS — 2 tests passing.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/export/export-docx.ts frontend/apps/web/src/features/documents/mddm-editor/engine/export/__tests__/export-docx.test.ts
git commit -m "feat(mddm-engine): add exportDocx wiring canonicalize+migrate into emitter"
```

### Task 33: Implement exportPdf

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/export/export-pdf.ts`

- [ ] **Step 1: Implement export-pdf.ts**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/export/export-pdf.ts`:

```ts
import { wrapInPrintDocument } from "./wrap-print-document";
import { PRINT_STYLESHEET } from "../print-stylesheet";
import { RESOURCE_CEILINGS, ResourceCeilingExceededError } from "../asset-resolver";

export type ExportPdfParams = {
  /** Body HTML produced by blocksToFullHTML with assets already inlined as data URIs. */
  bodyHtml: string;
  /** Document ID — used in the backend endpoint path. */
  documentId: string;
};

const PDF_MIME = "application/pdf";

export async function exportPdf({ bodyHtml, documentId }: ExportPdfParams): Promise<Blob> {
  const fullHtml = wrapInPrintDocument(bodyHtml);

  const htmlBytes = new TextEncoder().encode(fullHtml).byteLength;
  if (htmlBytes > RESOURCE_CEILINGS.maxHtmlPayloadBytes) {
    throw new ResourceCeilingExceededError(
      "maxHtmlPayloadBytes",
      htmlBytes,
      RESOURCE_CEILINGS.maxHtmlPayloadBytes,
    );
  }

  const formData = new FormData();
  formData.append("index.html", new Blob([fullHtml], { type: "text/html" }), "index.html");
  formData.append("style.css", new Blob([PRINT_STYLESHEET], { type: "text/css" }), "style.css");

  const response = await fetch(
    `/api/v1/documents/${encodeURIComponent(documentId)}/render/pdf`,
    {
      method: "POST",
      credentials: "same-origin",
      body: formData,
    },
  );

  if (!response.ok) {
    const text = await response.text().catch(() => "");
    throw new Error(`PDF render failed: ${response.status} ${text}`);
  }

  const contentType = (response.headers.get("Content-Type") ?? "").toLowerCase();
  if (!contentType.includes(PDF_MIME)) {
    throw new Error(`Unexpected Content-Type from PDF endpoint: ${contentType}`);
  }

  const arrayBuffer = await response.arrayBuffer();
  return new Blob([arrayBuffer], { type: PDF_MIME });
}
```

- [ ] **Step 2: Verify the file compiles**

Run: `cd frontend/apps/web && npx tsc --noEmit 2>&1 | grep export-pdf | head -5`
Expected: No errors.

- [ ] **Step 3: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/export/export-pdf.ts
git commit -m "feat(mddm-engine): add exportPdf client calling backend render endpoint"
```

### Task 34: Create export barrel

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/export/index.ts`

- [ ] **Step 1: Write barrel**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/export/index.ts`:

```ts
export { exportDocx } from "./export-docx";
export { exportPdf, type ExportPdfParams } from "./export-pdf";
export { wrapInPrintDocument } from "./wrap-print-document";
```

- [ ] **Step 2: Verify compilation**

Run: `cd frontend/apps/web && npx tsc --noEmit 2>&1 | grep "engine/export" | head -5`
Expected: No errors.

- [ ] **Step 3: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/export/index.ts
git commit -m "feat(mddm-engine): add export barrel"
```

---

## Part 12 — Backend PDF Endpoint (Go)

### Task 35: Add ConvertHTMLToPDF to Gotenberg client

**Files:**
- Modify: `internal/platform/render/gotenberg/client.go`
- Create or modify: `internal/platform/render/gotenberg/client_test.go`

- [ ] **Step 1: Write the failing Go test**

Write to `internal/platform/render/gotenberg/client_test.go` (create if missing; if it exists, append a new test function):

```go
package gotenberg

import (
    "bytes"
    "context"
    "io"
    "mime/multipart"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"
)

func TestConvertHTMLToPDF_SendsMultipartToChromiumRoute(t *testing.T) {
    var capturedPath string
    var capturedBody []byte
    var capturedContentType string

    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        capturedPath = r.URL.Path
        capturedContentType = r.Header.Get("Content-Type")
        body, _ := io.ReadAll(r.Body)
        capturedBody = body
        w.Header().Set("Content-Type", "application/pdf")
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte("%PDF-1.4 fake"))
    }))
    defer server.Close()

    client := NewClient(server.URL)

    pdf, err := client.ConvertHTMLToPDF(context.Background(), []byte("<html><body>Hi</body></html>"), []byte("body { color: black; }"))
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }

    if !bytes.HasPrefix(pdf, []byte("%PDF")) {
        t.Fatalf("expected PDF magic bytes, got %q", string(pdf[:8]))
    }
    if capturedPath != "/forms/chromium/convert/html" {
        t.Fatalf("expected chromium route, got %q", capturedPath)
    }
    if !strings.HasPrefix(capturedContentType, "multipart/form-data") {
        t.Fatalf("expected multipart request, got %q", capturedContentType)
    }
    if !bytes.Contains(capturedBody, []byte("index.html")) {
        t.Fatalf("expected body to include index.html part")
    }
    if !bytes.Contains(capturedBody, []byte("style.css")) {
        t.Fatalf("expected body to include style.css part")
    }

    // Defensive sanity: ensure multipart body parses.
    _, params, err := mimeParseMediaType(capturedContentType)
    if err != nil {
        t.Fatalf("parse media type: %v", err)
    }
    mr := multipart.NewReader(bytes.NewReader(capturedBody), params["boundary"])
    seen := map[string]bool{}
    for {
        part, err := mr.NextPart()
        if err == io.EOF {
            break
        }
        if err != nil {
            t.Fatalf("next part: %v", err)
        }
        seen[part.FileName()] = true
    }
    if !seen["index.html"] || !seen["style.css"] {
        t.Fatalf("missing parts; saw %v", seen)
    }
}

// small wrapper so the test doesn't drag in mime.ParseMediaType at the top;
// keeps the diff easy to read.
func mimeParseMediaType(v string) (string, map[string]string, error) {
    return mimeParseMediaTypeShim(v)
}
```

Also append at the bottom of the test file:

```go
import "mime"

func init() {
    mimeParseMediaTypeShim = func(v string) (string, map[string]string, error) {
        return mime.ParseMediaType(v)
    }
}

var mimeParseMediaTypeShim func(string) (string, map[string]string, error)
```

- [ ] **Step 2: Run the test — expect failure**

Run: `go test ./internal/platform/render/gotenberg/... -run TestConvertHTMLToPDF_SendsMultipartToChromiumRoute -v 2>&1 | tail -20`
Expected: FAIL — `ConvertHTMLToPDF` method does not exist.

- [ ] **Step 3: Implement ConvertHTMLToPDF**

Open `internal/platform/render/gotenberg/client.go` and add the following method at the bottom of the file (after `ConvertDocxToPDF`):

```go
// ConvertHTMLToPDF sends an HTML document plus an auxiliary stylesheet to
// Gotenberg's Chromium route and returns the rendered PDF bytes.
func (c *Client) ConvertHTMLToPDF(ctx context.Context, htmlBytes []byte, cssBytes []byte) ([]byte, error) {
    var body bytes.Buffer
    writer := multipart.NewWriter(&body)

    htmlPart, err := writer.CreateFormFile("files", "index.html")
    if err != nil {
        return nil, fmt.Errorf("gotenberg: create html form file: %w", err)
    }
    if _, err := htmlPart.Write(htmlBytes); err != nil {
        return nil, fmt.Errorf("gotenberg: write html content: %w", err)
    }

    if len(cssBytes) > 0 {
        cssPart, err := writer.CreateFormFile("files", "style.css")
        if err != nil {
            return nil, fmt.Errorf("gotenberg: create css form file: %w", err)
        }
        if _, err := cssPart.Write(cssBytes); err != nil {
            return nil, fmt.Errorf("gotenberg: write css content: %w", err)
        }
    }

    if err := writer.Close(); err != nil {
        return nil, fmt.Errorf("gotenberg: close multipart: %w", err)
    }

    url := c.baseURL + "/forms/chromium/convert/html"
    req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, &body)
    if err != nil {
        return nil, fmt.Errorf("gotenberg: create request: %w", err)
    }
    req.Header.Set("Content-Type", writer.FormDataContentType())

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("gotenberg: html request failed: %w", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        payload, _ := io.ReadAll(resp.Body)
        return nil, fmt.Errorf("gotenberg: html conversion returned status %d: %s", resp.StatusCode, string(payload))
    }

    pdfBytes, err := io.ReadAll(resp.Body)
    if err != nil {
        return nil, fmt.Errorf("gotenberg: read pdf response: %w", err)
    }
    return pdfBytes, nil
}
```

- [ ] **Step 4: Run tests — expect pass**

Run: `go test ./internal/platform/render/gotenberg/... -run TestConvertHTMLToPDF_SendsMultipartToChromiumRoute -v 2>&1 | tail -20`
Expected: PASS — test passes.

- [ ] **Step 5: Commit**

```bash
git add internal/platform/render/gotenberg/client.go internal/platform/render/gotenberg/client_test.go
git commit -m "feat(gotenberg): add ConvertHTMLToPDF method using Chromium route"
```

### Task 36: Implement the render/pdf HTTP handler

**Files:**
- Create: `internal/modules/documents/delivery/http/handler_render_pdf.go`
- Create: `internal/modules/documents/delivery/http/handler_render_pdf_test.go`

- [ ] **Step 1: Write the failing handler test**

Write to `internal/modules/documents/delivery/http/handler_render_pdf_test.go`:

```go
package httpdelivery

import (
    "bytes"
    "context"
    "io"
    "mime/multipart"
    "net/http"
    "net/http/httptest"
    "strings"
    "testing"
)

type fakePdfRenderer struct {
    lastHTML []byte
    lastCSS  []byte
    result   []byte
    err      error
}

func (f *fakePdfRenderer) ConvertHTMLToPDF(ctx context.Context, html []byte, css []byte) ([]byte, error) {
    f.lastHTML = html
    f.lastCSS = css
    return f.result, f.err
}

func TestHandleDocumentRenderPDF_ReturnsPDFBytes(t *testing.T) {
    fakeRenderer := &fakePdfRenderer{result: []byte("%PDF-1.4 fake")}
    handler := NewRenderPDFHandler(fakeRenderer)

    var body bytes.Buffer
    writer := multipart.NewWriter(&body)
    htmlPart, _ := writer.CreateFormFile("index.html", "index.html")
    _, _ = htmlPart.Write([]byte("<html><body>Hi</body></html>"))
    cssPart, _ := writer.CreateFormFile("style.css", "style.css")
    _, _ = cssPart.Write([]byte("body { color: black; }"))
    _ = writer.Close()

    req := httptest.NewRequest(http.MethodPost, "/api/v1/documents/d1/render/pdf", &body)
    req.Header.Set("Content-Type", writer.FormDataContentType())
    req = req.WithContext(withUserIDForTest(req.Context(), "u-1"))

    w := httptest.NewRecorder()
    handler.HandleRenderPDF(w, req, "d1")

    if w.Code != http.StatusOK {
        t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
    }
    if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/pdf") {
        t.Fatalf("expected application/pdf, got %q", ct)
    }
    if !bytes.HasPrefix(w.Body.Bytes(), []byte("%PDF")) {
        t.Fatalf("response body missing PDF magic")
    }
    if !bytes.Contains(fakeRenderer.lastHTML, []byte("Hi")) {
        t.Fatalf("renderer did not receive html body")
    }
}

func TestHandleDocumentRenderPDF_UnauthenticatedRejected(t *testing.T) {
    fakeRenderer := &fakePdfRenderer{result: []byte("%PDF")}
    handler := NewRenderPDFHandler(fakeRenderer)

    var body bytes.Buffer
    writer := multipart.NewWriter(&body)
    part, _ := writer.CreateFormFile("index.html", "index.html")
    _, _ = part.Write([]byte("<html></html>"))
    _ = writer.Close()

    req := httptest.NewRequest(http.MethodPost, "/api/v1/documents/d1/render/pdf", &body)
    req.Header.Set("Content-Type", writer.FormDataContentType())

    w := httptest.NewRecorder()
    handler.HandleRenderPDF(w, req, "d1")

    if w.Code != http.StatusUnauthorized {
        t.Fatalf("expected 401, got %d", w.Code)
    }
}

func TestHandleDocumentRenderPDF_RejectsOversizedPayload(t *testing.T) {
    fakeRenderer := &fakePdfRenderer{result: []byte("%PDF")}
    handler := NewRenderPDFHandler(fakeRenderer)
    handler.MaxPayloadBytes = 100 // tiny for test

    large := bytes.Repeat([]byte("X"), 500)
    var body bytes.Buffer
    writer := multipart.NewWriter(&body)
    part, _ := writer.CreateFormFile("index.html", "index.html")
    _, _ = part.Write(large)
    _ = writer.Close()

    req := httptest.NewRequest(http.MethodPost, "/api/v1/documents/d1/render/pdf", &body)
    req.Header.Set("Content-Type", writer.FormDataContentType())
    req = req.WithContext(withUserIDForTest(req.Context(), "u-1"))

    w := httptest.NewRecorder()
    handler.HandleRenderPDF(w, req, "d1")

    if w.Code != http.StatusRequestEntityTooLarge {
        t.Fatalf("expected 413, got %d", w.Code)
    }
}

func TestRenderPDFHandlerIgnoresReadToSilence(t *testing.T) {
    // Guard against accidental removal; keeps io in the import set.
    _, _ = io.ReadAll(bytes.NewReader(nil))
}
```

- [ ] **Step 2: Add test helper withUserIDForTest if missing**

Check if `withUserIDForTest` exists in the package:

Run: `grep -rn "withUserIDForTest" internal/modules/documents/delivery/http/ 2>&1 | head -5`

If it does NOT exist, append this helper to `internal/modules/documents/delivery/http/handler_render_pdf_test.go`:

```go
func withUserIDForTest(ctx context.Context, userID string) context.Context {
    return contextWithUserID(ctx, userID)
}
```

Then check whether `contextWithUserID` exists in the handler package:

Run: `grep -rn "func contextWithUserID\|userIDFromContext" internal/modules/documents/delivery/http/ 2>&1 | head -5`

If `contextWithUserID` does not exist (only `userIDFromContext` does), add it to the helper file you already identified OR inline it in the test file. Replace the `withUserIDForTest` body with whatever matches the existing context pattern discovered in the grep.

- [ ] **Step 3: Run the test — expect failure**

Run: `go test ./internal/modules/documents/delivery/http/... -run TestHandleDocumentRenderPDF -v 2>&1 | tail -30`
Expected: FAIL — `NewRenderPDFHandler` undefined.

- [ ] **Step 4: Implement handler_render_pdf.go**

Write to `internal/modules/documents/delivery/http/handler_render_pdf.go`:

```go
package httpdelivery

import (
    "context"
    "fmt"
    "io"
    "net/http"
)

// PDFRenderer is the minimal contract the render handler needs from Gotenberg.
// It matches *gotenberg.Client so the production wiring is a straight pass-through.
type PDFRenderer interface {
    ConvertHTMLToPDF(ctx context.Context, html []byte, css []byte) ([]byte, error)
}

// RenderPDFHandler converts editor-produced HTML/CSS to PDF via Gotenberg.
type RenderPDFHandler struct {
    renderer       PDFRenderer
    MaxPayloadBytes int64
}

const defaultRenderPDFMaxPayload = 10 * 1024 * 1024 // 10 MB

func NewRenderPDFHandler(renderer PDFRenderer) *RenderPDFHandler {
    return &RenderPDFHandler{
        renderer:       renderer,
        MaxPayloadBytes: defaultRenderPDFMaxPayload,
    }
}

func (h *RenderPDFHandler) HandleRenderPDF(w http.ResponseWriter, r *http.Request, documentID string) {
    traceID := requestTraceID(r)

    if userIDFromContext(r.Context()) == "" {
        writeAPIError(w, http.StatusUnauthorized, "AUTH_UNAUTHORIZED", "Authentication required", traceID)
        return
    }

    if documentID == "" {
        writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Document ID required", traceID)
        return
    }

    // Enforce payload ceiling before reading the full body into memory.
    r.Body = http.MaxBytesReader(w, r.Body, h.MaxPayloadBytes)

    if err := r.ParseMultipartForm(h.MaxPayloadBytes); err != nil {
        if err.Error() == "http: request body too large" {
            writeAPIError(w, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "Payload exceeds limit", traceID)
            return
        }
        writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", fmt.Sprintf("multipart parse: %v", err), traceID)
        return
    }

    htmlBytes, err := readFormFile(r, "index.html")
    if err != nil {
        writeAPIError(w, http.StatusBadRequest, "VALIDATION_ERROR", err.Error(), traceID)
        return
    }
    cssBytes, _ := readFormFile(r, "style.css") // optional

    pdf, err := h.renderer.ConvertHTMLToPDF(r.Context(), htmlBytes, cssBytes)
    if err != nil {
        writeAPIError(w, http.StatusBadGateway, "RENDER_UPSTREAM_ERROR", fmt.Sprintf("pdf render failed: %v", err), traceID)
        return
    }

    w.Header().Set("Content-Type", "application/pdf")
    w.Header().Set("Content-Length", fmt.Sprintf("%d", len(pdf)))
    w.WriteHeader(http.StatusOK)
    _, _ = w.Write(pdf)
}

func readFormFile(r *http.Request, name string) ([]byte, error) {
    file, _, err := r.FormFile(name)
    if err != nil {
        return nil, fmt.Errorf("missing form file %q: %w", name, err)
    }
    defer file.Close()
    return io.ReadAll(file)
}
```

- [ ] **Step 5: Run the test — expect pass**

Run: `go test ./internal/modules/documents/delivery/http/... -run TestHandleDocumentRenderPDF -v 2>&1 | tail -20`
Expected: PASS — all three subtests passing.

- [ ] **Step 6: Commit**

```bash
git add internal/modules/documents/delivery/http/handler_render_pdf.go internal/modules/documents/delivery/http/handler_render_pdf_test.go
git commit -m "feat(documents-http): add POST /render/pdf handler proxying to Gotenberg Chromium"
```

### Task 37: Register the render/pdf route in the main handler

**Files:**
- Modify: `internal/modules/documents/delivery/http/handler.go`

- [ ] **Step 1: Find the existing route registration for export/docx**

Run: `grep -n "export/docx\|render/pdf\|RegisterRoutes\|Handle(" internal/modules/documents/delivery/http/handler.go | head -20`
Expected: Output showing the router registration section.

- [ ] **Step 2: Add RenderPDFHandler field to the Handler struct**

In `internal/modules/documents/delivery/http/handler.go`, find the `Handler` struct definition. Add a new field `renderPDF *RenderPDFHandler` alongside the existing handler fields.

In the `Handler` struct:

```go
renderPDF *RenderPDFHandler
```

In the constructor or setup function that wires up dependencies, accept a `PDFRenderer` parameter and wire it:

```go
h.renderPDF = NewRenderPDFHandler(pdfRenderer)
```

- [ ] **Step 3: Register the route**

Find the route registration section (a `switch r.URL.Path` or `mux.HandleFunc` block). Add a new case matching `POST /api/v1/documents/{id}/render/pdf`. Extract the `{id}` segment using the existing pattern in this file.

Example (adjust to match the file's actual routing style):

```go
case strings.HasSuffix(path, "/render/pdf") && r.Method == http.MethodPost:
    documentID := extractDocumentIDFromPath(path, "/render/pdf")
    h.renderPDF.HandleRenderPDF(w, r, documentID)
    return
```

- [ ] **Step 4: Build and verify**

Run: `go build ./internal/modules/documents/delivery/http/...`
Expected: Clean build, exit code 0.

- [ ] **Step 5: Run the test suite for the package**

Run: `go test ./internal/modules/documents/delivery/http/... 2>&1 | tail -20`
Expected: All tests pass.

- [ ] **Step 6: Commit**

```bash
git add internal/modules/documents/delivery/http/handler.go
git commit -m "feat(documents-http): register render/pdf route on main handler"
```

### Task 38: Wire the render/pdf handler in service bootstrap

**Files:**
- Modify: `internal/modules/documents/delivery/http/handler.go` (if bootstrap lives there) OR the service bootstrap file (search for it).

- [ ] **Step 1: Find where Handler is constructed**

Run: `grep -rn "NewHandler\b" internal/modules/documents/delivery/http/ cmd/ 2>&1 | head -10`
Expected: Location(s) where the handler is constructed.

- [ ] **Step 2: Pass the Gotenberg client to NewHandler**

Locate the construction site identified above. The existing `gotenbergClient` is already available on the service (per `grep -rn "gotenbergClient" internal/modules/documents/ | head -5`). Wire it into the handler constructor — in the handler constructor, accept a `PDFRenderer` argument and store it:

```go
func NewHandler(/* existing args */, pdfRenderer PDFRenderer) *Handler {
    h := &Handler{/* existing fields */}
    h.renderPDF = NewRenderPDFHandler(pdfRenderer)
    return h
}
```

Update every call site of `NewHandler` found in step 1 to pass the Gotenberg client. When the Gotenberg client is nil (feature disabled), pass a nil renderer — the handler will return 502 on requests, which is the correct failure mode.

- [ ] **Step 3: Build**

Run: `go build ./...`
Expected: Clean build.

- [ ] **Step 4: Run package tests**

Run: `go test ./internal/modules/documents/delivery/http/... 2>&1 | tail -20`
Expected: All tests passing.

- [ ] **Step 5: Commit**

```bash
git add internal/modules/documents/delivery/http/handler.go
git commit -m "wire(documents-http): inject Gotenberg client into render/pdf handler"
```

---

## Part 13 — Golden File Infrastructure & First Fixture

### Task 39: Implement XML normalization helpers

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/golden/golden-helpers.ts`

- [ ] **Step 1: Implement golden-helpers.ts**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/golden/golden-helpers.ts`:

```ts
// Helpers for golden file tests. Normalizes DOCX document.xml and HTML
// output so tests compare semantics instead of engine-specific metadata.

// Tier-3 attributes that may differ between runs and should be stripped
// from comparisons. Kept intentionally narrow; update with explicit review.
const STRIP_ATTRIBUTES = new Set([
  "w:rsidR",
  "w:rsidRDefault",
  "w:rsidP",
  "w:rsidRPr",
  "w:paraId",
  "w:textId",
  "w:rsidTr",
]);

function stripRSIDs(xml: string): string {
  let result = xml;
  for (const attr of STRIP_ATTRIBUTES) {
    const re = new RegExp(`\\s${attr}="[^"]*"`, "g");
    result = result.replace(re, "");
  }
  return result;
}

function collapseWhitespace(xml: string): string {
  return xml
    .replace(/>\s+</g, "><")
    .replace(/\s+/g, " ")
    .trim();
}

export function normalizeDocxXml(xml: string): string {
  return collapseWhitespace(stripRSIDs(xml));
}

export function normalizeHtml(html: string): string {
  return collapseWhitespace(
    html
      .replace(/<!--[\s\S]*?-->/g, "")
      .replace(/\s(data-reactroot|data-bn-key)="[^"]*"/g, ""),
  );
}

export async function unzipDocxDocumentXml(blob: Blob): Promise<string> {
  // docx.js ships JSZip under the hood. Use the browser's DecompressionStream
  // when available; fall back to parsing the raw ZIP central directory.
  // For simplicity in this plan we import JSZip directly.
  const JSZip = (await import("jszip")).default;
  const zip = await JSZip.loadAsync(await blob.arrayBuffer());
  const documentXml = zip.file("word/document.xml");
  if (!documentXml) {
    throw new Error("word/document.xml not found in DOCX blob");
  }
  return await documentXml.async("string");
}
```

- [ ] **Step 2: Install jszip dependency**

Run: `cd frontend/apps/web && npm install jszip@^3.10.0`
Expected: `jszip` added to dependencies.

- [ ] **Step 3: Verify compilation**

Run: `cd frontend/apps/web && npx tsc --noEmit 2>&1 | grep golden | head -5`
Expected: No errors.

- [ ] **Step 4: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/golden/golden-helpers.ts frontend/apps/web/package.json frontend/apps/web/package-lock.json
git commit -m "feat(mddm-engine): add golden test XML/HTML normalization helpers + jszip"
```

### Task 40: Create the 01-simple-po golden fixture input

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/golden/fixtures/01-simple-po/input.mddm.json`

- [ ] **Step 1: Write the input fixture**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/golden/fixtures/01-simple-po/input.mddm.json`:

```json
{
  "mddm_version": 1,
  "template_ref": null,
  "blocks": [
    {
      "id": "00000000-0000-4000-8000-000000000001",
      "type": "section",
      "props": { "title": "1. Procedimento Operacional", "color": "red" },
      "children": []
    },
    {
      "id": "00000000-0000-4000-8000-000000000002",
      "type": "fieldGroup",
      "props": { "columns": 2 },
      "children": [
        {
          "id": "00000000-0000-4000-8000-000000000003",
          "type": "field",
          "props": { "label": "Responsável" },
          "content": [
            { "type": "text", "text": "João Silva", "styles": {} }
          ],
          "children": []
        },
        {
          "id": "00000000-0000-4000-8000-000000000004",
          "type": "field",
          "props": { "label": "Departamento" },
          "content": [
            { "type": "text", "text": "Qualidade", "styles": {} }
          ],
          "children": []
        }
      ]
    },
    {
      "id": "00000000-0000-4000-8000-000000000005",
      "type": "paragraph",
      "props": {},
      "content": [
        { "type": "text", "text": "Este documento descreve os passos do procedimento.", "styles": {} }
      ],
      "children": []
    }
  ]
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/golden/fixtures/01-simple-po/input.mddm.json
git commit -m "test(mddm-engine): add 01-simple-po golden fixture input"
```

### Task 41: Implement the golden runner (DOCX XML + HTML snapshot)

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/golden/__tests__/golden-runner.test.ts`

- [ ] **Step 1: Write the failing test (snapshot-style)**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/golden/__tests__/golden-runner.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { readFileSync, existsSync } from "node:fs";
import { resolve } from "node:path";
import { mddmToDocx } from "../../docx-emitter";
import { defaultLayoutTokens } from "../../layout-ir";
import { normalizeDocxXml, unzipDocxDocumentXml } from "../golden-helpers";
import type { MDDMEnvelope } from "../../../adapter";

const FIXTURE_DIR = resolve(__dirname, "../fixtures/01-simple-po");
const INPUT_PATH = resolve(FIXTURE_DIR, "input.mddm.json");
const EXPECTED_DOCX_XML = resolve(FIXTURE_DIR, "expected.document.xml");

describe("Golden fixture: 01-simple-po", () => {
  it("emits DOCX matching the approved document.xml", async () => {
    const envelope = JSON.parse(readFileSync(INPUT_PATH, "utf8")) as MDDMEnvelope;
    const blob = await mddmToDocx(envelope, defaultLayoutTokens);
    const xml = await unzipDocxDocumentXml(blob);
    const actual = normalizeDocxXml(xml);

    if (!existsSync(EXPECTED_DOCX_XML)) {
      throw new Error(
        `Golden file missing: ${EXPECTED_DOCX_XML}\n\n` +
          `Generate it once with:\n` +
          `  cd frontend/apps/web && npx vitest run <this-test-file> --reporter verbose\n` +
          `Then commit the file after manual review.`,
      );
    }

    const expected = normalizeDocxXml(readFileSync(EXPECTED_DOCX_XML, "utf8"));
    expect(actual).toBe(expected);
  });
});
```

- [ ] **Step 2: Run the test — expect failure (missing expected file)**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/golden/__tests__/golden-runner.test.ts 2>&1 | tail -30`
Expected: FAIL with "Golden file missing" error.

- [ ] **Step 3: Generate the expected file**

Add a temporary one-time helper. Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/golden/__tests__/generate-golden.test.ts`:

```ts
import { describe, it } from "vitest";
import { readFileSync, writeFileSync } from "node:fs";
import { resolve } from "node:path";
import { mddmToDocx } from "../../docx-emitter";
import { defaultLayoutTokens } from "../../layout-ir";
import { unzipDocxDocumentXml } from "../golden-helpers";
import type { MDDMEnvelope } from "../../../adapter";

const FIXTURE_DIR = resolve(__dirname, "../fixtures/01-simple-po");

describe.skipIf(!process.env.MDDM_GOLDEN_UPDATE)("Golden regenerator (01-simple-po)", () => {
  it("writes expected.document.xml", async () => {
    const envelope = JSON.parse(readFileSync(resolve(FIXTURE_DIR, "input.mddm.json"), "utf8")) as MDDMEnvelope;
    const blob = await mddmToDocx(envelope, defaultLayoutTokens);
    const xml = await unzipDocxDocumentXml(blob);
    writeFileSync(resolve(FIXTURE_DIR, "expected.document.xml"), xml, "utf8");
  });
});
```

Then run:

```bash
cd frontend/apps/web
MDDM_GOLDEN_UPDATE=1 npx vitest run src/features/documents/mddm-editor/engine/golden/__tests__/generate-golden.test.ts
```

Expected: PASS. The file `expected.document.xml` is now present.

- [ ] **Step 4: Manually inspect the generated file**

Run: `head -40 frontend/apps/web/src/features/documents/mddm-editor/engine/golden/fixtures/01-simple-po/expected.document.xml`
Expected: Well-formed OOXML containing the section header text, field labels, and paragraph content from the fixture. Review visually for anomalies. If anything looks wrong, fix the relevant emitter and regenerate.

- [ ] **Step 5: Run the golden runner — expect pass**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/golden/__tests__/golden-runner.test.ts`
Expected: PASS — golden matches.

- [ ] **Step 6: Commit the golden file and the runner**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/golden/fixtures/01-simple-po/expected.document.xml \
        frontend/apps/web/src/features/documents/mddm-editor/engine/golden/__tests__/golden-runner.test.ts \
        frontend/apps/web/src/features/documents/mddm-editor/engine/golden/__tests__/generate-golden.test.ts
git commit -m "test(mddm-engine): add 01-simple-po DOCX golden runner"
```

---

## Part 14 — Renderer Completeness Gate

### Task 42: Create block registry + completeness CI test

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/completeness-gate/block-registry.ts`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/completeness-gate/__tests__/completeness.test.ts`

- [ ] **Step 1: Implement block-registry.ts**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/completeness-gate/block-registry.ts`:

```ts
// Central registry of block types the MDDM engine is allowed to render.
// The completeness gate ensures every listed block has a React render,
// a toExternalHTML hook, and a DOCX emitter.

export type BlockSupport = Readonly<{
  type: string;
  hasReactRender: boolean;
  hasExternalHtml: boolean;
  hasDocxEmitter: boolean;
}>;

export const BLOCK_REGISTRY: readonly BlockSupport[] = [
  // Standard BlockNote blocks in Plan 1 scope
  { type: "paragraph", hasReactRender: true, hasExternalHtml: true, hasDocxEmitter: true },
  { type: "heading",   hasReactRender: true, hasExternalHtml: true, hasDocxEmitter: true },

  // MDDM custom blocks in Plan 1 scope
  { type: "section",    hasReactRender: true, hasExternalHtml: true, hasDocxEmitter: true },
  { type: "field",      hasReactRender: true, hasExternalHtml: true, hasDocxEmitter: true },
  { type: "fieldGroup", hasReactRender: true, hasExternalHtml: true, hasDocxEmitter: true },

  // MDDM custom blocks deferred to Plan 2. Listed with false so the gate
  // prevents accidental activation without implementation.
  { type: "repeatable",     hasReactRender: true, hasExternalHtml: false, hasDocxEmitter: false },
  { type: "repeatableItem", hasReactRender: true, hasExternalHtml: false, hasDocxEmitter: false },
  { type: "richBlock",      hasReactRender: true, hasExternalHtml: false, hasDocxEmitter: false },
  { type: "dataTable",      hasReactRender: true, hasExternalHtml: false, hasDocxEmitter: false },
  { type: "dataTableRow",   hasReactRender: true, hasExternalHtml: false, hasDocxEmitter: false },
  { type: "dataTableCell",  hasReactRender: true, hasExternalHtml: false, hasDocxEmitter: false },
];

export function getFullySupportedBlockTypes(): readonly string[] {
  return BLOCK_REGISTRY
    .filter((b) => b.hasReactRender && b.hasExternalHtml && b.hasDocxEmitter)
    .map((b) => b.type);
}
```

- [ ] **Step 2: Write the failing completeness test**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/completeness-gate/__tests__/completeness.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { BLOCK_REGISTRY, getFullySupportedBlockTypes } from "../block-registry";
import { mddmToDocx, MissingEmitterError } from "../../docx-emitter";
import { defaultLayoutTokens } from "../../layout-ir";
import type { MDDMEnvelope } from "../../../adapter";

describe("Renderer completeness gate", () => {
  it("includes every Plan 1 MVP block as fully supported", () => {
    const supported = getFullySupportedBlockTypes();
    expect(supported).toContain("paragraph");
    expect(supported).toContain("heading");
    expect(supported).toContain("section");
    expect(supported).toContain("field");
    expect(supported).toContain("fieldGroup");
  });

  it("DOCX emitter produces output for every fully-supported block type", async () => {
    for (const type of getFullySupportedBlockTypes()) {
      const envelope: MDDMEnvelope = {
        mddm_version: 1,
        template_ref: null,
        blocks: [
          {
            id: `test-${type}`,
            type,
            props: type === "field" || type === "section"
              ? { label: "L", title: "T" }
              : {},
            content: type === "paragraph" || type === "heading" || type === "field"
              ? [{ type: "text", text: "x", styles: {} }]
              : undefined,
            children: type === "fieldGroup"
              ? [{ id: "f1", type: "field", props: { label: "A" }, content: [], children: [] }]
              : [],
          } as any,
        ],
      };

      // Should NOT throw MissingEmitterError for any supported block type.
      await expect(mddmToDocx(envelope, defaultLayoutTokens)).resolves.toBeInstanceOf(Blob);
    }
  });

  it("DOCX emitter throws MissingEmitterError for unsupported types in the registry", async () => {
    const unsupported = BLOCK_REGISTRY.filter((b) => !b.hasDocxEmitter).map((b) => b.type);
    for (const type of unsupported) {
      const envelope: MDDMEnvelope = {
        mddm_version: 1,
        template_ref: null,
        blocks: [{ id: "x", type, props: {}, children: [] } as any],
      };
      await expect(mddmToDocx(envelope, defaultLayoutTokens)).rejects.toBeInstanceOf(MissingEmitterError);
    }
  });
});
```

- [ ] **Step 3: Run the test — expect pass (everything should already be in place)**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/completeness-gate/__tests__/completeness.test.ts`
Expected: PASS — 3 tests passing.

- [ ] **Step 4: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/completeness-gate/
git commit -m "feat(mddm-engine): add renderer completeness gate (Plan 1 MVP scope)"
```

---

## Part 15 — Feature Flag + Export State Contract

### Task 43: Implement feature flag registry

**Files:**
- Create: `frontend/apps/web/src/features/featureFlags.ts`

- [ ] **Step 1: Check if a feature flag module already exists**

Run: `grep -rn "featureFlags\|FeatureFlag" frontend/apps/web/src/ | head -10`
Expected: Either shows an existing module OR is empty (new module needed).

- [ ] **Step 2: Create the feature flags module (or integrate with existing)**

If no module exists, write to `frontend/apps/web/src/features/featureFlags.ts`:

```ts
// Feature flag registry. Flags are read once at module load time from a
// window-level config object injected by the backend via the HTML shell.
// Future work (Plan 4): replace with a per-user config endpoint.

type FeatureFlags = Readonly<{
  MDDM_NATIVE_EXPORT: boolean;
}>;

function readFlags(): FeatureFlags {
  const injected = typeof window !== "undefined"
    ? (window as unknown as { __METALDOCS_FEATURE_FLAGS?: Partial<FeatureFlags> }).__METALDOCS_FEATURE_FLAGS
    : undefined;

  return {
    MDDM_NATIVE_EXPORT: injected?.MDDM_NATIVE_EXPORT === true,
  };
}

export const featureFlags: FeatureFlags = readFlags();
```

If a module exists, add only the `MDDM_NATIVE_EXPORT: false` entry to it, matching the existing pattern.

- [ ] **Step 3: Verify compilation**

Run: `cd frontend/apps/web && npx tsc --noEmit 2>&1 | grep featureFlags | head -5`
Expected: No errors.

- [ ] **Step 4: Commit**

```bash
git add frontend/apps/web/src/features/featureFlags.ts
git commit -m "feat(feature-flags): add MDDM_NATIVE_EXPORT flag (off by default)"
```

### Task 44: Implement SaveBeforeExportDialog component

**Files:**
- Create: `frontend/apps/web/src/features/documents/browser-editor/SaveBeforeExportDialog.tsx`

- [ ] **Step 1: Implement the dialog**

Write to `frontend/apps/web/src/features/documents/browser-editor/SaveBeforeExportDialog.tsx`:

```tsx
import type { CSSProperties } from "react";

export type SaveBeforeExportDialogProps = {
  open: boolean;
  isReleased: boolean;
  onSaveAndExport: () => void;
  onExportSaved: () => void;
  onCancel: () => void;
};

const overlayStyle: CSSProperties = {
  position: "fixed",
  inset: 0,
  background: "rgba(15, 15, 15, 0.55)",
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
  zIndex: 9999,
};

const dialogStyle: CSSProperties = {
  background: "#ffffff",
  borderRadius: "8px",
  padding: "24px",
  width: "min(440px, 92vw)",
  boxShadow: "0 20px 40px rgba(0, 0, 0, 0.2)",
};

const actionsStyle: CSSProperties = {
  display: "flex",
  gap: "8px",
  justifyContent: "flex-end",
  marginTop: "20px",
};

const buttonStyle: CSSProperties = {
  padding: "8px 16px",
  borderRadius: "6px",
  border: "1px solid #cccccc",
  background: "#ffffff",
  cursor: "pointer",
};

const primaryButtonStyle: CSSProperties = {
  ...buttonStyle,
  background: "#6b1f2a",
  color: "#ffffff",
  border: "1px solid #6b1f2a",
};

export function SaveBeforeExportDialog({
  open,
  isReleased,
  onSaveAndExport,
  onExportSaved,
  onCancel,
}: SaveBeforeExportDialogProps) {
  if (!open) {
    return null;
  }

  const defaultActionLabel = isReleased
    ? "Exportar versão salva"
    : "Salvar e exportar";
  const defaultAction = isReleased ? onExportSaved : onSaveAndExport;
  const secondaryActionLabel = isReleased
    ? "Salvar e exportar"
    : "Exportar versão salva";
  const secondaryAction = isReleased ? onSaveAndExport : onExportSaved;

  return (
    <div role="dialog" aria-modal="true" aria-labelledby="mddm-save-before-export-title" style={overlayStyle}>
      <div style={dialogStyle}>
        <h3 id="mddm-save-before-export-title" style={{ margin: 0, fontSize: "1.15rem" }}>
          Você tem alterações não salvas
        </h3>
        <p style={{ marginTop: "12px", color: "#555" }}>
          {isReleased
            ? "Este documento está publicado. Por padrão, a exportação usa a versão salva. Para incluir suas edições locais, salve primeiro."
            : "Para garantir rastreabilidade, a exportação sempre usa a última versão salva. Deseja salvar agora e exportar, ou exportar a versão salva atual?"}
        </p>
        <div style={actionsStyle}>
          <button type="button" style={buttonStyle} onClick={onCancel}>
            Cancelar
          </button>
          <button type="button" style={buttonStyle} onClick={secondaryAction}>
            {secondaryActionLabel}
          </button>
          <button type="button" style={primaryButtonStyle} onClick={defaultAction} autoFocus>
            {defaultActionLabel}
          </button>
        </div>
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Verify compilation**

Run: `cd frontend/apps/web && npx tsc --noEmit 2>&1 | grep SaveBeforeExportDialog | head -5`
Expected: No errors.

- [ ] **Step 3: Commit**

```bash
git add frontend/apps/web/src/features/documents/browser-editor/SaveBeforeExportDialog.tsx
git commit -m "feat(browser-editor): add SaveBeforeExportDialog for export state contract"
```

---

## Part 16 — MDDMViewer Component

### Task 45: Create the read-only MDDM viewer

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/MDDMViewer.tsx`

- [ ] **Step 1: Inspect MDDMEditor for readOnly handling**

Run: `grep -n "readOnly\|editable" frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.tsx`
Expected: Shows whether `readOnly` prop exists on MDDMEditor. It does (see MDDMEditor.tsx).

- [ ] **Step 2: Implement MDDMViewer**

Write to `frontend/apps/web/src/features/documents/mddm-editor/MDDMViewer.tsx`:

```tsx
import type { PartialBlock } from "@blocknote/core";
import { MDDMEditor, type MDDMTheme } from "./MDDMEditor";

export type MDDMViewerProps = {
  initialContent?: PartialBlock[];
  theme?: MDDMTheme;
};

export function MDDMViewer({ initialContent, theme }: MDDMViewerProps) {
  return (
    <MDDMEditor
      initialContent={initialContent}
      theme={theme}
      readOnly={true}
    />
  );
}
```

- [ ] **Step 3: Verify compilation**

Run: `cd frontend/apps/web && npx tsc --noEmit 2>&1 | grep MDDMViewer | head -5`
Expected: No errors.

- [ ] **Step 4: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/MDDMViewer.tsx
git commit -m "feat(mddm-engine): add MDDMViewer read-only wrapper around MDDMEditor"
```

---

## Part 17 — BrowserDocumentEditorView Integration

### Task 46: Integrate new DOCX export behind feature flag

**Files:**
- Modify: `frontend/apps/web/src/features/documents/browser-editor/BrowserDocumentEditorView.tsx`

- [ ] **Step 1: Inspect the current handleExportDocx implementation**

Run: `grep -n "handleExportDocx\|exportDocumentDocx" frontend/apps/web/src/features/documents/browser-editor/BrowserDocumentEditorView.tsx | head -10`
Expected: Shows the current implementation that calls `exportDocumentDocx(document.documentId)`.

- [ ] **Step 2: Add imports**

At the top of `frontend/apps/web/src/features/documents/browser-editor/BrowserDocumentEditorView.tsx`, add these imports (preserving existing imports):

```tsx
import { featureFlags } from "../../featureFlags";
import { exportDocx as mddmExportDocx } from "../mddm-editor/engine/export";
import { defaultLayoutTokens } from "../mddm-editor/engine/layout-ir";
import type { MDDMEnvelope } from "../mddm-editor/adapter";
import { SaveBeforeExportDialog } from "./SaveBeforeExportDialog";
```

- [ ] **Step 3: Add dialog state**

In the `BrowserDocumentEditorView` component body, below the existing `useState` hooks, add:

```tsx
const [exportDialogOpen, setExportDialogOpen] = useState(false);
const [pendingExportKind, setPendingExportKind] = useState<"docx" | null>(null);
```

- [ ] **Step 4: Replace handleExportDocx with a feature-flagged version**

Replace the existing `handleExportDocx` function with:

```tsx
async function runDocxExport(useCurrentEditorState: boolean) {
  if (!document.documentId.trim() || isExporting) {
    return;
  }

  setIsExporting(true);
  try {
    if (featureFlags.MDDM_NATIVE_EXPORT) {
      // New client-side path
      const source = useCurrentEditorState && editorData ? editorData : bundle?.body ?? "";
      if (!source.trim() || !source.trim().startsWith("{")) {
        throw new Error("Document body is empty or not in MDDM format");
      }
      const envelope = JSON.parse(source) as MDDMEnvelope;
      const blob = await mddmExportDocx(envelope, defaultLayoutTokens);
      triggerBlobDownload(blob, `${(document.documentCode || "documento").trim().replace(/[^\w.-]+/g, "-")}.docx`);
      setErrorCode(null);
      setErrorMessage("");
    } else {
      // Legacy backend path
      const blob = await exportDocumentDocx(document.documentId);
      triggerBlobDownload(blob, `${(document.documentCode || "documento").trim().replace(/[^\w.-]+/g, "-")}.docx`);
      setErrorCode(null);
      setErrorMessage("");
    }
  } catch (error) {
    setErrorCode("save");
    setErrorMessage("Nao foi possivel exportar o DOCX deste documento.");
    const status = statusOf(error);
    if (status === 503) {
      setErrorMessage("Servico de render indisponivel. Inicie o docgen e tente novamente.");
    }
  } finally {
    setIsExporting(false);
  }
}

async function handleExportDocx() {
  if (!document.documentId.trim() || isExporting) {
    return;
  }
  if (!featureFlags.MDDM_NATIVE_EXPORT) {
    // Legacy path: same behavior as before the flag existed.
    void runDocxExport(false);
    return;
  }
  if (isDirty) {
    // Export State Contract — prompt the user.
    setPendingExportKind("docx");
    setExportDialogOpen(true);
    return;
  }
  void runDocxExport(false);
}
```

Also add this helper function inside the file (can be a local function or a top-level module function):

```tsx
function triggerBlobDownload(blob: Blob, filename: string) {
  const url = window.URL.createObjectURL(blob);
  const link = window.document.createElement("a");
  link.href = url;
  link.download = filename;
  window.document.body.appendChild(link);
  link.click();
  link.remove();
  window.URL.revokeObjectURL(url);
}
```

- [ ] **Step 5: Render the dialog at the bottom of the JSX tree**

Near the bottom of the component's returned JSX (before the closing `</section>`), add:

```tsx
<SaveBeforeExportDialog
  open={exportDialogOpen}
  isReleased={false /* Plan 3 adds true released detection */}
  onCancel={() => {
    setExportDialogOpen(false);
    setPendingExportKind(null);
  }}
  onSaveAndExport={async () => {
    setExportDialogOpen(false);
    await handleSave();
    if (pendingExportKind === "docx") {
      await runDocxExport(false);
    }
    setPendingExportKind(null);
  }}
  onExportSaved={async () => {
    setExportDialogOpen(false);
    if (pendingExportKind === "docx") {
      await runDocxExport(false);
    }
    setPendingExportKind(null);
  }}
/>
```

- [ ] **Step 6: Verify compilation**

Run: `cd frontend/apps/web && npx tsc --noEmit 2>&1 | grep BrowserDocumentEditorView | head -10`
Expected: No errors referencing `BrowserDocumentEditorView.tsx`.

- [ ] **Step 7: Run existing tests for the view**

Run: `cd frontend/apps/web && npx vitest run 2>&1 | tail -20`
Expected: All tests pass (no regression to existing behavior).

- [ ] **Step 8: Commit**

```bash
git add frontend/apps/web/src/features/documents/browser-editor/BrowserDocumentEditorView.tsx
git commit -m "feat(browser-editor): wire MDDM native DOCX export behind feature flag with save-before-export dialog"
```

---

## Part 18 — Gotenberg Container: Carlito Verification

### Task 47: Add Carlito to the Gotenberg Docker image

**Files:**
- Create: `docker/gotenberg/Dockerfile`
- Create: `docker/gotenberg/verify-carlito.sh`

- [ ] **Step 1: Check whether a Gotenberg Docker setup already exists**

Run: `find docker -type d -name "gotenberg" 2>/dev/null; find . -name "docker-compose*.yml" 2>/dev/null | head -5`
Expected: Shows existing docker config. If `docker/gotenberg/` already exists, inspect its contents first.

- [ ] **Step 2: Create the Dockerfile**

Write to `docker/gotenberg/Dockerfile`:

```dockerfile
# MetalDocs Gotenberg image — adds Carlito font (metric-compatible with
# Calibri) to the official Gotenberg image so Chromium HTML→PDF rendering
# uses the same metrics as the client editor and docx.js output.

FROM gotenberg/gotenberg:8

USER root

RUN apt-get update \
 && apt-get install -y --no-install-recommends fonts-crosextra-carlito fonts-liberation \
 && fc-cache -f \
 && rm -rf /var/lib/apt/lists/*

USER gotenberg

# Expose default Gotenberg port
EXPOSE 3000
```

- [ ] **Step 3: Create the verification script**

Write to `docker/gotenberg/verify-carlito.sh`:

```bash
#!/usr/bin/env bash
# Phase 0 gating check: verify the running Gotenberg container has Carlito
# installed. Run against a live container ID or name.
#
# Usage:
#   ./docker/gotenberg/verify-carlito.sh metaldocs-gotenberg
#
# Exit codes:
#   0  - Carlito present
#   1  - container not found
#   2  - Carlito missing

set -euo pipefail

CONTAINER="${1:-metaldocs-gotenberg}"

if ! docker ps --format '{{.Names}}' | grep -q "^${CONTAINER}$"; then
  echo "ERROR: container '${CONTAINER}' is not running" >&2
  exit 1
fi

if docker exec "${CONTAINER}" fc-list 2>/dev/null | grep -qi "carlito"; then
  echo "OK: Carlito is installed in container '${CONTAINER}'"
  docker exec "${CONTAINER}" fc-list | grep -i "carlito"
  exit 0
fi

echo "FAIL: Carlito font is missing from container '${CONTAINER}'" >&2
echo "Fix: rebuild the Gotenberg image from docker/gotenberg/Dockerfile" >&2
exit 2
```

- [ ] **Step 4: Make the verification script executable**

Run: `chmod +x docker/gotenberg/verify-carlito.sh`
Expected: No output, exit code 0.

- [ ] **Step 5: Document how to build and run**

Append a comment block to the top of `docker/gotenberg/Dockerfile`:

```dockerfile
# Build:   docker build -t metaldocs/gotenberg:local docker/gotenberg
# Run:     docker run --name metaldocs-gotenberg -p 3000:3000 -d metaldocs/gotenberg:local
# Verify:  ./docker/gotenberg/verify-carlito.sh metaldocs-gotenberg
```

- [ ] **Step 6: Commit**

```bash
git add docker/gotenberg/Dockerfile docker/gotenberg/verify-carlito.sh
git commit -m "infra(gotenberg): add Dockerfile with Carlito font and verification script"
```

---

## Part 19 — Engine Barrel & Final Smoke Test

### Task 48: Update the engine root barrel export

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/engine/index.ts`

- [ ] **Step 1: Overwrite engine/index.ts with complete barrel**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/index.ts`:

```ts
// MDDM Rendering Engine — public surface. Consumers import from here.

export * from "./layout-ir";
export * from "./helpers/units";
export * from "./asset-resolver";
export * from "./canonicalize-migrate";
export * from "./docx-emitter";
export * from "./external-html";
export * from "./print-stylesheet";
export * from "./export";
```

- [ ] **Step 2: Verify compilation**

Run: `cd frontend/apps/web && npx tsc --noEmit 2>&1 | tail -20`
Expected: Zero errors in the whole project.

- [ ] **Step 3: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/index.ts
git commit -m "feat(mddm-engine): expand engine barrel to expose all Plan 1 modules"
```

### Task 49: Run the full test suite

**Files:** (none — verification step)

- [ ] **Step 1: Run all vitest tests in the web app**

Run: `cd frontend/apps/web && npm test 2>&1 | tail -30`
Expected: All tests passing. Test counts roughly:
- Layout IR: 17 tests (6 tokens + 4 components + 7 contract)
- Helpers: 8 tests (units)
- Asset resolver: 11 tests (6 allowlist + 5 asset-resolver)
- Canonicalize/migrate: 4 tests
- DOCX emitter: 15+ tests (2 paragraph + 3 heading + 3 section + 2 field + 3 field-group + 3 main + 5 inline-content)
- External HTML: 10 tests (4 section + 3 field + 3 field-group)
- Export: 5 tests (3 wrap-print-document + 2 export-docx)
- Golden runner: 1 test
- Completeness gate: 3 tests
- Plus any existing adapter and styling-contract tests

Total: approximately 75-80 tests.

- [ ] **Step 2: Run Go tests for the documents package**

Run: `go test ./internal/modules/documents/... ./internal/platform/render/gotenberg/... 2>&1 | tail -20`
Expected: All existing tests pass plus the new `TestConvertHTMLToPDF_SendsMultipartToChromiumRoute` and `TestHandleDocumentRenderPDF_*` tests.

- [ ] **Step 3: Run TypeScript full build**

Run: `cd frontend/apps/web && npm run build 2>&1 | tail -20`
Expected: Clean build with no TypeScript errors.

- [ ] **Step 4: Commit any incidental fixes (if needed)**

If any cleanup commits were needed during Step 1-3, commit them now with descriptive messages. Otherwise skip.

### Task 50: Manual end-to-end smoke test

**Files:** (manual verification)

- [ ] **Step 1: Start backend and Gotenberg**

```bash
# Terminal 1 — backend
go run ./cmd/metaldocs

# Terminal 2 — Gotenberg (with Carlito)
docker build -t metaldocs/gotenberg:local docker/gotenberg
docker run --rm --name metaldocs-gotenberg -p 3000:3000 -d metaldocs/gotenberg:local
./docker/gotenberg/verify-carlito.sh metaldocs-gotenberg
```
Expected: Backend starts on configured port. Verify script prints `OK: Carlito is installed`.

- [ ] **Step 2: Start the frontend**

```bash
cd frontend/apps/web
npm run dev
```
Expected: Vite dev server prints URL (e.g., http://localhost:4173).

- [ ] **Step 3: Enable the feature flag locally**

In the browser dev console on the app page, run:

```js
window.__METALDOCS_FEATURE_FLAGS = { MDDM_NATIVE_EXPORT: true };
window.location.reload();
```
Expected: Page reloads with the new export path active.

- [ ] **Step 4: Export a saved document as DOCX**

- Open an existing MDDM document that uses only paragraph, heading, section, field, or fieldGroup blocks (or create a small one).
- Click "Exportar DOCX".
- If you had unsaved changes, the SaveBeforeExportDialog appears. Click "Salvar e exportar".

Expected: A `.docx` file downloads. Open it in Microsoft Word or LibreOffice. Verify:
- Text content matches the editor
- Section headers have the accent background color
- Fields render with a 35/65 label/value split with the shaded label cell
- Font is Carlito (or a metric-compatible fallback) at the expected sizes

- [ ] **Step 5: Export the same document as PDF**

Note: This step exercises the backend `/render/pdf` endpoint. In Plan 1 there is no UI button yet for "Export PDF" (that's added in later plans); use the browser console to trigger it directly:

```js
const { exportPdf } = await import("/src/features/documents/mddm-editor/engine/export/export-pdf.ts");
// Build body HTML — for smoke test, use any static snippet
const html = "<p>Smoke test body</p>";
const blob = await exportPdf({ bodyHtml: html, documentId: "<your-document-id>" });
window.open(URL.createObjectURL(blob));
```
Expected: A PDF opens in a new browser tab showing the body content. Verify:
- Page size is A4
- Margins match the tokens
- Font is Carlito (confirm in a PDF viewer's font inspector)

- [ ] **Step 6: Document any issues**

If any export failed or looked wrong, file follow-up work items in the plan's tracking issue. Do NOT proceed to Plan 2 approval until smoke test passes.

- [ ] **Step 7: Commit a smoke-test record (optional)**

If issues were fixed, commit the fixes with descriptive messages. No other commit needed for this step.

---

## Self-Review

### Spec coverage

| Spec requirement (section) | Task(s) covering it |
|---|---|
| Layout IR tokens + component rules | Tasks 4, 5, 7 |
| Three-tier compatibility contract | Task 6 |
| Unit conversions for docx.js | Task 8 |
| Asset Resolution Contract (allowlist, ceilings, magic bytes, allowed MIME) | Tasks 9, 10, 11, 12 |
| Canonicalize + Migrate Pipeline | Tasks 13, 14 |
| Inline content mapper (bold/italic/underline/strike/code) | Task 15 |
| DOCX emitters — paragraph, heading, section, field, fieldGroup | Tasks 16, 17, 18, 19, 20 |
| `mddmToDocx` entry point with page margins from tokens | Tasks 21, 22 |
| `toExternalHTML` on Section, Field, FieldGroup | Tasks 23, 24, 25, 26, 27, 28, 29 |
| Print stylesheet with Carlito/Liberation/Arial stack | Task 30 |
| `exportDocx` + `exportPdf` client functions | Tasks 31, 32, 33, 34 |
| Backend `POST /render/pdf` endpoint + Gotenberg Chromium route | Tasks 35, 36, 37, 38 |
| Golden file infrastructure + first fixture | Tasks 39, 40, 41 |
| Renderer completeness gate | Task 42 |
| Feature flag `MDDM_NATIVE_EXPORT` | Task 43 |
| Export State Contract (save-before-export dialog) | Tasks 44, 46 |
| MDDMViewer read-only component | Task 45 |
| BrowserDocumentEditorView integration | Task 46 |
| Carlito Gotenberg container verification | Task 47 |
| Engine barrel exposing public surface | Tasks 3, 48 |
| Full test + build verification | Task 49 |
| Smoke test | Task 50 |

**Out of scope by design** (deferred to later plans — NOT gaps):
- Version pinning + renderer bundle registry (Plan 3)
- Shadow testing telemetry + canary rollout + decommission (Plan 4)
- Remaining 11 emitters for repeatable, repeatableItem, richBlock, dataTable*, image, bulletListItem, numberedListItem, quote, divider (Plan 2)
- Full 7-fixture golden corpus + visual parity Playwright tests (Plan 2)
- `Version.rendererPin` domain model + schema migration (Plan 3)
- Complete `toExternalHTML` for the remaining 6 MDDM blocks (Plan 2)

### Placeholder scan

No "TBD", "TODO", "implement later", or "similar to Task N" placeholders remain. Every code step includes actual code to write, every command step includes the exact command and expected output.

### Type / signature consistency

- `mddmToDocx(envelope, tokens)` signature is identical across Tasks 21, 32, 41, and 42.
- `exportDocx(envelope, tokens)` signature identical across Tasks 32 and 46.
- `exportPdf({ bodyHtml, documentId })` parameter name consistent across Tasks 33 and Task 50.
- `LayoutTokens` type imported from `../layout-ir` consistently across all emitter tasks.
- `PDFRenderer` interface on the backend (Task 36) matches the method signature added in Task 35 (`ConvertHTMLToPDF(ctx, html, css) → ([]byte, error)`).
- Emitter functions all return arrays of docx.js types (`Paragraph[]`, `Table[]`) — composed via `children.push(...out)` in `mddmToDocx`.
- Block type strings (`"section"`, `"field"`, `"fieldGroup"`, `"paragraph"`, `"heading"`) are consistent between the emitter registry (Task 21), block registry (Task 42), and `toExternalHTML` registrations (Tasks 27-29).

---

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-04-10-mddm-engine-foundation.md`. Two execution options:

**1. Subagent-Driven (recommended)** — I dispatch a fresh subagent per task, review between tasks, fast iteration.
**REQUIRED SUB-SKILL:** `superpowers:subagent-driven-development`

**2. Inline Execution** — Execute tasks in this session using executing-plans, batch execution with checkpoints.
**REQUIRED SUB-SKILL:** `superpowers:executing-plans`

Which approach?
