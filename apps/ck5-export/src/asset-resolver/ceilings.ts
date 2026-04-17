// Resource ceilings for asset resolution and export. Mirrors spec section
// "Global Resource Ceilings".

export const RESOURCE_CEILINGS = {
  // Per asset
  maxImageSizeBytes: 5 * 1024 * 1024, // 5 MB
  maxImageDimensionPx: 10000,

  // Per document
  maxTotalAssetBytes: 50 * 1024 * 1024, // 50 MB
  maxImagesPerDocument: 200,

  // Content-level
  maxBlockCount: 5000,
  maxNestingDepth: 10,
  maxTextRunLength: 100000,

  // Pipeline timings
  maxDocxGenerationMs: 30_000,
  maxHtmlPayloadBytes: 10 * 1024 * 1024, // 10 MB
  maxGotenbergConversionMs: 60_000,
  maxConcurrentExportsPerUser: 3,
} as const;

export type ResourceCeilings = typeof RESOURCE_CEILINGS;

export class ResourceCeilingExceededError extends Error {
  constructor(
    public readonly limit: keyof ResourceCeilings,
    public readonly observed: number,
    public readonly allowed: number,
  ) {
    super(`Resource ceiling "${String(limit)}" exceeded: observed ${observed}, allowed ${allowed}`);
    this.name = "ResourceCeilingExceededError";
  }
}
