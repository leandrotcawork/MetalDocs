import { describe, it, expect } from 'vitest';
import { parseDocxTokens } from '../src/parser';
import { makeDocx, HAPPY_DOC } from './fixtures';

describe('parseDocxTokens (happy path)', () => {
  it('finds 2 var tokens with zero errors', async () => {
    const buf = await makeDocx(HAPPY_DOC);
    const result = await parseDocxTokens(buf);
    expect(result.errors).toEqual([]);
    expect(result.tokens).toHaveLength(2);
    expect(result.tokens.map(t => t.ident)).toEqual(['client_name','total_amount']);
    expect(result.tokens.every(t => t.kind === 'var')).toBe(true);
  });
});
