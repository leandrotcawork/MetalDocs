import { test, expect } from '@playwright/test';

test('two tabs: second is readonly until first releases', async ({ browser }) => {
  const ctx1 = await browser.newContext();
  const ctx2 = await browser.newContext();

  const tab1 = await ctx1.newPage();
  await tab1.goto('/documents-v2/new');
  await tab1.getByRole('button', { name: /purchase order/i }).click();
  await tab1.getByLabel(/document name/i).fill('concurrent');
  await tab1.getByRole('button', { name: /generate/i }).click();
  await tab1.waitForURL(/\/documents-v2\/.+/);

  const url = tab1.url();

  const tab2 = await ctx2.newPage();
  await tab2.goto(url);
  await expect(tab2.getByText(/read-only/i)).toBeVisible({ timeout: 10_000 });

  await tab1.goto('/documents-v2/new');

  await tab2.reload();
  await expect(tab2.locator('[data-status="saved"], [data-status="idle"]').first()).toBeVisible({ timeout: 15_000 });
  await expect(tab2.getByText(/read-only/i)).toHaveCount(0);
});