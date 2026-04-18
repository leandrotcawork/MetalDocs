import { describe, it, expect } from 'vitest';
import { computeSidebarModel } from '../src/plugins/mergefieldPlugin';

describe('mergefieldPlugin.computeSidebarModel', () => {
  it('returns used/missing/orphan segments', () => {
    const m = computeSidebarModel(
      [{ kind: 'var', ident: 'name', start: 0, end: 6, run_id: 'r0' }],
      [],
      { type: 'object', properties: { name: { type: 'string' }, age: { type: 'number' } } }
    );
    expect(m.used).toEqual(['name']);
    expect(m.missing).toEqual(['age']);
    expect(m.orphans).toEqual([]);
  });

  it('surfaces parse errors for red banner', () => {
    const m = computeSidebarModel(
      [],
      [{ type: 'unsupported_construct', element: 'w:ins', location: '', auto_fixable: false }],
      { type: 'object', properties: {} }
    );
    expect(m.bannerError).toBe(true);
    expect(m.errorCategories).toContain('tracked-changes');
  });
});
