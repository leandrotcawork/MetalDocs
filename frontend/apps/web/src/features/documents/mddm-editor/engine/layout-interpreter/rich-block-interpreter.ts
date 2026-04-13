import type { LayoutTokens } from "../layout-ir";
import { defaultComponentRules } from "../layout-ir";
import { RichBlockCodec } from "../codecs";
import { resolveThemeRef } from "../codecs/codec-utils";
import type { RichBlockViewModel } from "./view-models";

export function interpretRichBlock(
  block: { props: Record<string, unknown> },
  tokens: LayoutTokens,
): RichBlockViewModel {
  const style = RichBlockCodec.parseStyle((block.props.styleJson as string) ?? "{}");
  const caps = RichBlockCodec.parseCaps((block.props.capabilitiesJson as string) ?? "{}");
  const rule = defaultComponentRules.richBlock;

  return {
    label: (block.props.label as string) ?? "",
    chrome: (block.props.chrome as string) ?? "labeled",
    labelBackground: resolveThemeRef(style.labelBackground, tokens.theme) ?? tokens.theme.accentLight,
    labelFontSize: style.labelFontSize ?? `${rule.labelFontSizePt}pt`,
    labelColor: style.labelColor ?? rule.labelFontColor,
    borderColor: resolveThemeRef(style.borderColor, tokens.theme) ?? tokens.theme.accentBorder,
    locked: caps.locked,
    removable: caps.removable,
  };
}
