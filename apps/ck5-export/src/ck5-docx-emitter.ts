import {
  BorderStyle,
  Document,
  HeadingLevel,
  ImageRun,
  Paragraph,
  Table,
  TableCell,
  TableRow,
  TextRun,
  WidthType,
  type IBorderOptions,
  type IParagraphOptions,
} from "docx"
import type { ResolvedAsset } from "./asset-resolver"
import type {
  ExportNode,
  Heading,
  Hyperlink,
  Image,
  List,
  Paragraph as ExportParagraph,
  Repeatable,
  RepeatableItem,
  Section,
  Table as ExportTable,
  TableCell as ExportTableCell,
  TableRow as ExportTableRow,
} from "./export-node"
import type { LayoutTokens } from "./layout-ir"
import { mmToTwip, ptToHalfPt } from "./shared/helpers/units"

const DEFAULT_IMAGE_WIDTH_PX = 320
const DEFAULT_IMAGE_HEIGHT_PX = 240
const DEFAULT_NUMBERING_REFERENCE = "default-numbering"

type DocxBlock = Paragraph | Table
type HeadingLevelValue = (typeof HeadingLevel)[keyof typeof HeadingLevel]

function hexToFill(hex: string): string {
  return hex.replace(/^#/, "").toUpperCase()
}

function toHeadingLevel(level: Heading["level"]): HeadingLevelValue {
  switch (level) {
    case 1:
      return HeadingLevel.HEADING_1
    case 2:
      return HeadingLevel.HEADING_2
    case 3:
      return HeadingLevel.HEADING_3
    case 4:
      return HeadingLevel.HEADING_4
    case 5:
      return HeadingLevel.HEADING_5
    case 6:
      return HeadingLevel.HEADING_6
  }
}

function toDocxImageType(mimeType: ResolvedAsset["mimeType"]): "jpg" | "png" | "gif" | null {
  if (mimeType === "image/jpeg") return "jpg"
  if (mimeType === "image/png") return "png"
  if (mimeType === "image/gif") return "gif"
  return null
}

function buildTextRun(
  value: string,
  marks: readonly ("bold" | "italic" | "underline" | "strike")[] | undefined,
  tokens: LayoutTokens,
): TextRun {
  const set = new Set(marks ?? [])
  return new TextRun({
    text: value,
    font: tokens.typography.exportFont,
    size: ptToHalfPt(tokens.typography.baseSizePt),
    bold: set.has("bold"),
    italics: set.has("italic"),
    underline: set.has("underline") ? {} : undefined,
    strike: set.has("strike"),
  })
}

function getSectionBorder(variant: Section["variant"], tokens: LayoutTokens): IBorderOptions {
  if (variant === "plain") {
    return { style: BorderStyle.NONE, size: 0, color: "auto" }
  }

  if (variant === "solid") {
    return { style: BorderStyle.SINGLE, size: 8, color: hexToFill(tokens.theme.accent) }
  }

  return { style: BorderStyle.SINGLE, size: 4, color: hexToFill(tokens.theme.accentBorder) }
}

function mapHyperlinkRuns(node: Hyperlink, tokens: LayoutTokens): TextRun[] {
  return collectHyperlinkRuns(node.children, tokens)
}

function collectHyperlinkRuns(nodes: ExportNode[], tokens: LayoutTokens): TextRun[] {
  const runs: TextRun[] = []

  for (const node of nodes) {
    switch (node.kind) {
      case "text": {
        const set = new Set(node.marks ?? [])
        runs.push(
          new TextRun({
            text: node.value,
            font: tokens.typography.exportFont,
            size: ptToHalfPt(tokens.typography.baseSizePt),
            color: "0563C1",
            bold: set.has("bold"),
            italics: set.has("italic"),
            underline: {},
            strike: set.has("strike"),
          }),
        )
        break
      }
      case "lineBreak":
        runs.push(new TextRun({ break: 1 }))
        break
      case "hyperlink":
        runs.push(...collectHyperlinkRuns(node.children, tokens))
        break
      case "paragraph":
      case "heading":
      case "blockquote":
      case "listItem":
      case "tableCell":
      case "repeatableItem":
        runs.push(...collectHyperlinkRuns(node.children, tokens))
        break
      case "field":
        runs.push(
          new TextRun({
            text: node.value,
            font: tokens.typography.exportFont,
            size: ptToHalfPt(tokens.typography.baseSizePt),
            color: "0563C1",
            underline: {},
          }),
        )
        break
      default:
        break
    }
  }

  return runs
}

function collectInlineRuns(nodes: ExportNode[], tokens: LayoutTokens): TextRun[] {
  const runs: TextRun[] = []

  for (const node of nodes) {
    switch (node.kind) {
      case "text":
        runs.push(buildTextRun(node.value, node.marks, tokens))
        break
      case "lineBreak":
        runs.push(new TextRun({ break: 1 }))
        break
      case "hyperlink":
        runs.push(...mapHyperlinkRuns(node, tokens))
        break
      case "paragraph":
      case "heading":
      case "blockquote":
      case "listItem":
      case "tableCell":
      case "repeatableItem":
        runs.push(...collectInlineRuns(node.children, tokens))
        break
      case "field":
        runs.push(buildTextRun(node.value, undefined, tokens))
        break
      default:
        break
    }
  }

  return runs
}

function paragraphFromInlineChildren(
  children: ExportNode[],
  tokens: LayoutTokens,
  options: Omit<IParagraphOptions, "children"> = {},
): Paragraph {
  const runs = collectInlineRuns(children, tokens)
  return new Paragraph({
    ...options,
    children: runs.length > 0 ? runs : [new TextRun({ text: "" })],
  })
}

function emitHeading(node: Heading, tokens: LayoutTokens): Paragraph[] {
  return [
    paragraphFromInlineChildren(node.children, tokens, {
      heading: toHeadingLevel(node.level),
    }),
  ]
}

function emitParagraph(node: ExportParagraph, tokens: LayoutTokens): Paragraph[] {
  return [paragraphFromInlineChildren(node.children, tokens)]
}

function emitList(node: List, tokens: LayoutTokens): Paragraph[] {
  return node.items.map((item) =>
    paragraphFromInlineChildren(
      item.children,
      tokens,
      node.ordered
        ? { numbering: { reference: DEFAULT_NUMBERING_REFERENCE, level: 0 } }
        : { bullet: { level: 0 } },
    ),
  )
}

function emitImage(node: Image, assetMap: ReadonlyMap<string, ResolvedAsset>): Paragraph[] {
  const asset = assetMap.get(node.src)
  if (!asset) {
    return []
  }

  const imageType = toDocxImageType(asset.mimeType)
  if (!imageType) {
    return []
  }

  return [
    new Paragraph({
      children: [
        new ImageRun({
          type: imageType,
          data: asset.bytes,
          transformation: {
            width: node.width ?? DEFAULT_IMAGE_WIDTH_PX,
            height: node.height ?? DEFAULT_IMAGE_HEIGHT_PX,
          },
        }),
      ],
    }),
  ]
}

function emitField(node: ExportNode, tokens: LayoutTokens): Table[] {
  if (node.kind !== "field") {
    return []
  }

  const borderColor = hexToFill(tokens.theme.accentBorder)
  const borders = {
    top: { style: BorderStyle.SINGLE, size: 4, color: borderColor },
    bottom: { style: BorderStyle.SINGLE, size: 4, color: borderColor },
    left: { style: BorderStyle.SINGLE, size: 4, color: borderColor },
    right: { style: BorderStyle.SINGLE, size: 4, color: borderColor },
  }

  return [
    new Table({
      width: { size: 100, type: WidthType.PERCENTAGE },
      rows: [
        new TableRow({
          children: [
            new TableCell({
              width: { size: 35, type: WidthType.PERCENTAGE },
              shading: { fill: hexToFill(tokens.theme.accentLight), type: "clear", color: "auto" },
              borders,
              children: [new Paragraph({ children: [new TextRun({ text: node.id })] })],
            }),
            new TableCell({
              width: { size: 65, type: WidthType.PERCENTAGE },
              borders,
              children: [new Paragraph({ children: [new TextRun({ text: node.value })] })],
            }),
          ],
        }),
      ],
    }),
  ]
}

function emitTableCell(
  node: ExportTableCell,
  tokens: LayoutTokens,
  assetMap: ReadonlyMap<string, ResolvedAsset>,
): TableCell {
  const borderColor = hexToFill(tokens.theme.accentBorder)
  const children = emitBlocks(node.children, tokens, assetMap)

  return new TableCell({
    columnSpan: node.colspan,
    rowSpan: node.rowspan,
    shading: node.isHeader ? { fill: hexToFill(tokens.theme.accentLight), type: "clear", color: "auto" } : undefined,
    borders: {
      top: { style: BorderStyle.SINGLE, size: 4, color: borderColor },
      bottom: { style: BorderStyle.SINGLE, size: 4, color: borderColor },
      left: { style: BorderStyle.SINGLE, size: 4, color: borderColor },
      right: { style: BorderStyle.SINGLE, size: 4, color: borderColor },
    },
    children: children.length > 0 ? children : [new Paragraph({ children: [] })],
  })
}

function emitTableRow(
  node: ExportTableRow,
  tokens: LayoutTokens,
  assetMap: ReadonlyMap<string, ResolvedAsset>,
): TableRow {
  return new TableRow({
    children: node.cells.map((cell) => emitTableCell(cell, tokens, assetMap)),
  })
}

function emitTable(
  node: ExportTable,
  tokens: LayoutTokens,
  assetMap: ReadonlyMap<string, ResolvedAsset>,
): Table[] {
  return [
    new Table({
      width: { size: 100, type: WidthType.PERCENTAGE },
      rows: node.rows.map((row) => emitTableRow(row, tokens, assetMap)),
    }),
  ]
}

function emitRepeatableItem(
  item: RepeatableItem,
  tokens: LayoutTokens,
  assetMap: ReadonlyMap<string, ResolvedAsset>,
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

function emitRepeatable(
  node: Repeatable,
  tokens: LayoutTokens,
  assetMap: ReadonlyMap<string, ResolvedAsset>,
): Table[] {
  return node.items.flatMap((item) => emitRepeatableItem(item, tokens, assetMap))
}

function ensureCellChildren(children: DocxBlock[]): DocxBlock[] {
  return children.length > 0 ? children : [new Paragraph({ children: [] })]
}

function emitSection(
  node: Section,
  tokens: LayoutTokens,
  assetMap: ReadonlyMap<string, ResolvedAsset>,
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

  return [
    new Table({
      width: { size: 100, type: WidthType.PERCENTAGE },
      rows,
    }),
  ]
}

function emitBlock(node: ExportNode, tokens: LayoutTokens, assetMap: ReadonlyMap<string, ResolvedAsset>): DocxBlock[] {
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
      return emitSection(node, tokens, assetMap)
    case "table":
      return emitTable(node, tokens, assetMap)
    case "tableRow":
      return [new Table({ rows: [emitTableRow(node, tokens, assetMap)] })]
    case "tableCell":
      return [new Table({ rows: [new TableRow({ children: [emitTableCell(node, tokens, assetMap)] })] })]
    case "field":
      return emitField(node, tokens)
    case "repeatable":
      return emitRepeatable(node, tokens, assetMap)
    case "repeatableItem":
      return emitRepeatableItem(node, tokens, assetMap)
    case "hyperlink":
    case "text":
    case "lineBreak":
      return [new Paragraph({ children: collectInlineRuns([node], tokens) })]
  }
}

function emitBlocks(
  nodes: ExportNode[],
  tokens: LayoutTokens,
  assetMap: ReadonlyMap<string, ResolvedAsset>,
): DocxBlock[] {
  return nodes.flatMap((node) => emitBlock(node, tokens, assetMap))
}

export function emitDocxFromExportTree(
  nodes: ExportNode[],
  tokens: LayoutTokens,
  assetMap: ReadonlyMap<string, ResolvedAsset>,
): Document {
  const children = emitBlocks(nodes, tokens, assetMap)

  return new Document({
    numbering: {
      config: [
        {
          reference: DEFAULT_NUMBERING_REFERENCE,
          levels: [
            {
              level: 0,
              format: "decimal",
              text: "%1.",
              alignment: "left",
            },
          ],
        },
      ],
    },
    sections: [
      {
        properties: {
          page: {
            size: {
              width: mmToTwip(tokens.page.widthMm),
              height: mmToTwip(tokens.page.heightMm),
            },
            margin: {
              top: mmToTwip(tokens.page.marginTopMm),
              right: mmToTwip(tokens.page.marginRightMm),
              bottom: mmToTwip(tokens.page.marginBottomMm),
              left: mmToTwip(tokens.page.marginLeftMm),
            },
          },
        },
        children,
      },
    ],
  })
}

export function collectImageUrls(nodes: ExportNode[]): string[] {
  const urls = new Set<string>()

  const walk = (items: ExportNode[]) => {
    for (const node of items) {
      switch (node.kind) {
        case "image":
          if (node.src.length > 0) {
            urls.add(node.src)
          }
          break
        case "section":
          if (node.header) {
            walk(node.header)
          }
          walk(node.body)
          break
        case "repeatable":
          for (const item of node.items) {
            walk(item.children)
          }
          break
        case "repeatableItem":
        case "paragraph":
        case "heading":
        case "blockquote":
        case "hyperlink":
        case "listItem":
        case "tableCell":
          walk(node.children)
          break
        case "list":
          for (const item of node.items) {
            walk(item.children)
          }
          break
        case "table":
          for (const row of node.rows) {
            for (const cell of row.cells) {
              walk(cell.children)
            }
          }
          break
        case "tableRow":
          for (const cell of node.cells) {
            walk(cell.children)
          }
          break
        case "field":
        case "text":
        case "lineBreak":
          break
      }
    }
  }

  walk(nodes)
  return [...urls]
}
