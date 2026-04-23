import { describe, expect, test } from "vitest";
import { FooterPageNumbers } from "../footer_page_numbers";

describe("FooterPageNumbers", () => {
  test("emits PAGE and NUMPAGES field codes with 'Page X of Y' text", async () => {
    const ooxml = await FooterPageNumbers.render({ params: {}, values: {} });

    expect(ooxml).toContain(`<w:fldSimple w:instr="PAGE">`);
    expect(ooxml).toContain(`<w:fldSimple w:instr="NUMPAGES">`);
    expect(ooxml).toContain("Page ");
    expect(ooxml).toContain(" of ");
  });
});
