import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

let saveTemplate: typeof import('../apiPersistence').saveTemplate;
let loadTemplate: typeof import('../apiPersistence').loadTemplate;
let saveDocument: typeof import('../apiPersistence').saveDocument;
let loadDocument: typeof import('../apiPersistence').loadDocument;

const mockFetch = vi.fn();

beforeEach(async () => {
  globalThis.fetch = mockFetch;
  vi.resetModules();
  const mod = await import('../apiPersistence');
  saveTemplate = mod.saveTemplate;
  loadTemplate = mod.loadTemplate;
  saveDocument = mod.saveDocument;
  loadDocument = mod.loadDocument;
});

afterEach(() => {
  vi.restoreAllMocks();
});

function ok(body: unknown = {}): Response {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { 'Content-Type': 'application/json' },
  });
}

const templateDraftResponse = {
  templateKey: 'tpl-1',
  profileCode: 'PO',
  name: 'tpl-1',
  status: 'draft',
  lockVersion: 1,
  hasStrippedFields: false,
  blocks: { _ck5: { contentHtml: '<p>loaded</p>' } },
  theme: {},
  meta: {},
};

describe('saveTemplate', () => {
  it('PUTs to /api/v1/templates/:key/draft after loadTemplate primes cache', async () => {
    const saveResponse = { ...templateDraftResponse, lockVersion: 2 };
    mockFetch
      .mockResolvedValueOnce(ok(templateDraftResponse))
      .mockResolvedValueOnce(ok(saveResponse));
    await loadTemplate('tpl-1');
    await saveTemplate('tpl-1', '<p>x</p>', { fields: [] });
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/v1/templates/tpl-1/draft',
      expect.objectContaining({ method: 'PUT', credentials: 'include' }),
    );
  });

  it('skips fetch when cache not primed', async () => {
    await saveTemplate('no-load', '<p>x</p>', { fields: [] });
    expect(mockFetch).not.toHaveBeenCalled();
  });
});

describe('loadTemplate', () => {
  it('calls GET /api/v1/templates/:id and maps contentHtml from blocks', async () => {
    mockFetch.mockResolvedValue(ok(templateDraftResponse));
    const rec = await loadTemplate('tpl-1');
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/v1/templates/tpl-1',
      expect.objectContaining({ credentials: 'include' }),
    );
    expect(rec?.contentHtml).toBe('<p>loaded</p>');
  });

  it('returns null on 404', async () => {
    mockFetch.mockResolvedValue(new Response('', { status: 404 }));
    expect(await loadTemplate('missing')).toBeNull();
  });

  it('throws on 5xx', async () => {
    mockFetch.mockResolvedValue(new Response('err', { status: 500 }));
    await expect(loadTemplate('tpl-1')).rejects.toThrow();
  });
});

describe('saveDocument', () => {
  it('calls POST /api/v1/documents/:id/content/ck5 with body field', async () => {
    mockFetch.mockResolvedValue(new Response('', { status: 201 }));
    await saveDocument('doc-1', '<p>doc</p>');
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/v1/documents/doc-1/content/ck5',
      expect.objectContaining({
        method: 'POST',
        credentials: 'include',
        body: JSON.stringify({ body: '<p>doc</p>' }),
      }),
    );
  });
});

describe('loadDocument', () => {
  it('calls GET /api/v1/documents/:id/content/ck5 and returns body field', async () => {
    mockFetch.mockResolvedValue(ok({ body: '<p>doc content</p>' }));
    const html = await loadDocument('doc-2');
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/v1/documents/doc-2/content/ck5',
      expect.objectContaining({ credentials: 'include' }),
    );
    expect(html).toBe('<p>doc content</p>');
  });

  it('returns null on 404', async () => {
    mockFetch.mockResolvedValue(new Response('', { status: 404 }));
    expect(await loadDocument('missing')).toBeNull();
  });
});
