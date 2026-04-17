import { describe, expect, it } from "vitest"
import app from "../server"

describe("server health", () => {
  it("returns service health payload", async () => {
    const res = await app.request("/health")

    expect(res.status).toBe(200)
    expect(await res.json()).toEqual({ ok: true, service: "ck5-export" })
  })
})
