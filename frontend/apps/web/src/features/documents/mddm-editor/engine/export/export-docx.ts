import type { MDDMEnvelope } from "../../adapter";
import type { RendererPin } from "../../../../../lib.types";
import { canonicalizeAndMigrate } from "../canonicalize-migrate";
import { collectImageUrls } from "../docx-emitter";
import {
  AssetResolver,
  RESOURCE_CEILINGS,
  ResourceCeilingExceededError,
  type ResolvedAsset,
} from "../asset-resolver";
import { loadCurrentRenderer, loadPinnedRenderer } from "../renderers/registry";

export type ExportDocxOptions = {
  /** Renderer pin from the version record. `null` or omitted → current renderer. */
  rendererPin?: RendererPin | null;
  /** Optional resolver injection point — defaults to a fresh AssetResolver. */
  assetResolver?: AssetResolver;
};

export async function exportDocx(
  envelope: MDDMEnvelope,
  options: ExportDocxOptions = {},
): Promise<Blob> {
  const renderer = options.rendererPin
    ? await loadPinnedRenderer(options.rendererPin)
    : await loadCurrentRenderer();

  const canonical = await canonicalizeAndMigrate(envelope);

  // Resolve assets BEFORE emitter runs so the emitter receives bytes.
  const urls = collectImageUrls(canonical);
  if (urls.length > RESOURCE_CEILINGS.maxImagesPerDocument) {
    throw new ResourceCeilingExceededError(
      "maxImagesPerDocument",
      urls.length,
      RESOURCE_CEILINGS.maxImagesPerDocument,
    );
  }

  const resolver = options.assetResolver ?? new AssetResolver();
  const assetMap = new Map<string, ResolvedAsset>();
  let totalBytes = 0;
  for (const url of urls) {
    const asset = await resolver.resolveAsset(url);
    totalBytes += asset.sizeBytes;
    if (totalBytes > RESOURCE_CEILINGS.maxTotalAssetBytes) {
      throw new ResourceCeilingExceededError(
        "maxTotalAssetBytes",
        totalBytes,
        RESOURCE_CEILINGS.maxTotalAssetBytes,
      );
    }
    assetMap.set(url, asset);
  }

  // mddmToDocx guarantees the DOCX MIME type on the returned blob.
  return renderer.mddmToDocx(canonical, renderer.tokens, assetMap);
}
