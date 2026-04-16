import {
  BorderStyle,
  Table,
  TableCell,
  TableRow,
  WidthType,
  type IBorderOptions,
} from "docx"
import type { ResolvedAsset } from "../asset-resolver"
import type { Section } from "../export-node"
import type { LayoutTokens } from "../layout-tokens"
import { hexToFill, type EmitBlocks } from "./helpers"
import { ensureCellChildren } from "./repeatable"

export function getSectionBorder(variant: Section["variant"], tokens: LayoutTokens): IBorderOptions {
  if (variant === "plain") {
    return { style: BorderStyle.NONE, size: 0, color: "auto" }
  }

  if (variant === "solid") {
    return { style: BorderStyle.SINGLE, size: 8, color: hexToFill(tokens.theme.accent) }
  }

  return { style: BorderStyle.SINGLE, size: 4, color: hexToFill(tokens.theme.accentBorder) }
}

export function emitSection(
  node: Section,
  tokens: LayoutTokens,
  assetMap: ReadonlyMap<string, ResolvedAsset>,
  emitBlocks: EmitBlocks,
): Table[] {
  const border = getSectionBorder(node.variant, tokens)
  const headerFill =
    node.variant === "plain"
      ? undefined
      : { fill: hexToFill(tokens.theme.accent), type: "clear" as const, color: "auto" }

  const rows: TableRow[] = []

  if (node.header && node.header.length > 0) {
    rows.push(
      new TableRow({
        children: [
          new TableCell({
            shading: headerFill,
            borders: { top: border, bottom: border, left: border, right: border },
            children: ensureCellChildren(emitBlocks(node.header, tokens, assetMap)),
          }),
        ],
      }),
    )
  }

  rows.push(
    new TableRow({
      children: [
        new TableCell({
          borders: { top: border, bottom: border, left: border, right: border },
          children: ensureCellChildren(emitBlocks(node.body, tokens, assetMap)),
        }),
      ],
    }),
  )

  return [new Table({ width: { size: 100, type: WidthType.PERCENTAGE }, rows })]
}
