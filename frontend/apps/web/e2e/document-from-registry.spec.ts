import { test, expect } from '@playwright/test';

test.describe('Document from Registry', () => {
  test.setTimeout(30_000);

  test('pick controlled doc ? CD-first create flow shows the controlled document list', async ({ page }) => {
    await page.goto('/documents-v2/new');

    await expect(page.getByRole('heading', { name: /step\s*1:\s*pick controlled document/i })).toBeVisible();
  });

  test('selecting a CD enables the name input and submit button', async ({ page }) => {
    await page.goto('/documents-v2/new');

    await page.getByRole('row').nth(1).click(); // Assumption: first data row is selectable controlled document entry.
    const nameInput = page.getByLabel(/document\s*name|name/i);
    await expect(nameInput).toBeVisible();
    await nameInput.fill(`Generated ${Date.now()}`);

    await expect(page.getByRole('button', { name: /generate document/i })).toBeEnabled();
  });
});
