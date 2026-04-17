import { TextRun } from "docx"
import type { ExportNode, Hyperlink } from "../export-node"
import type { LayoutTokens } from "../layout-tokens"
import { ptToHalfPt } from "../shared/helpers/units"
import { hexToFill } from "./helpers"

export function buildTextRun(
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

export function mapHyperlinkRuns(node: Hyperlink, tokens: LayoutTokens): TextRun[] {
  return collectHyperlinkRuns(node.children, tokens)
}

export function collectHyperlinkRuns(nodes: ExportNode[], tokens: LayoutTokens): TextRun[] {
  const runs: TextRun[] = []
  const linkColor = hexToFill(tokens.theme.hyperlink)

  for (const node of nodes) {
    switch (node.kind) {
      case "text": {
        const set = new Set(node.marks ?? [])
        runs.push(
          new TextRun({
            text: node.value,
            font: tokens.typography.exportFont,
            size: ptToHalfPt(tokens.typography.baseSizePt),
            color: linkColor,
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
            color: linkColor,
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

export function collectInlineRuns(nodes: ExportNode[], tokens: LayoutTokens): TextRun[] {
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
