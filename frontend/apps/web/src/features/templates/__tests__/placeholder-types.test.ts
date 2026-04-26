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
  it('f_ prefix on digit-led 50-char label stays within 50 chars', () => {
    // A label of 50 digits: cleaned = '1...1' (50 chars), f_ prefix → 52, slice(0,50) = 'f_1...1' (50)
    const label = '1'.repeat(50);
    const result = slugifyLabel(label);
    expect(result.length).toBeLessThanOrEqual(50);
    expect(result.startsWith('f_')).toBe(true);
  });
});
