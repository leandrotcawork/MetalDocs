import { describe, expect, it } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import { FieldGroupExternalHTML } from "../field-group-html";
import { defaultLayoutTokens } from "../../layout-ir";

describe("FieldGroupExternalHTML", () => {
  it("renders a wrapping table with data-columns attribute", () => {
    const html = renderToStaticMarkup(
      <FieldGroupExternalHTML columns={2} tokens={defaultLayoutTokens}>
        <span>child</span>
      </FieldGroupExternalHTML>,
    );
    expect(html).toContain("<table");
    expect(html).toContain('data-columns="2"');
    expect(html).toContain("mddm-field-group");
  });

  it("supports columns=1", () => {
    const html = renderToStaticMarkup(
      <FieldGroupExternalHTML columns={1} tokens={defaultLayoutTokens}>
        <span>child</span>
      </FieldGroupExternalHTML>,
    );
    expect(html).toContain('data-columns="1"');
  });

  it("does not use flexbox or CSS grid fr units", () => {
    const html = renderToStaticMarkup(
      <FieldGroupExternalHTML columns={2} tokens={defaultLayoutTokens}>
        <span>x</span>
      </FieldGroupExternalHTML>,
    );
    expect(html).not.toContain("display:flex");
    expect(html).not.toContain("grid-template-columns:1fr");
  });
});
