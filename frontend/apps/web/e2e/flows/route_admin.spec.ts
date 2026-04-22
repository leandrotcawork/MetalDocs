/**
 * Task 11.8 — route_admin edit blocked when in_use
 *
 * Covers: create route, edit route, in_use=true disables edit,
 * deactivate confirmation, ESC focus trap.
 */
import { test, expect, Browser } from '@playwright/test';
import { seedTenant, resetTenant, SeedResult } from '../utils/seed';
import { loginAs, contextAs } from '../utils/auth';
import { randomUUID } from 'node:crypto';

const BASE_URL = process.env.E2E_BASE_URL || 'http://localhost:8080';

let seed: SeedResult;
let routeId: string;
let routeName: string;

test.describe('route admin', () => {
  test.beforeAll(async ({ request }) => {
    seed = await seedTenant(request, { workerIndex: 0, testTitle: 'route_admin' });
    routeName = `Route E2E ${seed.tenantId.slice(-6)}`;
  });

  test.afterAll(async ({ request }) => {
    await resetTenant(request, seed.tenantId);
  });

  test('admin creates route — lists with in_use=false', async ({ page }) => {
    await loginAs(page, seed.cookies, 'admin');
    await page.goto('/approval/admin/routes');

    // Create via UI
    await page.getByRole('button', { name: /nova rota/i }).click();
    await page.getByLabel(/nome/i).fill(routeName);

    // Add stage: sequential, 1 member
    await page.getByRole('button', { name: /adicionar etapa/i }).click();
    await page.getByLabel(/membro/i).first().fill(seed.users.reviewer.email);
    await page.keyboard.press('Enter');

    const routeResp = page.waitForResponse(
      r => r.url().includes('/api/v2/routes') && r.request().method() === 'POST'
    );
    await page.getByRole('button', { name: /salvar/i }).click();
    const resp = await routeResp;
    expect(resp.status()).toBeLessThan(300);
    const body = await resp.json();
    routeId = body.id as string;

    // Row visible with in_use = false
    const row = page.getByTestId(`route-row-${routeId}`);
    await expect(row).toBeVisible();
    await expect(row.getByTestId('in-use-badge')).toHaveText(/false|inativo|não/i);
  });

  test('admin edits route — toast + row updates', async ({ page }) => {
    await loginAs(page, seed.cookies, 'admin');
    await page.goto('/approval/admin/routes');

    const row = page.getByTestId(`route-row-${routeId}`);
    await row.getByRole('button', { name: /editar/i }).click();

    const updatedName = routeName + ' Updated';
    await page.getByLabel(/nome/i).clear();
    await page.getByLabel(/nome/i).fill(updatedName);

    await page.getByRole('button', { name: /salvar/i }).click();

    // Toast success
    await expect(page.locator('[data-testid="toast"]')).toBeVisible({ timeout: 5000 });

    // Row name updated
    await expect(row).toContainText(updatedName);
  });

  test('route in_use=true — edit button disabled with tooltip', async ({ page }) => {
    // Author submits doc using the route → creates active instance
    await loginAs(page, seed.cookies, 'author');
    await page.goto(`/docs/${seed.docId}`);
    await page.request.post(`/api/v2/documents/${seed.docId}/submit`, {
      headers: {
        'Content-Type': 'application/json',
        'Idempotency-Key': randomUUID(),
        'X-Tenant-ID': seed.tenantId,
      },
      data: { routeId },
    });

    // Admin reloads route list
    await loginAs(page, seed.cookies, 'admin');
    await page.goto('/approval/admin/routes');

    const row = page.getByTestId(`route-row-${routeId}`);
    await expect.poll(
      () => row.getByTestId('in-use-badge').textContent(),
      { timeout: 5000 }
    ).toMatch(/true|ativo|em uso/i);

    const editBtn = row.getByRole('button', { name: /editar/i });
    // Disabled
    await expect(editBtn).toBeDisabled();

    // Hover shows tooltip
    await editBtn.hover();
    await expect(page.locator('[role="tooltip"]')).toBeVisible({ timeout: 3000 });
  });

  test('deactivate confirmation — active=false, new submissions fail 409', async ({ page }) => {
    await loginAs(page, seed.cookies, 'admin');
    await page.goto('/approval/admin/routes');

    const row = page.getByTestId(`route-row-${routeId}`);
    await row.getByRole('button', { name: /desativar/i }).click();

    // Confirmation dialog
    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible();
    await expect(dialog).toContainText(/Desativar esta rota/i);

    await dialog.getByRole('button', { name: /confirmar/i }).click();
    await expect(dialog).not.toBeVisible({ timeout: 5000 });

    // Row shows inactive
    await expect(row.getByTestId('active-badge')).toHaveText(/false|inativo/i);

    // New submission with this route → 409
    const newDocId = randomUUID();
    const submitResp = await page.request.post(`/api/v2/documents/${newDocId}/submit`, {
      headers: {
        'Content-Type': 'application/json',
        'Idempotency-Key': randomUUID(),
        'X-Tenant-ID': seed.tenantId,
      },
      data: { routeId },
    });
    expect(submitResp.status()).toBe(409);

    // Existing in-progress instance unaffected
    const instanceResp = await page.request.get(
      `/api/v2/documents/${seed.docId}/instance`,
      { headers: { 'X-Tenant-ID': seed.tenantId } }
    );
    const instanceBody = await instanceResp.json();
    expect(instanceBody.status).toMatch(/in_progress|under_review/i);
  });

  test('ESC closes deactivate dialog without deactivating', async ({ page }) => {
    // Re-activate first (or use a fresh route)
    await loginAs(page, seed.cookies, 'admin');

    // Create a fresh route to test ESC without side effects
    const freshResp = await page.request.post('/api/v2/routes', {
      headers: {
        'Content-Type': 'application/json',
        'Idempotency-Key': randomUUID(),
        'X-Tenant-ID': seed.tenantId,
      },
      data: {
        name: 'Route ESC Test',
        stages: [{ order: 1, kind: 'sequential', members: [seed.users.reviewer.id] }],
      },
    });
    const freshRoute = await freshResp.json();
    const freshId = freshRoute.id as string;

    await page.goto('/approval/admin/routes');
    const freshRow = page.getByTestId(`route-row-${freshId}`);
    await freshRow.getByRole('button', { name: /desativar/i }).click();

    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible();

    // ESC closes dialog
    await page.keyboard.press('Escape');
    await expect(dialog).not.toBeVisible({ timeout: 3000 });

    // Route still active
    const checkResp = await page.request.get(`/api/v2/routes/${freshId}`, {
      headers: { 'X-Tenant-ID': seed.tenantId },
    });
    const routeBody = await checkResp.json();
    expect(routeBody.active).toBe(true);
  });

  test('dialog focus trap — Tab key cycles within dialog', async ({ page }) => {
    await loginAs(page, seed.cookies, 'admin');

    // Create another fresh route
    const freshResp = await page.request.post('/api/v2/routes', {
      headers: {
        'Content-Type': 'application/json',
        'Idempotency-Key': randomUUID(),
        'X-Tenant-ID': seed.tenantId,
      },
      data: {
        name: 'Route Focus Trap Test',
        stages: [{ order: 1, kind: 'sequential', members: [seed.users.reviewer.id] }],
      },
    });
    const freshRoute = await freshResp.json();
    const freshId = freshRoute.id as string;

    await page.goto('/approval/admin/routes');
    const freshRow = page.getByTestId(`route-row-${freshId}`);
    await freshRow.getByRole('button', { name: /desativar/i }).click();

    const dialog = page.getByRole('dialog');
    await expect(dialog).toBeVisible();

    // Tab twice — focus should stay within dialog
    await page.keyboard.press('Tab');
    await page.keyboard.press('Tab');

    const focusedElement = await page.evaluate(() => document.activeElement?.closest('[role="dialog"]') !== null);
    expect(focusedElement).toBe(true);

    // Cleanup
    await page.keyboard.press('Escape');
  });
});
