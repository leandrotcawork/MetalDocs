import type { ResolvedAsset } from "./asset-resolver";

function bytesToBase64(bytes: Uint8Array): string {
  // Chunked to avoid stack overflow on large arrays and quadratic string allocs.
  // btoa is available in browsers and modern Node test environments via jsdom.
  let binary = "";
  const chunkSize = 8192;
  for (let i = 0; i < bytes.byteLength; i += chunkSize) {
    binary += String.fromCharCode(...bytes.subarray(i, i + chunkSize));
  }
  return globalThis.btoa(binary);
}

function toDataUri(asset: ResolvedAsset): string {
  return `data:${asset.mimeType};base64,${bytesToBase64(asset.bytes)}`;
}

/**
 * Rewrite every <img src="..."> attribute whose URL is present in `assetMap`
 * to a data: URI. Untouched img tags whose URL is missing from the map are
 * preserved verbatim — Gotenberg will then fail or skip them, which surfaces
 * a clear missing-asset issue rather than silently dropping content.
 */
export function rewriteImgSrcToDataUri(
  html: string,
  assetMap: ReadonlyMap<string, ResolvedAsset>,
): string {
  // Match <img ... src="URL" ... /> with both single and double quoted src.
  return html.replace(
    /(<img\b[^>]*\bsrc\s*=\s*)(["'])([^"'>]+)\2/gi,
    (match, prefix: string, quote: string, url: string) => {
      const asset = assetMap.get(url);
      if (!asset) return match;
      return `${prefix}${quote}${toDataUri(asset)}${quote}`;
    },
  );
}
