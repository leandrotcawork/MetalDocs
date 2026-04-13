import { describe, it, expect } from "vitest";
import { RepeatableItemCodec } from "../repeatable-item-codec";

describe("RepeatableItemCodec.parseCaps", () => {
  it("parses editableZones", () => {
    const caps = RepeatableItemCodec.parseCaps(JSON.stringify({ editableZones: ["content"] }));
    expect(caps.editableZones).toEqual(["content"]);
  });
  it("defaults editableZones to ['content']", () => {
    const caps = RepeatableItemCodec.parseCaps("{}");
    expect(caps.editableZones).toEqual(["content"]);
  });
});
