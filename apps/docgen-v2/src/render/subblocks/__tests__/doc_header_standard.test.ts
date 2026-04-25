import { describe, expect, test } from "vitest";
import { DocHeaderStandard } from "../doc_header_standard";

describe("DocHeaderStandard", () => {
  test("emits title, doc code, effective date, and revision number from values", async () => {
    const ooxml = await DocHeaderStandard.render({
      params: {},
      values: {
        title: "Purchase Order Procedure",
        doc_code: "QMS-0001",
        effective_date: "2026-04-21",
        revision_number: 3,
      },
    });

    expect(ooxml).toContain("Purchase Order Procedure");
    expect(ooxml).toContain("QMS-0001");
    expect(ooxml).toContain("2026-04-21");
    expect(ooxml).toContain(">3<");
    expect(ooxml).toContain("<w:tbl>");
  });

  test("renders empty strings for missing values without throwing", async () => {
    const ooxml = await DocHeaderStandard.render({ params: {}, values: {} });
    expect(ooxml).toContain("<w:tbl>");
  });
});
