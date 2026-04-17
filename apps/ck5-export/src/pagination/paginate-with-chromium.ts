import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import type { PlaywrightPool } from './playwright-pool';
import { injectSentinels } from './sentinel';
import { wrapInPrintDocument } from '../print-stylesheet/wrap-print-document';
import { withWorker, WorkerCrash, PaginatorTimeoutError } from './pool-retry';
import { PaginationCache } from './cache';

export const paginationCache = new PaginationCache();

export { PaginatorTimeoutError } from './pool-retry';

export type ServerBreak = Readonly<{ bid: string; pageNumber: number }>;

export async function paginateWithChromium(
  pool: PlaywrightPool,
  rawHtml: string,
  opts: { timeoutMs: number },
): Promise<ServerBreak[]> {
  const cached = paginationCache.get(rawHtml);
  if (cached) return cached;

  const result = await withWorker(pool, async (browser: any) => {
    const withSentinels = injectSentinels(rawHtml);
    const fullHtml = wrapInPrintDocument(withSentinels);

    const ctx = await browser.newContext();
    const page = await ctx.newPage();

    try {
      await page.setContent(fullHtml, { waitUntil: 'networkidle', timeout: opts.timeoutMs })
        .catch((e: Error) => {
          if (/crash|closed/i.test(e.message)) throw new WorkerCrash(e.message);
          throw e;
        });

      const polyfillPath = join(process.cwd(), 'public', 'paged.polyfill.js');
      let polyfillContent = '';
      try { polyfillContent = readFileSync(polyfillPath, 'utf-8'); } catch { /* not available */ }

      if (polyfillContent) {
        await page.addScriptTag({ content: polyfillContent });
      }

      await page.waitForFunction(
        () => (document as any).querySelector('.pagedjs_page') !== null,
        { timeout: opts.timeoutMs },
      ).catch(() => {
        throw new PaginatorTimeoutError(opts.timeoutMs);
      });

      const breaks = await page.evaluate(() => {
        const result: { bid: string; pageNumber: number }[] = [];
        const markers = Array.from(document.querySelectorAll('[data-pb-marker]'));
        for (const m of markers) {
          const bid = (m as HTMLElement).dataset['pbMarker']!;
          const pageEl = m.closest('.pagedjs_page') as HTMLElement | null;
          const n = pageEl ? Number(pageEl.dataset['pageNumber'] ?? 1) : 1;
          result.push({ bid, pageNumber: n });
        }
        return result;
      });

      return breaks as ServerBreak[];
    } finally {
      await ctx.close();
    }
  });
  paginationCache.set(rawHtml, result);
  return result;
}
