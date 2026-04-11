import { describe, expect, it } from "vitest";
import { wrapInPrintDocument } from "../wrap-print-document";

describe("wrapInPrintDocument", () => {
  it("wraps body HTML in a full HTML document with DOCTYPE", () => {
    const result = wrapInPrintDocument("<p>hi</p>");
    expect(result).toContain("<!DOCTYPE html>");
    expect(result).toContain("<html");
    expect(result).toContain("<body");
    expect(result).toContain("<p>hi</p>");
  });

  it("injects the print stylesheet in <style>", () => {
    const result = wrapInPrintDocument("<p>x</p>");
    expect(result).toContain("<style");
    expect(result).toContain("@page");
    expect(result).toContain("Carlito");
  });

  it("sets UTF-8 meta charset", () => {
    const result = wrapInPrintDocument("<p>x</p>");
    expect(result).toContain('charset="UTF-8"');
  });
});
