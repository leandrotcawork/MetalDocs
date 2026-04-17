import { describe, it, expect } from 'vitest';
import { reconcile } from '../reconcile';

describe('reconcile', () => {
  it('exact agree → editor wins', () => {
    const r = reconcile(
      [{ afterBid: 'a', pageNumber: 2 }],
      [{ bid: 'a', pageNumber: 2 }],
    );
    expect(r.resolved).toHaveLength(1);
    expect(r.resolved[0].source).toBe('editor');
  });
  it('minor drift ±1 → editor wins with tag', () => {
    const r = reconcile(
      [{ afterBid: 'a', pageNumber: 2 }],
      [{ bid: 'a', pageNumber: 3 }],
    );
    expect(r.resolved[0].source).toBe('editor-minor-drift');
  });
  it('major drift >1 → server wins', () => {
    const r = reconcile(
      [{ afterBid: 'a', pageNumber: 2 }],
      [{ bid: 'a', pageNumber: 5 }],
    );
    expect(r.resolved[0].source).toBe('server');
    expect(r.resolved[0].pageNumber).toBe(5);
  });
  it('orphan editor break → dropped', () => {
    const r = reconcile(
      [{ afterBid: 'ghost', pageNumber: 2 }],
      [],
    );
    expect(r.resolved).toHaveLength(0);
  });
  it('server-only break → included', () => {
    const r = reconcile(
      [],
      [{ bid: 'a', pageNumber: 2 }],
    );
    expect(r.resolved).toHaveLength(1);
    expect(r.resolved[0].source).toBe('server');
  });
  it('output is monotonic', () => {
    const r = reconcile(
      [{ afterBid: 'a', pageNumber: 3 }, { afterBid: 'b', pageNumber: 2 }],
      [{ bid: 'a', pageNumber: 3 }, { bid: 'b', pageNumber: 2 }],
    );
    for (let i = 1; i < r.resolved.length; i++) {
      expect(r.resolved[i].pageNumber).toBeGreaterThanOrEqual(r.resolved[i - 1].pageNumber);
    }
  });
});
