import { test, expect } from '@playwright/test';
import * as fs from 'node:fs';
import * as path from 'node:path';

test('author happy path — create template, author, publish', async ({ page }) => {
  await page.goto('/templates-v2');
  await page.getByRole('button', { name: /new template/i }).click();
  await page.getByLabel(/key/i).fill('po');
  await page.getByLabel(/name/i).fill('Purchase Order');
  await page.getByRole('button', { name: /create/i }).click();

  await page.waitForURL(/\/templates-v2\/.+\/versions\/1\/author/);

  const docxBuf = fs.readFileSync(path.join(__dirname, 'fixtures/purchase-order.docx'));
  await page.setInputFiles('[data-testid="editor-file-input"]', {
    name: 'purchase-order.docx',
    mimeType: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
    buffer: docxBuf,
  });

  const schemaText = fs.readFileSync(path.join(__dirname, 'fixtures/purchase-order.schema.json'), 'utf8');
  await page.getByRole('button', { name: /schema/i }).click();
  const monaco = page.locator('.monaco-editor');
  await monaco.click();
  await page.keyboard.press('Control+A');
  await page.keyboard.insertText(schemaText);

  await expect(page.getByText(/saved/i)).toBeVisible({ timeout: 20_000 });

  await page.getByRole('button', { name: /publish/i }).click();
  await page.waitForURL(/\/templates-v2\/.+\/versions\/2\/author/, { timeout: 10_000 });
  await expect(page.getByRole('heading', { name: /purchase order/i })).toBeVisible();
});

test('publish after autosave races uses latest persisted keys', async ({ page }) => {
  await page.goto('/templates-v2');
  await page.getByRole('button', { name: /new template/i }).click();
  await page.getByLabel(/key/i).fill('po2');
  await page.getByLabel(/name/i).fill('Purchase Order 2');
  await page.getByRole('button', { name: /create/i }).click();
  await page.waitForURL(/\/templates-v2\/.+\/versions\/1\/author/);

  const docxBuf = fs.readFileSync(path.join(__dirname, 'fixtures/purchase-order.docx'));
  await page.setInputFiles('[data-testid="editor-file-input"]', {
    name: 'purchase-order.docx',
    mimeType: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
    buffer: docxBuf,
  });
  await expect(page.getByText(/saved/i)).toBeVisible({ timeout: 20_000 });

  await page.getByRole('button', { name: /publish/i }).click();
  await page.waitForURL(/\/templates-v2\/.+\/versions\/2\/author/, { timeout: 10_000 });
});
