import { HeadingLevel, type Paragraph } from "docx"
import type { Heading } from "../export-node"
import type { LayoutTokens } from "../layout-ir"
import type { HeadingLevelValue } from "./helpers"
import { paragraphFromInlineChildren } from "./paragraph"

export function toHeadingLevel(level: Heading["level"]): HeadingLevelValue {
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

export function emitHeading(node: Heading, tokens: LayoutTokens): Paragraph[] {
  return [
    paragraphFromInlineChildren(node.children, tokens, {
      heading: toHeadingLevel(node.level),
    }),
  ]
}
