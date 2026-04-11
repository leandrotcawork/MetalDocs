import { describe, expect, it } from "vitest";
import { normalizeVersionItem } from "../documents";
import type { VersionListItem, RendererPin } from "../../lib.types";

describe("normalizeVersionItem", () => {
  it("preserves renderer_pin when present", () => {
    const pin: RendererPin = {
      renderer_version: "1.0.0",
      layout_ir_hash: "h",
      template_key: "k",
      template_version: 1,
      pinned_at: "2026-04-10T12:00:00Z",
    };
    const input = {
      documentId: "doc-1",
      version: 1,
      contentHash: "ch",
      changeSummary: "cs",
      createdAt: "2026-04-10T00:00:00Z",
      renderer_pin: pin,
    } as unknown as VersionListItem;

    const out = normalizeVersionItem(input);
    expect(out.renderer_pin).toEqual(pin);
  });

  it("sets renderer_pin to null when missing from input", () => {
    const input = {
      documentId: "doc-2",
      version: 1,
      contentHash: "ch",
      changeSummary: "",
      createdAt: "2026-04-10T00:00:00Z",
    } as unknown as VersionListItem;

    const out = normalizeVersionItem(input);
    expect(out.renderer_pin).toBeNull();
  });
});
