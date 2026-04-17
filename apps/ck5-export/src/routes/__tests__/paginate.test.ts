import { beforeEach, describe, expect, it, vi } from "vitest";
import { Hono } from "hono";
import { paginateRoute } from "../paginate";
import { paginateWithChromium, PaginatorTimeoutError } from "../../pagination/paginate-with-chromium";
import { PaginationDegraded } from "../../pagination/pool-retry";

vi.mock("../../pagination/paginate-with-chromium", () => {
  class MockPaginatorTimeoutError extends Error {
    constructor(public readonly ms: number) {
      super(`pagination timed out after ${ms}ms`);
      this.name = "PaginatorTimeoutError";
    }
  }

  return {
    paginateWithChromium: vi.fn(),
    PaginatorTimeoutError: MockPaginatorTimeoutError,
  };
});

vi.mock("../../pagination/pool-retry", () => {
  class MockPaginationDegraded extends Error {
    constructor(public readonly reason: "worker-crash" | "pool-exhausted" | "runtime-error") {
      super(`pagination degraded: ${reason}`);
      this.name = "PaginationDegraded";
    }
  }

  return { PaginationDegraded: MockPaginationDegraded };
});

const fakePool = { acquire: vi.fn(), release: vi.fn(), replace: vi.fn() } as any;
const app = new Hono().route("/", paginateRoute(fakePool));

describe("POST /paginate", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("200 with breaks on clean input", async () => {
    vi.mocked(paginateWithChromium).mockResolvedValue([{ bid: "a", pageNumber: 2 }]);

    const res = await app.request("/paginate", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        html: '<p data-mddm-bid="aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa">x</p>',
      }),
    });

    expect(res.status).toBe(200);
    await expect(res.json()).resolves.toEqual({ breaks: [{ bid: "a", pageNumber: 2 }] });
  });

  it("400 on malformed input (no html field)", async () => {
    const res = await app.request("/paginate", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ editorBids: [] }),
    });

    expect(res.status).toBe(400);
  });

  it("422 on bid collision (duplicate bids in html)", async () => {
    const res = await app.request("/paginate", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        html: [
          '<p data-mddm-bid="aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa">x</p>',
          '<p data-mddm-bid="aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa">y</p>',
        ].join(""),
      }),
    });

    expect(res.status).toBe(422);
    await expect(res.json()).resolves.toMatchObject({
      error: "bid-collision",
      bids: ["aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa"],
    });
  });

  it("422 on editor-server-desync (editorBids contains bids not in html)", async () => {
    const res = await app.request("/paginate", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        html: '<p data-mddm-bid="aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa">x</p>',
        editorBids: [
          "aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa",
          "bbbbbbbb-bbbb-4bbb-bbbb-bbbbbbbbbbbb",
        ],
      }),
    });

    expect(res.status).toBe(422);
    await expect(res.json()).resolves.toEqual({
      error: "editor-server-desync",
      missingBids: ["bbbbbbbb-bbbb-4bbb-bbbb-bbbbbbbbbbbb"],
    });
  });

  it("503 on PaginationDegraded worker-crash", async () => {
    vi.mocked(paginateWithChromium).mockRejectedValue(new PaginationDegraded("worker-crash"));

    const res = await app.request("/paginate", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        html: '<p data-mddm-bid="aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa">x</p>',
      }),
    });

    expect(res.status).toBe(503);
  });

  it("503 on PaginationDegraded pool-exhausted", async () => {
    vi.mocked(paginateWithChromium).mockRejectedValue(new PaginationDegraded("pool-exhausted"));

    const res = await app.request("/paginate", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        html: '<p data-mddm-bid="aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa">x</p>',
      }),
    });

    expect(res.status).toBe(503);
  });

  it("200 with breaks:[] and degraded:true on PaginationDegraded runtime-error (graceful fallback)", async () => {
    vi.mocked(paginateWithChromium).mockRejectedValue(new PaginationDegraded("runtime-error"));

    const res = await app.request("/paginate", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        html: '<p data-mddm-bid="aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa">x</p>',
      }),
    });

    expect(res.status).toBe(200);
    await expect(res.json()).resolves.toEqual({ breaks: [], degraded: true });
  });

  it("504 on PaginatorTimeoutError", async () => {
    vi.mocked(paginateWithChromium).mockRejectedValue(new PaginatorTimeoutError(15000));

    const res = await app.request("/paginate", {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        html: '<p data-mddm-bid="aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa">x</p>',
      }),
    });

    expect(res.status).toBe(504);
  });
});
