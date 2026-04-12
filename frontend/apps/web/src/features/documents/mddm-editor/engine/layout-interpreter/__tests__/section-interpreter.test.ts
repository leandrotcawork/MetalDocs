import { describe, it, expect } from "vitest";
import { SectionInterpreter } from "../section-interpreter";
import { defaultLayoutTokens } from "../../layout-ir";

const makeBlock = (title = "Introduction") => ({
  props: { title, optional: false, variant: "bar" },
});

describe("SectionInterpreter", () => {
  it("maps section number from context.sectionIndex", () => {
    const vm = SectionInterpreter.interpret(makeBlock(), defaultLayoutTokens, { depth: 0, sectionIndex: 3 });
    expect(vm.sectionNumber).toBe(3);
  });

  it("uses theme accent as headerBackground", () => {
    const vm = SectionInterpreter.interpret(makeBlock(), defaultLayoutTokens, { depth: 0, sectionIndex: 1 });
    expect(vm.headerBackground).toBe(defaultLayoutTokens.theme.accent);
  });

  it("resolves header dimensions from ComponentRules", () => {
    const vm = SectionInterpreter.interpret(makeBlock(), defaultLayoutTokens, { depth: 0 });
    expect(vm.headerHeightMm).toBe(8);
    expect(vm.headerFontSizePt).toBe(13);
  });
});
