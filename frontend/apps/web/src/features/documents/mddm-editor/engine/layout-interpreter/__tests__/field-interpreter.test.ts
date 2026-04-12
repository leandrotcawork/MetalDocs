import { describe, it, expect } from "vitest";
import { FieldInterpreter } from "../field-interpreter";
import { defaultLayoutTokens } from "../../layout-ir";

describe("FieldInterpreter", () => {
  it("resolves label and value width percentages", () => {
    const block = { props: { label: "Name", valueMode: "inline", layout: "grid" } };
    const vm = FieldInterpreter.interpret(block, defaultLayoutTokens, { depth: 0 });
    expect(vm.labelWidthPct).toBe(35);
    expect(vm.valueWidthPct).toBe(65);
  });

  it("resolves token colors", () => {
    const block = { props: { label: "Test" } };
    const vm = FieldInterpreter.interpret(block, defaultLayoutTokens, { depth: 0 });
    expect(vm.labelBg).toBe(defaultLayoutTokens.theme.accentLight);
    expect(vm.borderColor).toBe(defaultLayoutTokens.theme.accentBorder);
  });
});
