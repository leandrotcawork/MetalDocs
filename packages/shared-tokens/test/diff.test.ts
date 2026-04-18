import { describe, it, expect } from 'vitest';
import { diffTokensVsSchema } from '../src/diff';
import type { Token } from '../src/types';

const t = (ident: string, kind: Token['kind'] = 'var'): Token => ({
  ident, kind, start: 0, end: 0, run_id: 'r0',
});

describe('diffTokensVsSchema', () => {
  const schema = {
    type: 'object',
    properties: {
      client_name: { type: 'string' },
      items: { type: 'array', items: { type: 'object', properties: { sku: { type: 'string' } } } },
    },
    required: ['client_name'],
  };

  it('reports missing (in schema, not in docx)', () => {
    const d = diffTokensVsSchema([t('items', 'section'), t('sku')], schema);
    expect(d.missing).toContain('client_name');
  });

  it('reports orphan (in docx, not in schema)', () => {
    const d = diffTokensVsSchema([t('client_name'), t('not_in_schema')], schema);
    expect(d.orphans).toContain('not_in_schema');
  });

  it('treats section idents as array-paths', () => {
    const d = diffTokensVsSchema([t('client_name'), t('items', 'section'), t('sku'), t('items', 'closing')], schema);
    expect(d.orphans).not.toContain('items');
    expect(d.orphans).not.toContain('sku');
  });
});
