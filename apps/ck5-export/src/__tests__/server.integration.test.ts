import { describe, it, expect, beforeAll, afterAll } from "vitest"
import { serve } from "@hono/node-server"
import { app } from "../server"
import type { ServerType } from "@hono/node-server"

describe("server integration", () => {
  let server: ServerType
  let baseUrl: string

  beforeAll(async () => {
    await new Promise<void>((resolve) => {
      server = serve({ fetch: app.fetch, port: 0 }, (info) => {
        baseUrl = `http://127.0.0.1:${info.port}`
        resolve()
      })
    })
  })

  afterAll(async () => {
    await new Promise<void>((resolve, reject) => {
      server.close((err) => (err ? reject(err) : resolve()))
    })
  })

  it("GET /health → 200 ok", async () => {
    const res = await fetch(`${baseUrl}/health`)
    expect(res.status).toBe(200)
    const json = await res.json()
    expect(json.ok).toBe(true)
  })

  it("POST /render/docx → 200 with DOCX content-type", async () => {
    const res = await fetch(`${baseUrl}/render/docx`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ html: "<p>Integration test</p>" }),
    })
    expect(res.status).toBe(200)
    expect(res.headers.get("content-type")).toContain("wordprocessingml")
    const buf = await res.arrayBuffer()
    expect(buf.byteLength).toBeGreaterThan(0)
  })

  it("POST /render/pdf-html → 200 with text/html content-type", async () => {
    const res = await fetch(`${baseUrl}/render/pdf-html`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ html: "<p>Integration test</p>" }),
    })
    expect(res.status).toBe(200)
    expect(res.headers.get("content-type")).toContain("text/html")
    const text = await res.text()
    expect(text).toContain("<!DOCTYPE html>")
  })
})
