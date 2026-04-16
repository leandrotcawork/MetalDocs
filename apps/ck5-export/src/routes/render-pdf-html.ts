import { type Context } from "hono"
import { type ResolvedAsset, AssetResolver } from "../asset-resolver"
import { rewriteImgSrcToDataUri } from "../inline-asset-rewriter"
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

    const urls = extractImgSrcs(body.html)
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

    const rewritten = rewriteImgSrcToDataUri(body.html, assetMap)
    const wrapped = wrapInPrintDocument(rewritten)

    return c.body(wrapped, 200, { "Content-Type": "text/html; charset=utf-8" })
  } catch (error) {
    const message = error instanceof Error ? error.message : "unknown error"
    return c.json({ error: message }, 500)
  }
}
