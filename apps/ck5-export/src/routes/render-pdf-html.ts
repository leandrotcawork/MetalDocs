import { type Context } from "hono"
import { type ResolvedAsset, AssetResolver } from "../asset-resolver"
import { rewriteImgSrcToDataUri } from "../inline-asset-rewriter"
import { validateBids } from "../pagination/validator"
import { wrapInPrintDocument } from "../print-stylesheet"

function extractImgSrcs(html: string): string[] {
  const urls: string[] = []
  const re = /\bsrc\s*=\s*["']([^"']+)["']/gi
  let m: RegExpExecArray | null
  while ((m = re.exec(html)) !== null) {
    urls.push(m[1])
  }
  return [...new Set(urls)]
}

export async function renderPdfHtmlHandler(c: Context): Promise<Response> {
  try {
    const body = await c.req.json().catch(() => null)
    if (body === null) {
      return c.json({ error: "invalid JSON" }, 400)
    }

    if (!body?.html || typeof body.html !== "string") {
      return c.json({ error: "html required" }, 400)
    }

    const html = body.html
    const v = validateBids(html);
    if (!v.ok && v.severity === 'error') {
      return c.json({ error: v.error, bids: (v as any).bids }, 422);
    }
    if (!v.ok && v.severity === 'warn') {
      console.warn(`mddm:${v.error}`, (v as any).elements);
    }

    const urls = extractImgSrcs(html)
    const assetResolver = new AssetResolver()
    const assetMap = new Map<string, ResolvedAsset>()

    for (const url of urls) {
      try {
        const asset = await assetResolver.resolveAsset(url)
        assetMap.set(url, asset)
      } catch (error) {
        console.warn(`Failed to resolve asset ${url}`, error)
      }
    }

    const rewritten = rewriteImgSrcToDataUri(html, assetMap)
    const wrapped = wrapInPrintDocument(rewritten)

    return c.body(wrapped, 200, { "Content-Type": "text/html; charset=utf-8" })
  } catch (error) {
    const message = error instanceof Error ? error.message : "unknown error"
    return c.json({ error: message }, 500)
  }
}
