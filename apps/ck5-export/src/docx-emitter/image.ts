import { ImageRun, Paragraph } from "docx"
import type { ResolvedAsset } from "../asset-resolver"
import type { Image } from "../export-node"

const DEFAULT_IMAGE_WIDTH_PX = 320
const DEFAULT_IMAGE_HEIGHT_PX = 240

function toDocxImageType(mimeType: ResolvedAsset["mimeType"]): "jpg" | "png" | "gif" | null {
  if (mimeType === "image/jpeg") return "jpg"
  if (mimeType === "image/png") return "png"
  if (mimeType === "image/gif") return "gif"
  return null
}

export function emitImage(node: Image, assetMap: ReadonlyMap<string, ResolvedAsset>): Paragraph[] {
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
