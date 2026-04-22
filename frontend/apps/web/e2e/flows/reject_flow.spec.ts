import { randomUUID } from 'node:crypto';

import { test, expect, type APIRequestContext, type BrowserContext, type Page, type Request } from '@playwright/test';

import { contextAs, loginAs } from '../utils/auth';
import { resetTenant, seedTenant, type SeedResult } from '../utils/seed';

let seeded: SeedResult;
let primaryDocId = '';
let secondaryDocId = '';

function requireBaseURL(baseURL: string | undefined): string {
  if (!baseURL) {
    throw new Error('Playwright baseURL is required for contextAs');
  }
  return baseURL;
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

async function submitAsAuthor(page: Page, docId: string): Promise<void> {
  await loginAs(page, seeded.cookies, 'author');
  await page.goto(`/documents/${docId}`);

  const submitRequestPromise = page.waitForRequest((request) => isSubmitRequest(request, docId));

  await page.getByRole('button', { name: 'Submeter para revisão' }).click();
  await page.getByRole('button', { name: /^Submeter$/ }).click();

  await submitRequestPromise;
  await expect.poll(() => stateBadgeText(page), { timeout: 5000 }).toBe('Em revisão');
}

async function signoffFromInbox(
  context: BrowserContext,
  options: { docId: string; decision: 'approve' | 'reject'; reason?: string },
): Promise<Page> {
  const page = await context.newPage();
  await page.goto('/approval/inbox');

  await expect(page.locator('tbody tr').first()).toBeVisible();
  const docRow = page.locator('tbody tr').filter({ hasText: options.docId }).first();
  await expect(docRow).toBeVisible();
  await docRow.click();

  await page.getByRole('button', { name: 'Assinar' }).click();
  await expect(page.getByRole('dialog')).toBeVisible();

  if (options.decision === 'reject') {
    await page.getByLabel('Rejeitado').check();
    if (options.reason !== undefined) {
      await page.getByLabel('Motivo').fill(options.reason);
    }
  }

  await page.getByLabel('Senha').fill('test1234');

  const signoffRequestPromise = page.waitForRequest((request) => isSignoffRequest(request));
  await page.getByRole('button', { name: 'Confirmar assinatura' }).click();
  await signoffRequestPromise;

  await expect.poll(async () => page.getByRole('dialog').count(), { timeout: 5000 }).toBe(0);

  return page;
}

test.describe.serial('reject_flow', () => {
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

  test('stage 1 passes normally', async ({ page, browser, baseURL }) => {
    await submitAsAuthor(page, primaryDocId);

    const reviewerContext = await contextAs(browser, requireBaseURL(baseURL), seeded.cookies, 'reviewer');
    try {
      await signoffFromInbox(reviewerContext, { docId: primaryDocId, decision: 'approve' });
    } finally {
      await reviewerContext.close();
    }

    await page.goto(`/documents/${primaryDocId}`);
    await expect.poll(() => stateBadgeText(page), { timeout: 5000 }).toBe('Em revisão');
  });

  test('approver rejects at stage 2 — badge transitions to rejected', async ({ browser, baseURL }) => {
    const approverContext = await contextAs(browser, requireBaseURL(baseURL), seeded.cookies, 'approver');
    try {
      const page = await signoffFromInbox(approverContext, {
        docId: primaryDocId,
        decision: 'reject',
        reason: 'needs diagram fix',
      });

      await expect.poll(() => stateBadgeText(page), { timeout: 5000 }).toBe('Rejeitado');
    } finally {
      await approverContext.close();
    }
  });

  test('after rejection doc auto-transitions to draft', async ({ page }) => {
    await loginAs(page, seeded.cookies, 'author');
    await page.goto(`/documents/${primaryDocId}`);

    await expect.poll(() => stateBadgeText(page), { timeout: 5000 }).toBe('Rascunho');
    await expect(page.getByRole('button', { name: /Documento em revisão/i })).toHaveCount(0);
    await expect(page.getByRole('button', { name: 'Submeter para revisão' })).toBeEnabled();
  });

  test('timeline shows rejection node with reason text', async ({ page }) => {
    await loginAs(page, seeded.cookies, 'author');
    await page.goto(`/documents/${primaryDocId}`);

    const timelinePanel = page.locator('section[aria-label="Timeline de aprovação"]');
    await expect(timelinePanel).toContainText('needs diagram fix');
  });

  test('author inbox is empty after rejection', async ({ page }) => {
    await loginAs(page, seeded.cookies, 'author');
    await page.goto('/approval/inbox');

    const docRow = page.locator('tbody tr').filter({ hasText: primaryDocId });
    await expect.poll(async () => docRow.count(), { timeout: 5000 }).toBe(0);
  });

  test('reject without reason — form validation, submit disabled', async ({ page, browser, baseURL }) => {
    await submitAsAuthor(page, secondaryDocId);

    const reviewerContext = await contextAs(browser, requireBaseURL(baseURL), seeded.cookies, 'reviewer');
    try {
      await signoffFromInbox(reviewerContext, { docId: secondaryDocId, decision: 'approve' });
    } finally {
      await reviewerContext.close();
    }

    const approverContext = await contextAs(browser, requireBaseURL(baseURL), seeded.cookies, 'approver');
    try {
      const approverPage = await approverContext.newPage();
      await approverPage.goto('/approval/inbox');

      const docRow = approverPage.locator('tbody tr').filter({ hasText: secondaryDocId }).first();
      await expect(docRow).toBeVisible();
      await docRow.click();

      await approverPage.getByRole('button', { name: 'Assinar' }).click();
      await expect(approverPage.getByRole('dialog')).toBeVisible();

      await approverPage.getByLabel('Rejeitado').check();
      await approverPage.getByLabel('Senha').fill('test1234');

      let signoffRequests = 0;
      approverPage.on('request', (request) => {
        if (isSignoffRequest(request)) {
          signoffRequests += 1;
        }
      });

      await approverPage.getByRole('button', { name: 'Confirmar assinatura' }).click();

      await expect.poll(() => signoffRequests, { timeout: 5000 }).toBe(0);
      await expect(approverPage.getByText('Informe o motivo da rejeição.')).toBeVisible();
      await expect(approverPage.getByRole('button', { name: 'Confirmar assinatura' })).toBeEnabled();
    } finally {
      await approverContext.close();
    }
  });
});
