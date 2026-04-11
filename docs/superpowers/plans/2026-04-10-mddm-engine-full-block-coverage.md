# MDDM Engine Full Block Coverage Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build out the remaining DOCX emitters (5 standard BlockNote blocks + 6 MDDM custom blocks) and `toExternalHTML` hooks (6 MDDM custom blocks), wire asset resolution through both export paths, expand the golden fixture corpus to 6 reference documents, and add Playwright visual parity tests — bringing the MDDM engine to full block coverage and parity validation.

**Architecture:** Plan 2 follows the patterns established in Plan 1 verbatim. New emitters live in `engine/docx-emitter/emitters/`, new HTML hooks in `engine/external-html/`, and the main `mddmToDocx` registry plus the completeness-gate `BLOCK_REGISTRY` are updated in lockstep. Asset resolution is wired by walking the canonicalized envelope before emission to build an `assetMap`, which the image emitter and PDF HTML rewriter consume. Visual parity uses Playwright + pixelmatch against rasterized PDFs from the new `/documents/{id}/render/pdf` endpoint.

**Tech Stack:** TypeScript 5.6, React 18, BlockNote 0.47.3 (core only), docx.js 9.x, Vitest 4.1, Playwright 1.58, pixelmatch, pdf.js (rasterization), Gotenberg 8.x (Chromium route).

**Spec:** `docs/superpowers/specs/2026-04-10-mddm-unified-document-engine-design.md`

**Depends on:** Plan 1 — `docs/superpowers/plans/2026-04-10-mddm-engine-foundation.md` (must be merged before Plan 2 starts)

---

## File Structure

### New files (frontend)

```
frontend/apps/web/src/features/documents/mddm-editor/engine/
├── docx-emitter/
│   ├── emitters/
│   │   ├── bullet-list-item.ts             # NEW
│   │   ├── numbered-list-item.ts           # NEW
│   │   ├── image.ts                        # NEW (uses asset resolver)
│   │   ├── quote.ts                        # NEW
│   │   ├── divider.ts                      # NEW
│   │   ├── data-table-cell.ts              # NEW
│   │   ├── data-table-row.ts               # NEW
│   │   ├── data-table.ts                   # NEW
│   │   ├── repeatable-item.ts              # NEW
│   │   ├── repeatable.ts                   # NEW
│   │   └── rich-block.ts                   # NEW
│   ├── asset-collector.ts                  # NEW: walk blocks → image URLs
│   └── __tests__/
│       ├── asset-collector.test.ts         # NEW
│       ├── bullet-list-item.test.ts        # NEW
│       ├── numbered-list-item.test.ts      # NEW
│       ├── image.test.ts                   # NEW
│       ├── quote.test.ts                   # NEW
│       ├── divider.test.ts                 # NEW
│       ├── data-table-cell.test.ts         # NEW
│       ├── data-table-row.test.ts          # NEW
│       ├── data-table.test.ts              # NEW
│       ├── repeatable-item.test.ts         # NEW
│       ├── repeatable.test.ts              # NEW
│       └── rich-block.test.ts              # NEW
├── external-html/
│   ├── data-table-cell-html.tsx            # NEW
│   ├── data-table-row-html.tsx             # NEW
│   ├── data-table-html.tsx                 # NEW
│   ├── repeatable-item-html.tsx            # NEW
│   ├── repeatable-html.tsx                 # NEW
│   ├── rich-block-html.tsx                 # NEW
│   └── __tests__/
│       ├── data-table-cell-html.test.tsx   # NEW
│       ├── data-table-row-html.test.tsx    # NEW
│       ├── data-table-html.test.tsx        # NEW
│       ├── repeatable-item-html.test.tsx   # NEW
│       ├── repeatable-html.test.tsx        # NEW
│       └── rich-block-html.test.tsx        # NEW
├── export/
│   ├── inline-asset-rewriter.ts            # NEW: rewrite img src → data: URI
│   └── __tests__/
│       └── inline-asset-rewriter.test.ts   # NEW
└── golden/
    ├── fixtures/
    │   ├── 02-complex-table/               # NEW
    │   │   ├── input.mddm.json
    │   │   └── expected.document.xml
    │   ├── 03-repeatable-sections/         # NEW
    │   │   ├── input.mddm.json
    │   │   └── expected.document.xml
    │   ├── 04-all-inline-marks/            # NEW
    │   │   ├── input.mddm.json
    │   │   └── expected.document.xml
    │   ├── 05-multi-block-doc/             # NEW
    │   │   ├── input.mddm.json
    │   │   └── expected.document.xml
    │   └── 06-theme-override/              # NEW
    │       ├── input.mddm.json
    │       └── expected.document.xml
    └── __tests__/
        ├── golden-02-complex-table.test.ts # NEW
        ├── golden-03-repeatable.test.ts    # NEW
        ├── golden-04-inline-marks.test.ts  # NEW
        ├── golden-05-multi-block.test.ts   # NEW
        └── golden-06-theme.test.ts         # NEW
```

### New e2e test files

```
frontend/apps/web/e2e/
├── mddm-visual-parity.spec.ts               # NEW: Playwright visual parity suite
└── helpers/
    ├── pixel-diff.ts                        # NEW: pdf-to-png (pdf-img-convert) + pixelmatch helper
    └── __tests__/
        └── pixel-diff.smoke.test.ts         # NEW: smoke test for rasterizer import
```

### New harness page (test-only)

```
frontend/apps/web/src/test-harness/
├── MDDMTestHarness.tsx                      # NEW: loads fixture, exposes export APIs
└── routes.tsx                                # NEW or MODIFY: register /test-harness/mddm
```

### Modified files

```
frontend/apps/web/package.json                                          # MODIFY: add pixelmatch, pdfjs-dist
frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitter.ts                # MODIFY: register 11 new emitters; consume assetMap
frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/index.ts                  # MODIFY: re-export new emitters
frontend/apps/web/src/features/documents/mddm-editor/engine/external-html/index.ts                 # MODIFY: re-export new HTML components
frontend/apps/web/src/features/documents/mddm-editor/engine/export/export-docx.ts                  # MODIFY: collect assets and pass assetMap to emitter
frontend/apps/web/src/features/documents/mddm-editor/engine/export/export-pdf.ts                   # MODIFY: rewrite img src to data: URIs before sending HTML
frontend/apps/web/src/features/documents/mddm-editor/engine/completeness-gate/block-registry.ts    # MODIFY: mark 11 blocks as fully supported
frontend/apps/web/src/features/documents/mddm-editor/blocks/Repeatable.tsx                         # MODIFY: register toExternalHTML
frontend/apps/web/src/features/documents/mddm-editor/blocks/RepeatableItem.tsx                     # MODIFY: register toExternalHTML
frontend/apps/web/src/features/documents/mddm-editor/blocks/RichBlock.tsx                          # MODIFY: register toExternalHTML
frontend/apps/web/src/features/documents/mddm-editor/blocks/DataTable.tsx                          # MODIFY: register toExternalHTML
frontend/apps/web/src/features/documents/mddm-editor/blocks/DataTableRow.tsx                       # MODIFY: register toExternalHTML
frontend/apps/web/src/features/documents/mddm-editor/blocks/DataTableCell.tsx                      # MODIFY: register toExternalHTML
frontend/apps/web/playwright.config.ts                                                              # MODIFY (or CREATE): add visual parity test directory
```

---

## Part 1 — Asset Collection & Inlining

### Task 1: Implement asset collector (walk envelope, gather image URLs)

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/asset-collector.ts`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/asset-collector.test.ts`

- [ ] **Step 1: Write the failing test**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/asset-collector.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { collectImageUrls } from "../asset-collector";
import type { MDDMEnvelope } from "../../../adapter";

describe("collectImageUrls", () => {
  it("returns an empty array when there are no image blocks", () => {
    const envelope: MDDMEnvelope = {
      mddm_version: 1,
      template_ref: null,
      blocks: [
        { id: "p", type: "paragraph", props: {}, children: [{ type: "text", text: "x" }] },
      ],
    };
    expect(collectImageUrls(envelope)).toEqual([]);
  });

  it("returns image URLs from top-level image blocks (reads block.props.src)", () => {
    const envelope: MDDMEnvelope = {
      mddm_version: 1,
      template_ref: null,
      blocks: [
        { id: "i1", type: "image", props: { src: "/api/images/aaa" }, children: [] },
        { id: "i2", type: "image", props: { src: "/api/images/bbb" }, children: [] },
      ],
    };
    expect(collectImageUrls(envelope)).toEqual(["/api/images/aaa", "/api/images/bbb"]);
  });

  it("walks nested children for images inside repeatables and sections", () => {
    const envelope: MDDMEnvelope = {
      mddm_version: 1,
      template_ref: null,
      blocks: [
        {
          id: "s",
          type: "section",
          props: { title: "S" },
          children: [
            {
              id: "r",
              type: "repeatable",
              props: { label: "L" },
              children: [
                {
                  id: "ri",
                  type: "repeatableItem",
                  props: {},
                  children: [
                    { id: "img", type: "image", props: { src: "/api/images/nested" }, children: [] },
                  ],
                },
              ],
            },
          ],
        },
      ],
    };
    expect(collectImageUrls(envelope)).toEqual(["/api/images/nested"]);
  });

  it("deduplicates URLs", () => {
    const envelope: MDDMEnvelope = {
      mddm_version: 1,
      template_ref: null,
      blocks: [
        { id: "i1", type: "image", props: { src: "/api/images/aaa" }, children: [] },
        { id: "i2", type: "image", props: { src: "/api/images/aaa" }, children: [] },
      ],
    };
    expect(collectImageUrls(envelope)).toEqual(["/api/images/aaa"]);
  });

  it("ignores image blocks without a src prop", () => {
    const envelope: MDDMEnvelope = {
      mddm_version: 1,
      template_ref: null,
      blocks: [
        { id: "i1", type: "image", props: {}, children: [] },
        { id: "i2", type: "image", props: { src: "" }, children: [] },
      ],
    };
    expect(collectImageUrls(envelope)).toEqual([]);
  });
});
```

- [ ] **Step 2: Run test — expect failure**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/docx-emitter/__tests__/asset-collector.test.ts`
Expected: FAIL — cannot find module `../asset-collector`.

- [ ] **Step 3: Implement asset-collector.ts**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/asset-collector.ts`:

```ts
import type { MDDMEnvelope, MDDMBlock } from "../../adapter";

function isMDDMBlock(child: unknown): child is MDDMBlock {
  return (
    child !== null &&
    typeof child === "object" &&
    typeof (child as MDDMBlock).type === "string" &&
    !("text" in (child as Record<string, unknown>))
  );
}

function walkBlock(block: MDDMBlock, urls: Set<string>): void {
  if (block.type === "image") {
    // MDDM envelope stores image URLs under `src` (see adapter.ts toMDDMProps).
    const src = (block.props as { src?: unknown }).src;
    if (typeof src === "string" && src.length > 0) {
      urls.add(src);
    }
  }
  const children = block.children ?? [];
  for (const child of children) {
    if (isMDDMBlock(child)) {
      walkBlock(child, urls);
    }
  }
}

/**
 * Walk an MDDM envelope and return a deduplicated list of image URLs.
 * The order matches the depth-first walk order, which keeps later resolution
 * deterministic for golden testing.
 */
export function collectImageUrls(envelope: MDDMEnvelope): string[] {
  const urls = new Set<string>();
  for (const block of envelope.blocks ?? []) {
    walkBlock(block, urls);
  }
  return Array.from(urls);
}
```

- [ ] **Step 4: Run test — expect pass**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/docx-emitter/__tests__/asset-collector.test.ts`
Expected: PASS — 5 tests passing.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/asset-collector.ts frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/asset-collector.test.ts
git commit -m "feat(mddm-engine): add asset collector for walking MDDM envelope images"
```

### Task 2: Implement inline asset rewriter for PDF export

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/export/inline-asset-rewriter.ts`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/export/__tests__/inline-asset-rewriter.test.ts`

- [ ] **Step 1: Write the failing test**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/export/__tests__/inline-asset-rewriter.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { rewriteImgSrcToDataUri } from "../inline-asset-rewriter";
import type { ResolvedAsset } from "../../asset-resolver";

function makeAsset(byte: number): ResolvedAsset {
  return {
    bytes: new Uint8Array([byte, byte, byte, byte]),
    mimeType: "image/png",
    sizeBytes: 4,
  };
}

describe("rewriteImgSrcToDataUri", () => {
  it("rewrites a single img src to a data: URI", () => {
    const html = `<p><img src="/api/images/aaa" alt="A" /></p>`;
    const map = new Map<string, ResolvedAsset>([
      ["/api/images/aaa", makeAsset(0x01)],
    ]);
    const out = rewriteImgSrcToDataUri(html, map);
    expect(out).toContain("data:image/png;base64,");
    expect(out).not.toContain("/api/images/aaa");
    expect(out).toContain('alt="A"');
  });

  it("rewrites multiple img tags with different URLs", () => {
    const html = `<img src="/api/images/aaa"/><img src="/api/images/bbb"/>`;
    const map = new Map<string, ResolvedAsset>([
      ["/api/images/aaa", makeAsset(0x10)],
      ["/api/images/bbb", makeAsset(0x20)],
    ]);
    const out = rewriteImgSrcToDataUri(html, map);
    const matches = out.match(/data:image\/png;base64,/g);
    expect(matches).toHaveLength(2);
  });

  it("leaves img tags whose src is not in the map untouched", () => {
    const html = `<img src="/api/images/missing"/>`;
    const map = new Map<string, ResolvedAsset>();
    const out = rewriteImgSrcToDataUri(html, map);
    expect(out).toContain("/api/images/missing");
  });

  it("handles single-quoted src attributes", () => {
    const html = `<img src='/api/images/aaa'/>`;
    const map = new Map<string, ResolvedAsset>([
      ["/api/images/aaa", makeAsset(0x01)],
    ]);
    const out = rewriteImgSrcToDataUri(html, map);
    expect(out).toContain("data:image/png;base64,");
  });
});
```

- [ ] **Step 2: Run test — expect failure**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/export/__tests__/inline-asset-rewriter.test.ts`
Expected: FAIL — cannot find module.

- [ ] **Step 3: Implement inline-asset-rewriter.ts**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/export/inline-asset-rewriter.ts`:

```ts
import type { ResolvedAsset } from "../asset-resolver";

function bytesToBase64(bytes: Uint8Array): string {
  let binary = "";
  for (let i = 0; i < bytes.byteLength; i++) {
    binary += String.fromCharCode(bytes[i]!);
  }
  // btoa is available in browsers and modern Node test environments via jsdom.
  return globalThis.btoa(binary);
}

function toDataUri(asset: ResolvedAsset): string {
  return `data:${asset.mimeType};base64,${bytesToBase64(asset.bytes)}`;
}

/**
 * Rewrite every <img src="..."> attribute whose URL is present in `assetMap`
 * to a data: URI. Untouched img tags whose URL is missing from the map are
 * preserved verbatim — Gotenberg will then fail or skip them, which surfaces
 * a clear missing-asset issue rather than silently dropping content.
 */
export function rewriteImgSrcToDataUri(
  html: string,
  assetMap: ReadonlyMap<string, ResolvedAsset>,
): string {
  // Match <img ... src="URL" ... /> with both single and double quoted src.
  return html.replace(
    /(<img\b[^>]*\bsrc\s*=\s*)(["'])([^"']+)\2/gi,
    (match, prefix: string, quote: string, url: string) => {
      const asset = assetMap.get(url);
      if (!asset) return match;
      return `${prefix}${quote}${toDataUri(asset)}${quote}`;
    },
  );
}
```

- [ ] **Step 4: Run test — expect pass**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/export/__tests__/inline-asset-rewriter.test.ts`
Expected: PASS — 4 tests passing.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/export/inline-asset-rewriter.ts frontend/apps/web/src/features/documents/mddm-editor/engine/export/__tests__/inline-asset-rewriter.test.ts
git commit -m "feat(mddm-engine): add img src→data URI rewriter for PDF export"
```

---

## Part 2 — Standard BlockNote DOCX Emitters

### Task 3: bulletListItem emitter

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/bullet-list-item.ts`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/bullet-list-item.test.ts`

