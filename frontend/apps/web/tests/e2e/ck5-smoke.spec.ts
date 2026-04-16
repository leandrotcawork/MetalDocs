import { test, expect } from '@playwright/test';

const AUTHOR_URL = '/#/test-harness/ck5?mode=author&tpl=smoke';
const FILL_URL = '/#/test-harness/ck5?mode=fill&tpl=smoke&doc=smoke-doc';

test.describe('CK5 smoke', () => {
  test('Author inserts a section; Fill loads it with the exception region', async ({ page }) => {
    await page.addInitScript(() => window.localStorage.clear());

    await page.goto(AUTHOR_URL);
    await expect(page.getByTestId('ck5-author-page')).toBeVisible();

    await page.getByRole('button', { name: 'Insert section' }).click();
    await expect(page.locator('.mddm-section')).toBeVisible();

    await page.waitForFunction(
      () => {
        const raw = window.localStorage.getItem('ck5.tpl.smoke');
        return raw && raw.includes('mddm-section');
      },
      { timeout: 5000 },
    );

    await page.goto(FILL_URL);
    await expect(page.getByTestId('ck5-fill-page')).toBeVisible();
    await expect(page.locator('.mddm-section')).toBeVisible();

    await expect(page.locator('.restricted-editing-exception, [class*="restricted-editing"]').first()).toBeVisible({
      timeout: 5000,
    });
  });
});
