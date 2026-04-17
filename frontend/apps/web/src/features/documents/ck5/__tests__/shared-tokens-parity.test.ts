import { describe, it, expect } from 'vitest';
import { defaultLayoutTokens } from '@metaldocs/mddm-layout-tokens';

describe('shared tokens parity (frontend)', () => {
  it('alias resolves and exposes canonical token shape', () => {
    expect(defaultLayoutTokens.page.widthMm).toBe(210);
    expect(defaultLayoutTokens.page.heightMm).toBe(297);
    expect(defaultLayoutTokens.page.marginLeftMm).toBe(25);
    expect(defaultLayoutTokens.page.marginRightMm).toBe(25);
    expect(defaultLayoutTokens.typography.exportFont).toBe('Carlito');
  });
});
