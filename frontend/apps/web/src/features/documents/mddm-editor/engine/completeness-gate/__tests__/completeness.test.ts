import { describe, expect, it } from "vitest";
import { BLOCK_REGISTRY, getFullySupportedBlockTypes } from "../block-registry";
import { mddmToDocx, MissingEmitterError, REGISTERED_EMITTER_TYPES } from "../../docx-emitter";
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
      let children: unknown[] = [];
      let props: Record<string, unknown> = {};

      if (type === "paragraph" || type === "heading" || type === "field") {
        children = [{ type: "text", text: "x" }];
        if (type === "field") props = { label: "L" };
        if (type === "heading") props = { level: 1 };
      } else if (type === "section") {
        props = { title: "T" };
      } else if (type === "fieldGroup") {
        props = { columns: 2 };
        children = [{ id: "nested-f1", type: "field", props: { label: "A" }, children: [] }];
      }

      const envelope: MDDMEnvelope = {
        mddm_version: 1,
        template_ref: null,
        blocks: [{ id: `test-${type}`, type, props, children } as any],
      };

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

  it("block-registry hasDocxEmitter flags exactly match the actual emitter registration", () => {
    const registeredInEmitter = new Set(REGISTERED_EMITTER_TYPES);
    const registeredInRegistry = new Set(
      BLOCK_REGISTRY.filter((b) => b.hasDocxEmitter).map((b) => b.type),
    );

    // Every type in the emitter should be marked hasDocxEmitter: true in registry
    for (const type of registeredInEmitter) {
      expect(registeredInRegistry.has(type), `${type} is in emitter but not in registry with hasDocxEmitter:true`).toBe(true);
    }

    // Every type marked hasDocxEmitter: true in registry should be in the emitter
    for (const type of registeredInRegistry) {
      expect(registeredInEmitter.has(type), `${type} is in registry with hasDocxEmitter:true but not in emitter`).toBe(true);
    }
  });
});
