import { describe, expect, it, vi } from "vitest";

vi.mock("@blocknote/react", () => ({
  createReactBlockSpec: vi.fn((config: object, spec: object) => ({
    config,
    ...spec,
  })),
}));

import { findItemIndex } from "../RepeatableItem";

type TestBlock = {
  id: string;
  type: string;
  props?: Record<string, unknown>;
  children?: TestBlock[];
};

describe("RepeatableItem numbering", () => {
  it("returns flat repeatable item indexes as 1, 2, and 3", () => {
    const item1: TestBlock = { id: "item-1", type: "repeatableItem", children: [] };
    const item2: TestBlock = { id: "item-2", type: "repeatableItem", children: [] };
    const item3: TestBlock = { id: "item-3", type: "repeatableItem", children: [] };
    const document: TestBlock[] = [
      {
        id: "repeatable-flat",
        type: "repeatable",
        children: [item1, item2, item3],
      },
    ];

    expect(findItemIndex(document, item1.id)).toBe(1);
    expect(findItemIndex(document, item2.id)).toBe(2);
    expect(findItemIndex(document, item3.id)).toBe(3);
  });

  it("numbers outer and inner repeatables independently", () => {
    const inner1: TestBlock = { id: "inner-1", type: "repeatableItem", children: [] };
    const inner2: TestBlock = { id: "inner-2", type: "repeatableItem", children: [] };
    const inner3: TestBlock = { id: "inner-3", type: "repeatableItem", children: [] };

    const outer1: TestBlock = {
      id: "outer-1",
      type: "repeatableItem",
      children: [
        {
          id: "nested-repeatable-1",
          type: "repeatable",
          children: [
            { id: "nested-helper-1", type: "paragraph", children: [] },
            inner1,
            inner2,
          ],
        },
      ],
    };

    const outer2: TestBlock = {
      id: "outer-2",
      type: "repeatableItem",
      children: [
        {
          id: "nested-repeatable-2",
          type: "repeatable",
          children: [
            { id: "nested-helper-2", type: "paragraph", children: [] },
            inner3,
          ],
        },
      ],
    };

    const document: TestBlock[] = [
      {
        id: "outer-repeatable",
        type: "repeatable",
        children: [
          { id: "outer-helper", type: "paragraph", children: [] },
          outer1,
          outer2,
        ],
      },
    ];

    expect(findItemIndex(document, outer1.id)).toBe(1);
    expect(findItemIndex(document, outer2.id)).toBe(2);
    expect(findItemIndex(document, inner1.id)).toBe(1);
    expect(findItemIndex(document, inner2.id)).toBe(2);
    expect(findItemIndex(document, inner3.id)).toBe(1);
  });
});
