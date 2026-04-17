import { Hono } from "hono"
import { serve } from "@hono/node-server"
import { serveStatic } from "@hono/node-server/serve-static"
import { existsSync } from "node:fs"
import { join } from "node:path"
import { fileURLToPath } from "node:url"
import { renderDocxHandler } from "./routes/render-docx"
import { renderPdfHtmlHandler } from "./routes/render-pdf-html"
import { PlaywrightPool } from "./pagination/playwright-pool"
import { paginateRoute } from "./routes/paginate"

export const app = new Hono()
const pool = new PlaywrightPool({ size: Number(process.env.CHROMIUM_POOL_SIZE ?? 3) })

app.onError((err, c) => c.json({ error: err.message ?? "internal error" }, 500))
app.notFound((c) => c.json({ error: "not found" }, 404))

app.use("/assets/*", serveStatic({ root: "./public" }))
app.get("/health", (c) => c.json({ ok: true, service: "ck5-export" }))
app.post("/render/docx", renderDocxHandler)
app.post("/render/pdf-html", renderPdfHtmlHandler)
app.route("/", paginateRoute(pool))

export const start = (port: number) => {
  serve({
    fetch: app.fetch,
    port,
  })
}

const isMain = process.argv[1] === fileURLToPath(import.meta.url)

if (isMain) {
  const main = async () => {
    for (const variant of ["Regular", "Bold", "Italic", "BoldItalic"]) {
      if (!existsSync(join("./fonts", `Carlito-${variant}.ttf`))) {
        console.error(`FATAL: missing fonts/Carlito-${variant}.ttf`)
        process.exit(1)
      }
    }
    await pool.init()
    process.on("SIGTERM", () => {
      void pool.shutdown().finally(() => process.exit(0))
    })
    start(parseInt(process.env.PORT ?? "9001", 10))
  }
  void main().catch((error) => {
    console.error(error)
    process.exit(1)
  })
}

export default app
