import { describe, it, expect } from "vitest";
import { interpretDataTable } from "../data-table-interpreter";
import { defaultLayoutTokens } from "../../layout-ir";

describe("interpretDataTable", () => {
  it("defaults to dynamic mode", () => {
    const block = { props: { label: "Checklist", styleJson: "{}", capabilitiesJson: "{}" } };
    const vm = interpretDataTable(block as any, defaultLayoutTokens);
    expect(vm.mode).toBe("dynamic");
    expect(vm.canAddRows).toBe(true);
  });

  it("respects fixed mode from capabilities", () => {
    const block = {
      props: {
        label: "Fixed Table",
        styleJson: "{}",
        capabilitiesJson: JSON.stringify({ mode: "fixed", addRows: false, removeRows: false }),
      },
    };
    const vm = interpretDataTable(block as any, defaultLayoutTokens);
    expect(vm.mode).toBe("fixed");
    expect(vm.canAddRows).toBe(false);
    expect(vm.canRemoveRows).toBe(false);
  });

  it("resolves theme colors for header", () => {
    const block = { props: { label: "Table", styleJson: "{}", capabilitiesJson: "{}" } };
    const vm = interpretDataTable(block as any, defaultLayoutTokens);
    expect(vm.headerBg).toBe(defaultLayoutTokens.theme.accentLight);
  });

  it("applies style overrides for header background", () => {
    const block = {
      props: {
        label: "Styled Table",
        styleJson: JSON.stringify({ headerBackground: "#ff0000" }),
        capabilitiesJson: "{}",
      },
    };
    const vm = interpretDataTable(block as any, defaultLayoutTokens);
    expect(vm.headerBg).toBe("#ff0000");
  });
});
