import { Document } from "docx"
import type { ResolvedAsset } from "../asset-resolver"
import type { ExportNode } from "../export-node"
import type { LayoutTokens } from "../layout-ir"
import { mmToTwip } from "../shared/helpers/units"
import { emitBlocks } from "./block-dispatch"
import { DEFAULT_NUMBERING_REFERENCE } from "./list"

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
