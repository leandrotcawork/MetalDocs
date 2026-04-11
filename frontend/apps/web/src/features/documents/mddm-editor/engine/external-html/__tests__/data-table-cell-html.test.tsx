import { describe, expect, it } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import { DataTableCellExternalHTML } from "../data-table-cell-html";
import { defaultLayoutTokens } from "../../layout-ir";

describe("DataTableCellExternalHTML", () => {
  it("renders a <td> with mddm-data-table-cell class", () => {
    const html = renderToStaticMarkup(
      <DataTableCellExternalHTML tokens={defaultLayoutTokens}>
        <span>100</span>
      </DataTableCellExternalHTML>,
    );
    expect(html).toContain("<td");
    expect(html).toContain("mddm-data-table-cell");
    expect(html).toContain("100");
  });

  it("uses absolute padding (mm) and accentBorder color", () => {
    const html = renderToStaticMarkup(
      <DataTableCellExternalHTML tokens={defaultLayoutTokens}>x</DataTableCellExternalHTML>,
    );
    expect(html).toMatch(/padding:\s*\d+(?:\.\d+)?mm/);
    expect(html.toLowerCase()).toContain(defaultLayoutTokens.theme.accentBorder.toLowerCase());
  });
});
