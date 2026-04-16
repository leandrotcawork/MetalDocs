import type { LayoutTokens } from "../layout-ir";
import { defaultComponentRules } from "../layout-ir";
import { RepeatableCodec } from "../codecs";
import { resolveThemeRef } from "../codecs/codec-utils";
import type { RepeatableViewModel } from "./view-models";

export function interpretRepeatable(
  block: { props: Record<string, unknown>; children?: unknown[] },
  tokens: LayoutTokens,
): RepeatableViewModel {
  const style = RepeatableCodec.parseStyle((block.props.styleJson as string) ?? "{}");
  const caps = RepeatableCodec.parseCaps((block.props.capabilitiesJson as string) ?? "{}");
  const rule = defaultComponentRules.repeatable;

  const currentItemCount = block.children?.length ?? 0;

  return {
    label: (block.props.label as string) ?? "",
    itemPrefix: (block.props.itemPrefix as string) ?? "Item",
    borderColor: resolveThemeRef(style.borderColor, tokens.theme) ?? tokens.theme.accentBorder,
    itemAccentBorder: resolveThemeRef(style.itemAccentBorder, tokens.theme) ?? tokens.theme.accent,
    itemAccentWidth: style.itemAccentWidth ?? `${rule.itemAccentWidthPt}pt`,
    locked: caps.locked,
    removable: caps.removable,
    canAddItems: caps.addItems && currentItemCount < caps.maxItems,
    canRemoveItems: caps.removeItems && currentItemCount > caps.minItems,
    maxItems: caps.maxItems,
    minItems: caps.minItems,
    currentItemCount,
  };
}
