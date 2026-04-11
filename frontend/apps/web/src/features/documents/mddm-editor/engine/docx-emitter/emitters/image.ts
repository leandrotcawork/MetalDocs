import { Paragraph, ImageRun } from "docx";
import type { LayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../../adapter";
import type { ResolvedAsset } from "../../asset-resolver";
import { mmToEmu } from "../../helpers/units";

export class MissingAssetError extends Error {
  constructor(public readonly url: string) {
    super(`Image asset not found in asset map: ${url}`);
    this.name = "MissingAssetError";
  }
}

const DEFAULT_IMAGE_WIDTH_MM = 80;
const DEFAULT_IMAGE_HEIGHT_MM = 60;

export function emitImage(
  block: MDDMBlock,
  tokens: LayoutTokens,
  assetMap: ReadonlyMap<string, ResolvedAsset>,
): Paragraph[] {
  const src = (block.props as { src?: string }).src;

  if (typeof src !== "string" || src.length === 0) {
    return [new Paragraph({ children: [] })];
  }

  const asset = assetMap.get(src);
  if (!asset) {
    throw new MissingAssetError(src);
  }

  const widthMm = (block.props as { widthMm?: number }).widthMm ?? DEFAULT_IMAGE_WIDTH_MM;
  const heightMm = (block.props as { heightMm?: number }).heightMm ?? DEFAULT_IMAGE_HEIGHT_MM;

  const docxImageType = asset.mimeType === "image/jpeg" ? "jpg"
    : asset.mimeType === "image/png" ? "png"
    : asset.mimeType === "image/gif" ? "gif"
    : "png";

  return [
    new Paragraph({
      children: [
        new ImageRun({
          type: docxImageType as any,
          data: asset.bytes,
          transformation: {
            width: Math.round(mmToEmu(widthMm) / 9525),  // EMU → px (1 px = 9525 EMU)
            height: Math.round(mmToEmu(heightMm) / 9525),
          },
        }),
      ],
    }),
  ];
}
