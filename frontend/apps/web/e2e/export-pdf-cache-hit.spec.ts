import { test, expect } from '@playwright/test';

test('export-pdf cache hit — second export returns cached=true and is faster', async ({ page }) => {
  await page.goto('/documents-v2/new');
  await page.getByRole('button', { name: /purchase order/i }).click();
  await page.getByLabel(/document name/i).fill('export-cache-e2e');
  await page.getByRole('button', { name: /generate document/i }).click();
  await page.waitForURL(/\/documents-v2\/.+/);

  await expect(page.locator('[data-status="saved"]')).toBeVisible({ timeout: 15_000 });

  // First export — cold miss
  const [firstResponse] = await Promise.all([
    page.waitForResponse((r) => r.url().includes('/export/pdf') && r.request().method() === 'POST'),
    page.locator('[data-export-pdf]').click(),
  ]);
  expect(firstResponse.status()).toBe(200);
  const firstBody = await firstResponse.json() as { cached: boolean };
  expect(firstBody.cached).toBe(false);
  await expect(page.locator('[data-export-status="done"]')).toBeVisible({ timeout: 30_000 });

  const t0 = Date.now();

  // Second export — warm hit (same composite hash, no content change)
  const [secondResponse] = await Promise.all([
    page.waitForResponse((r) => r.url().includes('/export/pdf') && r.request().method() === 'POST'),
    page.locator('[data-export-pdf]').click(),
  ]);
  const elapsed = Date.now() - t0;

  expect(secondResponse.status()).toBe(200);
  const secondBody = await secondResponse.json() as { cached: boolean };
  expect(secondBody.cached).toBe(true);

  // Cache hit should be noticeably faster than cold PDF generation (< 3 s)
  expect(elapsed).toBeLessThan(3_000);

  await expect(page.locator('[data-export-status="done"][data-export-cached="true"]')).toBeVisible({ timeout: 10_000 });
});
