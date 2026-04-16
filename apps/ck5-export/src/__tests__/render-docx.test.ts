import { describe, expect, it } from "vitest"
import { app } from "../server"

describe("POST /render/docx", () => {
  it("returns 200 with ZIP magic bytes for valid fixture HTML", async () => {
    const html = `<p>Hello world</p>`
    const res = await app.request("/render/docx", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ html }),
    })

    expect(res.status).toBe(200)
    expect(res.headers.get("content-type")).toContain("wordprocessingml")
    const buf = await res.arrayBuffer()
    expect(buf.byteLength).toBeGreaterThan(0)
    const bytes = new Uint8Array(buf)
    expect(bytes[0]).toBe(0x50)
    expect(bytes[1]).toBe(0x4b)
    expect(bytes[2]).toBe(0x03)
    expect(bytes[3]).toBe(0x04)
  })

  it("returns 400 when html is missing", async () => {
    const res = await app.request("/render/docx", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({}),
    })
    expect(res.status).toBe(400)
    const json = await res.json()
    expect(json.error).toBeTruthy()
  })

  it("returns 400 for malformed JSON", async () => {
    const res = await app.request("/render/docx", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: "not json",
    })
    expect(res.status).toBe(400)
  })
})
