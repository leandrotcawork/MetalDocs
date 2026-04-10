import { describe, expect, it } from "vitest";
import { parseDataTableColumns } from "../DataTable";

describe("parseDataTableColumns", () => {
  it("returns only valid column objects", () => {
    const result = parseDataTableColumns(
      JSON.stringify([
        { key: "c1", label: "Nome" },
        { key: "c2", label: "Cargo" },
        { key: "bad" },
      ]),
    );

    expect(result).toEqual([
      { key: "c1", label: "Nome" },
      { key: "c2", label: "Cargo" },
    ]);
  });

  it("returns empty array for invalid json", () => {
    expect(parseDataTableColumns("{")).toEqual([]);
  });

  it("returns empty array for non-array json", () => {
    expect(parseDataTableColumns(JSON.stringify({ key: "c1", label: "Nome" }))).toEqual([]);
  });

  it("skips primitive entries and rejects empty keys or labels", () => {
    const result = parseDataTableColumns(
      JSON.stringify([
        null,
        true,
        1,
        "column",
        { key: "", label: "Nome" },
        { key: "c1", label: "" },
        { key: "  ", label: "  " },
        { key: "c2", label: "Aceito" },
      ]),
    );

    expect(result).toEqual([{ key: "c2", label: "Aceito" }]);
  });

  it("deduplicates repeated keys and keeps the first valid entry", () => {
    const result = parseDataTableColumns(
      JSON.stringify([
        { key: "c1", label: "Nome" },
        { key: "c1", label: "Outro Nome" },
        { key: "c2", label: "Cargo" },
      ]),
    );

    expect(result).toEqual([
      { key: "c1", label: "Nome" },
      { key: "c2", label: "Cargo" },
    ]);
  });
});
