import { describe, expect, it } from "vitest";
import { emitRepeatable } from "../emitters/repeatable";
import { defaultLayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../shared/adapter";

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
    // Header paragraph + 2 repeatable-item tables = exactly 3 elements
    expect(out.length).toBe(3);
  });

  it("emits only the header when there are no items", () => {
    const block: MDDMBlock = {
      id: "rp2",
      type: "repeatable",
      props: { label: "Empty" },
      children: [],
    };
    const out = emitRepeatable(block, defaultLayoutTokens, () => []);
    // Only the header paragraph â€” exactly 1 element
    expect(out.length).toBe(1);
  });
});

