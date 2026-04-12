import { describe, expect, it } from "vitest";
import { BLOCK_REGISTRY, getFullySupportedBlockTypes } from "../block-registry";
import { mddmToDocx, MissingEmitterError, REGISTERED_EMITTER_TYPES } from "../../docx-emitter";
import { defaultLayoutTokens } from "../../layout-ir";
import type { MDDMEnvelope, MDDMBlock } from "../../../adapter";

// Minimal valid fixture for each block type so the DOCX emitter can exercise it.
function makeMinimalBlock(type: string): MDDMBlock {
  switch (type) {
    case "paragraph":
      return { id: `test-${type}`, type, props: {}, children: [{ type: "text", text: "x" }] };
    case "heading":
      return { id: `test-${type}`, type, props: { level: 1 }, children: [{ type: "text", text: "x" }] };
    case "bulletListItem":
    case "numberedListItem":
      return { id: `test-${type}`, type, props: {}, children: [{ type: "text", text: "x" }] };
    case "quote":
      return { id: `test-${type}`, type, props: {}, children: [{ type: "text", text: "x" }] };
    case "divider":
      return { id: `test-${type}`, type, props: {}, children: [] };
    case "image":
      return { id: `test-${type}`, type, props: { url: "", caption: "" }, children: [] };
    case "section":
      return { id: `test-${type}`, type, props: { title: "T" }, children: [] };
    case "field":
      return { id: `test-${type}`, type, props: { label: "L" }, children: [{ type: "text", text: "v" }] };
    case "fieldGroup":
      return {
        id: `test-${type}`, type, props: { columns: 2 },
        children: [{ id: "nested-f1", type: "field", props: { label: "A" }, children: [] }],
      };
    case "dataTable":
      return {
        id: `test-${type}`, type,
        props: { label: "T", locked: false, density: "normal" },
        content: {
          type: "tableContent",
          columnWidths: [null, null],
          headerRows: 1,
          rows: [
            { cells: [[{ type: "text", text: "A" }], [{ type: "text", text: "B" }]] },
            { cells: [[{ type: "text", text: "1" }], [{ type: "text", text: "2" }]] },
          ],
        },
        children: [],
      } as unknown as MDDMBlock;
    case "repeatable":
      return {
        id: `test-${type}`, type,
        props: { label: "Items", itemPrefix: "Item", locked: false },
        children: [
          {
            id: "nested-ri1", type: "repeatableItem",
            props: { title: "Item 1", style: "bordered" },
            children: [{ id: "nested-p1", type: "paragraph", props: {}, children: [{ type: "text", text: "x" }] }],
          },
        ],
      };
    case "repeatableItem":
      return {
        id: `test-${type}`, type,
        props: { title: "Item 1", style: "bordered" },
        children: [{ id: "nested-p2", type: "paragraph", props: {}, children: [{ type: "text", text: "x" }] }],
      };
    case "richBlock":
      return {
        id: `test-${type}`, type,
        props: { label: "Notes", chrome: "labeled" },
        children: [{ id: "nested-p3", type: "paragraph", props: {}, children: [{ type: "text", text: "x" }] }],
      };
    default:
      return { id: `test-${type}`, type, props: {}, children: [] };
  }
}

describe("Renderer completeness gate", () => {
  it("includes every MVP block type as fully supported", () => {
    const supported = getFullySupportedBlockTypes();
    const required = [
      "paragraph", "heading", "section", "field", "fieldGroup",
      "bulletListItem", "numberedListItem", "quote", "divider",
      "dataTable", "repeatable", "repeatableItem", "richBlock",
    ];
    for (const type of required) {
      expect(supported, `${type} missing from fully-supported list`).toContain(type);
    }
  });

  it("DOCX emitter produces output for every fully-supported block type", async () => {
    for (const type of getFullySupportedBlockTypes()) {
      const block = makeMinimalBlock(type);
      const envelope: MDDMEnvelope = {
        mddm_version: 2,
        template_ref: null,
        blocks: [block],
      };
      await expect(
        mddmToDocx(envelope, defaultLayoutTokens),
        `DOCX emitter failed for block type "${type}"`,
      ).resolves.toBeInstanceOf(Blob);
    }
  });

  it("DOCX emitter throws MissingEmitterError for unsupported types in the registry", async () => {
    const unsupported = BLOCK_REGISTRY.filter((b) => !b.hasDocxEmitter).map((b) => b.type);
    for (const type of unsupported) {
      const envelope: MDDMEnvelope = {
        mddm_version: 2,
        template_ref: null,
        blocks: [{ id: "x", type, props: {}, children: [] } as unknown as MDDMBlock],
      };
      await expect(mddmToDocx(envelope, defaultLayoutTokens)).rejects.toBeInstanceOf(MissingEmitterError);
    }
  });

  it("block-registry hasDocxEmitter flags exactly match the actual emitter registration", () => {
    const registeredInEmitter = new Set(REGISTERED_EMITTER_TYPES);
    const registeredInRegistry = new Set(
      BLOCK_REGISTRY.filter((b) => b.hasDocxEmitter).map((b) => b.type),
    );

    for (const type of registeredInEmitter) {
      expect(registeredInRegistry.has(type), `${type} is in emitter but not in registry with hasDocxEmitter:true`).toBe(true);
    }

    for (const type of registeredInRegistry) {
      expect(registeredInEmitter.has(type), `${type} is in registry with hasDocxEmitter:true but not in emitter`).toBe(true);
    }
  });
});
