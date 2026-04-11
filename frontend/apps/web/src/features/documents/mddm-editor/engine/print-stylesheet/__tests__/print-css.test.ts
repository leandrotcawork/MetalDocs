import { describe, expect, it } from "vitest";
import { PRINT_STYLESHEET } from "../print-css";

describe("PRINT_STYLESHEET", () => {
  it("has @page with A4 size", () => {
    expect(PRINT_STYLESHEET).toContain("@page");
    expect(PRINT_STYLESHEET).toContain("A4");
  });

  it("has Carlito font reference", () => {
    expect(PRINT_STYLESHEET).toContain("Carlito");
  });

  it("has @media print with -webkit-print-color-adjust: exact", () => {
    expect(PRINT_STYLESHEET).toContain("@media print");
    expect(PRINT_STYLESHEET).toContain("-webkit-print-color-adjust: exact");
  });

  it("hides .bn-side-menu", () => {
    expect(PRINT_STYLESHEET).toContain(".bn-side-menu");
    expect(PRINT_STYLESHEET).toContain("display: none");
  });
});
