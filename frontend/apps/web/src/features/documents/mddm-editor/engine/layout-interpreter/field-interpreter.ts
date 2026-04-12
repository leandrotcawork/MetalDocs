import type { LayoutTokens } from "../layout-ir";
import { defaultComponentRules } from "../layout-ir";
import type { InterpretContext, BlockLayoutInterpreter } from "./types";

type FieldBlock = {
  props: {
    label?: string;
    valueMode?: string;
    locked?: boolean;
    hint?: string;
    layout?: string;
  };
};

type FieldViewModel = {
  label: string;
  hint: string;
  layout: string;
  locked: boolean;
  labelWidthPct: number;
  valueWidthPct: number;
  labelBg: string;
  labelColor: string;
  borderColor: string;
  borderWidthPt: number;
  minHeightMm: number;
  labelFontSizePt: number;
};

export const FieldInterpreter: BlockLayoutInterpreter<FieldBlock, FieldViewModel> = {
  interpret(block, tokens: LayoutTokens, _context: InterpretContext): FieldViewModel {
    const rule = defaultComponentRules.field;
    return {
      label: block.props.label ?? "",
      hint: block.props.hint ?? "",
      layout: block.props.layout ?? "grid",
      locked: block.props.locked ?? false,
      labelWidthPct: rule.labelWidthPercent,
      valueWidthPct: rule.valueWidthPercent,
      labelBg: tokens.theme.accentLight,
      labelColor: tokens.theme.accentDark,
      borderColor: tokens.theme.accentBorder,
      borderWidthPt: rule.borderWidthPt,
      minHeightMm: rule.minHeightMm,
      labelFontSizePt: rule.labelFontSizePt,
    };
  },
};
