import { describe, it, expect, vi } from 'vitest';
import { paginateWithChromium } from '../paginate-with-chromium';
import { PlaywrightPool } from '../playwright-pool';

vi.mock('../playwright-pool', () => ({
  PlaywrightPool: vi.fn(),
}));

describe('paginateWithChromium (unit)', () => {
  it('returns ServerBreak[] from pool worker', async () => {
    const fakeBreaks = [
      { bid: 'bid-0001', pageNumber: 1 },
      { bid: 'bid-0050', pageNumber: 2 },
    ];

    const fakePool = {
      acquire: vi.fn(async () => fakeBrowser),
      release: vi.fn(),
      replace: vi.fn(),
    };

    const fakePage = {
      setContent: vi.fn(async () => {}),
      addScriptTag: vi.fn(async () => {}),
      waitForFunction: vi.fn(async () => {}),
      evaluate: vi.fn(async () => fakeBreaks),
    };
    const fakeCtx = {
      newPage: vi.fn(async () => fakePage),
      close: vi.fn(async () => {}),
    };
    const fakeBrowser = {
      newContext: vi.fn(async () => fakeCtx),
    };
    fakePool.acquire.mockResolvedValue(fakeBrowser);

    const result = await paginateWithChromium(fakePool as any, '<p data-mddm-bid="bid-0001">x</p>', { timeoutMs: 5000 });
    expect(result).toEqual(fakeBreaks);
  });
});
