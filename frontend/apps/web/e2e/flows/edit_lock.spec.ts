import { type BrowserContext, type Locator, type Page } from '@playwright/test';

import { test, expect } from '../fixtures/isolation';
import type { IsolatedFixture } from '../fixtures/isolation';
import { contextAs, loginAs } from '../utils/auth';

type RoleCookies = Record<'author' | 'reviewer' | 'approver' | 'admin', string>;

function roleCookies(isolated: IsolatedFixture): RoleCookies {
  return {
    author: isolated.users.author.cookie,
    reviewer: isolated.users.reviewer.cookie,
    approver: isolated.users.approver.cookie,
    admin: isolated.users.admin.cookie,
  };
}

function requireBaseURL(baseURL: string | undefined): string {
  if (!baseURL) {
    throw new Error('Playwright baseURL is required for contextAs');
  }
  return baseURL;
}

function stateBadge(page: Page): Locator {
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

function lockBanner(page: Page): Locator {
  return page.getByRole('button', { name: /documento em revis/i }).first();
}

async function submitAsAuthor(page: Page, isolated: IsolatedFixture): Promise<void> {
  await loginAs(page, roleCookies(isolated), 'author');
  await page.goto(`/docs/${isolated.docId}`);

  await page.getByRole('button', { name: 'Submeter para revisão' }).click();
  await page.getByRole('button', { name: /^Submeter$/ }).click();

  await expect.poll(() => stateBadgeText(page), { timeout: 5000 }).toMatch(/em revis/i);
}

async function signoffFromInbox(
  context: BrowserContext,
  options: { docId: string; decision: 'approve' | 'reject'; reason?: string },
): Promise<void> {
  const page = await context.newPage();
  await page.goto('/approval/inbox');

  const docRow = page.locator('tbody tr').filter({ hasText: options.docId }).first();
  await expect(docRow).toBeVisible();
  await docRow.click();

  await page.getByRole('button', { name: 'Assinar' }).click();
  await expect(page.getByRole('dialog')).toBeVisible();

  if (options.decision === 'reject') {
    await page.getByLabel('Rejeitado').check();
    if (options.reason) {
      await page.getByLabel('Motivo').fill(options.reason);
    }
  }

  await page.getByLabel('Senha').fill('test1234');
  await page.getByRole('button', { name: /Aprovar|Confirmar assinatura/i }).click();

  await expect.poll(async () => page.getByRole('dialog').count(), { timeout: 5000 }).toBe(0);
  await page.close();
}

async function resolveEditButton(page: Page): Promise<Locator> {
  const candidates: Locator[] = [
    page.getByRole('button', { name: /editar documento/i }),
    page.getByRole('button', { name: /editar/i }),
    page.locator('[data-testid="document-edit-button"]'),
    page.locator('[data-testid="edit-button"]'),
    page.locator('button[aria-label*="Editar"]'),
    page.locator('button[title*="edição"], button[title*="edicao"]'),
  ];

  for (const candidate of candidates) {
    if ((await candidate.count()) > 0) {
      return candidate.first();
    }
  }

  throw new Error('Edit button not found on locked document page');
}

async function isDisabled(button: Locator): Promise<boolean> {
  const hasDisabledAttr = (await button.getAttribute('disabled')) !== null;
  const ariaDisabled = await button.getAttribute('aria-disabled');
  return hasDisabledAttr || ariaDisabled === 'true';
}

async function readTooltipText(page: Page, button: Locator): Promise<string> {
  const title = await button.getAttribute('title');
  if (title && title.trim()) {
    return title.trim();
  }

  const tooltip = page.locator('[role="tooltip"], [data-testid*="tooltip"], [class*="tooltip"]').first();
  if ((await tooltip.count()) > 0) {
    const text = ((await tooltip.textContent()) ?? '').trim();
    if (text) {
      return text;
    }
  }

  return '';
}

test.describe('edit_lock', () => {
  test('submitted doc shows lock banner to second user', async ({ page, browser, baseURL, isolated }) => {
    await submitAsAuthor(page, isolated);

    const reviewerContext = await contextAs(browser, requireBaseURL(baseURL), roleCookies(isolated), 'reviewer');
    try {
      const reviewerPage = await reviewerContext.newPage();
      await reviewerPage.goto(`/docs/${isolated.docId}`);

      const banner = lockBanner(reviewerPage);
      await expect(banner).toBeVisible();

      const bannerText = (((await banner.textContent()) ?? '').replace(/\s+/g, ' ')).trim();
      expect(bannerText).toContain(isolated.users.author.email);

      const afterActor = bannerText.split(isolated.users.author.email).pop()?.trim() ?? '';
      expect(afterActor.length).toBeGreaterThan(0);
    } finally {
      await reviewerContext.close();
    }
  });

  test('edit button disabled with tooltip when locked', async ({ page, browser, baseURL, isolated }) => {
    await submitAsAuthor(page, isolated);

    const reviewerContext = await contextAs(browser, requireBaseURL(baseURL), roleCookies(isolated), 'reviewer');
    try {
      const reviewerPage = await reviewerContext.newPage();
      await reviewerPage.goto(`/docs/${isolated.docId}`);

      const editButton = await resolveEditButton(reviewerPage);
      expect(await isDisabled(editButton)).toBeTruthy();

      await editButton.hover();
      await expect
        .poll(() => readTooltipText(reviewerPage, editButton), { timeout: 5000 })
        .toMatch(/documento.*revis/i);
    } finally {
      await reviewerContext.close();
    }
  });

  test('direct API PUT returns 423 locked', async ({ page, isolated }) => {
    await submitAsAuthor(page, isolated);

    await loginAs(page, roleCookies(isolated), 'reviewer');
    const response = await page.request.put(`/api/v2/documents/${isolated.docId}`, {
      data: { title: 'hacked' },
      headers: { 'If-Match': 'any' },
    });

    expect(response.status()).toBe(423);
    const body = (await response.json()) as { code?: string; error?: { code?: string } };
    expect(body.code ?? body.error?.code).toBe('doc.locked');
  });

  test('rejection releases lock — edit re-enabled within 2s', async ({ page, browser, baseURL, isolated }) => {
    await submitAsAuthor(page, isolated);

    const reviewerContext = await contextAs(browser, requireBaseURL(baseURL), roleCookies(isolated), 'reviewer');
    const approverContext = await contextAs(browser, requireBaseURL(baseURL), roleCookies(isolated), 'approver');

    try {
      const reviewerPage = await reviewerContext.newPage();
      await reviewerPage.goto(`/docs/${isolated.docId}`);
      await expect(lockBanner(reviewerPage)).toBeVisible();

      await signoffFromInbox(reviewerContext, { docId: isolated.docId, decision: 'approve' });
      await signoffFromInbox(approverContext, {
        docId: isolated.docId,
        decision: 'reject',
        reason: 'needs rework',
      });

      await expect
        .poll(
          async () => {
            await reviewerPage.reload();
            return lockBanner(reviewerPage).count();
          },
          { timeout: 5000 },
        )
        .toBe(0);

      await reviewerPage.reload();
      const editButton = await resolveEditButton(reviewerPage);
      expect(await isDisabled(editButton)).toBeFalsy();
    } finally {
      await reviewerContext.close();
      await approverContext.close();
    }
  });
});
