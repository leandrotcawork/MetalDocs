import { describe, it, expect } from "vitest";
import { interpretRichBlock } from "../rich-block-interpreter";
import { defaultLayoutTokens } from "../../layout-ir";

describe("interpretRichBlock", () => {
  it("uses Layout IR defaults when no style override", () => {
    const block = { props: { label: "Objetivo", chrome: "labeled", styleJson: "{}", capabilitiesJson: "{}" } };
    const vm = interpretRichBlock(block as any, defaultLayoutTokens);

    expect(vm.label).toBe("Objetivo");
    expect(vm.chrome).toBe("labeled");
    expect(vm.labelBackground).toBe(defaultLayoutTokens.theme.accentLight);
    expect(vm.labelFontSize).toBe("9pt");
    expect(vm.locked).toBe(true);
  });

  it("applies style overrides", () => {
    const block = {
      props: {
        label: "Custom",
        styleJson: JSON.stringify({ labelBackground: "#ff0000", labelFontSize: "12pt" }),
        capabilitiesJson: "{}",
      },
    };
    const vm = interpretRichBlock(block as any, defaultLayoutTokens);
    expect(vm.labelBackground).toBe("#ff0000");
    expect(vm.labelFontSize).toBe("12pt");
  });

  it("reads capabilities from codec", () => {
    const block = {
      props: {
        label: "Editable",
        styleJson: "{}",
        capabilitiesJson: JSON.stringify({ locked: false }),
      },
    };
    const vm = interpretRichBlock(block as any, defaultLayoutTokens);
    expect(vm.locked).toBe(false);
  });
});
