import { describe, expect, test } from "vitest";

import {
  extractZones,
  wrapZone,
  type BlockNode,
  type EditableZone,
} from "../eigenpal-template-mode";

function paragraph(text: string): BlockNode {
  return {
    type: "paragraph",
    content: [{ type: "run", content: [{ type: "text", text }] }],
  };
}

describe("zone round-trip", () => {
  test("single zone preserves id and inner blocks", () => {
    const zone: EditableZone = { id: "intro", label: "Intro" };
    const inner = [paragraph("hello"), paragraph("world")];

    const wrapped = wrapZone(zone, inner, 1);
    const extracted = extractZones(wrapped);

    expect(extracted).toHaveLength(1);
    expect(extracted[0].zone.id).toBe("intro");
    expect(extracted[0].blocks).toEqual(inner);
  });

  test("two adjacent zones preserve ids, ordering, and per-zone content", () => {
    const introInner = [paragraph("intro-body")];
    const bodyInner = [paragraph("body-1"), paragraph("body-2")];

    const wrappedIntro = wrapZone({ id: "intro", label: "Intro" }, introInner, 1);
    const wrappedBody = wrapZone({ id: "body", label: "Body" }, bodyInner, 2);
    const content = [...wrappedIntro, ...wrappedBody];

    const extracted = extractZones(content);

    expect(extracted.map((z) => z.zone.id)).toEqual(["intro", "body"]);
    expect(extracted[0].blocks).toEqual(introInner);
    expect(extracted[1].blocks).toEqual(bodyInner);
  });

  test("unmatched end bookmark is ignored (no zone returned)", () => {
    const zone: EditableZone = { id: "orphan", label: "Orphan" };
    const wrapped = wrapZone(zone, [paragraph("x")], 7);
    const truncated = wrapped.slice(0, wrapped.length - 1);

    expect(extractZones(truncated)).toHaveLength(0);
  });

  test("wrapZone rejects empty zone id", () => {
    expect(() => wrapZone({ id: "  ", label: "x" }, [], 1)).toThrow();
  });
});
