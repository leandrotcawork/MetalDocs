import { test, expect } from '@playwright/test';

test('filler happy path: pick template fill form generate edit checkpoint finalize', async ({ page }) => {
  await page.goto('/documents-v2/new');
  await page.getByRole('button', { name: /purchase order/i }).click();
  await page.getByLabel(/document name/i).fill('PO-2026-0001');
  await page.getByRole('button', { name: /generate document/i }).click();

  await page.waitForURL(/\/documents-v2\/.+/);
  await expect(page.locator('[data-status="saved"], [data-status="idle"]').first()).toBeVisible({ timeout: 30_000 });

  await page.getByLabel(/placeholder label/i).fill('initial');
  await page.getByRole('button', { name: /create checkpoint/i }).click();
  await expect(page.getByText(/v1 initial/i)).toBeVisible();

  await page.getByRole('button', { name: /finalize/i }).click();
  await page.waitForURL(/\/documents-v2/, { timeout: 5_000 });
});