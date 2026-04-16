import { describe, expect, it } from "vitest";
import { wrapInPrintDocument } from "../wrap-print-document";
import { PRINT_STYLESHEET } from "../print-css";

describe("wrapInPrintDocument", () => {
  it("wraps body HTML in a full HTML document with DOCTYPE", () => {
    const result = wrapInPrintDocument("<p>hi</p>");
    expect(result).toContain("<!DOCTYPE html>");
    expect(result).toContain("<html");
    expect(result).toContain("<body");
    expect(result).toContain("<p>hi</p>");
  });

  it("contains PRINT_STYLESHEET verbatim", () => {
    const result = wrapInPrintDocument("<p>hello</p>");
    expect(result).toContain(PRINT_STYLESHEET);
  });

  it("sets UTF-8 meta charset", () => {
    const result = wrapInPrintDocument("<p>x</p>");
    expect(result).toContain('charset="UTF-8"');
  });

  it("includes viewport meta tag", () => {
    const result = wrapInPrintDocument("<p>x</p>");
    expect(result).toContain('name="viewport"');
    expect(result).toContain("width=device-width");
  });
});
