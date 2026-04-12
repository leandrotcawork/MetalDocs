import type { LayoutTokens } from "../layout-ir";
import { defaultComponentRules } from "../layout-ir";
import type { InterpretContext, BlockLayoutInterpreter } from "./types";

type SectionBlock = {
  props: {
    title?: string;
    optional?: boolean;
    variant?: string;
  };
};

type SectionViewModel = {
  title: string;
  optional: boolean;
  variant: string;
  headerHeightMm: number;
  headerFontSizePt: number;
  headerFontColor: string;
  headerBackground: string;
  sectionNumber: number;
};

export const SectionInterpreter: BlockLayoutInterpreter<SectionBlock, SectionViewModel> = {
  interpret(block, tokens: LayoutTokens, context: InterpretContext): SectionViewModel {
    const rule = defaultComponentRules.section;
    return {
      title: block.props.title ?? "",
      optional: block.props.optional ?? false,
      variant: block.props.variant ?? "bar",
      headerHeightMm: rule.headerHeightMm,
      headerFontSizePt: rule.headerFontSizePt,
      headerFontColor: rule.headerFontColor,
      headerBackground: tokens.theme.accent,
      sectionNumber: context.sectionIndex ?? 1,
    };
  },
};
