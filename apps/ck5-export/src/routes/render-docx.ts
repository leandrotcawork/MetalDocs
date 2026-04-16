import { type Context } from "hono"
import { Packer } from "docx"
import { type ResolvedAsset, AssetResolver } from "../asset-resolver"
import { collectImageUrls, emitDocxFromExportTree } from "../ck5-docx-emitter"
import { htmlToExportTree } from "../html-to-export-tree"
import { defaultLayoutTokens } from "../layout-tokens"

export async function renderDocxHandler(c: Context): Promise<Response> {
  try {
    const body = await c.req.json().catch(() => null)
    if (body === null) {
      return c.json({ error: "invalid JSON" }, 400)
    }

    if (!body?.html || typeof body.html !== "string") {
      return c.json({ error: "html required" }, 400)
    }

    const tree = htmlToExportTree(body.html)
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
