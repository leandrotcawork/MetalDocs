import type { LayoutTokens } from "../layout-ir";
import { defaultComponentRules } from "../layout-ir";
import { SectionCodec } from "../codecs";
import { resolveThemeRef } from "../codecs/codec-utils";
import type { SectionViewModel } from "./view-models";

type InterpretSectionContext = {
  sectionIndex: number;
};

export function interpretSection(
  block: { props: Record<string, unknown> },
  tokens: LayoutTokens,
  context: InterpretSectionContext,
): SectionViewModel {
  const style = SectionCodec.parseStyle((block.props.styleJson as string) ?? "{}");
  const caps = SectionCodec.parseCaps((block.props.capabilitiesJson as string) ?? "{}");
  const rule = defaultComponentRules.section;

  return {
    number: String(context.sectionIndex + 1),
    title: (block.props.title as string) ?? "",
    optional: (block.props.optional as boolean) ?? false,
    headerHeight: style.headerHeight ?? `${rule.headerHeightMm}mm`,
    headerBg: resolveThemeRef(style.headerBackground, tokens.theme) ?? tokens.theme.accent,
    headerColor: style.headerColor ?? rule.headerFontColor,
    headerFontSize: style.headerFontSize ?? `${rule.headerFontSizePt}pt`,
    headerFontWeight: style.headerFontWeight ?? rule.headerFontWeight,
    locked: caps.locked,
    removable: caps.removable,
  };
}
