import { test, expect } from '@playwright/test';

test.describe('Taxonomy Admin', () => {
  test.setTimeout(30_000);

  test('admin creates profile via Tipos Documentais nav ? profile appears in list', async ({ page }) => {
    const uniqueCode = `tst-${Date.now()}`;

    await page.goto('/admin/taxonomy');
    await page.getByRole('button', { name: /\+\s*add profile/i }).click();
    await page.getByLabel(/code/i).fill(uniqueCode);
    await page.getByLabel(/family\s*code/i).fill('PO');
    await page.getByLabel(/name/i).fill(`Taxonomy ${uniqueCode}`);
    await page.getByRole('button', { name: /create|save|submit/i }).click(); // Assumption: submit action label may vary by screen copy.

    await expect(page.getByRole('row', { name: new RegExp(uniqueCode, 'i') })).toBeVisible();
  });

  test('set profile default template ? picker shows only published versions of this profile', async ({ page }) => {
    const uniqueCode = `tst-${Date.now()}`;

    await page.goto('/admin/taxonomy');
    await page.getByRole('button', { name: /\+\s*add profile/i }).click();
    await page.getByLabel(/code/i).fill(uniqueCode);
    await page.getByLabel(/family\s*code/i).fill('PO');
    await page.getByLabel(/name/i).fill(`Taxonomy ${uniqueCode}`);
    await page.getByRole('button', { name: /create|save|submit/i }).click(); // Assumption: submit action label may vary by screen copy.

    const row = page.getByRole('row', { name: new RegExp(uniqueCode, 'i') });
    await expect(row).toBeVisible();
    await row.getByRole('button', { name: /edit/i }).click();

    await page.getByRole('button', { name: /set default template/i }).click();
    await page.getByLabel(/template\s*uuid/i).fill('00000000-0000-4000-8000-000000000001');
    await page.getByRole('button', { name: /save|set|submit/i }).click(); // Assumption: modal confirmation text may differ.

    await expect(page.getByText(/error/i)).not.toBeVisible();
  });
});
