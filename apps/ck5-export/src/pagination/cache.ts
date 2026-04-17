import { createHash } from 'node:crypto';
import type { ServerBreak } from './paginate-with-chromium';

type Entry = { breaks: ServerBreak[]; expiresAt: number };

export class PaginationCache {
  private readonly ttlMs: number;
  private readonly maxEntries: number;
  private readonly store = new Map<string, Entry>();

  public constructor(opts: { ttlMs?: number; maxEntries?: number } = {}) {
    this.ttlMs = opts.ttlMs ?? 300_000;
    this.maxEntries = opts.maxEntries ?? 64;
  }

  private static key(html: string): string {
    return createHash('sha256').update(html.trim()).digest('hex');
  }

  public get(html: string): ServerBreak[] | undefined {
    const k = PaginationCache.key(html);
    const entry = this.store.get(k);
    if (!entry) return undefined;
    if (entry.expiresAt <= Date.now()) { this.store.delete(k); return undefined; }
    // LRU bump
    this.store.delete(k);
    this.store.set(k, entry);
    return entry.breaks;
  }

  public set(html: string, breaks: ServerBreak[]): void {
    const k = PaginationCache.key(html);
    this.store.set(k, { breaks, expiresAt: Date.now() + this.ttlMs });
    while (this.store.size > this.maxEntries) {
      const oldest = this.store.keys().next().value;
      if (oldest !== undefined) this.store.delete(oldest);
    }
  }

  public clear(): void { this.store.clear(); }
  public get size(): number { return this.store.size; }
}
