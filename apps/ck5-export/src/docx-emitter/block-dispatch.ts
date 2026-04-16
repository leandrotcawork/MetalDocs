import { Paragraph, Table, TableRow } from "docx"
import type { ResolvedAsset } from "../asset-resolver"
import type { ExportNode } from "../export-node"
import type { LayoutTokens } from "../layout-tokens"
import { emitField } from "./field"
import type { DocxBlock } from "./helpers"
import { emitHeading } from "./heading"
import { emitImage } from "./image"
import { collectInlineRuns } from "./inline"
import { emitList } from "./list"
import { emitParagraph, paragraphFromInlineChildren } from "./paragraph"
import { emitRepeatable, emitRepeatableItem } from "./repeatable"
import { emitSection } from "./section"
import { emitTable, emitTableCell, emitTableRow } from "./table"

export function emitBlocks(
  nodes: ExportNode[],
  tokens: LayoutTokens,
  assetMap: ReadonlyMap<string, ResolvedAsset>,
): DocxBlock[] {
  return nodes.flatMap((node) => emitBlock(node, tokens, assetMap))
}

function emitBlock(
  node: ExportNode,
  tokens: LayoutTokens,
  assetMap: ReadonlyMap<string, ResolvedAsset>,
): DocxBlock[] {
  switch (node.kind) {
    case "paragraph":
      return emitParagraph(node, tokens)
    case "heading":
      return emitHeading(node, tokens)
    case "list":
      return emitList(node, tokens)
    case "listItem":
      return [paragraphFromInlineChildren(node.children, tokens)]
    case "image":
      return emitImage(node, assetMap)
    case "blockquote":
      return [paragraphFromInlineChildren(node.children, tokens, { indent: { left: 720 } })]
    case "section":
      return emitSection(node, tokens, assetMap, emitBlocks)
    case "table":
      return emitTable(node, tokens, assetMap, emitBlocks)
    case "tableRow":
      return [new Table({ rows: [emitTableRow(node, tokens, assetMap, emitBlocks)] })]
    case "tableCell":
      return [
        new Table({
          rows: [new TableRow({ children: [emitTableCell(node, tokens, assetMap, emitBlocks)] })],
        }),
      ]
    case "field":
      return emitField(node, tokens)
    case "repeatable":
      return emitRepeatable(node, tokens, assetMap, emitBlocks)
    case "repeatableItem":
      return emitRepeatableItem(node, tokens, assetMap, emitBlocks)
    case "hyperlink":
    case "text":
    case "lineBreak":
      return [new Paragraph({ children: collectInlineRuns([node], tokens) })]
  }
}
