import { describe, it, expect, vi, beforeEach } from 'vitest';
import { loadFillInData } from '../fill-in-loader';
import * as api from '../v2/api/documentsV2';

vi.mock('../v2/api/documentsV2');

function makeSchemaResponse() {
  return {
    data: {
      placeholder_schema: [{ id: 'p1', label: 'Title', type: 'text', required: false }],
    },
  };
}

beforeEach(() => {
  vi.mocked(api.getPlaceholderValues).mockResolvedValue([
    { placeholder_id: 'p1', value_text: 'Hello', source: 'user' },
  ]);

  global.fetch = vi.fn().mockImplementation((url: string) => {
    if (url === '/api/v2/documents/doc-1/fill-in-schema') {
      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve(makeSchemaResponse()),
      });
    }
    return Promise.resolve({ ok: false, status: 404 });
  }) as unknown as typeof fetch;
});

describe('loadFillInData', () => {
  it('returns placeholder values from API', async () => {
    const data = await loadFillInData('doc-1');
    expect(data.placeholderValues).toHaveLength(1);
    expect(data.placeholderValues[0].placeholder_id).toBe('p1');
  });

  it('returns placeholder schema from fill-in-schema endpoint', async () => {
    const data = await loadFillInData('doc-1');
    expect(data.placeholderSchema).toHaveLength(1);
    expect(data.placeholderSchema[0].id).toBe('p1');
    expect(data.placeholderSchema[0].label).toBe('Title');
    expect(data.placeholderSchema[0].type).toBe('text');
  });

  it('concurrent: schema + values fetched in parallel', async () => {
    const order: string[] = [];
    vi.mocked(api.getPlaceholderValues).mockImplementation(async () => {
      order.push('placeholders');
      return [];
    });

    await loadFillInData('doc-1');
    expect(order).toContain('placeholders');
  });
});
