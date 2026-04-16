import { describe, expect, it } from "vitest";
import { Table } from "docx";
import { emitFieldGroup } from "../emitters/field-group";
import { defaultLayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../shared/adapter";

function makeField(id: string, label: string): MDDMBlock {
  return { id, type: "field", props: { label }, children: [] };
}

describe("emitFieldGroup", () => {
  it("emits a single outer Table wrapping child fields", () => {
    const block: MDDMBlock = {
      id: "fg1",
      type: "fieldGroup",
      props: { columns: 2 },
      children: [makeField("f1", "A"), makeField("f2", "B")],
    };
    const out = emitFieldGroup(block, defaultLayoutTokens);
    expect(out).toHaveLength(1);
    expect(out[0]).toBeInstanceOf(Table);
  });

  it("arranges two fields side-by-side for columns=2", () => {
    const block: MDDMBlock = {
      id: "fg2",
      type: "fieldGroup",
      props: { columns: 2 },
      children: [makeField("f1", "A"), makeField("f2", "B")],
    };
    const out = emitFieldGroup(block, defaultLayoutTokens);
    const rows = (out[0] as any).options.rows;
    expect(rows).toHaveLength(1);
    expect(rows[0].options.children).toHaveLength(2);
  });

  it("stacks fields vertically for columns=1", () => {
    const block: MDDMBlock = {
      id: "fg3",
      type: "fieldGroup",
      props: { columns: 1 },
      children: [makeField("f1", "A"), makeField("f2", "B")],
    };
    const out = emitFieldGroup(block, defaultLayoutTokens);
    const rows = (out[0] as any).options.rows;
    expect(rows).toHaveLength(2);
  });
});

