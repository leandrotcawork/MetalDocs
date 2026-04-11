import { describe, expect, it } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import { FieldExternalHTML } from "../field-html";
import { defaultLayoutTokens } from "../../layout-ir";

describe("FieldExternalHTML", () => {
  it("renders a two-column table with label and value cells", () => {
    const html = renderToStaticMarkup(
      <FieldExternalHTML label="Responsável" tokens={defaultLayoutTokens}>
        <span>João Silva</span>
      </FieldExternalHTML>,
    );
    expect(html).toContain("<table");
    expect(html).toContain("Responsável");
    expect(html).toContain("João Silva");
    expect(html).toContain("mddm-field");
  });

  it("renders label cell with 35% width and value cell with 65% width", () => {
    const html = renderToStaticMarkup(
      <FieldExternalHTML label="L" tokens={defaultLayoutTokens}>
        V
      </FieldExternalHTML>,
    );
    expect(html).toContain("35%");
    expect(html).toContain("65%");
  });

  it("does not use flexbox", () => {
    const html = renderToStaticMarkup(
      <FieldExternalHTML label="L" tokens={defaultLayoutTokens}>
        V
      </FieldExternalHTML>,
    );
    expect(html).not.toContain("display:flex");
    expect(html).not.toContain("display: flex");
  });
});
