import { APIRequestContext } from "@playwright/test";
import { createHash, randomUUID } from "node:crypto";

export interface SeedResult {
  tenantId: string;
  docId: string;
  users: {
    author: { id: string; email: string };
    reviewer: { id: string; email: string };
    approver: { id: string; email: string };
    admin: { id: string; email: string };
  };
  cookies: Record<string, string>;
}

function shortHash(input: string): string {
  return createHash("sha256").update(input).digest("hex").slice(0, 8);
}

export async function seedTenant(
  request: APIRequestContext,
  opts: { workerIndex: number; testTitle: string }
): Promise<SeedResult> {
  const tenantId = `e2e_${opts.workerIndex}_${shortHash(opts.testTitle)}`;
  const docId = randomUUID();

  const response = await request.post("/internal/test/seed", {
    data: {
      tenantId,
      docId,
      roles: ["author", "reviewer", "approver", "admin"],
    },
  });

  if (!response.ok()) {
    throw new Error(`Seed API failed: ${response.status()} ${response.statusText()}`);
  }

  return (await response.json()) as SeedResult;
}

export async function resetTenant(
  request: APIRequestContext,
  tenantId: string
): Promise<void> {
  const response = await request.post("/internal/test/reset", {
    data: { tenantId },
  });

  if (!response.ok() && response.status() !== 204) {
    throw new Error(`Reset API failed: ${response.status()} ${response.statusText()}`);
  }
}
