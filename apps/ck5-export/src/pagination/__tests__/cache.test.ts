import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { PaginationCache } from '../cache';

describe('PaginationCache', () => {
  beforeEach(() => { vi.useFakeTimers(); });
  afterEach(() => { vi.useRealTimers(); });

  it('returns undefined for unseen key', () => {
    const c = new PaginationCache();
    expect(c.get('<p>x</p>')).toBeUndefined();
  });

  it('round-trips breaks for same html', () => {
    const c = new PaginationCache();
    c.set('<p>x</p>', [{ bid: 'a', pageNumber: 1 }]);
    expect(c.get('<p>x</p>')).toEqual([{ bid: 'a', pageNumber: 1 }]);
  });

  it('expires entries after TTL', () => {
    const c = new PaginationCache({ ttlMs: 1000 });
    c.set('<p>x</p>', [{ bid: 'a', pageNumber: 1 }]);
    vi.advanceTimersByTime(1001);
    expect(c.get('<p>x</p>')).toBeUndefined();
  });

  it('evicts LRU beyond maxEntries', () => {
    const c = new PaginationCache({ maxEntries: 2 });
    c.set('a', [{ bid: 'a', pageNumber: 1 }]);
    c.set('b', [{ bid: 'b', pageNumber: 1 }]);
    c.set('c', [{ bid: 'c', pageNumber: 1 }]);
    expect(c.get('a')).toBeUndefined();
    expect(c.get('b')).toBeDefined();
    expect(c.get('c')).toBeDefined();
  });
});
