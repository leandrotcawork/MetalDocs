/**
 * Per-worker isolation fixture.
 *
 * Each parallel Playwright worker gets its own tenant + deterministic IDs.
 * No test can affect another worker's data.
 *
 * Usage:
 *   import { test } from '../fixtures/isolation';
 *   test('my test', async ({ isolated }) => {
 *     const { tenantId, users, docId } = isolated;
 *   });
 */

import { test as base, expect } from '@playwright/test';
import { v5 as uuidv5 } from 'uuid';

// Stable namespace UUID for deterministic test IDs (generated once, never change).
const WORKER_NAMESPACE = '6ba7b810-9dad-11d1-80b4-00c04fd430c8';

export interface SeedUsers {
  author: { id: string; email: string; cookie: string };
  reviewer: { id: string; email: string; cookie: string };
  approver: { id: string; email: string; cookie: string };
  admin: { id: string; email: string; cookie: string };
}

export interface IsolatedFixture {
  tenantId: string;
  docId: string;
  users: SeedUsers;
  /** Resets tenant scope — call in afterEach for aggressive cleanup */
  teardown: () => Promise<void>;
}

type IsolatedTest = {
  isolated: IsolatedFixture;
};

function shortId(title: string, workerIndex: number): string {
  // 8-char stable slug from test title + worker index
  return uuidv5(`${workerIndex}:${title}`, WORKER_NAMESPACE).replace(/-/g, '').slice(0, 8);
}

export const test = base.extend<IsolatedTest>({
  isolated: async ({ request, page }, use, testInfo) => {
    const workerIndex = testInfo.workerIndex;
    const sid = shortId(testInfo.title, workerIndex);

    // Seed via admin API (only available when METALDOCS_E2E=1)
    const seedResp = await request.post('/internal/test/seed', {
      data: {
        tenantId: `e2e_${workerIndex}_${sid}`,
        // Deterministic doc ID: reruns reproduce same UUID
        docId: uuidv5(`doc:${sid}`, WORKER_NAMESPACE),
        roles: ['author', 'reviewer', 'approver', 'admin'],
      },
    });

    if (!seedResp.ok()) {
      throw new Error(
        `Seed API failed: ${seedResp.status()} — is METALDOCS_E2E=1 set?`
      );
    }

    const seed = await seedResp.json() as {
      tenantId: string;
      docId: string;
      users: SeedUsers;
    };

    const teardown = async () => {
      await request.post('/internal/test/reset', {
        data: { tenantId: seed.tenantId },
      });
    };

    await use({
      tenantId: seed.tenantId,
      docId: seed.docId,
      users: seed.users,
      teardown,
    });

    // Always tear down after test, even on failure.
    await teardown();
  },
});

export { expect };

/**
 * Playwright project split for Task 11.1:
 *
 *   parallel-flows: workers=3  (stateless flows — isolation fixture handles tenant separation)
 *   serial-clock:   workers=1  (clock-advance tests — scheduled_publish.spec.ts)
 *
 * These are configured in playwright.config.ts under `projects`.
 */
