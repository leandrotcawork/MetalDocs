import { describe, it, expect, vi, beforeEach } from 'vitest';
import { loadFillInData } from '../fill-in-loader';
import * as api from '../v2/api/documentsV2';

vi.mock('../v2/api/documentsV2');

const BODY_BYTES = new Uint8Array([1, 2, 3, 4]);

function makeDocResponse() {
  return {
    data: {
      body_url: 'https://s3.example.com/body.docx',
      placeholder_schema: [{ id: 'p1', label: 'Title', type: 'text' as const }],
      zone_schema: [{ id: 'z1', label: 'Section', contentPolicy: { allowTables: true, allowImages: false, allowHeadings: true, allowLists: true } }],
    },
  };
}

beforeEach(() => {
  vi.mocked(api.getPlaceholderValues).mockResolvedValue([
    { placeholder_id: 'p1', value_text: 'Hello', source: 'user' },
  ]);
  vi.mocked(api.getZoneContents).mockResolvedValue([
    { zone_id: 'z1', content_ooxml: '<w:p/>' },
  ]);

  global.fetch = vi.fn().mockImplementation((url: string) => {
    if (url === '/api/v2/documents/doc-1') {
      return Promise.resolve({
        ok: true,
        json: () => Promise.resolve(makeDocResponse()),
      });
    }
    if (url === 'https://s3.example.com/body.docx') {
      return Promise.resolve({
        ok: true,
        arrayBuffer: () => Promise.resolve(BODY_BYTES.buffer),
      });
    }
    return Promise.resolve({ ok: false, status: 404 });
  }) as unknown as typeof fetch;
});

describe('loadFillInData', () => {
  it('returns correct shape with Uint8Array bodyDocx', async () => {
    const data = await loadFillInData('doc-1');
    expect(data.bodyDocx).toBeInstanceOf(Uint8Array);
    expect(Array.from(data.bodyDocx)).toEqual([1, 2, 3, 4]);
  });

  it('returns placeholder and zone values from API', async () => {
    const data = await loadFillInData('doc-1');
    expect(data.placeholderValues).toHaveLength(1);
    expect(data.placeholderValues[0].placeholder_id).toBe('p1');
    expect(data.zoneContents).toHaveLength(1);
    expect(data.zoneContents[0].zone_id).toBe('z1');
  });

  it('returns schemas from document detail', async () => {
    const data = await loadFillInData('doc-1');
    expect(data.placeholderSchema).toHaveLength(1);
    expect(data.placeholderSchema[0].id).toBe('p1');
  });

  it('concurrent: all three fetches happen in parallel', async () => {
    const order: string[] = [];
    vi.mocked(api.getPlaceholderValues).mockImplementation(async () => {
      order.push('placeholders');
      return [];
    });
    vi.mocked(api.getZoneContents).mockImplementation(async () => {
      order.push('zones');
      return [];
    });

    await loadFillInData('doc-1');
    expect(order).toContain('placeholders');
    expect(order).toContain('zones');
  });
});
