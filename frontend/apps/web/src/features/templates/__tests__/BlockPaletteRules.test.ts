import { describe, test, expect } from "vitest";
import { canInsertBlock } from "../block-palette-rules";

describe("canInsertBlock", () => {
  test("section can insert at root level", () => {
    expect(canInsertBlock("section", null)).toBeNull();
  });

  test("section can insert from nested context (palette inserts it at root)", () => {
    expect(canInsertBlock("section", "section", "richBlock")).toBeNull();
  });

  test("richBlock needs section context", () => {
    expect(canInsertBlock("richBlock", null, null)).not.toBeNull();
    expect(canInsertBlock("richBlock", "section", "paragraph")).toBeNull();
  });

  test("dataTable can insert when section itself is selected", () => {
    expect(canInsertBlock("dataTable", null, "section")).toBeNull();
  });

  test("unknown block type is rejected", () => {
    expect(canInsertBlock("UNKNOWN", null)).not.toBeNull();
  });
});
