import { HeadingLevel, type Paragraph, type Table } from "docx"
import type { ResolvedAsset } from "../asset-resolver"
import type { ExportNode } from "../export-node"
import type { LayoutTokens } from "../layout-tokens"

export type DocxBlock = Paragraph | Table
export type HeadingLevelValue = (typeof HeadingLevel)[keyof typeof HeadingLevel]

// Callback type used to break the cycle between block-dispatch and the
// leaf emitters that recurse into child blocks (table, section, repeatable).
// Passed in by block-dispatch; never imported from block-dispatch by leaves.
export type EmitBlocks = (
  nodes: ExportNode[],
  tokens: LayoutTokens,
  assetMap: ReadonlyMap<string, ResolvedAsset>,
) => DocxBlock[]

export function hexToFill(hex: string): string {
  return hex.replace(/^#/, "").toUpperCase()
}
