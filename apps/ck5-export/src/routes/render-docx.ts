import { type Context } from "hono"
import { Packer } from "docx"
import { type ResolvedAsset, AssetResolver } from "../asset-resolver"
import { collectImageUrls, emitDocxFromExportTree } from "../docx-emitter"
import { htmlToExportTree } from "../html-to-export-tree"
import { defaultLayoutTokens } from "../layout-ir"
import { validateBids } from "../pagination/validator"

export async function renderDocxHandler(c: Context): Promise<Response> {
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

    const tree = htmlToExportTree(html)
    const urls = collectImageUrls(tree)

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

    const doc = emitDocxFromExportTree(tree, defaultLayoutTokens, assetMap)
    const buf = await Packer.toBuffer(doc)

    return c.body(buf as unknown as ReadableStream, 200, {
      "Content-Type": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
    })
  } catch (error) {
    const message = error instanceof Error ? error.message : "unknown error"
    return c.json({ error: message }, 500)
  }
}
