import { describe, it, expect } from 'vitest';
import { IDENT_RE, RESERVED_IDENTS, isValidIdent } from '../src/grammar';

describe('grammar', () => {
  it.each([
    ['client_name', true],
    ['Item1', true],
    ['_internal', true],
    ['1starts_number', false],
    ['client.name', false],
    ['has space', false],
    ['', false],
  ])('ident %s → valid=%s', (s, want) => {
    expect(isValidIdent(s)).toBe(want);
  });

  it('RESERVED_IDENTS rejects docgen internals', () => {
    expect(RESERVED_IDENTS.has('__proto__')).toBe(true);
    expect(RESERVED_IDENTS.has('constructor')).toBe(true);
  });

  it('IDENT_RE matches spec EBNF', () => {
    expect('client_name').toMatch(IDENT_RE);
    expect('1bad').not.toMatch(IDENT_RE);
  });
});
