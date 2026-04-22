import { randomUUID } from 'node:crypto';

import { test, expect, type APIRequestContext, type BrowserContext, type Page, type Request } from '@playwright/test';

import { contextAs, loginAs } from '../utils/auth';
import { resetTenant, seedTenant, type SeedResult } from '../utils/seed';

type SubmitBody = {
  route_id: string;
  content_hash: string;
};

type SubmitResponseBody = {
  instance_id: string;
  was_replay: boolean;
  etag: string;
};

type GovernanceEventRow = {
  id: string;
  tenant_id: string;
  event_type: string;
  actor_user_id: string;
  resource_type: string;
  resource_id: string;
  created_at: string;
  instance_id?: string;
  doc_id?: string;
};

const UUID_HEADER_RE = /^[0-9a-f-]{36}$/i;

let seeded: SeedResult;
let primaryDocId = '';
let secondaryDocId = '';

async function seedExtraDocument(request: APIRequestContext, tenantId: string): Promise<string> {
  const docId = randomUUID();
  const response = await request.post('/internal/test/seed', {
    data: {
      tenantId,
      docId,
      roles: ['author', 'reviewer', 'approver', 'admin'],
    },
  });

  expect(response.ok()).toBeTruthy();
  return docId;
}

function requireBaseURL(baseURL: string | undefined): string {
  if (!baseURL) {
    throw new Error('Playwright baseURL is required for contextAs');
  }
  return baseURL;
}

function isSubmitRequest(request: Request, docId: string): boolean {
  return request.method() === 'POST' && request.url().includes(`/api/v2/documents/${docId}/submit`);
}

function isSignoffRequest(request: Request): boolean {
  if (request.method() !== 'POST') {
    return false;
  }
  const url = request.url();
  return url.includes('/api/v2/signoff') || url.includes('/signoff');
}

function stateBadge(page: Page) {
  return page.locator('[data-testid="state-badge"], [aria-label^="Estado:"]').first();
}

async function stateBadgeText(page: Page): Promise<string> {
  const locator = stateBadge(page);
  const ariaLabel = await locator.getAttribute('aria-label');
  if (ariaLabel?.startsWith('Estado:')) {
    return ariaLabel.slice('Estado:'.length).trim();
  }
  const rawText = (await locator.textContent()) ?? '';
  return rawText.replace(/\s+/g, ' ').trim();
}

async function submitAsAuthor(page: Page, docId: string): Promise<{ request: Request; responseBody: SubmitResponseBody }> {
  await loginAs(page, seeded.cookies, 'author');
  await page.goto(`/documents/${docId}`);

  const submitRequestPromise = page.waitForRequest((request) => isSubmitRequest(request, docId));
  const submitResponsePromise = page.waitForResponse(
    (response) => response.request().method() === 'POST' && response.url().includes(`/api/v2/documents/${docId}/submit`),
  );

  await page.getByRole('button', { name: 'Submeter para revisão' }).click();
  await page.getByRole('button', { name: /^Submeter$/ }).click();

  const submitRequest = await submitRequestPromise;
  const submitResponse = await submitResponsePromise;
  expect(submitResponse.ok()).toBeTruthy();

  const responseBody = (await submitResponse.json()) as SubmitResponseBody;
  return { request: submitRequest, responseBody };
}

async function reviewerOrApproverSignoff(
  context: BrowserContext,
  docId: string,
): Promise<{ request: Request; page: Page }> {
  const page = await context.newPage();
  await page.goto('/approval/inbox');

  await expect(page.locator('tbody tr').first()).toBeVisible();
  const docRow = page.locator('tbody tr').filter({ hasText: docId }).first();
  if ((await docRow.count()) > 0) {
    await docRow.click();
  } else {
    await page.locator('tbody tr').first().click();
  }

  await page.getByRole('button', { name: /Assinar/i }).click();
  await expect(page.getByRole('dialog')).toBeVisible();

  const signoffRequestPromise = page.waitForRequest((request) => isSignoffRequest(request));

  await page.getByLabel('Senha').fill('test1234');
  await page.getByRole('button', { name: /Aprovar|Confirmar assinatura/i }).click();

  const request = await signoffRequestPromise;

  await expect.poll(async () => page.getByRole('dialog').count(), { timeout: 5000 }).toBe(0);

  return { request, page };
}

async function governanceEvents(
  request: APIRequestContext,
  params: { tenantId: string; docId?: string; instanceId?: string },
): Promise<GovernanceEventRow[]> {
  const query = new URLSearchParams({ tenantId: params.tenantId });
  if (params.docId) {
    query.set('docId', params.docId);
  }
  if (params.instanceId) {
    query.set('instanceId', params.instanceId);
  }

  const response = await request.get(`/internal/test/governance-events?${query.toString()}`);
  expect(response.ok()).toBeTruthy();
  return (await response.json()) as GovernanceEventRow[];
}

