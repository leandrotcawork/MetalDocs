import { renderToStaticMarkup } from "react-dom/server";
import { describe, expect, it } from "vitest";
import { SectionExternalHTML } from "../section-html";
import { defaultLayoutTokens } from "../../layout-ir";

describe("SectionExternalHTML token threading", () => {
  it("SectionExternalHTML renders custom accent when custom tokens are provided", () => {
    const tokens = {
      ...defaultLayoutTokens,
      theme: {
        ...defaultLayoutTokens.theme,
        accent: "#123456",
      },
    };

    const html = renderToStaticMarkup(<SectionExternalHTML title="Section" tokens={tokens} />);

    expect(html.toLowerCase()).toContain("#123456");
    expect(html.toLowerCase()).not.toContain(defaultLayoutTokens.theme.accent.toLowerCase());
  });

  it("SectionExternalHTML renders default accent when default tokens are provided", () => {
    const html = renderToStaticMarkup(
      <SectionExternalHTML title="Section" tokens={defaultLayoutTokens} />,
    );

    expect(html.toLowerCase()).toContain(defaultLayoutTokens.theme.accent.toLowerCase());
  });
});
