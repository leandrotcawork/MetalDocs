import type { LayoutTokens } from "../layout-ir";
import { defaultComponentRules } from "../layout-ir";
import { DataTableCodec } from "../codecs";
import { resolveThemeRef } from "../codecs/codec-utils";
import type { DataTableViewModel } from "./view-models";

export function interpretDataTable(
  block: { props: Record<string, unknown> },
  tokens: LayoutTokens,
): DataTableViewModel {
  const style = DataTableCodec.parseStyle((block.props.styleJson as string) ?? "{}");
  const caps = DataTableCodec.parseCaps((block.props.capabilitiesJson as string) ?? "{}");
  const rule = defaultComponentRules.dataTable;

  return {
    label: (block.props.label as string) ?? "",
    mode: caps.mode,
    headerBg: resolveThemeRef(style.headerBackground, tokens.theme) ?? tokens.theme.accentLight,
    headerColor: style.headerColor ?? rule.headerFontColor,
    headerFontWeight: style.headerFontWeight ?? rule.headerFontWeight,
    cellBorderColor: resolveThemeRef(style.cellBorderColor, tokens.theme) ?? tokens.theme.accentBorder,
    cellPadding: style.cellPadding ?? `${rule.cellPaddingMm}mm`,
    density: style.density ?? rule.defaultDensity,
    locked: caps.locked,
    removable: caps.removable,
    canAddRows: caps.addRows,
    canRemoveRows: caps.removeRows,
    canAddColumns: caps.addColumns,
    canRemoveColumns: caps.removeColumns,
    canResizeColumns: caps.resizeColumns,
    headerLocked: caps.headerLocked,
    maxRows: caps.maxRows,
  };
}
