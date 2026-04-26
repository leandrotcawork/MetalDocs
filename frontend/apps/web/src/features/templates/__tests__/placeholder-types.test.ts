import { describe, expect, it } from 'vitest';
import { slugifyLabel } from '../placeholder-types';

describe('slugifyLabel', () => {
  it('lowercases and underscores spaces', () => {
    expect(slugifyLabel('Customer Name')).toBe('customer_name');
  });
  it('strips special chars', () => {
    expect(slugifyLabel('Effective Date (ISO)')).toBe('effective_date_iso');
  });
  it('prefixes f_ when starts with non-letter', () => {
    expect(slugifyLabel('123abc')).toBe('f_123abc');
  });
  it('caps at 50 chars', () => {
    expect(slugifyLabel('a'.repeat(80)).length).toBeLessThanOrEqual(50);
  });
  it('fallback for empty', () => {
    expect(slugifyLabel('')).toBe('field');
  });
  it('trims leading/trailing underscores', () => {
    expect(slugifyLabel('  hello  ')).toBe('hello');
  });
});
