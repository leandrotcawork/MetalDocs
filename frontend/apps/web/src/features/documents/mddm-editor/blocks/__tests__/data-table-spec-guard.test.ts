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
});
