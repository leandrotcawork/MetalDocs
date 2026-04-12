import { describe, expect, it, vi } from "vitest";

vi.mock("@blocknote/react", () => ({
  createReactBlockSpec: vi.fn((config: object, spec: object) => {
    const factory = () => ({
      config,
      ...spec,
    });
    return factory;
  }),
}));

import { DataTable } from "../DataTable";

describe("DataTable block spec guard", () => {
  it("keeps the runtime table content patch and required props", () => {
    const spec = DataTable();

    expect(spec.config.type).toBe("dataTable");
    expect(spec.config.content).toBe("table");
    expect(spec.config.propSchema).toHaveProperty("label");
    expect(spec.config.propSchema).toHaveProperty("locked");
    expect(spec.config.propSchema).toHaveProperty("density");
  });
});
