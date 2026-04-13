import type { LayoutTokens } from "../layout-ir";
import { defaultComponentRules } from "../layout-ir";
import { RepeatableItemCodec } from "../codecs";
import { resolveThemeRef } from "../codecs/codec-utils";
import type { RepeatableItemViewModel } from "./view-models";

export function interpretRepeatableItem(
  block: { props: Record<string, unknown> },
  tokens: LayoutTokens,
  context: { itemIndex: number; parentNumber?: string },
): RepeatableItemViewModel {
  const style = RepeatableItemCodec.parseStyle((block.props.styleJson as string) ?? "{}");
  const caps = RepeatableItemCodec.parseCaps((block.props.capabilitiesJson as string) ?? "{}");
  const rule = defaultComponentRules.repeatable;

  const number = context.parentNumber
    ? `${context.parentNumber}.${context.itemIndex + 1}`
    : String(context.itemIndex + 1);

  return {
    title: (block.props.title as string) ?? "",
    number,
    accentBorderColor: resolveThemeRef(style.accentBorderColor, tokens.theme) ?? tokens.theme.accent,
    accentBorderWidth: style.accentBorderWidth ?? `${rule.itemAccentWidthPt}pt`,
    locked: caps.locked,
    removable: caps.removable,
  };
}
