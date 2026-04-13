import { describe, it, expect } from "vitest";
import { interpretRepeatableItem } from "../repeatable-item-interpreter";
import { defaultLayoutTokens } from "../../layout-ir";

describe("interpretRepeatableItem", () => {
  it("generates number from itemIndex", () => {
    const block = { props: { title: "Etapa 1", styleJson: "{}", capabilitiesJson: "{}" } };
    const vm = interpretRepeatableItem(block as any, defaultLayoutTokens, { itemIndex: 0 });
    expect(vm.number).toBe("1");
    expect(vm.title).toBe("Etapa 1");
  });

  it("generates prefixed number with parentNumber", () => {
    const block = { props: { title: "Etapa", styleJson: "{}", capabilitiesJson: "{}" } };
    const vm = interpretRepeatableItem(block as any, defaultLayoutTokens, { itemIndex: 2, parentNumber: "4" });
    expect(vm.number).toBe("4.3");
  });

  it("resolves theme color for accentBorderColor", () => {
    const block = { props: { title: "E", styleJson: "{}", capabilitiesJson: "{}" } };
    const vm = interpretRepeatableItem(block as any, defaultLayoutTokens, { itemIndex: 0 });
    expect(vm.accentBorderColor).toBe(defaultLayoutTokens.theme.accent);
  });
});
