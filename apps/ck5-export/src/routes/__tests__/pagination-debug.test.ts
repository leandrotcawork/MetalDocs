import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { Hono } from "hono";
import { paginationDebugRoute } from "../pagination-debug";
import { paginationStore } from "../../pagination/parity-store";

const app = new Hono().route("/", paginationDebugRoute());

describe("GET /pagination-debug/:docId", () => {
  beforeEach(() => {
    paginationStore.clear();
  });

  afterEach(() => {
    vi.unstubAllEnvs();
  });

  it("returns 200 with report and updatedAt for known doc in development", async () => {
    vi.stubEnv("NODE_ENV", "development");
    paginationStore.put("doc-1", {
      docId: "doc-1",
      editorBreaks: [{ afterBid: "aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa", pageNumber: 1, yPx: 0 }],
      serverBreaks: [{ bid: "aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa", pageNumber: 1 }],
      reconciled: [{ afterBid: "aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa", pageNumber: 1, source: "editor" }],
      logs: {
        exactMatches: 1,
        minorDrift: 0,
        majorDrift: 0,
        orphanedEditor: 0,
        serverOnly: 0,
      },
      driftStats: {
        totalBreaks: 1,
        driftRatio: 0,
      },
    });

    const res = await app.request("/pagination-debug/doc-1");
    const body = await res.json();

    expect(res.status).toBe(200);
    expect(body).toMatchObject({
      docId: "doc-1",
      reconciled: expect.any(Array),
      driftStats: {
        totalBreaks: 1,
        driftRatio: 0,
      },
      updatedAt: expect.any(Number),
    });
  });

  it("returns 404 unknown-doc for missing doc in development", async () => {
    vi.stubEnv("NODE_ENV", "development");

    const res = await app.request("/pagination-debug/missing");

    expect(res.status).toBe(404);
    await expect(res.json()).resolves.toEqual({ error: "unknown-doc" });
  });

  it("returns 404 in production even when seeded", async () => {
    vi.stubEnv("NODE_ENV", "production");
    paginationStore.put("doc-2", {
      docId: "doc-2",
      editorBreaks: [{ afterBid: "aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa", pageNumber: 1, yPx: 0 }],
      serverBreaks: [{ bid: "aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa", pageNumber: 1 }],
      reconciled: [{ afterBid: "aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa", pageNumber: 1, source: "editor" }],
      logs: {
        exactMatches: 1,
        minorDrift: 0,
        majorDrift: 0,
        orphanedEditor: 0,
        serverOnly: 0,
      },
      driftStats: {
        totalBreaks: 1,
        driftRatio: 0,
      },
    });

    const res = await app.request("/pagination-debug/doc-2");

    expect(res.status).toBe(404);
  });
});
