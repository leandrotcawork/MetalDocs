import { test, expect } from '@playwright/test';
import JSZip from 'jszip';

test('pagination — 10 pages end-to-end', async ({ page }) => {
  await page.goto('/templates/demo/editor');
  await page.locator('[contenteditable]').click();
  const lorem = 'Lorem ipsum dolor sit amet, consectetur adipiscing elit. '.repeat(300);
  await page.keyboard.type(lorem, { delay: 0 });
  await expect(page.locator('.mddm-page-counter')).toContainText(/Page \d+ of \d+/, { timeout: 10000 });
  const countText = await page.locator('.mddm-page-counter').textContent();
  const counter = Number((countText ?? '').match(/of (\d+)/)![1]);

  const downloadPromise = page.waitForEvent('download');
  await page.getByRole('button', { name: 'Export DOCX' }).click();
  const dl = await downloadPromise;
  const path = await dl.path();
  const fs = await import('node:fs');
  const zip = await JSZip.loadAsync(fs.readFileSync(path!));
  const xml = await zip.file('word/document.xml')!.async('string');
  const pageBreaks = (xml.match(/<w:br w:type="page"\/>/g) ?? []).length;
  expect(pageBreaks + 1).toBeGreaterThanOrEqual(counter - 1);
  expect(pageBreaks + 1).toBeLessThanOrEqual(counter + 1);
});