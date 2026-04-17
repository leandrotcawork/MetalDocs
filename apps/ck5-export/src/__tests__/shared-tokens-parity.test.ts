import { describe, it, expect } from 'vitest';
import { defaultLayoutTokens as shared } from '@metaldocs/mddm-layout-tokens';

describe('shared tokens parity (export)', () => {
  it('shared module resolves and has expected shape', () => {
    expect(shared.page.widthMm).toBe(210);
    expect(shared.page.heightMm).toBe(297);
    expect(shared.page.marginLeftMm).toBe(25);
    expect(shared.typography.exportFont).toBe('Carlito');
  });
});
