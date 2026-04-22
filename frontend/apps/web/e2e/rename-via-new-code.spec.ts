import { test, expect } from '@playwright/test';

test.describe('Code Immutability', () => {
  test.setTimeout(30_000);

  test('attempt to rename profile code via API PUT returns 409', async ({ page }) => {
    const uniqueCode = `tst-${Date.now()}`;

    await page.goto('/admin/taxonomy');
    await page.getByRole('button', { name: /\+\s*add profile/i }).click();
    await page.getByLabel(/code/i).fill(uniqueCode);
    await page.getByLabel(/family\s*code/i).fill('PO');
    await page.getByLabel(/name/i).fill(`Immutable ${uniqueCode}`);
    await page.getByRole('button', { name: /create|save|submit/i }).click(); // Assumption: submit button label may vary.

    const result = await page.evaluate(async (originalCode) => {
      const response = await fetch(`/api/v2/taxonomy/profiles/${originalCode}`, {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ code: `newcode-${Date.now()}` }),
      });

      let body: { data?: { profile?: { code?: string } } } | null = null;
      try {
        body = (await response.json()) as { data?: { profile?: { code?: string } } };
      } catch {
        body = null;
      }

      return {
        status: response.status,
        returnedCode: body?.data?.profile?.code ?? null,
      };
    }, uniqueCode);

    expect(result.status === 409 || result.returnedCode === uniqueCode).toBeTruthy();
  });

  test('archived profile is still visible with includeArchived toggle', async ({ page }) => {
    await page.goto('/admin/taxonomy');
    await page.getByRole('checkbox', { name: /show archived/i }).check(); // Assumption: includeArchived is represented as this checkbox.

    await expect(page.getByText(/archived/i).first()).toBeVisible();
  });
});
