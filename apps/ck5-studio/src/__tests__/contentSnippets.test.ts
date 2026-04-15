import { describe, expect, it } from "vitest";
import { DEFAULT_EDITORIAL_HTML, snippetFor } from "../lib/contentSnippets";

describe("contentSnippets", () => {
  it("returns starter text snippet", () => {
    expect(snippetFor("text")).toContain("<p>");
  });

  it("returns starter heading snippet", () => {
    expect(snippetFor("heading")).toContain("<h2>");
  });

  it("returns section snippet with heading and body", () => {
    expect(snippetFor("section")).toContain("<h2>New Section</h2>");
    expect(snippetFor("section")).toContain("Describe this section.");
  });

  it("returns starter table markup", () => {
    expect(snippetFor("table")).toContain("<table>");
  });

  it("returns template note block with mixed lock/edit fields", () => {
    expect(snippetFor("note")).toContain("Template Note Block");
    expect(snippetFor("note")).toContain("restricted-editing-exception");
    expect(snippetFor("note")).toContain("CK5-TEMPLATE-BLOCK-NOTE");
  });

  it("returns mixed section with locked header and editable body", () => {
    expect(snippetFor("mixed")).toContain("Section 2 - Locked Header");
    expect(snippetFor("mixed")).toContain("Editable body:");
    expect(snippetFor("mixed")).toContain('<div class="restricted-editing-exception">');
  });

  it("seeds restricted editing exception markers for fill mode", () => {
    expect(DEFAULT_EDITORIAL_HTML).toContain("restricted-editing-exception");
  });

  it("includes mixed section in default template", () => {
    expect(DEFAULT_EDITORIAL_HTML).toContain("Section 2 - Locked Header");
    expect(DEFAULT_EDITORIAL_HTML).toContain('<div class="restricted-editing-exception">');
  });
});
