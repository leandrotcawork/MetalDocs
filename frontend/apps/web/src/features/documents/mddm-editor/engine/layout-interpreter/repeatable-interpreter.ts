import type { LayoutTokens } from "../layout-ir";
import type { InterpretContext, BlockLayoutInterpreter } from "./types";

type RepeatableBlock = {
  props: {
    label?: string;
    itemPrefix?: string;
    locked?: boolean;
    minItems?: number;
    maxItems?: number;
  };
  children?: Array<{ type: string; id?: string }>;
};

type RepeatableItemViewModel = {
  id?: string;
  number: number;
  displayNumber: string;
};

type RepeatableViewModel = {
  label: string;
  itemPrefix: string;
  items: RepeatableItemViewModel[];
  canAddItem: boolean;
};

export const RepeatableInterpreter: BlockLayoutInterpreter<RepeatableBlock, RepeatableViewModel> = {
  interpret(block, _tokens: LayoutTokens, context: InterpretContext): RepeatableViewModel {
    const label = block.props.label ?? "";
    const itemPrefix = block.props.itemPrefix ?? "Item";
    const locked = block.props.locked ?? false;
    const maxItems = block.props.maxItems ?? 100;

    const children = block.children ?? [];
    const items = children
      .filter((c) => c.type === "repeatableItem")
      .map((child, index) => {
        const number = index + 1;
        const parentNum = context.parentNumber;
        const displayNumber = parentNum ? `${parentNum}.${number}` : String(number);
        return {
          id: child.id,
          number,
          displayNumber,
        };
      });

    const canAddItem = !locked && items.length < maxItems;

    return { label, itemPrefix, items, canAddItem };
  },
};
