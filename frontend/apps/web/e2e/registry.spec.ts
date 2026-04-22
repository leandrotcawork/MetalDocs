import { test, expect } from '@playwright/test';

test.describe('Controlled Document Registry', () => {
  test.setTimeout(30_000);

  test('auto-code increments per profile — create 3 CDs ? codes match PO-01, PO-02, PO-03 pattern', async ({ page }) => {
    const uniqueCode = `tst-${Date.now()}`;

    await page.goto('/registry-v2');

    for (let i = 1; i <= 3; i += 1) {
      await page.getByRole('button', { name: /\+\s*novo/i }).click();
      await page.getByLabel(/profile/i).fill('PO');
      await page.getByLabel(/area/i).fill('QA');
      await page.getByLabel(/title/i).fill(`Registry ${uniqueCode}-${i}`);
      await page.getByLabel(/owner\s*user\s*id/i).fill('user-1');
      await page.getByRole('button', { name: /create|save|submit/i }).click(); // Assumption: submit button wording may vary.
    }

    await expect(page.getByRole('row', { name: /PO-01/i })).toBeVisible();
    await expect(page.getByRole('row', { name: /PO-02/i })).toBeVisible();
    await expect(page.getByRole('row', { name: /PO-03/i })).toBeVisible();
  });

  test('manual code + reason works', async ({ page }) => {
    const uniqueCode = `tst-${Date.now()}`;

    await page.goto('/registry-v2');
    await page.getByRole('button', { name: /\+\s*novo/i }).click();
    await page.getByLabel(/profile/i).fill('PO');
    await page.getByLabel(/area/i).fill('QA');
    await page.getByLabel(/title/i).fill(`Manual ${uniqueCode}`);
    await page.getByLabel(/owner\s*user\s*id/i).fill('user-1');
    await page.getByLabel(/manual\s*code/i).check(); // Assumption: manual code is exposed as a checkbox/toggle with this label.
    await page.getByLabel(/code/i).fill('PO-MANUAL-01');
    await page.getByLabel(/reason/i).fill('Legacy migration');
    await page.getByRole('button', { name: /create|save|submit/i }).click();

    await expect(page.getByRole('row', { name: /PO-MANUAL-01/i })).toBeVisible();
  });

  test('duplicate code shows conflict error in UI', async ({ page }) => {
    const uniqueCode = `tst-${Date.now()}`;
    const manualCode = `PO-DUP-${uniqueCode}`;

    await page.goto('/registry-v2');

    for (let i = 0; i < 2; i += 1) {
      await page.getByRole('button', { name: /\+\s*novo/i }).click();
      await page.getByLabel(/profile/i).fill('PO');
      await page.getByLabel(/area/i).fill('QA');
      await page.getByLabel(/title/i).fill(`Duplicate ${uniqueCode}-${i}`);
      await page.getByLabel(/owner\s*user\s*id/i).fill('user-1');
      await page.getByLabel(/manual\s*code/i).check();
      await page.getByLabel(/code/i).fill(manualCode);
      await page.getByLabel(/reason/i).fill('Forced duplicate test');
      await page.getByRole('button', { name: /create|save|submit/i }).click();
    }

    await expect(page.getByText(/conflict|already exists|duplicate/i)).toBeVisible();
  });

  test('missing reason for manual code shows validation error', async ({ page }) => {
    const uniqueCode = `tst-${Date.now()}`;

    await page.goto('/registry-v2');
    await page.getByRole('button', { name: /\+\s*novo/i }).click();
    await page.getByLabel(/profile/i).fill('PO');
    await page.getByLabel(/area/i).fill('QA');
    await page.getByLabel(/title/i).fill(`Missing reason ${uniqueCode}`);
    await page.getByLabel(/owner\s*user\s*id/i).fill('user-1');
    await page.getByLabel(/manual\s*code/i).check();
    await page.getByLabel(/code/i).fill(`PO-NOREASON-${uniqueCode}`);
    await page.getByRole('button', { name: /create|save|submit/i }).click();

    await expect(page.getByText(/reason.*required|required.*reason/i)).toBeVisible();
  });
});
