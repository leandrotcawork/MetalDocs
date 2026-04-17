import type { Paragraph } from "docx"
import type { List } from "../export-node"
import type { LayoutTokens } from "../layout-ir"
import { paragraphFromInlineChildren } from "./paragraph"

export const DEFAULT_NUMBERING_REFERENCE = "default-numbering"

export function emitList(node: List, tokens: LayoutTokens): Paragraph[] {
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
