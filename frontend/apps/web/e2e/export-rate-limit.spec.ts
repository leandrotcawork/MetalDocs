import { test, expect } from '@playwright/test';

test('export-pdf rate limit — 21st request in 1 min returns 429', async ({ page }) => {
  await page.goto('/documents-v2/new');
  await page.getByRole('button', { name: /purchase order/i }).click();
  await page.getByLabel(/document name/i).fill('export-ratelimit-e2e');
  await page.getByRole('button', { name: /generate document/i }).click();
  await page.waitForURL(/\/documents-v2\/.+/);

  const docURL = page.url();
  const docID = docURL.split('/').pop()!;

  await expect(page.locator('[data-status="saved"]')).toBeVisible({ timeout: 15_000 });

  // Drive 20 requests directly via fetch to exhaust the 20/min bucket.
  // Use page.evaluate so requests carry the session cookie.
  const statuses = await page.evaluate(async ({ id, count }: { id: string; count: number }) => {
    const results: number[] = [];
    for (let i = 0; i < count; i++) {
      const r = await fetch(`/api/v2/documents/${id}/export/pdf`, {
        method: 'POST',
        headers: { 'content-type': 'application/json' },
        body: JSON.stringify({ paper_size: 'A4' }),
        credentials: 'include',
      });
      results.push(r.status);
    }
    return results;
  }, { id: docID, count: 20 });

  // All 20 within burst+refill window should succeed (200) or be cached (still 200).
  expect(statuses.every((s) => s === 200)).toBe(true);

  // 21st request should be rate-limited
  const [limitedResponse] = await Promise.all([
    page.waitForResponse((r) => r.url().includes('/export/pdf') && r.request().method() === 'POST'),
    page.locator('[data-export-pdf]').click(),
  ]);

  expect(limitedResponse.status()).toBe(429);
  await expect(page.locator('[data-export-status="rate_limited"]')).toBeVisible({ timeout: 5_000 });
  await expect(page.getByRole('alert')).toContainText(/retry in/i);
});
