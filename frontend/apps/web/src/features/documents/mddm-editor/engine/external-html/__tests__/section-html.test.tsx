import { describe, expect, it } from "vitest";
import { renderToStaticMarkup } from "react-dom/server";
import { SectionExternalHTML } from "../section-html";
import { defaultLayoutTokens } from "../../layout-ir";

describe("SectionExternalHTML", () => {
  it("renders a table-based header with the section title", () => {
    const html = renderToStaticMarkup(
      <SectionExternalHTML title="1. Procedimento" tokens={defaultLayoutTokens} />,
    );
    expect(html).toContain("<table");
    expect(html).toContain("1. Procedimento");
    expect(html).toContain("mddm-section-header");
  });

  it("does NOT use display:flex (flexbox is forbidden)", () => {
    const html = renderToStaticMarkup(
      <SectionExternalHTML title="x" tokens={defaultLayoutTokens} />,
    );
    expect(html).not.toContain("display:flex");
    expect(html).not.toContain("display: flex");
  });

  it("uses the theme accent color for the header background", () => {
    const tokens = {
      ...defaultLayoutTokens,
      theme: { ...defaultLayoutTokens.theme, accent: "#abcdef" },
    };
    const html = renderToStaticMarkup(<SectionExternalHTML title="x" tokens={tokens} />);
    expect(html.toLowerCase()).toContain("#abcdef");
  });

  it("uses absolute pt font size (no em or percent)", () => {
    const html = renderToStaticMarkup(
      <SectionExternalHTML title="x" tokens={defaultLayoutTokens} />,
    );
    expect(html).toMatch(/font-size:\s*\d+pt/);
    expect(html).not.toMatch(/font-size:\s*\d+em/);
  });
});
