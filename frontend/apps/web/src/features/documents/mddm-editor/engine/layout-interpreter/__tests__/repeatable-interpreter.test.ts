import { describe, it, expect } from "vitest";
import { RepeatableInterpreter } from "../repeatable-interpreter";
import { defaultLayoutTokens } from "../../layout-ir";

const makeBlock = (items: number, locked = false, maxItems = 100) => ({
  props: { label: "Etapas", itemPrefix: "Etapa", locked, minItems: 0, maxItems },
  children: Array.from({ length: items }, (_, i) => ({
    type: "repeatableItem",
    id: `item-${i}`,
  })),
});

describe("RepeatableInterpreter", () => {
  it("numbers items sequentially", () => {
    const block = makeBlock(3);
    const vm = RepeatableInterpreter.interpret(block, defaultLayoutTokens, { depth: 0 });
    expect(vm.items[0].number).toBe(1);
    expect(vm.items[1].number).toBe(2);
    expect(vm.items[2].number).toBe(3);
  });

  it("uses parentNumber for displayNumber", () => {
    const block = makeBlock(3);
    const vm = RepeatableInterpreter.interpret(block, defaultLayoutTokens, { depth: 1, parentNumber: "4" });
    expect(vm.items[0].displayNumber).toBe("4.1");
    expect(vm.items[1].displayNumber).toBe("4.2");
    expect(vm.items[2].displayNumber).toBe("4.3");
  });

  it("canAddItem is true when not locked and under maxItems", () => {
    const block = makeBlock(2);
    const vm = RepeatableInterpreter.interpret(block, defaultLayoutTokens, { depth: 0 });
    expect(vm.canAddItem).toBe(true);
  });

  it("canAddItem is false when locked", () => {
    const block = makeBlock(2, true);
    const vm = RepeatableInterpreter.interpret(block, defaultLayoutTokens, { depth: 0 });
    expect(vm.canAddItem).toBe(false);
  });

  it("canAddItem is false when at maxItems", () => {
    const block = makeBlock(3, false, 3);
    const vm = RepeatableInterpreter.interpret(block, defaultLayoutTokens, { depth: 0 });
    expect(vm.canAddItem).toBe(false);
  });
});
