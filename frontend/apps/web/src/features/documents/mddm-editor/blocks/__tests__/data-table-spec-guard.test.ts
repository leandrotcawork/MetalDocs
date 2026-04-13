import { describe, expect, it } from "vitest";

import { DataTable } from "../DataTable";

describe("DataTable block spec guard", () => {
  it("keeps the runtime table content patch and required props", () => {
    const spec = DataTable();
    const propSchemaKeys = Object.keys(spec.config.propSchema ?? {}).sort();

    expect(spec.config.type).toBe("dataTable");
    expect(spec.config.content).toBe("table");
    expect(propSchemaKeys).toEqual(expect.arrayContaining(["density", "label", "locked"]));
    expect(propSchemaKeys).toEqual(expect.arrayContaining(["__template_block_id"]));
    expect(propSchemaKeys.filter((key) => ["density", "label", "locked"].includes(key))).toEqual([
      "density",
      "label",
      "locked",
    ]);
  });

  it("provides a PM node with tableRow+ content so blockToNode can round-trip tableContent", () => {
    const spec = DataTable() as any;
    // implementation.node must be present so BlockNoteSchema.create() uses it
    // instead of creating a content:"" node from blockConfig.content:"none".
    expect(spec.implementation.node).toBeDefined();
    // The Tiptap node extension carries the content spec in its config.
    const pmNodeContent = spec.implementation.node?.config?.content;
    expect(pmNodeContent).toBe("tableRow+");
  });
});
