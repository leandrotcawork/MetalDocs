import { describe, it, expect } from "vitest";
import { interpretSection } from "../section-interpreter";
import { defaultLayoutTokens } from "../../layout-ir";

describe("interpretSection", () => {
  it("uses Layout IR defaults when no style override", () => {
    const block = { props: { title: "OBJETIVO", styleJson: "{}", capabilitiesJson: "{}" } };
    const vm = interpretSection(block as any, defaultLayoutTokens, { sectionIndex: 2 });

    expect(vm.number).toBe("3");
    expect(vm.title).toBe("OBJETIVO");
    expect(vm.headerBg).toBe(defaultLayoutTokens.theme.accent);
    expect(vm.headerHeight).toBe("8mm");
    expect(vm.headerFontSize).toBe("13pt");
    expect(vm.locked).toBe(true);
  });

  it("applies style overrides", () => {
    const block = {
      props: {
        title: "CUSTOM",
        styleJson: JSON.stringify({ headerHeight: "12mm", headerBackground: "#ff0000" }),
        capabilitiesJson: "{}",
      },
    };
    const vm = interpretSection(block as any, defaultLayoutTokens, { sectionIndex: 0 });

    expect(vm.headerHeight).toBe("12mm");
    expect(vm.headerBg).toBe("#ff0000");
    expect(vm.headerColor).toBe("#ffffff"); // default from rule
  });

  it("resolves theme references in style", () => {
    const block = {
      props: {
        title: "THEMED",
        styleJson: JSON.stringify({ headerBackground: "theme.accentDark" }),
        capabilitiesJson: "{}",
      },
    };
    const vm = interpretSection(block as any, defaultLayoutTokens, { sectionIndex: 0 });

    expect(vm.headerBg).toBe(defaultLayoutTokens.theme.accentDark);
  });

  it("reads capabilities from codec", () => {
    const block = {
      props: {
        title: "UNLOCKED",
        styleJson: "{}",
        capabilitiesJson: JSON.stringify({ locked: false, removable: true }),
      },
    };
    const vm = interpretSection(block as any, defaultLayoutTokens, { sectionIndex: 0 });

    expect(vm.locked).toBe(false);
    expect(vm.removable).toBe(true);
  });
});
