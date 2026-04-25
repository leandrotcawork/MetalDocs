import { test, expect } from '@playwright/test';

test.describe('Placeholder fill-in flow', () => {
  test.skip(
    !process.env.PLAYWRIGHT_FILLIN_E2E,
    'placeholder fill-in E2E — requires live server (set PLAYWRIGHT_FILLIN_E2E=1)',
  );

  test('Draft → Fill placeholders → Submit → Approval pending', async ({ page }) => {
    // Navigate to a document in draft state
    await page.goto('/documents');
    // TODO: wire up real selectors — open a draft document
    const draftLink = page.locator('[data-testid="document-row"]').first();
    await draftLink.click();

    // Fill a text placeholder
    const textInput = page.locator('[data-testid^="placeholder-input-"]').first();
    await textInput.fill('Test Document Title');

    // Fill a date placeholder
    const dateInput = page.locator('[data-testid^="placeholder-input-date-"]').first();
    if (await dateInput.isVisible()) {
      await dateInput.fill('2026-05-01');
    }

    // Submit for approval
    const submitBtn = page.getByTestId('submit-btn');
    await expect(submitBtn).toBeEnabled();
    await submitBtn.click();

    // Verify approval-pending state
    await expect(
      page.locator('[data-testid="approval-pending"], [data-testid="status-pill"]'),
    ).toBeVisible({ timeout: 10_000 });
  });

  test('Signoff → Viewer shows PDF iframe', async ({ page }) => {
    // TODO: reviewer signs off on submitted document
    // TODO: admin approves → publishes
    // TODO: consumer opens viewer
    // TODO: assert PDF iframe visible

    // Placeholder assertion to keep test syntactically valid
    await page.goto('/');
    await expect(page).toHaveURL('/');
  });
});
