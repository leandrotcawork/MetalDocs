import { test, expect } from '@playwright/test';

test('export-pdf happy path — PDF generated, response has %PDF- magic bytes', async ({ page }) => {
  await page.goto('/documents-v2/new');
  await page.getByRole('button', { name: /purchase order/i }).click();
  await page.getByLabel(/document name/i).fill('export-pdf-e2e');
  await page.getByRole('button', { name: /generate document/i }).click();
  await page.waitForURL(/\/documents-v2\/.+/);

  const docURL = page.url();
  const docID = docURL.split('/').pop()!;

  await expect(page.locator('[data-status="saved"]')).toBeVisible({ timeout: 15_000 });

  const [response] = await Promise.all([
    page.waitForResponse((r) => r.url().includes(`/export/pdf`) && r.request().method() === 'POST'),
    page.locator('[data-export-pdf]').click(),
  ]);

  expect(response.status()).toBe(200);
  const body = await response.json() as { signed_url: string; cached: boolean };
  expect(typeof body.signed_url).toBe('string');
  expect(body.signed_url.length).toBeGreaterThan(0);

  const pdfRes = await page.evaluate(async (url: string) => {
    const r = await fetch(url);
    const buf = await r.arrayBuffer();
    const bytes = new Uint8Array(buf, 0, 5);
    return Array.from(bytes).map((b) => String.fromCharCode(b)).join('');
  }, body.signed_url);

  expect(pdfRes).toBe('%PDF-');

  await expect(page.locator('[data-export-status="done"]')).toBeVisible({ timeout: 30_000 });
  const cachedAttr = await page.locator('[data-export-status="done"]').getAttribute('data-export-cached');
  expect(cachedAttr).toBe('false');
});