test.describe.serial('happy_path', () => {
  test.beforeAll(async ({ request }, testInfo) => {
    seeded = await seedTenant(request, {
      workerIndex: testInfo.workerIndex,
      testTitle: `${testInfo.title}-primary`,
    });
    primaryDocId = seeded.docId;
    secondaryDocId = await seedExtraDocument(request, seeded.tenantId);
  });

  test.afterAll(async ({ request }) => {
    await resetTenant(request, seeded.tenantId);
  });

  test('submits document and badge transitions to under_review', async ({ page }) => {
    const { request } = await submitAsAuthor(page, primaryDocId);

    const idempotencyKey = request.headerValue('idempotency-key');
    expect(idempotencyKey).toMatch(UUID_HEADER_RE);

    await expect.poll(() => stateBadgeText(page), { timeout: 5000 }).toBe('Em revisão');
    await expect(page.getByRole('button', { name: /Documento em revisão/i })).toBeVisible();
  });

  test('reviewer signs stage 1', async ({ browser, baseURL }) => {
    const context = await contextAs(browser, requireBaseURL(baseURL), seeded.cookies, 'reviewer');
    try {
      const { request } = await reviewerOrApproverSignoff(context, primaryDocId);
      expect(request.headerValue('idempotency-key')).toBeTruthy();
      expect(request.headerValue('if-match')).toBeTruthy();
    } finally {
      await context.close();
    }
  });

  test('approver signs stage 2 -> doc published', async ({ browser, baseURL }) => {
    const context = await contextAs(browser, requireBaseURL(baseURL), seeded.cookies, 'approver');
    try {
      const { page } = await reviewerOrApproverSignoff(context, primaryDocId);
      await expect.poll(() => stateBadgeText(page), { timeout: 5000 }).toBe('Publicado');
    } finally {
      await context.close();
    }
  });

  test('timeline shows 4 nodes with actors', async ({ page }) => {
    await loginAs(page, seeded.cookies, 'author');
    await page.goto(`/documents/${primaryDocId}`);

    const timelineNodes = page.locator('section[aria-label="Timeline de aprovação"] li');
    await expect(timelineNodes).toHaveCount(4);
    await expect(page.getByText(seeded.users.author.id)).toBeVisible();
    await expect(page.getByText(seeded.users.reviewer.id)).toBeVisible();
    await expect(page.getByText(seeded.users.approver.id)).toBeVisible();
  });

  test('idempotent replay — same key returns Idempotent-Replay: true', async ({ page, request }) => {
    const { request: firstSubmitRequest, responseBody } = await submitAsAuthor(page, secondaryDocId);
    const firstBody = firstSubmitRequest.postDataJSON() as SubmitBody;
    const idempotencyKey = firstSubmitRequest.headerValue('idempotency-key');

    expect(idempotencyKey).toMatch(UUID_HEADER_RE);

    const replayHeaders: Record<string, string> = {
      'Idempotency-Key': idempotencyKey ?? '',
    };
    const ifMatch = firstSubmitRequest.headerValue('if-match');
    if (ifMatch) {
      replayHeaders['If-Match'] = ifMatch;
    }

    const replayResponse = await page.request.post(`/api/v2/documents/${secondaryDocId}/submit`, {
      data: firstBody,
      headers: replayHeaders,
    });

    expect(replayResponse.headers()['idempotent-replay']).toBe('true');

    const events = await governanceEvents(request, {
      tenantId: seeded.tenantId,
      instanceId: responseBody.instance_id,
    });
    expect(events.length).toBe(1);
  });

  test('same key + mutated body -> 409 key_conflict', async ({ page, request }) => {
    const docId = await seedExtraDocument(request, seeded.tenantId);
    const { request: firstSubmitRequest } = await submitAsAuthor(page, docId);

    const firstBody = firstSubmitRequest.postDataJSON() as SubmitBody;
    const idempotencyKey = firstSubmitRequest.headerValue('idempotency-key') ?? randomUUID();

    const response = await page.request.post(`/api/v2/documents/${docId}/submit`, {
      data: {
        ...firstBody,
        content_hash: `${firstBody.content_hash}-mutated`,
      },
      headers: {
        'Idempotency-Key': idempotencyKey,
        ...(firstSubmitRequest.headerValue('if-match')
          ? { 'If-Match': firstSubmitRequest.headerValue('if-match') ?? '' }
          : {}),
      },
    });

    expect(response.status()).toBe(409);
    const body = (await response.json()) as {
      code?: string;
      error?: { code?: string };
    };
    expect(body.code ?? body.error?.code).toBe('idempotency.key_conflict');
  });

  test('stale If-Match -> 412, badge unchanged, toast shown', async ({ page, request }) => {
    const docId = await seedExtraDocument(request, seeded.tenantId);
    const { request: firstSubmitRequest } = await submitAsAuthor(page, docId);
    const firstBody = firstSubmitRequest.postDataJSON() as SubmitBody;

    const staleResponse = await page.request.post(`/api/v2/documents/${docId}/submit`, {
      data: firstBody,
      headers: {
        'Idempotency-Key': randomUUID(),
        'If-Match': '"stale-etag-000"',
      },
    });

    expect(staleResponse.status()).toBe(412);

    await expect(page.locator('[data-testid="app-toast-error"], [role="alert"]')).toContainText(/documento foi alterado/i);
    await expect.poll(() => stateBadgeText(page), { timeout: 5000 }).toBe('Em revisão');
  });

  test('governance event chain — exact types and order', async ({ request }) => {
    const events = await governanceEvents(request, {
      tenantId: seeded.tenantId,
      docId: primaryDocId,
    });

    expect(events.map((event) => event.event_type)).toEqual([
      'doc.submitted',
      'stage.activated',
      'signoff.recorded',
      'stage.passed',
      'signoff.recorded',
      'stage.passed',
      'doc.published',
    ]);

    let previousTimestamp = 0;
    for (const event of events) {
      expect(event.instance_id).toBeTruthy();
      expect(event.actor_user_id).toBeTruthy();
      const ts = Date.parse(event.created_at);
      expect(Number.isNaN(ts)).toBeFalsy();
      expect(ts).toBeGreaterThanOrEqual(previousTimestamp);
      previousTimestamp = ts;
    }
  });
});
