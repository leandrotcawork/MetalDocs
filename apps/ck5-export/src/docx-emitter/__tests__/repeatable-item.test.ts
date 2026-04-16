import { describe, expect, it } from "vitest";
import { Table } from "docx";
import { emitRepeatableItem } from "../emitters/repeatable-item";
import { defaultLayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../shared/adapter";

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

