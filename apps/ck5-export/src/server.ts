import { Hono } from "hono"
import { serve } from "@hono/node-server"
import { fileURLToPath } from "node:url"
import { renderDocxHandler } from "./routes/render-docx"
import { renderPdfHtmlHandler } from "./routes/render-pdf-html"

export const app = new Hono()

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
  start(parseInt(process.env.PORT ?? "9001", 10))
}

export default app
