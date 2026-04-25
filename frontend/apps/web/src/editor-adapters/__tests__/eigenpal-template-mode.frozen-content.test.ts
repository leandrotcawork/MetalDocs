import { describe, expect, test } from "vitest";

import { wrapFrozenContent } from "../eigenpal-template-mode";

describe("wrapFrozenContent", () => {
  test("wraps blocks in locked blockSdt", () => {
    const block = { type: "paragraph", content: [] } as any;
    const result = wrapFrozenContent([block]);
    expect(result.type).toBe("blockSdt");
    expect(result.properties.lock).toBe("sdtContentLocked");
    expect(result.content).toEqual([block]);
  });
});
