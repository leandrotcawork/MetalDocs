import { describe, it, expect, vi } from 'vitest';
import { PlaywrightPool } from '../playwright-pool';

const mockBrowser = { close: vi.fn(), newContext: vi.fn() };
vi.mock('playwright', () => ({
  chromium: { launch: vi.fn(async () => mockBrowser) },
}));

describe('PlaywrightPool', () => {
  it('launches POOL_SIZE browsers and acquires/releases', async () => {
    const pool = new PlaywrightPool({ size: 2 });
    await pool.init();
    const w1 = await pool.acquire();
    const w2 = await pool.acquire();
    expect(w1).toBeDefined();
    expect(w2).toBeDefined();
    pool.release(w1);
    const w3 = await pool.acquire();
    expect(w3).toBe(w1);
    await pool.shutdown();
    expect(mockBrowser.close).toHaveBeenCalledTimes(2);
  });
});