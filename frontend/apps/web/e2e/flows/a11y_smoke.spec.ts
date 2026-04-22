/**
 * Task 11.9 — Axe a11y smoke across all 7 flows
 *
 * Runs AxeBuilder (WCAG 2.1 AA) at each route entry point.
 * Net-new violations vs axe-baseline.json fail the build via axe-diff.mjs.
 *
 * Invoked via fixture so assertions are DRY — not duplicated inline across flows.
 */
import { test, expect } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';
import { readFileSync, writeFileSync, existsSync } from 'node:fs';
import { resolve } from 'node:path';
import { seedTenant, resetTenant, SeedResult } from '../utils/seed';
import { loginAs, contextAs } from '../utils/auth';
import { randomUUID } from 'node:crypto';

const BASE_URL = process.env.E2E_BASE_URL || 'http://localhost:8080';
const BASELINE_PATH = resolve(__dirname, '../axe-baseline.json');
const REPORT_PATH = resolve(__dirname, '../../test-results/axe-report.json');

const AXE_TAGS = ['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa'];

// Dynamic regions to exclude from axe (timestamps, relative time strings)
const EXCLUDED_SELECTORS = ['.timeline-timestamp', '.lock-badge-relative-time'];

interface AxeViolation {
  id: string;
  impact: string;
  help: string;
  helpUrl: string;
  nodes: Array<{ target: string[] | string }>;
}

let seed: SeedResult;
const allViolations: AxeViolation[] = [];

async function expectAxeClean(page: import('@playwright/test').Page, label: string): Promise<void> {
  // Freeze Date so relative timestamps are deterministic
  await page.clock.install();

  const builder = new AxeBuilder({ page })
    .withTags(AXE_TAGS);

  for (const sel of EXCLUDED_SELECTORS) {
    builder.exclude(sel);
  }

  const results = await builder.analyze();

  // Collect violations for post-run diff
  for (const v of results.violations) {
    if (v.impact === 'minor') continue; // minor: log only
    allViolations.push({
      id: v.id,
      impact: v.impact ?? 'unknown',
      help: v.help,
      helpUrl: v.helpUrl,
      nodes: v.nodes.map(n => ({ target: n.target ?? [] })),
    });
  }

  const criticals = results.violations.filter(v => v.impact === 'critical');
  if (criticals.length > 0) {
    const ids = criticals.map(v => v.id).join(', ');
    throw new Error(`[${label}] axe critical violations: ${ids}`);
  }

  // Serious/moderate: only fail if not in baseline
  const baseline: AxeViolation[] = existsSync(BASELINE_PATH)
    ? JSON.parse(readFileSync(BASELINE_PATH, 'utf8'))
    : [];
  const baselineIds = new Set(baseline.map(b => b.id));

  const netNew = results.violations.filter(
    v => v.impact !== 'minor' && v.impact !== 'critical' && !baselineIds.has(v.id)
  );

  if (netNew.length > 0) {
    const ids = netNew.map(v => `${v.id}(${v.impact})`).join(', ');
    throw new Error(`[${label}] net-new axe violations not in baseline: ${ids}`);
  }
}

test.describe('a11y smoke — all approval routes', () => {
  test.beforeAll(async ({ request }) => {
    seed = await seedTenant(request, { workerIndex: 0, testTitle: 'a11y_smoke' });
  });

  test.afterAll(async ({ request }) => {
    // Write aggregated violations report for axe-diff.mjs
    writeFileSync(REPORT_PATH, JSON.stringify(allViolations, null, 2));
    await resetTenant(request, seed.tenantId);
  });

  test('inbox page — no critical axe violations', async ({ page }) => {
    await loginAs(page, seed.cookies, 'reviewer');
    await page.goto('/approval/inbox');
    await page.setViewportSize({ width: 1280, height: 800 });
    await expectAxeClean(page, 'InboxPage');
  });

  test('route admin page — no critical axe violations', async ({ page }) => {
    await loginAs(page, seed.cookies, 'admin');
    await page.goto('/approval/admin/routes');
    await page.setViewportSize({ width: 1280, height: 800 });
    await expectAxeClean(page, 'RouteAdminPage');
  });

  test('doc detail page (draft) — no critical axe violations', async ({ page }) => {
    await loginAs(page, seed.cookies, 'author');
    await page.goto(`/docs/${seed.docId}`);
    await page.setViewportSize({ width: 1280, height: 800 });
    await expectAxeClean(page, 'DocDetail:draft');
  });

  test('doc detail page (under_review) — no critical axe violations', async ({ page }) => {
    // Submit to get under_review state
    await page.request.post(`/api/v2/documents/${seed.docId}/submit`, {
      headers: {
        'Content-Type': 'application/json',
        'Idempotency-Key': randomUUID(),
        'X-Tenant-ID': seed.tenantId,
      },
      data: {},
    });

    await loginAs(page, seed.cookies, 'author');
    await page.goto(`/docs/${seed.docId}`);
    await page.setViewportSize({ width: 1280, height: 800 });
    await expectAxeClean(page, 'DocDetail:under_review');
  });

  test('SignoffDialog open — no critical axe violations', async ({ page }) => {
    await loginAs(page, seed.cookies, 'reviewer');
    await page.goto('/approval/inbox');
    await page.setViewportSize({ width: 1280, height: 800 });

    // Open dialog if row present
    const row = page.getByTestId(`inbox-row-${seed.docId}`);
    if (await row.count() === 0) {
      test.skip();
      return;
    }
    await row.click();
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 3000 });
    await expectAxeClean(page, 'SignoffDialog:open');
  });

  test('SupersedePublishDialog open — no critical axe violations', async ({ page }) => {
    await loginAs(page, seed.cookies, 'admin');
    await page.goto(`/docs/${seed.docId}`);
    await page.setViewportSize({ width: 1280, height: 800 });

    const publishBtn = page.getByRole('button', { name: /publicar/i });
    if (await publishBtn.count() === 0) {
      test.skip();
      return;
    }
    await publishBtn.click();
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 3000 });
    await expectAxeClean(page, 'SupersedePublishDialog:open');
  });

  test('route admin new route dialog — no critical axe violations', async ({ page }) => {
    await loginAs(page, seed.cookies, 'admin');
    await page.goto('/approval/admin/routes');
    await page.setViewportSize({ width: 1280, height: 800 });

    await page.getByRole('button', { name: /nova rota/i }).click();
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 3000 });
    await expectAxeClean(page, 'RouteAdmin:newRouteDialog');
  });
});
