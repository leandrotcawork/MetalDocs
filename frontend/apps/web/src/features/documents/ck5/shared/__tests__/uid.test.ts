import { describe, it, expect } from 'vitest';
import { uid } from '../uid';

describe('uid', () => {
  it('returns a non-empty string', () => {
    expect(typeof uid()).toBe('string');
    expect(uid().length).toBeGreaterThan(0);
  });

  it('returns unique values on successive calls', () => {
    const set = new Set<string>();
    for (let i = 0; i < 100; i++) set.add(uid());
    expect(set.size).toBe(100);
  });

  it('accepts a prefix', () => {
    expect(uid('sec')).toMatch(/^sec-/);
  });
});