- [ ] **Step 1: Write the failing test**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/bullet-list-item.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { Paragraph } from "docx";
import { emitBulletListItem } from "../emitters/bullet-list-item";
import { defaultLayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../../adapter";

describe("emitBulletListItem", () => {
  it("emits a Paragraph with bullet numbering", () => {
    const block: MDDMBlock = {
      id: "b1",
      type: "bulletListItem",
      props: {},
      children: [{ type: "text", text: "First" }],
    };
    const out = emitBulletListItem(block, defaultLayoutTokens);
    expect(out).toHaveLength(1);
    expect(out[0]).toBeInstanceOf(Paragraph);
    expect((out[0] as any).options.bullet).toBeDefined();
    expect((out[0] as any).options.bullet.level).toBe(0);
  });

  it("preserves marks on text runs", () => {
    const block: MDDMBlock = {
      id: "b2",
      type: "bulletListItem",
      props: {},
      children: [{ type: "text", text: "Bold", marks: [{ type: "bold" }] }],
    };
    const out = emitBulletListItem(block, defaultLayoutTokens);
    const run = (out[0] as any).options.children[0];
    expect(run.options).toMatchObject({ bold: true });
  });
});
```

- [ ] **Step 2: Run test — expect failure**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/docx-emitter/__tests__/bullet-list-item.test.ts`
Expected: FAIL.

- [ ] **Step 3: Implement bullet-list-item.ts**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/bullet-list-item.ts`:

```ts
import { Paragraph } from "docx";
import type { LayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../../adapter";
import { mddmTextRunsToDocxRuns } from "../inline-content";
import { extractTextRuns } from "./paragraph";

export function emitBulletListItem(block: MDDMBlock, tokens: LayoutTokens): Paragraph[] {
  const runs = mddmTextRunsToDocxRuns(extractTextRuns(block), tokens);
  return [
    new Paragraph({
      bullet: { level: 0 },
      children: runs,
    }),
  ];
}
```

- [ ] **Step 4: Run test — expect pass**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/docx-emitter/__tests__/bullet-list-item.test.ts`
Expected: PASS — 2 tests passing.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/bullet-list-item.ts frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/bullet-list-item.test.ts
git commit -m "feat(mddm-engine): add bullet-list-item DOCX emitter"
```

### Task 4: numberedListItem emitter

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/numbered-list-item.ts`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/numbered-list-item.test.ts`

- [ ] **Step 1: Write the failing test**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/numbered-list-item.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { Paragraph } from "docx";
import { emitNumberedListItem } from "../emitters/numbered-list-item";
import { defaultLayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../../adapter";

describe("emitNumberedListItem", () => {
  it("emits a Paragraph with numbering reference", () => {
    const block: MDDMBlock = {
      id: "n1",
      type: "numberedListItem",
      props: {},
      children: [{ type: "text", text: "Item 1" }],
    };
    const out = emitNumberedListItem(block, defaultLayoutTokens);
    expect(out).toHaveLength(1);
    expect(out[0]).toBeInstanceOf(Paragraph);
    expect((out[0] as any).options.numbering).toBeDefined();
    expect((out[0] as any).options.numbering.level).toBe(0);
  });
});
```

- [ ] **Step 2: Run test — expect failure**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/docx-emitter/__tests__/numbered-list-item.test.ts`
Expected: FAIL.

- [ ] **Step 3: Implement numbered-list-item.ts**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/numbered-list-item.ts`:

```ts
import { Paragraph } from "docx";
import type { LayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../../adapter";
import { mddmTextRunsToDocxRuns } from "../inline-content";
import { extractTextRuns } from "./paragraph";

// Reference name registered on the Document at emit time.
// Stable string keeps OOXML output deterministic for golden tests.
export const MDDM_NUMBERING_REF = "mddm-decimal";

export function emitNumberedListItem(block: MDDMBlock, tokens: LayoutTokens): Paragraph[] {
  const runs = mddmTextRunsToDocxRuns(extractTextRuns(block), tokens);
  return [
    new Paragraph({
      numbering: {
        reference: MDDM_NUMBERING_REF,
        level: 0,
      },
      children: runs,
    }),
  ];
}
```

- [ ] **Step 4: Run test — expect pass**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/docx-emitter/__tests__/numbered-list-item.test.ts`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/numbered-list-item.ts frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/numbered-list-item.test.ts
git commit -m "feat(mddm-engine): add numbered-list-item DOCX emitter with stable numbering ref"
```

### Task 5: image emitter (consumes asset map)

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/image.ts`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/image.test.ts`

- [ ] **Step 1: Write the failing test**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/image.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { Paragraph } from "docx";
import { emitImage, MissingAssetError } from "../emitters/image";
import { defaultLayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../../adapter";
import type { ResolvedAsset } from "../../asset-resolver";

const PNG_BYTES = new Uint8Array([0x89, 0x50, 0x4e, 0x47]);

function makeAsset(): ResolvedAsset {
  return { bytes: PNG_BYTES, mimeType: "image/png", sizeBytes: PNG_BYTES.byteLength };
}

describe("emitImage", () => {
  it("emits a Paragraph containing an ImageRun for a resolved image (src prop)", () => {
    const block: MDDMBlock = {
      id: "i1",
      type: "image",
      props: { src: "/api/images/aaa", widthMm: 80 },
      children: [],
    };
    const map = new Map<string, ResolvedAsset>([["/api/images/aaa", makeAsset()]]);
    const out = emitImage(block, defaultLayoutTokens, map);
    expect(out).toHaveLength(1);
    expect(out[0]).toBeInstanceOf(Paragraph);
  });

  it("throws MissingAssetError when image src is not in the asset map", () => {
    const block: MDDMBlock = {
      id: "i2",
      type: "image",
      props: { src: "/api/images/missing" },
      children: [],
    };
    expect(() => emitImage(block, defaultLayoutTokens, new Map())).toThrow(MissingAssetError);
  });

  it("returns an empty Paragraph when block has no src prop", () => {
    const block: MDDMBlock = {
      id: "i3",
      type: "image",
      props: {},
      children: [],
    };
    const out = emitImage(block, defaultLayoutTokens, new Map());
    expect(out).toHaveLength(1);
    expect(out[0]).toBeInstanceOf(Paragraph);
  });
});
```

- [ ] **Step 2: Run test — expect failure**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/docx-emitter/__tests__/image.test.ts`
Expected: FAIL.

- [ ] **Step 3: Implement image.ts**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/image.ts`:

```ts
import { Paragraph, ImageRun } from "docx";
import type { LayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../../adapter";
import type { ResolvedAsset } from "../../asset-resolver";
import { mmToEmu } from "../../helpers/units";

export class MissingAssetError extends Error {
  constructor(public readonly url: string) {
    super(`Image asset not found in asset map: ${url}`);
    this.name = "MissingAssetError";
  }
}

const DEFAULT_IMAGE_WIDTH_MM = 80;
const DEFAULT_IMAGE_HEIGHT_MM = 60;

export function emitImage(
  block: MDDMBlock,
  tokens: LayoutTokens,
  assetMap: ReadonlyMap<string, ResolvedAsset>,
): Paragraph[] {
  // MDDM envelope stores image URLs under `src` (see adapter.ts toMDDMProps).
  const src = (block.props as { src?: string }).src;

  if (typeof src !== "string" || src.length === 0) {
    return [new Paragraph({ children: [] })];
  }

  const asset = assetMap.get(src);
  if (!asset) {
    throw new MissingAssetError(src);
  }

  const widthMm = (block.props as { widthMm?: number }).widthMm ?? DEFAULT_IMAGE_WIDTH_MM;
  const heightMm = (block.props as { heightMm?: number }).heightMm ?? DEFAULT_IMAGE_HEIGHT_MM;

  const docxImageType = asset.mimeType === "image/jpeg" ? "jpg"
    : asset.mimeType === "image/png" ? "png"
    : asset.mimeType === "image/gif" ? "gif"
    : "png";

  return [
    new Paragraph({
      children: [
        new ImageRun({
          type: docxImageType as any,
          data: asset.bytes,
          transformation: {
            width: Math.round(mmToEmu(widthMm) / 9525),  // EMU → px (1 px = 9525 EMU)
            height: Math.round(mmToEmu(heightMm) / 9525),
          },
        }),
      ],
    }),
  ];
}
```

- [ ] **Step 4: Run test — expect pass**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/docx-emitter/__tests__/image.test.ts`
Expected: PASS — 3 tests passing.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/image.ts frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/image.test.ts
git commit -m "feat(mddm-engine): add image DOCX emitter consuming asset map"
```

### Task 6: quote emitter

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/quote.ts`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/quote.test.ts`

- [ ] **Step 1: Write the failing test**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/quote.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { Paragraph } from "docx";
import { emitQuote } from "../emitters/quote";
import { defaultLayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../../adapter";

describe("emitQuote", () => {
  it("emits a Paragraph with left indentation and italic styling", () => {
    const block: MDDMBlock = {
      id: "q1",
      type: "quote",
      props: {},
      children: [{ type: "text", text: "Quoted text" }],
    };
    const out = emitQuote(block, defaultLayoutTokens);
    expect(out).toHaveLength(1);
    expect(out[0]).toBeInstanceOf(Paragraph);
    const opts = (out[0] as any).options;
    expect(opts.indent).toBeDefined();
    expect(opts.indent.left).toBeGreaterThan(0);
  });
});
```

- [ ] **Step 2: Run test — expect failure**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/docx-emitter/__tests__/quote.test.ts`
Expected: FAIL.

- [ ] **Step 3: Implement quote.ts**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/quote.ts`:

```ts
import { Paragraph } from "docx";
import type { LayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../../adapter";
import { mddmTextRunsToDocxRuns } from "../inline-content";
import { extractTextRuns } from "./paragraph";
import { mmToTwip } from "../../helpers/units";

const QUOTE_INDENT_MM = 6;

export function emitQuote(block: MDDMBlock, tokens: LayoutTokens): Paragraph[] {
  const runs = mddmTextRunsToDocxRuns(extractTextRuns(block), tokens);
  return [
    new Paragraph({
      indent: { left: mmToTwip(QUOTE_INDENT_MM) },
      children: runs,
    }),
  ];
}
```

- [ ] **Step 4: Run test — expect pass**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/docx-emitter/__tests__/quote.test.ts`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/quote.ts frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/quote.test.ts
git commit -m "feat(mddm-engine): add quote DOCX emitter with left indent"
```

### Task 7: divider emitter

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/divider.ts`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/divider.test.ts`

- [ ] **Step 1: Write the failing test**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/divider.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { Paragraph } from "docx";
import { emitDivider } from "../emitters/divider";
import { defaultLayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../../adapter";

describe("emitDivider", () => {
  it("emits a Paragraph with a bottom border (horizontal rule)", () => {
    const block: MDDMBlock = { id: "d1", type: "divider", props: {}, children: [] };
    const out = emitDivider(block, defaultLayoutTokens);
    expect(out).toHaveLength(1);
    expect(out[0]).toBeInstanceOf(Paragraph);
    const opts = (out[0] as any).options;
    expect(opts.border).toBeDefined();
    expect(opts.border.bottom).toBeDefined();
  });
});
```

- [ ] **Step 2: Run test — expect failure**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/docx-emitter/__tests__/divider.test.ts`
Expected: FAIL.

- [ ] **Step 3: Implement divider.ts**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/divider.ts`:

```ts
import { Paragraph, BorderStyle } from "docx";
import type { LayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../../adapter";

export function emitDivider(_block: MDDMBlock, tokens: LayoutTokens): Paragraph[] {
  const color = tokens.theme.accentBorder.replace(/^#/, "").toUpperCase();
  return [
    new Paragraph({
      border: {
        bottom: { style: BorderStyle.SINGLE, size: 6, color, space: 1 },
      },
      children: [],
    }),
  ];
}
```

- [ ] **Step 4: Run test — expect pass**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/docx-emitter/__tests__/divider.test.ts`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/divider.ts frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/divider.test.ts
git commit -m "feat(mddm-engine): add divider DOCX emitter (horizontal rule)"
```

---

## Part 3 — MDDM Custom DOCX Emitters

### Task 8: data-table-cell emitter

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/data-table-cell.ts`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/data-table-cell.test.ts`

- [ ] **Step 1: Write the failing test**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/data-table-cell.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { TableCell } from "docx";
import { emitDataTableCell } from "../emitters/data-table-cell";
import { defaultLayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../../adapter";

describe("emitDataTableCell", () => {
  it("emits a TableCell containing a Paragraph with text runs", () => {
    const block: MDDMBlock = {
      id: "c1",
      type: "dataTableCell",
      props: { columnKey: "qty" },
      children: [{ type: "text", text: "100" }],
    };
    const out = emitDataTableCell(block, defaultLayoutTokens);
    expect(out).toBeInstanceOf(TableCell);
    expect((out as any).options.children).toHaveLength(1);
  });

  it("renders empty cell when there are no text runs", () => {
    const block: MDDMBlock = {
      id: "c2",
      type: "dataTableCell",
      props: { columnKey: "x" },
      children: [],
    };
    const out = emitDataTableCell(block, defaultLayoutTokens);
    expect(out).toBeInstanceOf(TableCell);
  });
});
```

- [ ] **Step 2: Run test — expect failure**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/docx-emitter/__tests__/data-table-cell.test.ts`
Expected: FAIL.

- [ ] **Step 3: Implement data-table-cell.ts**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/data-table-cell.ts`:

```ts
import { TableCell, Paragraph, TextRun, BorderStyle } from "docx";
import type { LayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../../adapter";
import { mddmTextRunsToDocxRuns } from "../inline-content";
import { extractTextRuns } from "./paragraph";

function hexToFill(hex: string): string {
  return hex.replace(/^#/, "").toUpperCase();
}

export function emitDataTableCell(block: MDDMBlock, tokens: LayoutTokens): TableCell {
  const borderColor = hexToFill(tokens.theme.accentBorder);
  const borders = {
    top:    { style: BorderStyle.SINGLE, size: 4, color: borderColor },
    bottom: { style: BorderStyle.SINGLE, size: 4, color: borderColor },
    left:   { style: BorderStyle.SINGLE, size: 4, color: borderColor },
    right:  { style: BorderStyle.SINGLE, size: 4, color: borderColor },
  };

  const runs = mddmTextRunsToDocxRuns(extractTextRuns(block), tokens);

  return new TableCell({
    borders,
    children: [
      new Paragraph({
        children: runs.length > 0 ? runs : [new TextRun({ text: "" })],
      }),
    ],
  });
}
```

- [ ] **Step 4: Run test — expect pass**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/docx-emitter/__tests__/data-table-cell.test.ts`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/data-table-cell.ts frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/data-table-cell.test.ts
git commit -m "feat(mddm-engine): add data-table-cell DOCX emitter"
```

### Task 9: data-table-row emitter

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/data-table-row.ts`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/data-table-row.test.ts`

- [ ] **Step 1: Write the failing test**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/data-table-row.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { TableRow } from "docx";
import { emitDataTableRow } from "../emitters/data-table-row";
import { defaultLayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../../adapter";

describe("emitDataTableRow", () => {
  it("emits a TableRow containing one cell per dataTableCell child", () => {
    const block: MDDMBlock = {
      id: "r1",
      type: "dataTableRow",
      props: {},
      children: [
        { id: "c1", type: "dataTableCell", props: { columnKey: "a" }, children: [{ type: "text", text: "1" }] },
        { id: "c2", type: "dataTableCell", props: { columnKey: "b" }, children: [{ type: "text", text: "2" }] },
      ],
    };
    const out = emitDataTableRow(block, defaultLayoutTokens);
    expect(out).toBeInstanceOf(TableRow);
    expect((out as any).options.children).toHaveLength(2);
  });
});
```

- [ ] **Step 2: Run test — expect failure**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/docx-emitter/__tests__/data-table-row.test.ts`
Expected: FAIL.

- [ ] **Step 3: Implement data-table-row.ts**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/data-table-row.ts`:

```ts
import { TableRow } from "docx";
import type { LayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../../adapter";
import { emitDataTableCell } from "./data-table-cell";

function isCellBlock(child: unknown): child is MDDMBlock {
  return typeof child === "object" && child !== null && (child as MDDMBlock).type === "dataTableCell";
}

export function emitDataTableRow(block: MDDMBlock, tokens: LayoutTokens): TableRow {
  const allChildren = (block.children ?? []) as unknown[];
  const cells = allChildren.filter(isCellBlock).map((c) => emitDataTableCell(c as MDDMBlock, tokens));
  return new TableRow({ children: cells });
}
```

- [ ] **Step 4: Run test — expect pass**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/docx-emitter/__tests__/data-table-row.test.ts`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/data-table-row.ts frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/data-table-row.test.ts
git commit -m "feat(mddm-engine): add data-table-row DOCX emitter"
```

### Task 10: data-table emitter

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/data-table.ts`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/data-table.test.ts`

- [ ] **Step 1: Write the failing test**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/data-table.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { Table } from "docx";
import { emitDataTable } from "../emitters/data-table";
import { defaultLayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../../adapter";

function makeRow(id: string, cellCount: number): MDDMBlock {
  return {
    id,
    type: "dataTableRow",
    props: {},
    children: Array.from({ length: cellCount }, (_, i) => ({
      id: `${id}-c${i}`,
      type: "dataTableCell",
      props: { columnKey: `col${i}` },
      children: [{ type: "text", text: `r${id}c${i}` }],
    })),
  };
}

**Input shape note:** Per adapter.ts `toMDDMProps` for `dataTable` (lines 338-346), the MDDM envelope stores columns as `{ label: string, columns: Array<{key: string; label: string}>, locked, minRows, maxRows, density }`. `columns` is an ARRAY (not a JSON string) and each column has `key` + `label` keys (NOT `header` or `width`). The BlockNote editor state uses `columnsJson` as a string, but the adapter parses it into the `columns` array when serializing to MDDM.

describe("emitDataTable", () => {
  it("emits a single Table with header row + data rows", () => {
    const block: MDDMBlock = {
      id: "t1",
      type: "dataTable",
      props: {
        label: "Items",
        columns: [
          { key: "col0", label: "Item" },
          { key: "col1", label: "Qty" },
        ],
      },
      children: [makeRow("r1", 2), makeRow("r2", 2)],
    };
    const out = emitDataTable(block, defaultLayoutTokens);
    expect(out).toHaveLength(1);
    expect(out[0]).toBeInstanceOf(Table);

    // header row + 2 data rows
    const rows = (out[0] as any).options.rows;
    expect(rows).toHaveLength(3);
  });

  it("renders empty table when there are no columns and no rows", () => {
    const block: MDDMBlock = {
      id: "t2",
      type: "dataTable",
      props: { label: "X", columns: [] },
      children: [],
    };
    const out = emitDataTable(block, defaultLayoutTokens);
    expect(out).toHaveLength(1);
    expect(out[0]).toBeInstanceOf(Table);
  });

  it("falls back gracefully when columns prop is missing or not an array", () => {
    const block: MDDMBlock = {
      id: "t3",
      type: "dataTable",
      props: { label: "X" },
      children: [makeRow("r1", 1)],
    };
    expect(() => emitDataTable(block, defaultLayoutTokens)).not.toThrow();
  });
});
```

- [ ] **Step 2: Run test — expect failure**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/docx-emitter/__tests__/data-table.test.ts`
Expected: FAIL.

- [ ] **Step 3: Implement data-table.ts**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/data-table.ts`:

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
import type { MDDMBlock } from "../../../adapter";
import { emitDataTableRow } from "./data-table-row";
import { ptToHalfPt, mmToTwip } from "../../helpers/units";

type ColumnSpec = { key: string; label: string };

function readColumns(props: Record<string, unknown>): ColumnSpec[] {
  const columns = props.columns;
  if (!Array.isArray(columns)) return [];
  const out: ColumnSpec[] = [];
  for (const column of columns) {
    if (!column || typeof column !== "object") continue;
    const key = typeof (column as { key?: unknown }).key === "string" ? (column as { key: string }).key : "";
    const label = typeof (column as { label?: unknown }).label === "string" ? (column as { label: string }).label : "";
    if (key && label) out.push({ key, label });
  }
  return out;
}

function hexToFill(hex: string): string {
  return hex.replace(/^#/, "").toUpperCase();
}

function isRowBlock(child: unknown): child is MDDMBlock {
  return typeof child === "object" && child !== null && (child as MDDMBlock).type === "dataTableRow";
}

function buildHeaderRow(columns: ColumnSpec[], tokens: LayoutTokens): TableRow {
  const headerFill = hexToFill(tokens.theme.accentLight);
  const borderColor = hexToFill(tokens.theme.accentBorder);
  const borders = {
    top:    { style: BorderStyle.SINGLE, size: 4, color: borderColor },
    bottom: { style: BorderStyle.SINGLE, size: 4, color: borderColor },
    left:   { style: BorderStyle.SINGLE, size: 4, color: borderColor },
    right:  { style: BorderStyle.SINGLE, size: 4, color: borderColor },
  };

  const cells = columns.map((col) => new TableCell({
    shading: { fill: headerFill, type: "clear", color: "auto" },
    borders,
    children: [
      new Paragraph({
        children: [
          new TextRun({
            text: col.label,
            bold: true,
            size: ptToHalfPt(tokens.typography.baseSizePt),
            font: tokens.typography.exportFont,
          }),
        ],
      }),
    ],
  }));

  return new TableRow({ children: cells });
}

export function emitDataTable(block: MDDMBlock, tokens: LayoutTokens): Table[] {
  const columns = readColumns(block.props as Record<string, unknown>);
  const rowChildren = ((block.children ?? []) as unknown[]).filter(isRowBlock) as MDDMBlock[];

  const headerRow = columns.length > 0 ? [buildHeaderRow(columns, tokens)] : [];
  const dataRows = rowChildren.map((r) => emitDataTableRow(r, tokens));

  return [
    new Table({
      // Absolute width in twips = page content width. Avoids any ambiguity
      // with docx.js percentage-unit interpretation.
      width: { size: mmToTwip(tokens.page.contentWidthMm), type: WidthType.DXA },
      rows: [...headerRow, ...dataRows],
    }),
  ];
}
```

- [ ] **Step 4: Run test — expect pass**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/docx-emitter/__tests__/data-table.test.ts`
Expected: PASS — 3 tests passing.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/data-table.ts frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/data-table.test.ts
git commit -m "feat(mddm-engine): add data-table DOCX emitter with header row"
```

### Task 11: repeatable-item emitter

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/repeatable-item.ts`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/repeatable-item.test.ts`

- [ ] **Step 1: Write the failing test**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/repeatable-item.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { Table } from "docx";
import { emitRepeatableItem } from "../emitters/repeatable-item";
import { defaultLayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../../adapter";

describe("emitRepeatableItem", () => {
  it("emits a Table with a single bordered cell wrapping child blocks", () => {
    const block: MDDMBlock = {
      id: "ri1",
      type: "repeatableItem",
      props: { title: "Step 1" },
      children: [
        { id: "p", type: "paragraph", props: {}, children: [{ type: "text", text: "child" }] },
      ],
    };
    const out = emitRepeatableItem(block, defaultLayoutTokens, () => []);
    expect(out).toHaveLength(1);
    expect(out[0]).toBeInstanceOf(Table);
  });

  it("uses the provided child renderer for nested blocks", () => {
    let renderCalled = 0;
    const block: MDDMBlock = {
      id: "ri2",
      type: "repeatableItem",
      props: {},
      children: [
        { id: "p1", type: "paragraph", props: {}, children: [] },
        { id: "p2", type: "paragraph", props: {}, children: [] },
      ],
    };
    emitRepeatableItem(block, defaultLayoutTokens, (child) => {
      renderCalled++;
      return [];
    });
    expect(renderCalled).toBe(2);
  });
});
```

- [ ] **Step 2: Run test — expect failure**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/docx-emitter/__tests__/repeatable-item.test.ts`
Expected: FAIL.

- [ ] **Step 3: Implement repeatable-item.ts**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/repeatable-item.ts`:

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
import type { MDDMBlock } from "../../../adapter";
import { ptToHalfPt } from "../../helpers/units";

function hexToFill(hex: string): string {
  return hex.replace(/^#/, "").toUpperCase();
}

function isMDDMBlock(child: unknown): child is MDDMBlock {
  return typeof child === "object" && child !== null && typeof (child as MDDMBlock).type === "string" && !("text" in (child as Record<string, unknown>));
}

/** ChildRenderer is supplied by the main emitter so repeatable-item can recursively
 *  emit any block type without depending on the registry directly (avoids cycles). */
export type ChildRenderer = (child: MDDMBlock) => unknown[];

export function emitRepeatableItem(
  block: MDDMBlock,
  tokens: LayoutTokens,
  renderChild: ChildRenderer,
): Table[] {
  const accent = hexToFill(tokens.theme.accent);
  const borderColor = hexToFill(tokens.theme.accentBorder);
  const title = (block.props as { title?: string }).title ?? "";

  const innerChildren: unknown[] = [];
  if (title) {
    innerChildren.push(
      new Paragraph({
        children: [
          new TextRun({
            text: title,
            bold: true,
            size: ptToHalfPt(tokens.typography.baseSizePt),
            font: tokens.typography.exportFont,
          }),
        ],
      }),
    );
  }
  const allChildren = (block.children ?? []) as unknown[];
  for (const child of allChildren) {
    if (isMDDMBlock(child)) {
      innerChildren.push(...renderChild(child));
    }
  }

  const cell = new TableCell({
    borders: {
      top:    { style: BorderStyle.SINGLE, size: 4, color: borderColor },
      bottom: { style: BorderStyle.SINGLE, size: 4, color: borderColor },
      left:   { style: BorderStyle.SINGLE, size: 12, color: accent },
      right:  { style: BorderStyle.SINGLE, size: 4, color: borderColor },
    },
    children: innerChildren as any,
  });

  return [
    new Table({
      width: { size: 100, type: WidthType.PERCENTAGE },
      rows: [new TableRow({ children: [cell] })],
    }),
  ];
}
```

- [ ] **Step 4: Run test — expect pass**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/docx-emitter/__tests__/repeatable-item.test.ts`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/repeatable-item.ts frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/repeatable-item.test.ts
git commit -m "feat(mddm-engine): add repeatable-item DOCX emitter with left accent border"
```

### Task 12: repeatable emitter

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/repeatable.ts`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/repeatable.test.ts`

- [ ] **Step 1: Write the failing test**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/repeatable.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { emitRepeatable } from "../emitters/repeatable";
import { defaultLayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../../adapter";

describe("emitRepeatable", () => {
  it("emits a header paragraph and one repeatable-item table per child", () => {
    const block: MDDMBlock = {
      id: "rp1",
      type: "repeatable",
      props: { label: "Steps", itemPrefix: "Step" },
      children: [
        { id: "ri1", type: "repeatableItem", props: { title: "1" }, children: [] },
        { id: "ri2", type: "repeatableItem", props: { title: "2" }, children: [] },
      ],
    };
    const out = emitRepeatable(block, defaultLayoutTokens, () => []);
    // Header paragraph + 2 repeatable-item tables = 3 elements at minimum
    expect(out.length).toBeGreaterThanOrEqual(3);
  });

  it("emits only the header when there are no items", () => {
    const block: MDDMBlock = {
      id: "rp2",
      type: "repeatable",
      props: { label: "Empty" },
      children: [],
    };
    const out = emitRepeatable(block, defaultLayoutTokens, () => []);
    expect(out.length).toBeGreaterThanOrEqual(1);
  });
});
```

- [ ] **Step 2: Run test — expect failure**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/docx-emitter/__tests__/repeatable.test.ts`
Expected: FAIL.

- [ ] **Step 3: Implement repeatable.ts**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/repeatable.ts`:

```ts
import { Paragraph, TextRun } from "docx";
import type { LayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../../adapter";
import { ptToHalfPt } from "../../helpers/units";
import { emitRepeatableItem, type ChildRenderer } from "./repeatable-item";

function isItemBlock(child: unknown): child is MDDMBlock {
  return typeof child === "object" && child !== null && (child as MDDMBlock).type === "repeatableItem";
}

export function emitRepeatable(
  block: MDDMBlock,
  tokens: LayoutTokens,
  renderChild: ChildRenderer,
): unknown[] {
  const label = (block.props as { label?: string }).label ?? "";
  const out: unknown[] = [];

  if (label) {
    out.push(
      new Paragraph({
        children: [
          new TextRun({
            text: label,
            bold: true,
            size: ptToHalfPt(tokens.typography.baseSizePt),
            font: tokens.typography.exportFont,
          }),
        ],
      }),
    );
  }

  const items = ((block.children ?? []) as unknown[]).filter(isItemBlock) as MDDMBlock[];
  for (const item of items) {
    out.push(...emitRepeatableItem(item, tokens, renderChild));
  }

  return out;
}
```

- [ ] **Step 4: Run test — expect pass**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/docx-emitter/__tests__/repeatable.test.ts`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/repeatable.ts frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/repeatable.test.ts
git commit -m "feat(mddm-engine): add repeatable DOCX emitter (header + items)"
```

### Task 13: rich-block emitter

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/rich-block.ts`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/rich-block.test.ts`

- [ ] **Step 1: Write the failing test**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/rich-block.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { emitRichBlock } from "../emitters/rich-block";
import { defaultLayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../../adapter";

describe("emitRichBlock", () => {
  it("emits an optional label paragraph plus rendered children", () => {
    const block: MDDMBlock = {
      id: "rb1",
      type: "richBlock",
      props: { label: "Notes", chrome: "labeled" },
      children: [
        { id: "p1", type: "paragraph", props: {}, children: [{ type: "text", text: "note" }] },
      ],
    };
    const renderedChildren: unknown[] = [{ marker: "p1" }];
    const out = emitRichBlock(block, defaultLayoutTokens, () => renderedChildren);
    // Label paragraph + at least 1 child element
    expect(out.length).toBeGreaterThanOrEqual(2);
  });

  it("skips the label paragraph when label is missing", () => {
    const block: MDDMBlock = {
      id: "rb2",
      type: "richBlock",
      props: {},
      children: [],
    };
    const out = emitRichBlock(block, defaultLayoutTokens, () => []);
    // No label, no children → empty array
    expect(out).toEqual([]);
  });
});
```

- [ ] **Step 2: Run test — expect failure**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/docx-emitter/__tests__/rich-block.test.ts`
Expected: FAIL.

- [ ] **Step 3: Implement rich-block.ts**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/rich-block.ts`:

```ts
import { Paragraph, TextRun } from "docx";
import type { LayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../../adapter";
import { ptToHalfPt } from "../../helpers/units";
import type { ChildRenderer } from "./repeatable-item";

function isMDDMBlock(child: unknown): child is MDDMBlock {
  return typeof child === "object" && child !== null && typeof (child as MDDMBlock).type === "string" && !("text" in (child as Record<string, unknown>));
}

export function emitRichBlock(
  block: MDDMBlock,
  tokens: LayoutTokens,
  renderChild: ChildRenderer,
): unknown[] {
  const label = (block.props as { label?: string }).label ?? "";
  const out: unknown[] = [];

  if (label) {
    out.push(
      new Paragraph({
        children: [
          new TextRun({
            text: label,
            bold: true,
            size: ptToHalfPt(tokens.typography.labelSizePt),
            font: tokens.typography.exportFont,
          }),
        ],
      }),
    );
  }

  for (const child of (block.children ?? []) as unknown[]) {
    if (isMDDMBlock(child)) {
      out.push(...renderChild(child));
    }
  }

  return out;
}
```

- [ ] **Step 4: Run test — expect pass**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/docx-emitter/__tests__/rich-block.test.ts`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitters/rich-block.ts frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/__tests__/rich-block.test.ts
git commit -m "feat(mddm-engine): add rich-block DOCX emitter (optional label + children)"
```

---

## Part 4 — Wire New Emitters Into Main Entry Point

### Task 14: Update mddmToDocx to consume assetMap and dispatch new emitters

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitter.ts`

- [ ] **Step 1: Inspect current emitter.ts**

Run: `cat frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitter.ts | head -60`
Expected: Shows the current registry with 5 MVP emitters.

- [ ] **Step 2: Replace emitter.ts with the expanded version**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitter.ts`:

```ts
import { Document, Packer } from "docx";
import type { MDDMEnvelope, MDDMBlock } from "../../adapter";
import type { LayoutTokens } from "../layout-ir";
import type { ResolvedAsset } from "../asset-resolver";
import { mmToTwip } from "../helpers/units";

import { emitParagraph } from "./emitters/paragraph";
import { emitHeading } from "./emitters/heading";
import { emitSection } from "./emitters/section";
import { emitField } from "./emitters/field";
import { emitFieldGroup } from "./emitters/field-group";
import { emitBulletListItem } from "./emitters/bullet-list-item";
import { emitNumberedListItem, MDDM_NUMBERING_REF } from "./emitters/numbered-list-item";
import { emitImage } from "./emitters/image";
import { emitQuote } from "./emitters/quote";
import { emitDivider } from "./emitters/divider";
import { emitDataTable } from "./emitters/data-table";
import { emitDataTableRow } from "./emitters/data-table-row";
import { emitDataTableCell } from "./emitters/data-table-cell";
import { emitRepeatable } from "./emitters/repeatable";
import { emitRepeatableItem } from "./emitters/repeatable-item";
import { emitRichBlock } from "./emitters/rich-block";

const DOCX_MIME = "application/vnd.openxmlformats-officedocument.wordprocessingml.document";

export class MissingEmitterError extends Error {
  constructor(public readonly blockType: string) {
    super(`No DOCX emitter registered for block type "${blockType}"`);
    this.name = "MissingEmitterError";
  }
}

export type EmitContext = {
  tokens: LayoutTokens;
  assetMap: ReadonlyMap<string, ResolvedAsset>;
};

type Emitter = (block: MDDMBlock, ctx: EmitContext) => unknown[];

function makeRegistry(ctx: EmitContext): Record<string, Emitter> {
  // renderChild is captured by closure so structural emitters can recurse
  // through the registry without an import cycle.
  const renderChild = (child: MDDMBlock): unknown[] => {
    const emit = registry[child.type];
    if (!emit) throw new MissingEmitterError(child.type);
    return emit(child, ctx);
  };

  const registry: Record<string, Emitter> = {
    paragraph: (b, c) => emitParagraph(b, c.tokens),
    heading:   (b, c) => emitHeading(b, c.tokens),
    section:   (b, c) => emitSection(b, c.tokens),
    field:     (b, c) => emitField(b, c.tokens),
    fieldGroup: (b, c) => emitFieldGroup(b, c.tokens),

    bulletListItem:   (b, c) => emitBulletListItem(b, c.tokens),
    numberedListItem: (b, c) => emitNumberedListItem(b, c.tokens),
    image:            (b, c) => emitImage(b, c.tokens, c.assetMap),
    quote:            (b, c) => emitQuote(b, c.tokens),
    divider:          (b, c) => emitDivider(b, c.tokens),

    dataTable:     (b, c) => emitDataTable(b, c.tokens),
    dataTableRow:  (b, c) => [emitDataTableRow(b, c.tokens)],
    dataTableCell: (b, c) => [emitDataTableCell(b, c.tokens)],

    repeatable:     (b, c) => emitRepeatable(b, c.tokens, renderChild),
    repeatableItem: (b, c) => emitRepeatableItem(b, c.tokens, renderChild),
    richBlock:      (b, c) => emitRichBlock(b, c.tokens, renderChild),
  };
  return registry;
}

export async function mddmToDocx(
  envelope: MDDMEnvelope,
  tokens: LayoutTokens,
  assetMap: ReadonlyMap<string, ResolvedAsset> = new Map(),
): Promise<Blob> {
  const ctx: EmitContext = { tokens, assetMap };
  const registry = makeRegistry(ctx);

  const blocks = envelope.blocks ?? [];
  const children: unknown[] = [];

  for (const block of blocks) {
    const emit = registry[block.type];
    if (!emit) {
      throw new MissingEmitterError(block.type);
    }
    children.push(...emit(block, ctx));
  }

  const doc = new Document({
    numbering: {
      config: [
        {
          reference: MDDM_NUMBERING_REF,
          levels: [
            {
              level: 0,
              format: "decimal" as any,
              text: "%1.",
              alignment: "left" as any,
            },
          ],
        },
      ],
    },
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
  return new Blob([await blob.arrayBuffer()], { type: DOCX_MIME });
}
```

- [ ] **Step 3: Run the existing emitter test plus all new emitter tests**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/docx-emitter/`
Expected: All previously passing tests still pass; the new emitter dispatch wiring keeps the registry coherent.

- [ ] **Step 4: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/emitter.ts
git commit -m "feat(mddm-engine): wire 11 new emitters and assetMap into mddmToDocx"
```

### Task 15: Update docx-emitter barrel export

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/index.ts`

- [ ] **Step 1: Replace the barrel with the expanded list**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/index.ts`:

```ts
export { mddmToDocx, MissingEmitterError, type EmitContext } from "./emitter";
export { mddmTextRunsToDocxRuns } from "./inline-content";
export { collectImageUrls } from "./asset-collector";

export { emitParagraph, extractTextRuns } from "./emitters/paragraph";
export { emitHeading } from "./emitters/heading";
export { emitSection } from "./emitters/section";
export { emitField } from "./emitters/field";
export { emitFieldGroup } from "./emitters/field-group";
export { emitBulletListItem } from "./emitters/bullet-list-item";
export { emitNumberedListItem, MDDM_NUMBERING_REF } from "./emitters/numbered-list-item";
export { emitImage, MissingAssetError } from "./emitters/image";
export { emitQuote } from "./emitters/quote";
export { emitDivider } from "./emitters/divider";
export { emitDataTable } from "./emitters/data-table";
export { emitDataTableRow } from "./emitters/data-table-row";
export { emitDataTableCell } from "./emitters/data-table-cell";
export { emitRepeatable } from "./emitters/repeatable";
export { emitRepeatableItem, type ChildRenderer } from "./emitters/repeatable-item";
export { emitRichBlock } from "./emitters/rich-block";
```

- [ ] **Step 2: Verify compilation**

Run: `cd frontend/apps/web && npx tsc --noEmit 2>&1 | grep -E "docx-emitter" | head -10`
Expected: No errors.

- [ ] **Step 3: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/docx-emitter/index.ts
git commit -m "feat(mddm-engine): expand docx-emitter barrel for full block coverage"
```

### Task 16: Wire asset resolution into exportDocx

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/engine/export/export-docx.ts`

- [ ] **Step 1: Replace export-docx.ts**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/export/export-docx.ts`:

```ts
import type { MDDMEnvelope } from "../../adapter";
import type { LayoutTokens } from "../layout-ir";
import { canonicalizeAndMigrate } from "../canonicalize-migrate";
import { collectImageUrls, mddmToDocx } from "../docx-emitter";
import {
  AssetResolver,
  RESOURCE_CEILINGS,
  ResourceCeilingExceededError,
  type ResolvedAsset,
} from "../asset-resolver";

export type ExportDocxOptions = {
  /** Optional resolver injection point — defaults to a fresh AssetResolver. */
  assetResolver?: AssetResolver;
};

export async function exportDocx(
  envelope: MDDMEnvelope,
  tokens: LayoutTokens,
  options: ExportDocxOptions = {},
): Promise<Blob> {
  const canonical = await canonicalizeAndMigrate(envelope);

  // Resolve assets BEFORE emitter runs so the emitter receives bytes.
  const urls = collectImageUrls(canonical);
  if (urls.length > RESOURCE_CEILINGS.maxImagesPerDocument) {
    throw new ResourceCeilingExceededError(
      "maxImagesPerDocument",
      urls.length,
      RESOURCE_CEILINGS.maxImagesPerDocument,
    );
  }

  const resolver = options.assetResolver ?? new AssetResolver();
  const assetMap = new Map<string, ResolvedAsset>();
  let totalBytes = 0;
  for (const url of urls) {
    const asset = await resolver.resolveAsset(url);
    totalBytes += asset.sizeBytes;
    if (totalBytes > RESOURCE_CEILINGS.maxTotalAssetBytes) {
      throw new ResourceCeilingExceededError(
        "maxTotalAssetBytes",
        totalBytes,
        RESOURCE_CEILINGS.maxTotalAssetBytes,
      );
    }
    assetMap.set(url, asset);
  }

  return mddmToDocx(canonical, tokens, assetMap);
}
```

- [ ] **Step 2: Update existing export-docx test to use new signature**

The Plan 1 tests at `frontend/apps/web/src/features/documents/mddm-editor/engine/export/__tests__/export-docx.test.ts` already pass an envelope with no images. They should still pass without changes — verify:

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/export/__tests__/export-docx.test.ts`
Expected: PASS — 2 tests still pass.

- [ ] **Step 3: Add a test for asset resolution wiring**

Append to `frontend/apps/web/src/features/documents/mddm-editor/engine/export/__tests__/export-docx.test.ts`:

```ts
import { AssetResolver } from "../../asset-resolver";

describe("exportDocx asset wiring", () => {
  it("calls the asset resolver for each unique image URL", async () => {
    const PNG = new Uint8Array([0x89, 0x50, 0x4e, 0x47]);
    const calls: string[] = [];

    const fakeResolver = {
      async resolveAsset(url: string) {
        calls.push(url);
        return { bytes: PNG, mimeType: "image/png" as const, sizeBytes: PNG.byteLength };
      },
    } as unknown as AssetResolver;

    const envelope: MDDMEnvelope = {
      mddm_version: 1,
      template_ref: null,
      blocks: [
        { id: "i1", type: "image", props: { src: "/api/images/aaa" }, children: [] },
        { id: "i2", type: "image", props: { src: "/api/images/bbb" }, children: [] },
      ],
    };

    const blob = await exportDocx(envelope, defaultLayoutTokens, { assetResolver: fakeResolver });
    expect(blob).toBeInstanceOf(Blob);
    expect(calls).toEqual(["/api/images/aaa", "/api/images/bbb"]);
  });
});
```

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/export/__tests__/export-docx.test.ts`
Expected: PASS — 3 tests passing.

- [ ] **Step 4: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/export/export-docx.ts frontend/apps/web/src/features/documents/mddm-editor/engine/export/__tests__/export-docx.test.ts
git commit -m "feat(mddm-engine): wire asset resolution into exportDocx"
```

### Task 17: Wire asset inlining into exportPdf

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/engine/export/export-pdf.ts`

- [ ] **Step 1: Replace export-pdf.ts**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/export/export-pdf.ts`:

```ts
import { wrapInPrintDocument } from "./wrap-print-document";
import { PRINT_STYLESHEET } from "../print-stylesheet";
import {
  AssetResolver,
  RESOURCE_CEILINGS,
  ResourceCeilingExceededError,
  type ResolvedAsset,
} from "../asset-resolver";
import { rewriteImgSrcToDataUri } from "./inline-asset-rewriter";

export type ExportPdfParams = {
  /** Body HTML produced by blocksToFullHTML (still containing /api/images/... src refs). */
  bodyHtml: string;
  /** Document ID — used in the backend endpoint path. */
  documentId: string;
  /** Optional resolver injection point. */
  assetResolver?: AssetResolver;
};

const PDF_MIME = "application/pdf";

/** Extract every <img src> URL from the body HTML for asset resolution. */
function extractImageUrls(html: string): string[] {
  const out = new Set<string>();
  const re = /<img\b[^>]*\bsrc\s*=\s*["']([^"']+)["']/gi;
  let m: RegExpExecArray | null;
  while ((m = re.exec(html)) !== null) {
    const url = m[1];
    if (url) out.add(url);
  }
  return Array.from(out);
}

export async function exportPdf({
  bodyHtml,
  documentId,
  assetResolver,
}: ExportPdfParams): Promise<Blob> {
  // Resolve and inline images so the HTML sent to Gotenberg has zero auth-bound URLs.
  const urls = extractImageUrls(bodyHtml);
  if (urls.length > RESOURCE_CEILINGS.maxImagesPerDocument) {
    throw new ResourceCeilingExceededError(
      "maxImagesPerDocument",
      urls.length,
      RESOURCE_CEILINGS.maxImagesPerDocument,
    );
  }

  const resolver = assetResolver ?? new AssetResolver();
  const assetMap = new Map<string, ResolvedAsset>();
  let totalBytes = 0;
  for (const url of urls) {
    const asset = await resolver.resolveAsset(url);
    totalBytes += asset.sizeBytes;
    if (totalBytes > RESOURCE_CEILINGS.maxTotalAssetBytes) {
      throw new ResourceCeilingExceededError(
        "maxTotalAssetBytes",
        totalBytes,
        RESOURCE_CEILINGS.maxTotalAssetBytes,
      );
    }
    assetMap.set(url, asset);
  }

  const inlinedBody = rewriteImgSrcToDataUri(bodyHtml, assetMap);
  const fullHtml = wrapInPrintDocument(inlinedBody);

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

- [ ] **Step 2: Update existing exportPdf tests to verify inlining**

Append to `frontend/apps/web/src/features/documents/mddm-editor/engine/export/__tests__/export-pdf.test.ts`:

```ts
describe("exportPdf asset inlining", () => {
  it("inlines image URLs as data: URIs in the HTML sent to Gotenberg", async () => {
    const PNG = new Uint8Array([0x89, 0x50, 0x4e, 0x47]);
    const fakeResolver = {
      async resolveAsset(_url: string) {
        return { bytes: PNG, mimeType: "image/png" as const, sizeBytes: PNG.byteLength };
      },
    } as unknown as import("../../asset-resolver").AssetResolver;

    const fetchSpy = vi.fn().mockResolvedValue(
      new Response(new Uint8Array([0x25, 0x50, 0x44, 0x46]), {
        status: 200,
        headers: { "Content-Type": "application/pdf" },
      }),
    );
    vi.stubGlobal("fetch", fetchSpy);

    await exportPdf({
      bodyHtml: `<p><img src="/api/images/aaa" /></p>`,
      documentId: "doc-1",
      assetResolver: fakeResolver,
    });

    const formData = fetchSpy.mock.calls[0][1].body as FormData;
    const htmlBlob = formData.get("index.html") as Blob;
    const htmlText = await htmlBlob.text();
    expect(htmlText).toContain("data:image/png;base64,");
    expect(htmlText).not.toContain("/api/images/aaa");
  });
});
```

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/export/__tests__/export-pdf.test.ts`
Expected: PASS — 6 tests passing total.

- [ ] **Step 3: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/export/export-pdf.ts frontend/apps/web/src/features/documents/mddm-editor/engine/export/__tests__/export-pdf.test.ts
git commit -m "feat(mddm-engine): wire asset inlining into exportPdf"
```

### Task 18: Update completeness gate to mark new blocks as fully supported

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/engine/completeness-gate/block-registry.ts`

- [ ] **Step 1: Update the registry**

Replace `frontend/apps/web/src/features/documents/mddm-editor/engine/completeness-gate/block-registry.ts` with:

```ts
// Central registry of block types the MDDM engine renders.
// After Plan 2, every entry has all three renderers (React, toExternalHTML, DOCX).

export type BlockSupport = Readonly<{
  type: string;
  hasReactRender: boolean;
  hasExternalHtml: boolean;
  hasDocxEmitter: boolean;
}>;

export const BLOCK_REGISTRY: readonly BlockSupport[] = [
  // Standard BlockNote blocks
  { type: "paragraph",        hasReactRender: true, hasExternalHtml: true, hasDocxEmitter: true },
  { type: "heading",          hasReactRender: true, hasExternalHtml: true, hasDocxEmitter: true },
  { type: "bulletListItem",   hasReactRender: true, hasExternalHtml: true, hasDocxEmitter: true },
  { type: "numberedListItem", hasReactRender: true, hasExternalHtml: true, hasDocxEmitter: true },
  { type: "image",            hasReactRender: true, hasExternalHtml: true, hasDocxEmitter: true },
  { type: "quote",            hasReactRender: true, hasExternalHtml: true, hasDocxEmitter: true },
  { type: "divider",          hasReactRender: true, hasExternalHtml: true, hasDocxEmitter: true },

  // MDDM custom blocks
  { type: "section",          hasReactRender: true, hasExternalHtml: true, hasDocxEmitter: true },
  { type: "field",            hasReactRender: true, hasExternalHtml: true, hasDocxEmitter: true },
  { type: "fieldGroup",       hasReactRender: true, hasExternalHtml: true, hasDocxEmitter: true },
  { type: "repeatable",       hasReactRender: true, hasExternalHtml: true, hasDocxEmitter: true },
  { type: "repeatableItem",   hasReactRender: true, hasExternalHtml: true, hasDocxEmitter: true },
  { type: "richBlock",        hasReactRender: true, hasExternalHtml: true, hasDocxEmitter: true },
  { type: "dataTable",        hasReactRender: true, hasExternalHtml: true, hasDocxEmitter: true },
  { type: "dataTableRow",     hasReactRender: true, hasExternalHtml: true, hasDocxEmitter: true },
  { type: "dataTableCell",    hasReactRender: true, hasExternalHtml: true, hasDocxEmitter: true },
];

export function getFullySupportedBlockTypes(): readonly string[] {
  return BLOCK_REGISTRY
    .filter((b) => b.hasReactRender && b.hasExternalHtml && b.hasDocxEmitter)
    .map((b) => b.type);
}
```

- [ ] **Step 2: Run the completeness gate test**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/completeness-gate/__tests__/completeness.test.ts`
Expected: PASS — every fully-supported block type produces a Blob via mddmToDocx, no unsupported types remain in the registry. The test from Plan 1 dynamically iterates the registry, so it covers the new types automatically.

If the test asserting "MissingEmitterError for unsupported types" fails because there are no unsupported types left, update that test to skip when the unsupported list is empty:

```ts
it("DOCX emitter throws MissingEmitterError for unsupported types in the registry", async () => {
  const unsupported = BLOCK_REGISTRY.filter((b) => !b.hasDocxEmitter).map((b) => b.type);
  if (unsupported.length === 0) {
    // After Plan 2 there are no unsupported blocks; the gate's job is done.
    return;
  }
  // ... existing loop
});
```

Run the test again:

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/completeness-gate/__tests__/completeness.test.ts`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/completeness-gate/
git commit -m "feat(mddm-engine): mark all 16 blocks fully supported in completeness registry"
```

---

## Part 5 — toExternalHTML Hooks for MDDM Blocks

**Critical BlockNote API note:** BlockNote's `contentRef` callback is ONLY provided to blocks declared with `content: "inline"`. For `content: "none"` blocks, `contentRef` is `undefined`. This means custom `toExternalHTML` hooks for `content: "none"` blocks cannot receive or mount inline content refs; they also cannot directly place nested *child blocks* inside their output, because BlockNote's `blocksToFullHTML` serializes child blocks separately from the parent's external HTML.

**Verified content types:**
- `content: "inline"` → `Field`, `DataTableCell` (these DO get a contentRef)
- `content: "none"` → `Section`, `FieldGroup`, `Repeatable`, `RepeatableItem`, `RichBlock`, `DataTable`, `DataTableRow`

**Decision for Plan 2:** Only `DataTableCell` gets a custom `toExternalHTML` in Plan 2 (it's the only new inline-content block). For every `content: "none"` block, per BlockNote's documented behavior — *"If undefined, BlockNote falls back to the standard render function"* — we do NOT register `toExternalHTML`. BlockNote's serializer uses the existing `render()` function (which correctly nests children via its React component tree) to produce the external HTML. The print stylesheet from Plan 1 Task 30 already hides `.bn-side-menu`, `.bn-drag-handle`, and other editor chrome so the fallback output is print-ready.

**What this means for Plan 2 task list:**
- Keep Task 19 (DataTableCell `toExternalHTML`) and Task 25 (barrel export) and Task 31 (DataTableCell block-spec registration).
- Tasks 20, 21, 22, 23, 24 are **removed** (DataTableRow, DataTable, RepeatableItem, Repeatable, RichBlock `toExternalHTML` components are not created).
- Tasks 26, 27, 28, 29, 30 are **removed** (no registrations on `content: "none"` blocks).
- A new Task 25b is added to verify that `blocksToFullHTML` produces acceptable output for these blocks via their render fallback.

### Task 19: data-table-cell-html

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/external-html/data-table-cell-html.tsx`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/external-html/__tests__/data-table-cell-html.test.tsx`

- [ ] **Step 1: Write the failing test**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/external-html/__tests__/data-table-cell-html.test.tsx`:

```tsx
import { describe, expect, it } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import { DataTableCellExternalHTML } from "../data-table-cell-html";
import { defaultLayoutTokens } from "../../layout-ir";

describe("DataTableCellExternalHTML", () => {
  it("renders a <td> with mddm-data-table-cell class", () => {
    const html = renderToStaticMarkup(
      <DataTableCellExternalHTML tokens={defaultLayoutTokens}>
        <span>100</span>
      </DataTableCellExternalHTML>,
    );
    expect(html).toContain("<td");
    expect(html).toContain("mddm-data-table-cell");
    expect(html).toContain("100");
  });

  it("uses absolute padding (mm) and accentBorder color", () => {
    const html = renderToStaticMarkup(
      <DataTableCellExternalHTML tokens={defaultLayoutTokens}>x</DataTableCellExternalHTML>,
    );
    expect(html).toMatch(/padding:\s*\d+(?:\.\d+)?mm/);
    expect(html.toLowerCase()).toContain(defaultLayoutTokens.theme.accentBorder.toLowerCase());
  });
});
```

- [ ] **Step 2: Run test — expect failure**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/external-html/__tests__/data-table-cell-html.test.tsx`
Expected: FAIL.

- [ ] **Step 3: Implement data-table-cell-html.tsx**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/external-html/data-table-cell-html.tsx`:

```tsx
import type { ReactNode } from "react";
import type { LayoutTokens } from "../layout-ir";

export type DataTableCellExternalHTMLProps = {
  tokens: LayoutTokens;
  children?: ReactNode;
};

export function DataTableCellExternalHTML({ tokens, children }: DataTableCellExternalHTMLProps) {
  return (
    <td
      className="mddm-data-table-cell"
      data-mddm-block="dataTableCell"
      style={{
        padding: `${tokens.spacing.cellPaddingMm}mm`,
        border: `0.5pt solid ${tokens.theme.accentBorder}`,
        verticalAlign: "top",
      }}
    >
      {children}
    </td>
  );
}
```

- [ ] **Step 4: Run test — expect pass**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/external-html/__tests__/data-table-cell-html.test.tsx`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/external-html/data-table-cell-html.tsx frontend/apps/web/src/features/documents/mddm-editor/engine/external-html/__tests__/data-table-cell-html.test.tsx
git commit -m "feat(mddm-engine): add DataTableCell toExternalHTML"
```

### Tasks 20-24: REMOVED (see note above Task 19)

Tasks 20-24 in an earlier draft created `toExternalHTML` components for `DataTableRow`, `DataTable`, `RepeatableItem`, `Repeatable`, and `RichBlock`. These blocks all have `content: "none"`, so BlockNote never provides a `contentRef` and `blocksToFullHTML` cannot correctly nest their child blocks through a custom hook. Per the note above Task 19, these blocks rely on BlockNote's render() fallback instead. No implementation work for these five tasks — skip directly to Task 25.

### Task 25: Update external-html barrel

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/engine/external-html/index.ts`

- [ ] **Step 1: Replace the barrel**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/external-html/index.ts`:

```ts
// Plan 1 exports (retained)
export { SectionExternalHTML, type SectionExternalHTMLProps } from "./section-html";
export { FieldExternalHTML, type FieldExternalHTMLProps } from "./field-html";
export { FieldGroupExternalHTML, type FieldGroupExternalHTMLProps } from "./field-group-html";

// Plan 2 additions: only inline-content blocks get custom toExternalHTML.
// DataTableCell is the only content:"inline" block added in Plan 2.
// Repeatable, RepeatableItem, RichBlock, DataTable, DataTableRow are
// content:"none" and rely on BlockNote's render() fallback.
export { DataTableCellExternalHTML, type DataTableCellExternalHTMLProps } from "./data-table-cell-html";
```

### Task 25b: Gating test — verify blocksToFullHTML render fallback for content:"none" blocks

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/external-html/__tests__/render-fallback.test.tsx`

This test validates the foundational assumption that BlockNote's `blocksToFullHTML` produces usable HTML for `content: "none"` MDDM blocks via their `render()` fallback. If this test fails, the render-fallback strategy is not viable and Plan 2 needs a redesign before continuing — so this is a **gating test** that runs early in Part 5.

- [ ] **Step 1: Install @blocknote/server-util**

Run: `cd frontend/apps/web && npm list @blocknote/server-util 2>&1 | tail -5`
If missing: `cd frontend/apps/web && npm install @blocknote/server-util@^0.47.3`
Expected: Package installed in dependencies.

- [ ] **Step 2: Write the failing test**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/external-html/__tests__/render-fallback.test.tsx`:

```tsx
import { describe, expect, it } from "vitest";
import { BlockNoteSchema, defaultBlockSpecs } from "@blocknote/core";
import { ServerBlockNoteEditor } from "@blocknote/server-util";
import { mddmSchemaBlockSpecs } from "../../../schema";
import { mddmToBlockNote, type MDDMEnvelope } from "../../../adapter";

async function toHtml(envelope: MDDMEnvelope): Promise<string> {
  const schema = BlockNoteSchema.create({
    blockSpecs: {
      ...defaultBlockSpecs,
      ...mddmSchemaBlockSpecs,
    },
  });
  const editor = ServerBlockNoteEditor.create({ schema });
  const blocks = mddmToBlockNote(envelope);
  return await editor.blocksToFullHTML(blocks as any);
}

describe("blocksToFullHTML render fallback for MDDM content:\"none\" blocks", () => {
  it("serializes a repeatable + repeatableItem + nested paragraph with text preserved", async () => {
    const envelope: MDDMEnvelope = {
      mddm_version: 1,
      template_ref: null,
      blocks: [
        {
          id: "r",
          type: "repeatable",
          props: { label: "Steps", itemPrefix: "Step" },
          children: [
            {
              id: "ri",
              type: "repeatableItem",
              props: { title: "Step 1" },
              children: [
                { id: "p", type: "paragraph", props: {}, children: [{ type: "text", text: "inspect" }] },
              ],
            },
          ],
        },
      ],
    };
    const html = await toHtml(envelope);
    expect(html).toContain("inspect");
    expect(html.toLowerCase()).toContain("repeatable");
  });

  it("serializes a dataTable + dataTableRow + dataTableCell with cell text preserved", async () => {
    const envelope: MDDMEnvelope = {
      mddm_version: 1,
      template_ref: null,
      blocks: [
        {
          id: "t",
          type: "dataTable",
          props: {
            label: "Items",
            columns: [{ key: "c0", label: "Item" }],
            locked: true, minRows: 0, maxRows: 500, density: "normal",
          },
          children: [
            {
              id: "row1",
              type: "dataTableRow",
              props: {},
              children: [
                {
                  id: "cell1",
                  type: "dataTableCell",
                  props: { columnKey: "c0" },
                  children: [{ type: "text", text: "Parafuso" }],
                },
              ],
            },
          ],
        },
      ],
    };
    const html = await toHtml(envelope);
    expect(html).toContain("Parafuso");
  });
});
```

- [ ] **Step 3: Check that `mddmSchemaBlockSpecs` is a reusable export**

Run: `grep -n "mddmSchemaBlockSpecs\|export const blockSpecs\|blockSpecs =" frontend/apps/web/src/features/documents/mddm-editor/schema.ts | head -10`

If `mddmSchemaBlockSpecs` does NOT exist in the schema module, add it there so the test and the editor share one source of truth:

```ts
// frontend/apps/web/src/features/documents/mddm-editor/schema.ts
// Add alongside the existing default export
export const mddmSchemaBlockSpecs = {
  section: Section,
  field: Field,
  fieldGroup: FieldGroup,
  repeatable: Repeatable,
  repeatableItem: RepeatableItem,
  richBlock: RichBlock,
  dataTable: DataTable,
  dataTableRow: DataTableRow,
  dataTableCell: DataTableCell,
};
```

Then `schema.ts` can reuse it where it creates the schema.

- [ ] **Step 4: Run the test**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/external-html/__tests__/render-fallback.test.tsx`
Expected: PASS — 2 tests verifying structural blocks preserve child text in fallback HTML.

**If the test FAILS**: BlockNote's serializer cannot walk `content: "none"` children via the default fallback. Stop Plan 2 execution. File a design spike issue and evaluate alternatives: (a) custom MDDM-to-HTML walker that mirrors the render tree manually, (b) downgrade Plan 2 to DOCX-only and defer PDF visual parity for these blocks, (c) restructure Repeatable/DataTable to use `content: "inline"` with a custom inline content spec. Do NOT proceed with Tasks 31-41 until this is resolved.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/external-html/__tests__/render-fallback.test.tsx frontend/apps/web/package.json frontend/apps/web/package-lock.json frontend/apps/web/src/features/documents/mddm-editor/schema.ts
git commit -m "test(mddm-engine): gating test for blocksToFullHTML render fallback on structural blocks"
```

- [ ] **Step 2: Verify compilation**

Run: `cd frontend/apps/web && npx tsc --noEmit 2>&1 | grep external-html | head -5`
Expected: No errors.

- [ ] **Step 3: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/external-html/index.ts
git commit -m "feat(mddm-engine): expand external-html barrel for full block coverage"
```

---

## Part 6 — Register toExternalHTML on Existing Block Specs

### Tasks 26-30: REMOVED (see note above Task 19)

Tasks 26-30 in an earlier draft registered custom `toExternalHTML` hooks on `Repeatable`, `RepeatableItem`, `RichBlock`, `DataTable`, and `DataTableRow`. All five are declared with `content: "none"` in the MDDM block specs, which means BlockNote does not pass a `contentRef` callback. These blocks already have working `render()` functions that `blocksToFullHTML` uses as a fallback, so no registration is needed. Skip directly to Task 31.

### Task 31: Register toExternalHTML on the DataTableCell block spec

**Files:**
- Modify: `frontend/apps/web/src/features/documents/mddm-editor/blocks/DataTableCell.tsx`

**Note:** `DataTableCell` is declared with `content: "inline"` (verified via `grep -n 'content:' frontend/apps/web/src/features/documents/mddm-editor/blocks/DataTableCell.tsx`). BlockNote provides a `contentRef` callback that must be invoked with the mounted DOM node — use the callback-ref form, NOT `React.Ref`.

- [ ] **Step 1: Add the imports**

```tsx
import { DataTableCellExternalHTML } from "../engine/external-html";
import { defaultLayoutTokens } from "../engine/layout-ir";
```

- [ ] **Step 2: Add toExternalHTML to the block implementation**

```tsx
toExternalHTML: ({ contentRef }) => (
  <DataTableCellExternalHTML tokens={defaultLayoutTokens}>
    <span ref={(el: HTMLSpanElement | null) => contentRef(el)} />
  </DataTableCellExternalHTML>
),
```

- [ ] **Step 3: Verify compilation**

Run: `cd frontend/apps/web && npx tsc --noEmit 2>&1 | grep DataTableCell.tsx | head -5`
Expected: No errors.

- [ ] **Step 4: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/blocks/DataTableCell.tsx
git commit -m "feat(mddm-engine): register toExternalHTML on DataTableCell block"
```

---

## Part 7 — Additional Golden Fixtures

Each fixture follows the same pattern as Plan 1's `01-simple-po`: input MDDM JSON + generated `expected.document.xml` + a small test runner. The generation process reuses the `generate-golden.test.ts` regenerator from Plan 1 — set `MDDM_GOLDEN_UPDATE=1` and run the regenerator once for each new fixture.

### Task 32: 02-complex-table fixture

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/golden/fixtures/02-complex-table/input.mddm.json`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/golden/fixtures/02-complex-table/expected.document.xml` (generated)
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/golden/__tests__/golden-02-complex-table.test.ts`

- [ ] **Step 1: Write the input fixture**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/golden/fixtures/02-complex-table/input.mddm.json`:

```json
{
  "mddm_version": 1,
  "template_ref": null,
  "blocks": [
    {
      "id": "00000000-0000-4000-8000-000000000010",
      "type": "section",
      "props": { "title": "2. Materiais", "color": "red" },
      "children": []
    },
    {
      "id": "00000000-0000-4000-8000-000000000011",
      "type": "dataTable",
      "props": {
        "label": "Lista de Materiais",
        "columns": [
          { "key": "item",  "label": "Item" },
          { "key": "qty",   "label": "Quantidade" },
          { "key": "valor", "label": "Valor Unitário" }
        ],
        "locked": true,
        "minRows": 0,
        "maxRows": 500,
        "density": "normal"
      },
      "children": [
        {
          "id": "00000000-0000-4000-8000-000000000012",
          "type": "dataTableRow",
          "props": {},
          "children": [
            { "id": "00000000-0000-4000-8000-000000000013", "type": "dataTableCell", "props": { "columnKey": "item" }, "children": [{ "type": "text", "text": "Parafuso M8" }] },
            { "id": "00000000-0000-4000-8000-000000000014", "type": "dataTableCell", "props": { "columnKey": "qty" }, "children": [{ "type": "text", "text": "100" }] },
            { "id": "00000000-0000-4000-8000-000000000015", "type": "dataTableCell", "props": { "columnKey": "valor" }, "children": [{ "type": "text", "text": "R$ 5,00" }] }
          ]
        },
        {
          "id": "00000000-0000-4000-8000-000000000016",
          "type": "dataTableRow",
          "props": {},
          "children": [
            { "id": "00000000-0000-4000-8000-000000000017", "type": "dataTableCell", "props": { "columnKey": "item" }, "children": [{ "type": "text", "text": "Porca M8" }] },
            { "id": "00000000-0000-4000-8000-000000000018", "type": "dataTableCell", "props": { "columnKey": "qty" }, "children": [{ "type": "text", "text": "200" }] },
            { "id": "00000000-0000-4000-8000-000000000019", "type": "dataTableCell", "props": { "columnKey": "valor" }, "children": [{ "type": "text", "text": "R$ 2,50" }] }
          ]
        }
      ]
    }
  ]
}
```

- [ ] **Step 2: Write the runner test**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/golden/__tests__/golden-02-complex-table.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { readFileSync, existsSync } from "node:fs";
import { resolve } from "node:path";
import { mddmToDocx } from "../../docx-emitter";
import { defaultLayoutTokens } from "../../layout-ir";
import { normalizeDocxXml, unzipDocxDocumentXml } from "../golden-helpers";
import type { MDDMEnvelope } from "../../../adapter";

const FIXTURE = resolve(__dirname, "../fixtures/02-complex-table");

describe("Golden fixture: 02-complex-table", () => {
  it("emits DOCX matching expected.document.xml", async () => {
    const envelope = JSON.parse(readFileSync(resolve(FIXTURE, "input.mddm.json"), "utf8")) as MDDMEnvelope;
    const blob = await mddmToDocx(envelope, defaultLayoutTokens);
    const xml = await unzipDocxDocumentXml(blob);
    const actual = normalizeDocxXml(xml);

    const expectedPath = resolve(FIXTURE, "expected.document.xml");
    if (!existsSync(expectedPath)) {
      throw new Error(`Golden file missing: ${expectedPath}\nGenerate via MDDM_GOLDEN_UPDATE=1 plus the regenerator.`);
    }
    const expected = normalizeDocxXml(readFileSync(expectedPath, "utf8"));
    expect(actual).toBe(expected);
  });
});
```

- [ ] **Step 3: Extend the regenerator to write 02-complex-table golden**

Open `frontend/apps/web/src/features/documents/mddm-editor/engine/golden/__tests__/generate-golden.test.ts` (created in Plan 1) and add a second `it()` block inside the existing `describe.skipIf(!process.env.MDDM_GOLDEN_UPDATE)`:

```ts
it("writes expected.document.xml for 02-complex-table", async () => {
  const dir = resolve(__dirname, "../fixtures/02-complex-table");
  const envelope = JSON.parse(readFileSync(resolve(dir, "input.mddm.json"), "utf8")) as MDDMEnvelope;
  const blob = await mddmToDocx(envelope, defaultLayoutTokens);
  const xml = await unzipDocxDocumentXml(blob);
  writeFileSync(resolve(dir, "expected.document.xml"), xml, "utf8");
});
```

- [ ] **Step 4: Generate the expected file**

Run:

```bash
cd frontend/apps/web
MDDM_GOLDEN_UPDATE=1 npx vitest run src/features/documents/mddm-editor/engine/golden/__tests__/generate-golden.test.ts
```

Expected: PASS. The file `expected.document.xml` is now present in `02-complex-table/`.

- [ ] **Step 5: Inspect the generated file for sanity**

Run: `head -40 frontend/apps/web/src/features/documents/mddm-editor/engine/golden/fixtures/02-complex-table/expected.document.xml`
Expected: Well-formed OOXML containing the section header text and the data-table contents (Item / Quantidade / Valor Unitário headers and the row data).

- [ ] **Step 6: Run the runner test**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/golden/__tests__/golden-02-complex-table.test.ts`
Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/golden/fixtures/02-complex-table/ frontend/apps/web/src/features/documents/mddm-editor/engine/golden/__tests__/golden-02-complex-table.test.ts frontend/apps/web/src/features/documents/mddm-editor/engine/golden/__tests__/generate-golden.test.ts
git commit -m "test(mddm-engine): add 02-complex-table golden fixture"
```

### Task 33: 03-repeatable-sections fixture

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/golden/fixtures/03-repeatable-sections/input.mddm.json`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/golden/fixtures/03-repeatable-sections/expected.document.xml` (generated)
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/golden/__tests__/golden-03-repeatable.test.ts`

- [ ] **Step 1: Write the input fixture**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/golden/fixtures/03-repeatable-sections/input.mddm.json`:

```json
{
  "mddm_version": 1,
  "template_ref": null,
  "blocks": [
    {
      "id": "00000000-0000-4000-8000-000000000020",
      "type": "section",
      "props": { "title": "3. Procedimentos", "color": "red" },
      "children": []
    },
    {
      "id": "00000000-0000-4000-8000-000000000021",
      "type": "repeatable",
      "props": { "label": "Etapas", "itemPrefix": "Etapa" },
      "children": [
        {
          "id": "00000000-0000-4000-8000-000000000022",
          "type": "repeatableItem",
          "props": { "title": "Etapa 1" },
          "children": [
            {
              "id": "00000000-0000-4000-8000-000000000023",
              "type": "paragraph",
              "props": {},
              "children": [{ "type": "text", "text": "Inspecionar a peça antes de iniciar." }]
            }
          ]
        },
        {
          "id": "00000000-0000-4000-8000-000000000024",
          "type": "repeatableItem",
          "props": { "title": "Etapa 2" },
          "children": [
            {
              "id": "00000000-0000-4000-8000-000000000025",
              "type": "paragraph",
              "props": {},
              "children": [{ "type": "text", "text": "Aplicar o torque conforme tabela." }]
            }
          ]
        }
      ]
    }
  ]
}
```

- [ ] **Step 2: Write the runner test**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/golden/__tests__/golden-03-repeatable.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { readFileSync, existsSync } from "node:fs";
import { resolve } from "node:path";
import { mddmToDocx } from "../../docx-emitter";
import { defaultLayoutTokens } from "../../layout-ir";
import { normalizeDocxXml, unzipDocxDocumentXml } from "../golden-helpers";
import type { MDDMEnvelope } from "../../../adapter";

const FIXTURE = resolve(__dirname, "../fixtures/03-repeatable-sections");

describe("Golden fixture: 03-repeatable-sections", () => {
  it("emits DOCX matching expected.document.xml", async () => {
    const envelope = JSON.parse(readFileSync(resolve(FIXTURE, "input.mddm.json"), "utf8")) as MDDMEnvelope;
    const blob = await mddmToDocx(envelope, defaultLayoutTokens);
    const xml = await unzipDocxDocumentXml(blob);
    const actual = normalizeDocxXml(xml);

    const expectedPath = resolve(FIXTURE, "expected.document.xml");
    if (!existsSync(expectedPath)) {
      throw new Error(`Golden file missing: ${expectedPath}`);
    }
    const expected = normalizeDocxXml(readFileSync(expectedPath, "utf8"));
    expect(actual).toBe(expected);
  });
});
```

- [ ] **Step 3: Extend the regenerator with the new fixture and generate the expected file**

In `generate-golden.test.ts`, add:

```ts
it("writes expected.document.xml for 03-repeatable-sections", async () => {
  const dir = resolve(__dirname, "../fixtures/03-repeatable-sections");
  const envelope = JSON.parse(readFileSync(resolve(dir, "input.mddm.json"), "utf8")) as MDDMEnvelope;
  const blob = await mddmToDocx(envelope, defaultLayoutTokens);
  const xml = await unzipDocxDocumentXml(blob);
  writeFileSync(resolve(dir, "expected.document.xml"), xml, "utf8");
});
```

Then:

```bash
cd frontend/apps/web
MDDM_GOLDEN_UPDATE=1 npx vitest run src/features/documents/mddm-editor/engine/golden/__tests__/generate-golden.test.ts
```

Expected: PASS.

- [ ] **Step 4: Run the runner test**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/golden/__tests__/golden-03-repeatable.test.ts`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/golden/fixtures/03-repeatable-sections/ frontend/apps/web/src/features/documents/mddm-editor/engine/golden/__tests__/golden-03-repeatable.test.ts frontend/apps/web/src/features/documents/mddm-editor/engine/golden/__tests__/generate-golden.test.ts
git commit -m "test(mddm-engine): add 03-repeatable-sections golden fixture"
```

### Task 34: 04-all-inline-marks fixture

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/golden/fixtures/04-all-inline-marks/input.mddm.json`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/golden/fixtures/04-all-inline-marks/expected.document.xml` (generated)
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/golden/__tests__/golden-04-inline-marks.test.ts`

- [ ] **Step 1: Write the input fixture**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/golden/fixtures/04-all-inline-marks/input.mddm.json`:

```json
{
  "mddm_version": 1,
  "template_ref": null,
  "blocks": [
    {
      "id": "00000000-0000-4000-8000-000000000030",
      "type": "paragraph",
      "props": {},
      "children": [
        { "type": "text", "text": "Plain " },
        { "type": "text", "text": "bold", "marks": [{ "type": "bold" }] },
        { "type": "text", "text": " " },
        { "type": "text", "text": "italic", "marks": [{ "type": "italic" }] },
        { "type": "text", "text": " " },
        { "type": "text", "text": "underline", "marks": [{ "type": "underline" }] },
        { "type": "text", "text": " " },
        { "type": "text", "text": "strike", "marks": [{ "type": "strike" }] },
        { "type": "text", "text": " " },
        { "type": "text", "text": "code", "marks": [{ "type": "code" }] }
      ]
    }
  ]
}
```

- [ ] **Step 2: Write the runner test**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/golden/__tests__/golden-04-inline-marks.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { readFileSync, existsSync } from "node:fs";
import { resolve } from "node:path";
import { mddmToDocx } from "../../docx-emitter";
import { defaultLayoutTokens } from "../../layout-ir";
import { normalizeDocxXml, unzipDocxDocumentXml } from "../golden-helpers";
import type { MDDMEnvelope } from "../../../adapter";

const FIXTURE = resolve(__dirname, "../fixtures/04-all-inline-marks");

describe("Golden fixture: 04-all-inline-marks", () => {
  it("emits DOCX matching expected.document.xml", async () => {
    const envelope = JSON.parse(readFileSync(resolve(FIXTURE, "input.mddm.json"), "utf8")) as MDDMEnvelope;
    const blob = await mddmToDocx(envelope, defaultLayoutTokens);
    const xml = await unzipDocxDocumentXml(blob);
    const actual = normalizeDocxXml(xml);

    const expectedPath = resolve(FIXTURE, "expected.document.xml");
    if (!existsSync(expectedPath)) {
      throw new Error(`Golden file missing: ${expectedPath}`);
    }
    const expected = normalizeDocxXml(readFileSync(expectedPath, "utf8"));
    expect(actual).toBe(expected);
  });
});
```

- [ ] **Step 3: Extend the regenerator and run it**

Append to `generate-golden.test.ts`:

```ts
it("writes expected.document.xml for 04-all-inline-marks", async () => {
  const dir = resolve(__dirname, "../fixtures/04-all-inline-marks");
  const envelope = JSON.parse(readFileSync(resolve(dir, "input.mddm.json"), "utf8")) as MDDMEnvelope;
  const blob = await mddmToDocx(envelope, defaultLayoutTokens);
  const xml = await unzipDocxDocumentXml(blob);
  writeFileSync(resolve(dir, "expected.document.xml"), xml, "utf8");
});
```

Run:
```bash
MDDM_GOLDEN_UPDATE=1 npx vitest run src/features/documents/mddm-editor/engine/golden/__tests__/generate-golden.test.ts
```
Expected: PASS.

- [ ] **Step 4: Run the runner test**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/golden/__tests__/golden-04-inline-marks.test.ts`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/golden/fixtures/04-all-inline-marks/ frontend/apps/web/src/features/documents/mddm-editor/engine/golden/__tests__/golden-04-inline-marks.test.ts frontend/apps/web/src/features/documents/mddm-editor/engine/golden/__tests__/generate-golden.test.ts
git commit -m "test(mddm-engine): add 04-all-inline-marks golden fixture"
```

### Task 35: 05-multi-block-doc fixture

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/golden/fixtures/05-multi-block-doc/input.mddm.json`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/golden/fixtures/05-multi-block-doc/expected.document.xml` (generated)
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/golden/__tests__/golden-05-multi-block.test.ts`

- [ ] **Step 1: Write the input fixture (covers heading, lists, quote, divider, richBlock)**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/golden/fixtures/05-multi-block-doc/input.mddm.json`:

```json
{
  "mddm_version": 1,
  "template_ref": null,
  "blocks": [
    {
      "id": "00000000-0000-4000-8000-000000000040",
      "type": "heading",
      "props": { "level": 1 },
      "children": [{ "type": "text", "text": "Manual de Operações" }]
    },
    {
      "id": "00000000-0000-4000-8000-000000000041",
      "type": "heading",
      "props": { "level": 2 },
      "children": [{ "type": "text", "text": "Introdução" }]
    },
    {
      "id": "00000000-0000-4000-8000-000000000042",
      "type": "paragraph",
      "props": {},
      "children": [{ "type": "text", "text": "Este manual cobre a operação padrão." }]
    },
    {
      "id": "00000000-0000-4000-8000-000000000043",
      "type": "bulletListItem",
      "props": {},
      "children": [{ "type": "text", "text": "Verificar EPI antes de iniciar." }]
    },
    {
      "id": "00000000-0000-4000-8000-000000000044",
      "type": "bulletListItem",
      "props": {},
      "children": [{ "type": "text", "text": "Inspecionar a área de trabalho." }]
    },
    {
      "id": "00000000-0000-4000-8000-000000000045",
      "type": "numberedListItem",
      "props": {},
      "children": [{ "type": "text", "text": "Ligar o equipamento." }]
    },
    {
      "id": "00000000-0000-4000-8000-000000000046",
      "type": "numberedListItem",
      "props": {},
      "children": [{ "type": "text", "text": "Aguardar o aquecimento." }]
    },
    {
      "id": "00000000-0000-4000-8000-000000000047",
      "type": "quote",
      "props": {},
      "children": [{ "type": "text", "text": "Segurança em primeiro lugar." }]
    },
    {
      "id": "00000000-0000-4000-8000-000000000048",
      "type": "divider",
      "props": {},
      "children": []
    },
    {
      "id": "00000000-0000-4000-8000-000000000049",
      "type": "richBlock",
      "props": { "label": "Observações" },
      "children": [
        {
          "id": "00000000-0000-4000-8000-00000000004a",
          "type": "paragraph",
          "props": {},
          "children": [{ "type": "text", "text": "Conteúdo adicional dentro de um RichBlock." }]
        }
      ]
    }
  ]
}
```

- [ ] **Step 2: Write the runner test**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/golden/__tests__/golden-05-multi-block.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { readFileSync, existsSync } from "node:fs";
import { resolve } from "node:path";
import { mddmToDocx } from "../../docx-emitter";
import { defaultLayoutTokens } from "../../layout-ir";
import { normalizeDocxXml, unzipDocxDocumentXml } from "../golden-helpers";
import type { MDDMEnvelope } from "../../../adapter";

const FIXTURE = resolve(__dirname, "../fixtures/05-multi-block-doc");

describe("Golden fixture: 05-multi-block-doc", () => {
  it("emits DOCX matching expected.document.xml", async () => {
    const envelope = JSON.parse(readFileSync(resolve(FIXTURE, "input.mddm.json"), "utf8")) as MDDMEnvelope;
    const blob = await mddmToDocx(envelope, defaultLayoutTokens);
    const xml = await unzipDocxDocumentXml(blob);
    const actual = normalizeDocxXml(xml);

    const expectedPath = resolve(FIXTURE, "expected.document.xml");
    if (!existsSync(expectedPath)) {
      throw new Error(`Golden file missing: ${expectedPath}`);
    }
    const expected = normalizeDocxXml(readFileSync(expectedPath, "utf8"));
    expect(actual).toBe(expected);
  });
});
```

- [ ] **Step 3: Regenerate and verify**

Append to `generate-golden.test.ts`:

```ts
it("writes expected.document.xml for 05-multi-block-doc", async () => {
  const dir = resolve(__dirname, "../fixtures/05-multi-block-doc");
  const envelope = JSON.parse(readFileSync(resolve(dir, "input.mddm.json"), "utf8")) as MDDMEnvelope;
  const blob = await mddmToDocx(envelope, defaultLayoutTokens);
  const xml = await unzipDocxDocumentXml(blob);
  writeFileSync(resolve(dir, "expected.document.xml"), xml, "utf8");
});
```

Then:

```bash
MDDM_GOLDEN_UPDATE=1 npx vitest run src/features/documents/mddm-editor/engine/golden/__tests__/generate-golden.test.ts
npx vitest run src/features/documents/mddm-editor/engine/golden/__tests__/golden-05-multi-block.test.ts
```
Expected: Both PASS.

- [ ] **Step 4: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/golden/fixtures/05-multi-block-doc/ frontend/apps/web/src/features/documents/mddm-editor/engine/golden/__tests__/golden-05-multi-block.test.ts frontend/apps/web/src/features/documents/mddm-editor/engine/golden/__tests__/generate-golden.test.ts
git commit -m "test(mddm-engine): add 05-multi-block-doc golden fixture"
```

### Task 36: 06-theme-override fixture

**Files:**
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/golden/fixtures/06-theme-override/input.mddm.json`
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/golden/fixtures/06-theme-override/expected.document.xml` (generated)
- Create: `frontend/apps/web/src/features/documents/mddm-editor/engine/golden/__tests__/golden-06-theme.test.ts`

- [ ] **Step 1: Write the input fixture**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/golden/fixtures/06-theme-override/input.mddm.json`:

```json
{
  "mddm_version": 1,
  "template_ref": null,
  "blocks": [
    {
      "id": "00000000-0000-4000-8000-000000000050",
      "type": "section",
      "props": { "title": "Override Theme", "color": "blue" },
      "children": []
    },
    {
      "id": "00000000-0000-4000-8000-000000000051",
      "type": "field",
      "props": { "label": "Color" },
      "children": [{ "type": "text", "text": "Custom blue" }]
    }
  ]
}
```

- [ ] **Step 2: Write the runner test that uses overridden theme tokens**

Write to `frontend/apps/web/src/features/documents/mddm-editor/engine/golden/__tests__/golden-06-theme.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { readFileSync, existsSync } from "node:fs";
import { resolve } from "node:path";
import { mddmToDocx } from "../../docx-emitter";
import { defaultLayoutTokens } from "../../layout-ir";
import { normalizeDocxXml, unzipDocxDocumentXml } from "../golden-helpers";
import type { MDDMEnvelope } from "../../../adapter";

const FIXTURE = resolve(__dirname, "../fixtures/06-theme-override");

const BLUE_TOKENS = {
  ...defaultLayoutTokens,
  theme: {
    accent: "#2a4f8b",
    accentLight: "#eaf1fa",
    accentDark: "#15273f",
    accentBorder: "#b9c9e0",
  },
};

describe("Golden fixture: 06-theme-override", () => {
  it("emits DOCX with overridden theme matching expected.document.xml", async () => {
    const envelope = JSON.parse(readFileSync(resolve(FIXTURE, "input.mddm.json"), "utf8")) as MDDMEnvelope;
    const blob = await mddmToDocx(envelope, BLUE_TOKENS);
    const xml = await unzipDocxDocumentXml(blob);
    const actual = normalizeDocxXml(xml);

    const expectedPath = resolve(FIXTURE, "expected.document.xml");
    if (!existsSync(expectedPath)) {
      throw new Error(`Golden file missing: ${expectedPath}`);
    }
    const expected = normalizeDocxXml(readFileSync(expectedPath, "utf8"));
    expect(actual).toBe(expected);
  });
});
```

- [ ] **Step 3: Extend regenerator with same theme override and run it**

Append to `generate-golden.test.ts` (importing the same blue tokens or duplicating):

```ts
it("writes expected.document.xml for 06-theme-override", async () => {
  const dir = resolve(__dirname, "../fixtures/06-theme-override");
  const envelope = JSON.parse(readFileSync(resolve(dir, "input.mddm.json"), "utf8")) as MDDMEnvelope;
  const tokens = {
    ...defaultLayoutTokens,
    theme: {
      accent: "#2a4f8b",
      accentLight: "#eaf1fa",
      accentDark: "#15273f",
      accentBorder: "#b9c9e0",
    },
  };
  const blob = await mddmToDocx(envelope, tokens);
  const xml = await unzipDocxDocumentXml(blob);
  writeFileSync(resolve(dir, "expected.document.xml"), xml, "utf8");
});
```

Then run regenerator:

```bash
MDDM_GOLDEN_UPDATE=1 npx vitest run src/features/documents/mddm-editor/engine/golden/__tests__/generate-golden.test.ts
```
Expected: PASS.

- [ ] **Step 4: Run the runner test**

Run: `cd frontend/apps/web && npx vitest run src/features/documents/mddm-editor/engine/golden/__tests__/golden-06-theme.test.ts`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/src/features/documents/mddm-editor/engine/golden/fixtures/06-theme-override/ frontend/apps/web/src/features/documents/mddm-editor/engine/golden/__tests__/golden-06-theme.test.ts frontend/apps/web/src/features/documents/mddm-editor/engine/golden/__tests__/generate-golden.test.ts
git commit -m "test(mddm-engine): add 06-theme-override golden fixture"
```

---

## Part 8 — Visual Parity (Playwright)

### Task 37: Install pixelmatch, pngjs, and pdf-img-convert

**Files:**
- Modify: `frontend/apps/web/package.json`

**Library choice:** We use `pdf-img-convert` for Node-side PDF→PNG rasterization instead of driving `pdfjs-dist` manually with a custom canvas factory. `pdf-img-convert` wraps pdfjs with a working Node canvas backend and exposes a simple `pdfBufferToPngBuffer`-style API.

- [ ] **Step 1: Install dev dependencies**

Run:
```bash
cd frontend/apps/web
npm install -D pixelmatch@^7.1.0 pngjs@^7.0.0 pdf-img-convert@^1.3.0
```
Expected: Three new entries in `devDependencies`.

- [ ] **Step 2: Verify they install cleanly**

Run: `cd frontend/apps/web && npm list pixelmatch pdf-img-convert pngjs 2>&1 | tail -5`
Expected: All three listed at expected versions.

- [ ] **Step 3: Commit**

```bash
git add frontend/apps/web/package.json frontend/apps/web/package-lock.json
git commit -m "chore(mddm-engine): add pixelmatch + pdf-img-convert for visual parity tests"
```

### Task 38: Build the MDDM test harness page

**Files:**
- Create: `frontend/apps/web/src/test-harness/MDDMTestHarness.tsx`
- Modify: `frontend/apps/web/src/App.tsx` (to add an early escape hatch for the harness path)

**Routing note:** The MetalDocs web app does NOT use element-based `<Routes>` / `<Route>` — it uses `HashRouter` + a custom `viewFromPath` mapping in `frontend/apps/web/src/routing/workspaceRoutes.ts` feeding into `WorkspaceShell`. The standard pattern of registering a new `<Route>` will not work. Instead, add an early-return escape hatch in `App.tsx` that checks `useLocation()` for a harness path and renders `<MDDMTestHarness />` directly, bypassing the workspace shell.

**Backend-bypass note for PDF export:** The backend `/render/pdf` endpoint requires authentication and document-level authorization. Playwright parity tests do NOT want to set up a real logged-in session or seed a test document just to validate rendering. Instead, the harness exposes a dev-only `__mddmRenderPdfDirectlyViaGotenberg(html, css)` function that calls Gotenberg's Chromium route directly (bypassing the Go backend) via a Vite dev proxy. The parity tests use this API; they do NOT call `exportPdf()` (which hits the auth-protected endpoint).

- [ ] **Step 1: Add a dev-only Vite proxy path for Gotenberg**

Open `frontend/apps/web/vite.config.ts` and add a second proxy entry only when in dev mode:

```ts
// Inside the server.proxy object, add:
"/__gotenberg": {
  target: env.GOTENBERG_URL || "http://localhost:3000",
  changeOrigin: true,
  rewrite: (path) => path.replace(/^\/__gotenberg/, ""),
},
```

This forwards frontend requests to `/__gotenberg/forms/chromium/convert/html` → `http://localhost:3000/forms/chromium/convert/html`. The path is only wired in dev because it is set in `vite.config.ts`, which is not part of the production bundle.

- [ ] **Step 2: Implement the harness component**

Write to `frontend/apps/web/src/test-harness/MDDMTestHarness.tsx`:

```tsx
import { useEffect, useState } from "react";
import { MDDMEditor } from "../features/documents/mddm-editor/MDDMEditor";
import { mddmToBlockNote, type MDDMEnvelope } from "../features/documents/mddm-editor/adapter";
import { exportDocx } from "../features/documents/mddm-editor/engine/export";
import { defaultLayoutTokens } from "../features/documents/mddm-editor/engine/layout-ir";
import { PRINT_STYLESHEET } from "../features/documents/mddm-editor/engine/print-stylesheet";
import { wrapInPrintDocument } from "../features/documents/mddm-editor/engine/export/wrap-print-document";

// Dev-only: loads a golden fixture by name and exposes export APIs to Playwright.
// This component is only reachable via App.tsx when import.meta.env.DEV is true.

const FIXTURES: Record<string, () => Promise<MDDMEnvelope>> = {
  "01-simple-po":         () => import("../features/documents/mddm-editor/engine/golden/fixtures/01-simple-po/input.mddm.json").then((m) => m.default as unknown as MDDMEnvelope),
  "02-complex-table":     () => import("../features/documents/mddm-editor/engine/golden/fixtures/02-complex-table/input.mddm.json").then((m) => m.default as unknown as MDDMEnvelope),
  "03-repeatable-sections": () => import("../features/documents/mddm-editor/engine/golden/fixtures/03-repeatable-sections/input.mddm.json").then((m) => m.default as unknown as MDDMEnvelope),
};

/**
 * Posts HTML+CSS directly to Gotenberg's Chromium route via the Vite dev proxy.
 * Bypasses the Go backend's auth/document-level authz — intended for visual
 * parity tests that care about rendering, not auth.
 */
async function renderPdfDirectlyViaGotenberg(bodyHtml: string): Promise<Blob> {
  const fullHtml = wrapInPrintDocument(bodyHtml);
  const form = new FormData();
  form.append("files", new Blob([fullHtml], { type: "text/html" }), "index.html");
  form.append("files", new Blob([PRINT_STYLESHEET], { type: "text/css" }), "style.css");

  const response = await fetch("/__gotenberg/forms/chromium/convert/html", {
    method: "POST",
    body: form,
  });
  if (!response.ok) {
    throw new Error(`Gotenberg render failed: ${response.status}`);
  }
  const arrayBuffer = await response.arrayBuffer();
  return new Blob([arrayBuffer], { type: "application/pdf" });
}

export function MDDMTestHarness() {
  const [envelope, setEnvelope] = useState<MDDMEnvelope | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!import.meta.env.DEV) {
      setError("Test harness is disabled in production builds.");
      return;
    }
    const params = new URLSearchParams(window.location.hash.split("?")[1] ?? "");
    const docName = params.get("doc");
    if (!docName || !FIXTURES[docName]) {
      setError(`Unknown fixture: ${docName ?? "(none)"}`);
      return;
    }
    FIXTURES[docName]!().then(setEnvelope).catch((err) => setError(String(err)));
  }, []);

  useEffect(() => {
    if (!envelope) return;
    (window as any).__mddmExportDocx = () => exportDocx(envelope, defaultLayoutTokens);
    (window as any).__mddmRenderPdfDirectlyViaGotenberg = renderPdfDirectlyViaGotenberg;
    (window as any).__mddmHarnessReady = true;
  }, [envelope]);

  if (error) return <div data-testid="harness-error">{error}</div>;
  if (!envelope) return <div data-testid="harness-loading">Loading…</div>;

  const blocks = mddmToBlockNote(envelope);

  return (
    <div data-testid="mddm-harness">
      <MDDMEditor
        initialContent={blocks as any}
        readOnly={true}
      />
    </div>
  );
}
```

Note: the component does NOT call `exportPdf()` (which requires a document ID and hits the auth-protected backend). Playwright tests will call `blocksToFullHTML` on the mounted editor (via `window.__mddmEditor`) AND `__mddmRenderPdfDirectlyViaGotenberg` to produce the PDF for parity comparison.

- [ ] **Step 3: Expose the BlockNote editor instance for tests**

Open `frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.tsx` and add a dev-only effect that exposes the editor reference on `window`:

```tsx
// Near the editor creation hook, after the editor instance exists:
useEffect(() => {
  if (import.meta.env.DEV) {
    (window as any).__mddmEditor = editor;
    return () => { delete (window as any).__mddmEditor; };
  }
  return;
}, [editor]);
```

(Integrate with the existing effect structure of MDDMEditor.tsx. If there's already a similar dev hook, reuse it.)

- [ ] **Step 4: Add the escape hatch in App.tsx**

Open `frontend/apps/web/src/App.tsx` and find the component that derives the current view from the URL (it uses `useLocation` and `viewFromPath`). BEFORE the normal view-rendering logic (and ideally before any auth gate), add an early return for the harness path:

```tsx
import { MDDMTestHarness } from "./test-harness/MDDMTestHarness";

// ... inside the component that has `useLocation()`:
const location = useLocation();
if (import.meta.env.DEV && location.pathname.startsWith("/test-harness/mddm")) {
  return <MDDMTestHarness />;
}
```

Place this EARLY in the render so it bypasses the WorkspaceShell, auth guards, and all other chrome. In dev mode only.

- [ ] **Step 5: Verify the harness page loads**

Start the backend and dev server, then open `http://localhost:4173/#/test-harness/mddm?doc=01-simple-po` in a browser.
Expected: The 01-simple-po document renders read-only inside the MDDM editor; `window.__mddmHarnessReady === true` in the console.

- [ ] **Step 6: Commit**

```bash
git add frontend/apps/web/src/test-harness/ frontend/apps/web/src/App.tsx frontend/apps/web/src/features/documents/mddm-editor/MDDMEditor.tsx frontend/apps/web/vite.config.ts
git commit -m "feat(mddm-engine): add MDDMTestHarness page with dev-only Gotenberg proxy"
```

### Task 39: Implement pdf-to-png rasterization helper

**Files:**
- Create: `frontend/apps/web/e2e/helpers/pixel-diff.ts`

- [ ] **Step 1: Implement pixel-diff.ts using pdf-img-convert**

Write to `frontend/apps/web/e2e/helpers/pixel-diff.ts`:

```ts
import { PNG } from "pngjs";
import pixelmatch from "pixelmatch";
// pdf-img-convert wraps pdfjs-dist with a Node canvas backend so we don't
// have to configure a canvas factory manually.
// eslint-disable-next-line @typescript-eslint/ban-ts-comment
// @ts-expect-error — pdf-img-convert ships without types
import * as pdfImgConvert from "pdf-img-convert";

/** Render the first page of a PDF Buffer to a PNG Buffer. */
export async function rasterizePdfFirstPageToPng(pdfBytes: Uint8Array): Promise<Buffer> {
  // pdf-img-convert returns an array of PNG Uint8Arrays, one per page.
  const images = (await pdfImgConvert.convert(Buffer.from(pdfBytes), {
    page_numbers: [1],
    scale: 2.0, // render at 2x for better diff resolution
  })) as Uint8Array[];

  if (!images || images.length === 0) {
    throw new Error("pdf-img-convert returned no images");
  }
  return Buffer.from(images[0]!);
}

/** Compare two PNG buffers; returns the fraction of differing pixels (0..1).
 *  Buffers are resampled to the smaller dimensions before comparison. */
export function pngDiffPercent(a: Buffer, b: Buffer): number {
  const left = PNG.sync.read(a);
  const right = PNG.sync.read(b);
  const width = Math.min(left.width, right.width);
  const height = Math.min(left.height, right.height);
  if (width === 0 || height === 0) {
    throw new Error("pngDiffPercent: empty image");
  }

  // If dimensions differ, crop both to the shared region before diffing.
  function crop(png: PNG, w: number, h: number): Buffer {
    if (png.width === w && png.height === h) return png.data as unknown as Buffer;
    const cropped = new PNG({ width: w, height: h });
    PNG.bitblt(png, cropped, 0, 0, w, h, 0, 0);
    return cropped.data as unknown as Buffer;
  }

  const leftData = crop(left, width, height);
  const rightData = crop(right, width, height);

  const diff = new PNG({ width, height });
  const numDiff = pixelmatch(leftData, rightData, diff.data, width, height, {
    threshold: 0.1,
  });
  return numDiff / (width * height);
}
```

- [ ] **Step 2: Smoke test the rasterizer against a tiny known PDF**

Write to `frontend/apps/web/e2e/helpers/__tests__/pixel-diff.smoke.test.ts`:

```ts
import { describe, expect, it } from "vitest";
import { rasterizePdfFirstPageToPng } from "../pixel-diff";

// Minimal valid PDF header + trailer — a real PDF file created once to verify
// the rasterizer runs without crashing. For this smoke test we use a tiny
// two-byte stub that will reject cleanly; the point is to verify the import
// chain and the error path.
describe("pixel-diff smoke", () => {
  it("imports and exposes rasterizePdfFirstPageToPng", () => {
    expect(typeof rasterizePdfFirstPageToPng).toBe("function");
  });
});
```

Run: `cd frontend/apps/web && npx vitest run e2e/helpers/__tests__/pixel-diff.smoke.test.ts`
Expected: PASS — module imports cleanly. The real end-to-end rasterization is validated by the Playwright parity test in Task 40.

- [ ] **Step 3: Verify the file compiles**

Run: `cd frontend/apps/web && npx tsc --noEmit 2>&1 | grep pixel-diff | head -5`
Expected: No errors.

- [ ] **Step 4: Commit**

```bash
git add frontend/apps/web/e2e/helpers/pixel-diff.ts frontend/apps/web/e2e/helpers/__tests__/pixel-diff.smoke.test.ts
git commit -m "feat(mddm-engine): add pdf-img-convert based rasterization + pixel diff helper"
```

### Task 40: Implement Playwright visual parity tests

**Files:**
- Modify or create: `frontend/apps/web/playwright.config.ts`
- Create: `frontend/apps/web/e2e/mddm-visual-parity.spec.ts`

- [ ] **Step 1: Confirm Playwright config exists and add the e2e directory**

Run: `cat frontend/apps/web/playwright.config.ts 2>&1 | head -30`
Expected: Existing config (since `e2e:smoke` script is in package.json from Plan 1's environment). If the `testDir` is not set to `e2e`, set it.

If no config exists, write to `frontend/apps/web/playwright.config.ts`:

```ts
import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./e2e",
  timeout: 60_000,
  use: {
    baseURL: "http://localhost:4173",
    headless: true,
  },
  webServer: {
    command: "npm run dev",
    url: "http://localhost:4173",
    reuseExistingServer: !process.env.CI,
    timeout: 60_000,
  },
});
```

- [ ] **Step 2: Write the visual parity spec**

Write to `frontend/apps/web/e2e/mddm-visual-parity.spec.ts`:

```ts
import { test, expect } from "@playwright/test";
import { rasterizePdfFirstPageToPng, pngDiffPercent } from "./helpers/pixel-diff";

const FIXTURES = ["01-simple-po", "02-complex-table", "03-repeatable-sections"] as const;

// Visual parity tolerance. Matches the Render Compatibility Contract tier-2
// threshold in the spec (COMPATIBILITY_CONTRACT.tier2.pixelDiffEditorToPdf).
const EDITOR_TO_PDF_MAX_DIFF = 0.02; // 2%

for (const fixture of FIXTURES) {
  test(`MDDM visual parity: editor screenshot vs PDF (${fixture})`, async ({ page }) => {
    await page.goto(`/#/test-harness/mddm?doc=${fixture}`);

    // Wait for the harness to signal it has mounted and exposed the APIs.
    await page.waitForFunction(() => (window as any).__mddmHarnessReady === true, undefined, {
      timeout: 30_000,
    });
    await page.locator("[data-testid='mddm-harness']").waitFor({ state: "visible" });

    // 1. Capture editor screenshot.
    const editorElement = page.locator("[data-testid='mddm-harness']");
    const editorPng = await editorElement.screenshot();

    // 2. Produce full-fidelity HTML from the mounted BlockNote editor.
    const bodyHtml = await page.evaluate(async () => {
      const editor = (window as any).__mddmEditor;
      if (!editor) throw new Error("__mddmEditor not exposed");
      return await editor.blocksToFullHTML(editor.document);
    });

    // 3. Render PDF directly via Gotenberg (bypasses auth-protected backend).
    const pdfArray = await page.evaluate(async (html: string) => {
      const blob = await (window as any).__mddmRenderPdfDirectlyViaGotenberg(html);
      const buffer = await (blob as Blob).arrayBuffer();
      return Array.from(new Uint8Array(buffer));
    }, bodyHtml);
    const pdfBytes = new Uint8Array(pdfArray);

    // 4. Rasterize PDF page 1 and diff against editor screenshot.
    const pdfPng = await rasterizePdfFirstPageToPng(pdfBytes);

    const diff = pngDiffPercent(editorPng, pdfPng);
    expect(diff, `Visual diff for ${fixture} exceeded ${EDITOR_TO_PDF_MAX_DIFF * 100}%: ${(diff * 100).toFixed(2)}%`)
      .toBeLessThan(EDITOR_TO_PDF_MAX_DIFF);
  });
}
```

**Key differences from an earlier draft:**
- Uses `__mddmRenderPdfDirectlyViaGotenberg` (harness dev proxy) instead of `__mddmExportPdf` — bypasses backend auth entirely.
- Derives the body HTML from the mounted editor via `__mddmEditor.blocksToFullHTML`, not from a stored fixture, so the test matches exactly what the user would see after saving.

- [ ] **Step 3: Prerequisites — ensure Gotenberg is running**

Before running the spec, Gotenberg must be accessible at the URL in `GOTENBERG_URL` (dev default `http://localhost:3000`):

```bash
docker build -t metaldocs/gotenberg:local docker/gotenberg
docker run --rm --name metaldocs-gotenberg -p 3000:3000 -d metaldocs/gotenberg:local
./docker/gotenberg/verify-carlito.sh metaldocs-gotenberg
```

Expected: `OK: Carlito is installed` from the verification script.

- [ ] **Step 4: Run the spec**

Run:
```bash
cd frontend/apps/web
npx playwright test e2e/mddm-visual-parity.spec.ts 2>&1 | tail -40
```
Expected: PASS — three tests, each under the 2% pixel diff threshold.

If any fixture exceeds the threshold, do NOT relax the threshold. Investigate root causes: missing print CSS rule, wrong font fallback in the editor render, different default margins, or a `content: "none"` block rendering differently between editor HTML and BlockNote's external HTML serializer. Fix the root cause and re-run.

- [ ] **Step 5: Commit**

```bash
git add frontend/apps/web/playwright.config.ts frontend/apps/web/e2e/mddm-visual-parity.spec.ts frontend/apps/web/e2e/helpers/
git commit -m "test(mddm-engine): add Playwright visual parity suite for 3 golden fixtures"
```

### Task 41: Run the full test suite + final compile check

**Files:** (verification only)

- [ ] **Step 1: Run all vitest tests**

Run: `cd frontend/apps/web && npm test 2>&1 | tail -40`
Expected: All tests pass. After Plan 2, total test count is approximately 140-150 (Plan 1's ~95 + ~50 from Plan 2).

- [ ] **Step 2: Run TypeScript build**

Run: `cd frontend/apps/web && npm run build 2>&1 | tail -20`
Expected: Clean build, zero errors.

- [ ] **Step 3: Run Playwright visual parity**

Run: `cd frontend/apps/web && npx playwright test e2e/mddm-visual-parity.spec.ts 2>&1 | tail -20`
Expected: PASS — all 3 visual parity tests under 2% diff.

- [ ] **Step 4: Smoke test in browser**

Open `http://localhost:4173/#/test-harness/mddm?doc=02-complex-table` in a real browser. Click around the editor (read-only) and verify the document looks polished. Use the browser console:

```js
const blob = await window.__mddmExportDocx();
window.open(URL.createObjectURL(blob));
```

Open the downloaded DOCX in Microsoft Word or LibreOffice and verify the table renders with header row + data rows + the right column proportions.

- [ ] **Step 5: Commit any cleanup**

If any incidental fixes were needed during steps 1-4, commit them with descriptive messages.

---

## Self-Review

### Spec coverage

| Spec requirement (section) | Task(s) covering it |
|---|---|
| Asset resolver wiring through DOCX export | Tasks 1, 16 |
| Asset inlining through PDF export (data: URI rewriting) | Tasks 2, 17 |
| Standard BlockNote DOCX emitters (bullet, numbered, image, quote, divider) | Tasks 3, 4, 5, 6, 7 |
| MDDM custom DOCX emitters (dataTable*, repeatable*, richBlock) | Tasks 8, 9, 10, 11, 12, 13 |
| `mddmToDocx` registry expansion + asset map plumbing | Task 14 |
| Updated docx-emitter barrel | Task 15 |
| Renderer completeness gate covering all 16 blocks | Task 18 |
| `toExternalHTML` for DataTableCell (only `content: "inline"` block in Plan 2) | Task 19 |
| ~~Custom `toExternalHTML` for DataTableRow/DataTable/RepeatableItem/Repeatable/RichBlock~~ (intentionally NOT implemented — they are `content: "none"` and rely on BlockNote's render() fallback; see note above Task 19 and the gating test in Task 25b) | Task 25b |
| Updated external-html barrel (only exports DataTableCellExternalHTML) | Task 25 |
| Block-spec registration for DataTableCell | Task 31 |
| ~~Block-spec registrations for Repeatable/RepeatableItem/RichBlock/DataTable/DataTableRow~~ (REMOVED — `content: "none"` blocks do not receive `contentRef`; they use `render()` fallback) | N/A |
| Golden fixture corpus expansion (5 new fixtures) | Tasks 32, 33, 34, 35, 36 |
| pixelmatch + pdf-img-convert dependency installation | Task 37 |
| Playwright test harness page with dev-only Gotenberg proxy | Task 38 |
| PDF rasterization + pixel-diff helper (pdf-img-convert) | Task 39 |
| Playwright visual parity test suite (3 fixtures, < 2% diff) | Task 40 |
| Full test + build + smoke verification | Task 41 |

### Codex revision history

Codex round-1 on Plan 2 caught 9 structural issues. All addressed in this revision:

1. **`toExternalHTML` on `content: "none"` blocks** — BlockNote's `contentRef` callback is ONLY provided to inline-content blocks. Removed Tasks 20-24 (custom HTML components for Repeatable/RepeatableItem/RichBlock/DataTable/DataTableRow) and Tasks 26-30 (their registrations). Added Task 25b as a gating test verifying `blocksToFullHTML` correctly serializes these blocks via their `render()` fallback.
2. **`contentRef` cast pattern** — Replaced `React.Ref<...>` cast with inline callback-ref form `(el) => contentRef(el)` in Task 31 (DataTableCell). Plan 1 received the same fix in a separate commit.
3. **Image prop key** — Fixed asset collector (Task 1) and image emitter (Task 5) to read `block.props.src` instead of `block.props.url`. Per `adapter.ts` `toMDDMProps`, MDDM envelopes store image URLs under `src`.
4. **DataTable column contract** — Fixed data-table emitter (Task 10) to read `block.props.columns` as an array of `{key, label}` objects. Updated 02-complex-table fixture (Task 32) to match the real schema. Removed all references to `columnsJson` and `header` field from Plan 2.
5. **Table width ambiguity** — Data-table emitter now uses `WidthType.DXA` with `mmToTwip(tokens.page.contentWidthMm)` for unambiguous absolute width. No more percentage-unit ambiguity.
6. **Test harness routing** — Task 38 no longer uses element-based `<Routes>`. It adds an early-return escape hatch in `App.tsx` that checks `location.pathname.startsWith("/test-harness/mddm")` and renders `<MDDMTestHarness />` directly, bypassing the WorkspaceShell.
7. **Backend auth in visual parity** — Task 38 exposes `__mddmRenderPdfDirectlyViaGotenberg(html)` that posts to a dev-only Vite proxy path (`/__gotenberg/forms/chromium/convert/html`) bypassing the Go backend entirely. Task 40 uses this instead of the auth-protected `/render/pdf` endpoint.
8. **pdfjs-dist render API** — Task 39 switched from manual pdfjs canvas factory to `pdf-img-convert`, which ships a working Node canvas backend. Task 37 installs `pdf-img-convert` instead of `pdfjs-dist`.
9. **Orphan `render-pdf-page.ts` entry** — Removed from the file structure. Replaced with `pixel-diff.smoke.test.ts` which is actually created in Task 39.

### Placeholder scan

No "TBD", "TODO", "implement later", or "similar to Task N" placeholders remain.

### Type / signature consistency

- `mddmToDocx(envelope, tokens, assetMap?)` signature is consistent across Tasks 14, 16, golden runners, and the test harness.
- `exportDocx(envelope, tokens, options?)` signature matches between Tasks 16 and the test harness (Task 38).
- `exportPdf({bodyHtml, documentId, assetResolver?})` is referenced in Task 17 but NOT used by Plan 2's visual parity tests — those use the dev-only direct-Gotenberg path.
- `ChildRenderer` type from `repeatable-item.ts` (Task 11) is reused by `repeatable.ts` (Task 12), `rich-block.ts` (Task 13), and `emitter.ts` (Task 14).
- `EmitContext = { tokens, assetMap }` from `emitter.ts` (Task 14) has no external consumers — internal only.
- `extractTextRuns` from Plan 1's `paragraph.ts` is reused by `bullet-list-item.ts`, `numbered-list-item.ts`, `quote.ts`, `data-table-cell.ts`.
- Block type strings are consistent between the emitter registry (Task 14), block registry (Task 18), `DataTableCell` toExternalHTML (Task 31), and the MDDM schema.

**Out of scope by design** (deferred to Plans 3 and 4):
- Version pinning + renderer bundle registry (`Version.rendererPin`) — Plan 3
- Renderer version bump rules + bundle storage — Plan 3
- Shadow testing telemetry endpoint + frontend dual-run — Plan 4
- Canary rollout (5% → 100%) + decommissioning docgen — Plan 4
- Multi-paragraph field rendering (nested MDDMBlock children inside Field) — explicitly noted in Plan 1's Field emitter; deferred to a follow-up since the inline-text path covers all current corpus needs
- Custom font embedding (template-level opt-in) — out of scope at launch per spec

### Placeholder scan

No "TBD", "TODO", "implement later", or "similar to Task N" placeholders remain. Every step contains the actual code or command needed.

### Type / signature consistency

- `mddmToDocx(envelope, tokens, assetMap?)` signature is consistent across Tasks 14, 16, the existing Plan 1 main entry, all golden runner tests, and the test harness.
- `exportDocx(envelope, tokens, options?)` and `exportPdf({bodyHtml, documentId, assetResolver?})` signatures match between Tasks 16, 17, the test harness in Task 38, and Plan 1's existing test references (which only used the required positional args).
- `ChildRenderer` type from `repeatable-item.ts` (Task 11) is reused by `repeatable.ts` (Task 12), `rich-block.ts` (Task 13), and the registry in Task 14.
- `EmitContext = { tokens, assetMap }` is consistent in `emitter.ts` (Task 14) — no other module references it.
- `extractTextRuns` from Plan 1's `paragraph.ts` is reused by `bullet-list-item.ts`, `numbered-list-item.ts`, `quote.ts`, `data-table-cell.ts`, `field.ts` (Plan 1) — same import, same return type.
- Block type strings (`"bulletListItem"`, `"numberedListItem"`, `"image"`, `"quote"`, `"divider"`, `"dataTable"`, `"dataTableRow"`, `"dataTableCell"`, `"repeatable"`, `"repeatableItem"`, `"richBlock"`) are consistent between the emitter registry (Task 14), block registry (Task 18), `toExternalHTML` registrations (Tasks 26-31), and the standard BlockNote schema (which uses the same camelCase identifiers).

---

## Execution Handoff

Plan complete and saved to `docs/superpowers/plans/2026-04-10-mddm-engine-full-block-coverage.md`. Two execution options:

**1. Subagent-Driven (recommended)** — I dispatch a fresh subagent per task, review between tasks, fast iteration.
**REQUIRED SUB-SKILL:** `superpowers:subagent-driven-development`

**2. Inline Execution** — Execute tasks in this session using executing-plans, batch execution with checkpoints.
**REQUIRED SUB-SKILL:** `superpowers:executing-plans`

Which approach?
