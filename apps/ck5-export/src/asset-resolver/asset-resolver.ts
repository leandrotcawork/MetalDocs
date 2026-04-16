import { isAllowlistedAssetUrl } from "./allowlist";
import { RESOURCE_CEILINGS, ResourceCeilingExceededError } from "./ceilings";

export type AllowedMimeType = "image/png" | "image/jpeg" | "image/gif" | "image/webp";

export type ResolvedAsset = {
  bytes: Uint8Array;
  mimeType: AllowedMimeType;
  sizeBytes: number;
};

export class AssetResolverError extends Error {
  constructor(message: string, public readonly code: string) {
    super(message);
    this.name = "AssetResolverError";
  }
}

const ALLOWED_MIME: ReadonlySet<AllowedMimeType> = new Set([
  "image/png",
  "image/jpeg",
  "image/gif",
  "image/webp",
]);

// Magic-byte signatures used to validate declared Content-Type.
function detectMimeByMagic(bytes: Uint8Array): AllowedMimeType | null {
  if (bytes.length >= 8 &&
      bytes[0] === 0x89 && bytes[1] === 0x50 && bytes[2] === 0x4e && bytes[3] === 0x47) {
    return "image/png";
  }
  if (bytes.length >= 3 && bytes[0] === 0xff && bytes[1] === 0xd8 && bytes[2] === 0xff) {
    return "image/jpeg";
  }
  if (bytes.length >= 6 && bytes[0] === 0x47 && bytes[1] === 0x49 && bytes[2] === 0x46 &&
      bytes[3] === 0x38 && (bytes[4] === 0x37 || bytes[4] === 0x39) && bytes[5] === 0x61) {
    return "image/gif";
  }
  if (bytes.length >= 12 &&
      bytes[0] === 0x52 && bytes[1] === 0x49 && bytes[2] === 0x46 && bytes[3] === 0x46 &&
      bytes[8] === 0x57 && bytes[9] === 0x45 && bytes[10] === 0x42 && bytes[11] === 0x50) {
    return "image/webp";
  }
  return null;
}

export class AssetResolver {
  async resolveAsset(url: string): Promise<ResolvedAsset> {
    if (!isAllowlistedAssetUrl(url)) {
      throw new AssetResolverError(`Asset URL not allowlisted: ${url}`, "NOT_ALLOWLISTED");
    }

    const response = await fetch(url, {
      credentials: "same-origin",
      signal: AbortSignal.timeout(5_000),
    });
    if (!response.ok) {
      throw new AssetResolverError(
        `Asset fetch failed: ${response.status} ${response.statusText}`,
        "FETCH_FAILED",
      );
    }

    const declaredType = (response.headers.get("Content-Type") ?? "").split(";")[0]!.trim().toLowerCase() as AllowedMimeType;
    if (!ALLOWED_MIME.has(declaredType)) {
      throw new AssetResolverError(`Disallowed MIME type: ${declaredType}`, "MIME_NOT_ALLOWED");
    }

    const buffer = await response.arrayBuffer();
    const bytes = new Uint8Array(buffer);

    if (bytes.byteLength > RESOURCE_CEILINGS.maxImageSizeBytes) {
      throw new ResourceCeilingExceededError(
        "maxImageSizeBytes",
        bytes.byteLength,
        RESOURCE_CEILINGS.maxImageSizeBytes,
      );
    }

    const detected = detectMimeByMagic(bytes);
    if (detected === null || detected !== declaredType) {
      throw new AssetResolverError(
        `Asset magic bytes do not match declared Content-Type: declared=${declaredType}, detected=${detected ?? "unknown"}`,
        "MAGIC_MISMATCH",
      );
    }

    return {
      bytes,
      mimeType: detected,
      sizeBytes: bytes.byteLength,
    };
  }
}
