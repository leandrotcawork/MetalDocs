import { Paragraph, TextRun, type IParagraphOptions } from "docx"
import type { ExportNode, Paragraph as ExportParagraph } from "../export-node"
import type { LayoutTokens } from "../layout-ir"
import { collectInlineRuns } from "./inline"

export function paragraphFromInlineChildren(
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

export function emitParagraph(node: ExportParagraph, tokens: LayoutTokens): Paragraph[] {
  return [paragraphFromInlineChildren(node.children, tokens)]
}
