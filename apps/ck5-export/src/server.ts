import { Hono } from "hono"
import { serve } from "@hono/node-server"
import { serveStatic } from "@hono/node-server/serve-static"
import { existsSync } from "node:fs"
import { join } from "node:path"
import { fileURLToPath } from "node:url"
import { renderDocxHandler } from "./routes/render-docx"
import { renderPdfHtmlHandler } from "./routes/render-pdf-html"

export const app = new Hono()

app.onError((err, c) => c.json({ error: err.message ?? "internal error" }, 500))
app.notFound((c) => c.json({ error: "not found" }, 404))

app.use("/assets/*", serveStatic({ root: "./public" }))
app.get("/health", (c) => c.json({ ok: true, service: "ck5-export" }))
app.post("/render/docx", renderDocxHandler)
app.post("/render/pdf-html", renderPdfHtmlHandler)

export const start = (port: number) => {
  serve({
    fetch: app.fetch,
    port,
  })
}

const isMain = process.argv[1] === fileURLToPath(import.meta.url)

if (isMain) {
  for (const variant of ["Regular", "Bold", "Italic", "BoldItalic"]) {
    if (!existsSync(join("./fonts", `Carlito-${variant}.ttf`))) {
      console.error(`FATAL: missing fonts/Carlito-${variant}.ttf`)
      process.exit(1)
    }
  }
  start(parseInt(process.env.PORT ?? "9001", 10))
}

export default app
