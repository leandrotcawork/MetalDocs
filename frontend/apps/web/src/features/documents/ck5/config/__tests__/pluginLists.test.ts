import { describe, it, expect } from 'vitest';
import { AUTHOR_PLUGINS, FILL_PLUGINS } from '../pluginLists';

describe('plugin lists', () => {
  it('AUTHOR_PLUGINS is a non-empty array of constructors', () => {
    expect(Array.isArray(AUTHOR_PLUGINS)).toBe(true);
    expect(AUTHOR_PLUGINS.length).toBeGreaterThan(5);
    for (const p of AUTHOR_PLUGINS) {
      expect(typeof p).toBe('function');
    }
  });

  it('FILL_PLUGINS is a non-empty array of constructors', () => {
    expect(Array.isArray(FILL_PLUGINS)).toBe(true);
    expect(FILL_PLUGINS.length).toBeGreaterThan(5);
  });

  it('Author includes StandardEditingMode, not RestrictedEditingMode', () => {
    const names = AUTHOR_PLUGINS.map((p) => p.name);
    expect(names).toContain('StandardEditingMode');
    expect(names).not.toContain('RestrictedEditingMode');
  });

  it('Fill includes RestrictedEditingMode, not StandardEditingMode', () => {
    const names = FILL_PLUGINS.map((p) => p.name);
    expect(names).toContain('RestrictedEditingMode');
    expect(names).not.toContain('StandardEditingMode');
  });
});
