import { describe, expect, it } from "vitest";
import { Table } from "docx";
import { emitSection } from "../emitters/section";
import { defaultLayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../shared/adapter";

describe("emitSection", () => {
  it("emits a full-width Table wrapping the section header", () => {
    const block: MDDMBlock = {
      id: "s1",
      type: "section",
      props: { title: "1. Procedimento", color: "red" },
      children: [],
    };
    const out = emitSection(block, defaultLayoutTokens);
    expect(out).toHaveLength(1);
    expect(out[0]).toBeInstanceOf(Table);
  });

  it("uses the token accent color for header background", () => {
    const block: MDDMBlock = {
      id: "s2",
      type: "section",
      props: { title: "Header" },
      children: [],
    };
    const tokens = {
      ...defaultLayoutTokens,
      theme: { ...defaultLayoutTokens.theme, accent: "#123456" },
    };
    const out = emitSection(block, tokens);
    const tableOptions = (out[0] as any).options;
    const firstRow = tableOptions.rows[0];
    const firstCell = firstRow.options.children[0];
    expect(firstCell.options.shading.fill).toBe("123456");
  });

  it("renders empty title when title prop is missing", () => {
    const block: MDDMBlock = {
      id: "s3",
      type: "section",
      props: {},
      children: [],
    };
    const out = emitSection(block, defaultLayoutTokens);
    expect(out).toHaveLength(1);
  });
});

