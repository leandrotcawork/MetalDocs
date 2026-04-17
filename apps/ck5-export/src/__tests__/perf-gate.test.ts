import { describe, it, expect, beforeAll, afterAll } from 'vitest';
import { readFileSync, writeFileSync, mkdirSync } from 'node:fs';
import { join } from 'node:path';
import { PlaywrightPool } from '../pagination/playwright-pool';
import { paginateWithChromium, paginationCache } from '../pagination/paginate-with-chromium';

function percentile(sorted: number[], p: number): number {
  const idx = Math.ceil((p / 100) * sorted.length) - 1;
  return sorted[Math.max(0, idx)];
}

describe('paginator perf gate', () => {
  let pool: PlaywrightPool;
  beforeAll(async () => {
    pool = new PlaywrightPool({ size: 1 });
    await pool.init();
  }, 60000);
  afterAll(async () => { await pool.shutdown(); });

  it('p95 ≤ 3000ms for 100-page fixture', async () => {
    const html = readFileSync(
      join(__dirname, '../__fixtures__/pagination/100-page-contract.html'),
      'utf-8',
    );
    const durations: number[] = [];
    for (let i = 0; i < 22; i++) {
      paginationCache.clear();
      const t0 = performance.now();
      await paginateWithChromium(pool, html, { timeoutMs: 30000 });
      durations.push(performance.now() - t0);
    }
    const measured = durations.slice(2).sort((a, b) => a - b);
    const p50 = percentile(measured, 50);
    const p95 = percentile(measured, 95);

    mkdirSync(join(__dirname, '../../artifacts'), { recursive: true });
    writeFileSync(
      join(__dirname, '../../artifacts/perf-gate.json'),
      JSON.stringify({ p50, p95, durations: measured }, null, 2),
    );

    console.log(`[perf-gate] p50=${p50.toFixed(0)}ms p95=${p95.toFixed(0)}ms`);
    expect(p95).toBeLessThanOrEqual(3000);
  }, 120000);
});