import { wrapInPrintDocument } from "../print-stylesheet/wrap-print-document";
import { PRINT_STYLESHEET } from "../print-stylesheet";
import {
  AssetResolver,
  RESOURCE_CEILINGS,
  ResourceCeilingExceededError,
  type ResolvedAsset,
} from "../asset-resolver";
import { rewriteImgSrcToDataUri } from "./inline-asset-rewriter";

export type ExportPdfParams = {
  /** Body HTML produced by blocksToFullHTML (still containing /api/images/... src refs). */
  bodyHtml: string;
  /** Document ID — used in the backend endpoint path. */
  documentId: string;
  /** Optional resolver injection point. */
  assetResolver?: AssetResolver;
};

const PDF_MIME = "application/pdf";

/** Extract every <img src> URL from the body HTML for asset resolution. */
function extractImageUrls(html: string): string[] {
  const out = new Set<string>();
  const re = /<img\b[^>]*\bsrc\s*=\s*(["'])([^"']+)\1/gi;
  let m: RegExpExecArray | null;
  while ((m = re.exec(html)) !== null) {
    const url = m[2];
    if (url) out.add(url);
  }
  return Array.from(out);
}

export async function exportPdf({
  bodyHtml,
  documentId,
  assetResolver,
}: ExportPdfParams): Promise<Blob> {
  // Resolve and inline images so the HTML sent to Gotenberg has zero auth-bound URLs.
  const urls = extractImageUrls(bodyHtml);
  if (urls.length > RESOURCE_CEILINGS.maxImagesPerDocument) {
    throw new ResourceCeilingExceededError(
      "maxImagesPerDocument",
      urls.length,
      RESOURCE_CEILINGS.maxImagesPerDocument,
    );
  }

  const resolver = assetResolver ?? new AssetResolver();
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

  const inlinedBody = rewriteImgSrcToDataUri(bodyHtml, assetMap);
  const fullHtml = wrapInPrintDocument(inlinedBody);

  const htmlBytes = new TextEncoder().encode(fullHtml).byteLength;
  if (htmlBytes > RESOURCE_CEILINGS.maxHtmlPayloadBytes) {
    throw new ResourceCeilingExceededError(
      "maxHtmlPayloadBytes",
      htmlBytes,
      RESOURCE_CEILINGS.maxHtmlPayloadBytes,
    );
  }

  const formData = new FormData();
  formData.append("index.html", new Blob([fullHtml], { type: "text/html" }), "index.html");
  formData.append("style.css", new Blob([PRINT_STYLESHEET], { type: "text/css" }), "style.css");

  const response = await fetch(
    `/api/v1/documents/${encodeURIComponent(documentId)}/render/pdf`,
    {
      method: "POST",
      credentials: "same-origin",
      body: formData,
    },
  );

  if (!response.ok) {
    const text = await response.text().catch(() => "");
    throw new Error(`PDF render failed: ${response.status} ${text}`);
  }

  const contentType = (response.headers.get("Content-Type") ?? "").toLowerCase();
  if (!contentType.includes(PDF_MIME)) {
    throw new Error(`Unexpected Content-Type from PDF endpoint: ${contentType}`);
  }

  const arrayBuffer = await response.arrayBuffer();
  return new Blob([arrayBuffer], { type: PDF_MIME });
}
