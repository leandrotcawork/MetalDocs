import { test, expect } from '@playwright/test';

test.describe('Area Membership Admin', () => {
  test.setTimeout(30_000);

  test('admin grants user area/editor ? membership appears in list', async ({ page }) => {
    const uniqueCode = `tst-${Date.now()}`;
    const userId = `user-${uniqueCode}`;

    await page.goto('/admin/memberships');
    await page.getByLabel(/user\s*id/i).fill(userId);
    await page.getByRole('button', { name: /search/i }).click();

    await page.getByRole('button', { name: /grant/i }).click();
    await page.getByLabel(/user\s*id/i).fill(userId);
    await page.getByLabel(/area\s*code/i).fill('QA');
    await page.getByLabel(/role/i).fill('editor');
    await page.getByRole('button', { name: /grant|save|submit/i }).click(); // Assumption: modal submit label may vary.

    await expect(page.getByRole('row', { name: new RegExp(`${userId}.*QA.*editor`, 'i') })).toBeVisible();
  });

  test('revoke membership ? row disappears from list', async ({ page }) => {
    const uniqueCode = `tst-${Date.now()}`;
    const userId = `user-${uniqueCode}`;

    await page.goto('/admin/memberships');
    await page.getByRole('button', { name: /grant/i }).click();
    await page.getByLabel(/user\s*id/i).fill(userId);
    await page.getByLabel(/area\s*code/i).fill('QA');
    await page.getByLabel(/role/i).fill('editor');
    await page.getByRole('button', { name: /grant|save|submit/i }).click();

    const row = page.getByRole('row', { name: new RegExp(`${userId}.*QA.*editor`, 'i') });
    await expect(row).toBeVisible();
    await row.getByRole('button', { name: /revoke/i }).click();
    await page.getByRole('button', { name: /confirm|revoke/i }).click(); // Assumption: confirmation CTA may vary.

    await expect(row).not.toBeVisible();
  });
});
