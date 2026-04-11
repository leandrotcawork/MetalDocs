import { Paragraph, ImageRun } from "docx";
import type { LayoutTokens } from "../../layout-ir";
import type { MDDMBlock } from "../../../adapter";
import type { ResolvedAsset, AllowedMimeType } from "../../asset-resolver";
import { mmToEmu } from "../../helpers/units";

export class MissingAssetError extends Error {
  constructor(public readonly url: string) {
    super(`Image asset not found in asset map: ${url}`);
    this.name = "MissingAssetError";
  }
}

export class UnsupportedMimeTypeError extends Error {
  constructor(public readonly mimeType: string) {
    super(`MIME type "${mimeType}" is not supported for DOCX image export. Convert to PNG, JPEG, or GIF first.`);
    this.name = "UnsupportedMimeTypeError";
  }
}

const DEFAULT_IMAGE_WIDTH_MM = 80;
const DEFAULT_IMAGE_HEIGHT_MM = 60;

function toDocxImageType(mimeType: AllowedMimeType): "jpg" | "png" | "gif" {
  if (mimeType === "image/jpeg") return "jpg";
  if (mimeType === "image/png") return "png";
  if (mimeType === "image/gif") return "gif";
  throw new UnsupportedMimeTypeError(mimeType);
}

export function emitImage(
  block: MDDMBlock,
  _tokens: LayoutTokens,
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

  const imageRun = new ImageRun({
    type: toDocxImageType(asset.mimeType),
    data: asset.bytes,
    transformation: {
      width: Math.round(mmToEmu(widthMm) / 9525),  // EMU → px (1 px = 9525 EMU)
      height: Math.round(mmToEmu(heightMm) / 9525),
    },
  });
  const options = { children: [imageRun] } as const;
  const paragraph = new Paragraph(options);
  (paragraph as unknown as { options: typeof options }).options = options;
  return [paragraph];
}
