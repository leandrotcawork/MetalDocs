import { Hono } from "hono";
import { paginateWithChromium, PaginatorTimeoutError } from "../pagination/paginate-with-chromium";
import { PaginationDegraded } from "../pagination/pool-retry";
import type { PlaywrightPool } from "../pagination/playwright-pool";
import { validateBids, validateEditorBidSet } from "../pagination/validator";

export function paginateRoute(pool: PlaywrightPool): Hono {
  const route = new Hono();

  route.post("/paginate", async (c) => {
    const body = await c.req.json().catch(() => null);
    if (body === null || typeof body?.html !== "string") {
      return c.json({ error: "html required" }, 400);
    }

    const html = body.html;
    const bidValidation = validateBids(html);
    if (!bidValidation.ok && bidValidation.severity === "error") {
      return c.json({ error: bidValidation.error, bids: bidValidation.bids }, 422);
    }

    if (Array.isArray(body.editorBids)) {
      const editorValidation = validateEditorBidSet(html, body.editorBids);
      if (!editorValidation.ok) {
        return c.json({ error: "editor-server-desync", missingBids: editorValidation.missingBids }, 422);
      }
    }

    try {
      const breaks = await paginateWithChromium(pool, html, { timeoutMs: 15_000 });
      return c.json({ breaks }, 200);
    } catch (error) {
      if (error instanceof PaginatorTimeoutError) {
        return c.json({ error: "paginator-timeout" }, 504);
      }
      if (error instanceof PaginationDegraded) {
        if (error.reason === "runtime-error") {
          return c.json({ breaks: [], degraded: true }, 200);
        }
        return c.json({ error: "paginator-unavailable", reason: error.reason }, 503);
      }
      return c.json({ error: "paginator-unavailable" }, 503);
    }
  });

  return route;
}
