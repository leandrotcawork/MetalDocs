import { describe, it, expect } from 'vitest';
import { validateBids } from '../validator';

const CLEAN = '<p data-mddm-bid="aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa">x</p><p data-mddm-bid="bbbbbbbb-bbbb-4bbb-bbbb-bbbbbbbbbbbb">y</p>';
const COLLISION = '<p data-mddm-bid="aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa">x</p><p data-mddm-bid="aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa">y</p>';
const MISSING = '<p>x</p><p data-mddm-bid="aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa">y</p>';

describe('validateBids', () => {
  it('accepts clean html', () => {
    expect(validateBids(CLEAN).ok).toBe(true);
  });
  it('rejects on duplicate bid', () => {
    const r = validateBids(COLLISION);
    expect(r.ok).toBe(false);
    if (!r.ok) {
      expect(r.severity).toBe('error');
      expect(r.error).toBe('bid-collision');
      expect(r.bids).toContain('aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa');
    }
  });
  it('warns on paginable missing bid', () => {
    const r = validateBids(MISSING);
    expect(r.ok).toBe(false);
    if (!r.ok) {
      expect(r.severity).toBe('warn');
      expect(r.error).toBe('paginable-missing-bid');
    }
  });
  it('ignores inline elements without bid', () => {
    expect(validateBids('<p data-mddm-bid="aaaaaaaa-aaaa-4aaa-aaaa-aaaaaaaaaaaa"><span>x</span></p>').ok).toBe(true);
  });
});
