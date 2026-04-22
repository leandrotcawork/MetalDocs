// @serial-clock
import { expect, type Page } from '@playwright/test';

import { test } from '../fixtures/isolation';
import type { IsolatedFixture } from '../fixtures/isolation';
import { loginAs } from '../utils/auth';

type RoleCookies = Record<'author' | 'reviewer' | 'approver' | 'admin', string>;

function roleCookies(isolated: IsolatedFixture): RoleCookies {
  return {
    author: isolated.users.author.cookie,
    reviewer: isolated.users.reviewer.cookie,
    approver: isolated.users.approver.cookie,
    admin: isolated.users.admin.cookie,
  };
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

async function signoffFromInbox(page: Page, docId: string): Promise<boolean> {
  await page.goto('/approval/inbox');

  const docRow = page.locator('tbody tr').filter({ hasText: docId }).first();
  if ((await docRow.count()) === 0) {
    return false;
  }
  await expect(docRow).toBeVisible();
  await docRow.click();

  await page.getByRole('button', { name: /Assinar/i }).click();
  await expect(page.getByRole('dialog')).toBeVisible();

  await page.getByLabel('Senha').fill('test1234');
  await page.getByRole('button', { name: /Aprovar|Confirmar assinatura/i }).click();

  await expect.poll(async () => page.getByRole('dialog').count(), { timeout: 5000 }).toBe(0);
  return true;
}

async function prepareApprovedDocument(
  page: Page,
  isolated: IsolatedFixture,
): Promise<void> {
  const cookies = roleCookies(isolated);

  await loginAs(page, cookies, 'author');
  await page.goto(`/documents/${isolated.docId}`);

  await page.getByRole('button', { name: 'Submeter para revisão' }).click();
  await page.getByRole('button', { name: /^Submeter$/ }).click();
  await expect.poll(() => stateBadgeText(page), { timeout: 5000 }).toBe('Em revisão');

  await loginAs(page, cookies, 'reviewer');
  const reviewerSigned = await signoffFromInbox(page, isolated.docId);
  expect(reviewerSigned).toBeTruthy();

  await loginAs(page, cookies, 'approver');
  await signoffFromInbox(page, isolated.docId);

  await page.goto(`/documents/${isolated.docId}`);
  await expect(page.getByRole('button', { name: /Publicar/i })).toBeVisible();
}

async function datetimeLocalValue(page: Page, timestampMs: number): Promise<string> {
  return page.evaluate((ms) => {
    const date = new Date(ms);
    const pad = (value: number) => String(value).padStart(2, '0');
    const year = date.getFullYear();
    const month = pad(date.getMonth() + 1);
    const day = pad(date.getDate());
    const hours = pad(date.getHours());
    const minutes = pad(date.getMinutes());
    return `${year}-${month}-${day}T${hours}:${minutes}`;
  }, timestampMs);
}

async function localDateLabel(page: Page, timestampMs: number): Promise<string> {
  return page.evaluate((ms) => new Date(ms).toLocaleDateString('pt-BR'), timestampMs);
}

async function vigenciaText(page: Page): Promise<string> {
  const row = page
    .locator('section[aria-label="Painel de detalhes de aprovação"] div')
    .filter({ hasText: 'Vigência:' })
    .first();
  await expect(row).toBeVisible();
  return ((await row.textContent()) ?? '').replace(/\s+/g, ' ').trim();
}

test.describe.serial('scheduled_publish', () => {
  test.beforeEach(async ({ request }) => {
    const resetClock = await request.post('/internal/test/advance-clock?seconds=0');
    expect(resetClock.ok()).toBeTruthy();
  });

  test('schedules publish — badge transitions to scheduled', async ({ page, isolated }) => {
    await page.clock.install();
    await prepareApprovedDocument(page, isolated);

    await page.getByRole('button', { name: /Publicar/i }).click();
    await page.getByLabel(/Agendar publicação|Agendar para/i).check();

    const frozenNowMs = await page.evaluate(() => Date.now());
    const scheduledMs = frozenNowMs + 10 * 60 * 1000;
    const scheduledValue = await datetimeLocalValue(page, scheduledMs);
    await page.getByLabel('Data e hora da publicação').fill(scheduledValue);
    await page.getByRole('button', { name: 'Confirmar publicação' }).click();

    await expect.poll(() => stateBadgeText(page), { timeout: 10_000 }).toBe('Agendado');

    const expectedLocalDate = await localDateLabel(page, scheduledMs);
    await expect(page.getByText(new RegExp(`Vigência:.*${expectedLocalDate}`))).toBeVisible();
  });

  test('clock advance → doc publishes', async ({ page, request, isolated }) => {
    await page.clock.install();
    await prepareApprovedDocument(page, isolated);

    await page.getByRole('button', { name: /Publicar/i }).click();
    await page.getByLabel(/Agendar publicação|Agendar para/i).check();

    const frozenNowMs = await page.evaluate(() => Date.now());
    const scheduledMs = frozenNowMs + 10 * 60 * 1000;
    const scheduledValue = await datetimeLocalValue(page, scheduledMs);
    await page.getByLabel('Data e hora da publicação').fill(scheduledValue);
    await page.getByRole('button', { name: 'Confirmar publicação' }).click();
    await expect.poll(() => stateBadgeText(page), { timeout: 10_000 }).toBe('Agendado');

    const effectiveFromBefore = await vigenciaText(page);

    const advance = await request.post('/internal/test/advance-clock?seconds=700');
    expect(advance.ok()).toBeTruthy();

    const tick = await request.post('/internal/test/trigger-scheduler-tick');
    expect(tick.ok()).toBeTruthy();

    await expect
      .poll(
        async () => {
          await page.reload();
          return stateBadgeText(page);
        },
        { timeout: 10_000 },
      )
      .toBe('Publicado');

    const effectiveFromAfter = await vigenciaText(page);
    expect(effectiveFromAfter).toBe(effectiveFromBefore);
  });

  test('past datetime — dialog shows validation error', async ({ page, isolated }) => {
    await page.clock.install();
    await prepareApprovedDocument(page, isolated);

    await page.getByRole('button', { name: /Publicar/i }).click();
    await page.getByLabel(/Agendar publicação|Agendar para/i).check();

    const frozenNowMs = await page.evaluate(() => Date.now());
    const scheduledValue = await datetimeLocalValue(page, frozenNowMs - 60_000);
    await page.getByLabel('Data e hora da publicação').fill(scheduledValue);

    await expect(page.getByText('A data deve ser pelo menos 5 minutos no futuro.')).toBeVisible();
    await expect(page.getByRole('button', { name: 'Confirmar publicação' })).toBeDisabled();
  });

  test('datetime less than 5 min from now — validation error', async ({ page, isolated }) => {
    await page.clock.install();
    await prepareApprovedDocument(page, isolated);

    await page.getByRole('button', { name: /Publicar/i }).click();
    await page.getByLabel(/Agendar publicação|Agendar para/i).check();

    const frozenNowMs = await page.evaluate(() => Date.now());
    const scheduledValue = await datetimeLocalValue(page, frozenNowMs + 2 * 60 * 1000);
    await page.getByLabel('Data e hora da publicação').fill(scheduledValue);

    await expect(page.getByText('A data deve ser pelo menos 5 minutos no futuro.')).toBeVisible();
    await expect(page.getByRole('button', { name: 'Confirmar publicação' })).toBeDisabled();
  });
});
