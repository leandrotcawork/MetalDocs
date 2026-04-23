import { describe, expect, test } from "vitest";
import { RevisionBox } from "../revision_box";

describe("RevisionBox", () => {
  test("renders one row per revision_history entry", async () => {
    const ooxml = await RevisionBox.render({
      params: {},
      values: {
        revision_history: [
          { rev: "1", date: "2026-01-01", description: "Initial release" },
          { rev: "2", date: "2026-03-15", description: "Added section 4" },
        ],
      },
    });

    expect(ooxml).toContain("<w:tbl>");
    expect(ooxml).toContain("Initial release");
    expect(ooxml).toContain("Added section 4");
    expect(ooxml).toContain("2026-01-01");
    expect(ooxml.match(/<w:tr>/g)?.length).toBe(3);
  });

  test("empty history renders placeholder row with em-dashes", async () => {
    const ooxml = await RevisionBox.render({
      params: {},
      values: { revision_history: [] },
    });

    expect(ooxml).toContain("<w:tbl>");
    expect(ooxml.match(/<w:tr>/g)?.length).toBe(2);
    expect(ooxml).toContain("—");
  });

  test("missing revision_history treated as empty", async () => {
    const ooxml = await RevisionBox.render({ params: {}, values: {} });
    expect(ooxml).toContain("<w:tbl>");
    expect(ooxml).toContain("—");
  });
});
