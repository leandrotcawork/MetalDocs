import type { MDDMEnvelope } from "../../adapter";
import { defaultLayoutTokens, type LayoutTokens } from "../layout-ir";
import { canonicalizeAndMigrate } from "../canonicalize-migrate";
import { collectImageUrls, mddmToDocx } from "../docx-emitter";
import {
  AssetResolver,
  RESOURCE_CEILINGS,
  ResourceCeilingExceededError,
  type ResolvedAsset,
} from "../asset-resolver";

export type ExportDocxOptions = {
  /** Optional resolver injection point — defaults to a fresh AssetResolver. */
  assetResolver?: AssetResolver;
};

export async function exportDocx(
  envelope: MDDMEnvelope,
  tokens?: LayoutTokens,
  options: ExportDocxOptions = {},
): Promise<Blob> {
  const canonical = await canonicalizeAndMigrate(envelope);
  const resolvedTokens = tokens ?? defaultLayoutTokens;

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
  return mddmToDocx(canonical, resolvedTokens, assetMap);
}
