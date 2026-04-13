import { describe, it, expect } from "vitest";
import { interpretRepeatable } from "../repeatable-interpreter";
import { defaultLayoutTokens } from "../../layout-ir";

describe("interpretRepeatable", () => {
  it("uses Layout IR defaults when no style override", () => {
    const block = { props: { label: "Etapas", itemPrefix: "Etapa", styleJson: "{}", capabilitiesJson: "{}" }, children: [] };
    const vm = interpretRepeatable(block as any, defaultLayoutTokens);

    expect(vm.label).toBe("Etapas");
    expect(vm.itemPrefix).toBe("Etapa");
    expect(vm.borderColor).toBe(defaultLayoutTokens.theme.accentBorder);
    expect(vm.locked).toBe(true);
  });

  it("computes canAddItems based on count vs maxItems", () => {
    const block = {
      props: {
        label: "Etapas",
        styleJson: "{}",
        capabilitiesJson: JSON.stringify({ addItems: true, maxItems: 3, minItems: 0 }),
      },
      children: [1, 2, 3], // 3 items, at max
    };
    const vm = interpretRepeatable(block as any, defaultLayoutTokens);
    expect(vm.canAddItems).toBe(false);
  });

  it("allows adding items when under maxItems", () => {
    const block = {
      props: {
        label: "Etapas",
        styleJson: "{}",
        capabilitiesJson: JSON.stringify({ addItems: true, maxItems: 10, minItems: 0 }),
      },
      children: [1, 2],
    };
    const vm = interpretRepeatable(block as any, defaultLayoutTokens);
    expect(vm.canAddItems).toBe(true);
    expect(vm.currentItemCount).toBe(2);
  });

  it("resolves theme color for itemAccentBorder", () => {
    const block = { props: { label: "L", styleJson: "{}", capabilitiesJson: "{}" }, children: [] };
    const vm = interpretRepeatable(block as any, defaultLayoutTokens);
    expect(vm.itemAccentBorder).toBe(defaultLayoutTokens.theme.accent);
  });
});
