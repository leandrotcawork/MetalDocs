import { test, expect } from '@playwright/test';

test.describe.skip('Separation of Duties', () => {
  test.setTimeout(30_000);

  test('template.publish SoD — author cannot publish own template', async ({ page }) => {
    await expect(page.getByText(/requires multi-user session setup/i)).toBeVisible();
  });
});
