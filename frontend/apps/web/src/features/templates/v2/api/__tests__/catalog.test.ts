import { describe, expect, it, vi } from 'vitest';
import { fetchPlaceholderCatalog } from '../catalog';

describe('fetchPlaceholderCatalog', () => {
  it('returns 7 catalog entries from the API', async () => {
    vi.stubGlobal('fetch', vi.fn(() => Promise.resolve({
      ok: true,
      json: () => Promise.resolve({ items: [
        { key: 'doc_code', label: 'Código do documento', description: '' },
        { key: 'doc_title', label: 'Título do documento', description: '' },
        { key: 'revision_number', label: 'Número da revisão', description: '' },
        { key: 'author', label: 'Autor', description: '' },
        { key: 'effective_date', label: 'Data efetiva', description: '' },
        { key: 'approvers', label: 'Aprovadores', description: '' },
        { key: 'controlled_by_area', label: 'Área controladora', description: '' },
      ] }),
    })));
    const items = await fetchPlaceholderCatalog();
    expect(items).toHaveLength(7);
    expect(items[0].key).toBe('doc_code');
  });
});
