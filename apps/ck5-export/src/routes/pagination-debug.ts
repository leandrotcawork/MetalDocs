import { Hono } from "hono";
import { paginationStore } from "../pagination/parity-store";

export function paginationDebugRoute(): Hono {
  const route = new Hono();

  route.get("/pagination-debug/:docId", (c) => {
    if (process.env.NODE_ENV === "production") {
      return c.notFound();
    }

    const docId = c.req.param("docId");
    const stored = paginationStore.get(docId);

    if (!stored) {
      return c.json({ error: "unknown-doc" }, 404);
    }

    return c.json({ ...stored.report, updatedAt: stored.updatedAt }, 200);
  });

  return route;
}
