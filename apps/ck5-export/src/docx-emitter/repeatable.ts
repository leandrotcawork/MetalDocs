import { BorderStyle, Paragraph, Table, TableCell, TableRow, WidthType } from "docx"
import type { ResolvedAsset } from "../asset-resolver"
import type { Repeatable, RepeatableItem } from "../export-node"
import type { LayoutTokens } from "../layout-tokens"
import { hexToFill, type DocxBlock, type EmitBlocks } from "./helpers"

export function ensureCellChildren(children: DocxBlock[]): DocxBlock[] {
  return children.length > 0 ? children : [new Paragraph({ children: [] })]
}

export function emitRepeatableItem(
  item: RepeatableItem,
  tokens: LayoutTokens,
  assetMap: ReadonlyMap<string, ResolvedAsset>,
  emitBlocks: EmitBlocks,
): Table[] {
  const borderColor = hexToFill(tokens.theme.accentBorder)
  return [
    new Table({
      width: { size: 100, type: WidthType.PERCENTAGE },
      rows: [
        new TableRow({
          children: [
            new TableCell({
              borders: {
                top: { style: BorderStyle.SINGLE, size: 4, color: borderColor },
                bottom: { style: BorderStyle.SINGLE, size: 4, color: borderColor },
                left: { style: BorderStyle.SINGLE, size: 8, color: hexToFill(tokens.theme.accent) },
                right: { style: BorderStyle.SINGLE, size: 4, color: borderColor },
              },
              children: ensureCellChildren(emitBlocks(item.children, tokens, assetMap)),
            }),
          ],
        }),
      ],
    }),
  ]
}

export function emitRepeatable(
  node: Repeatable,
  tokens: LayoutTokens,
  assetMap: ReadonlyMap<string, ResolvedAsset>,
  emitBlocks: EmitBlocks,
): Table[] {
  return node.items.flatMap((item) => emitRepeatableItem(item, tokens, assetMap, emitBlocks))
}
