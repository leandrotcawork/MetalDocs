import { test, expect } from '@playwright/test';

test('autosave-crash real-blob replay advances current_revision_id', async ({ browser }) => {
  const context = await browser.newContext();
  const tab1 = await context.newPage();

  await tab1.goto('/documents-v2/new');
  await tab1.getByRole('button', { name: /purchase order/i }).click();
  await tab1.getByLabel(/document name/i).fill('crash-recovery');
  await tab1.getByRole('button', { name: /generate document/i }).click();
  await tab1.waitForURL(/\/documents-v2\/.+/);

  const docURL = tab1.url();
  const docID = docURL.split('/').pop()!;

  await expect(tab1.locator('[data-status="saved"]')).toBeVisible({ timeout: 15_000 });

  const revBefore = await tab1.evaluate(async (id) => {
    const res = await fetch(`/api/v2/documents/${id}`, { credentials: 'include' });
    const json = await res.json();
    return json.current_revision_id as string;
  }, docID);

  await tab1.route(`**/api/v2/documents/${docID}/autosave/commit`, (route) => route.abort('failed'));

  await tab1.locator('[data-editor-root]').click();
  await tab1.keyboard.type(' edit-for-crash-test');

  await expect(tab1.locator('[data-status="error"]')).toBeVisible({ timeout: 15_000 });

  await tab1.close();

  const tab2 = await context.newPage();
  await tab2.goto(docURL);
  await expect(tab2.locator('[data-status="saved"]')).toBeVisible({ timeout: 20_000 });

  const revAfter = await tab2.evaluate(async (id) => {
    const res = await fetch(`/api/v2/documents/${id}`, { credentials: 'include' });
    const json = await res.json();
    return json.current_revision_id as string;
  }, docID);

  expect(revAfter).not.toBe(revBefore);
});