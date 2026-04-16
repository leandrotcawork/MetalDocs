import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

// Import after mocking so fetch is patched before module loads.
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

describe('saveTemplate', () => {
  it('calls PUT /api/v1/templates/{id}/ck5-draft with correct shape', async () => {
    mockFetch.mockResolvedValue(ok());
    await saveTemplate('tpl-1', '<p>hi</p>', { fields: [] });
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/v1/templates/tpl-1/ck5-draft',
      expect.objectContaining({
        method: 'PUT',
        body: JSON.stringify({ contentHtml: '<p>hi</p>', manifest: { fields: [] } }),
      }),
    );
  });
});

describe('loadTemplate', () => {
  it('calls GET /api/v1/templates/{id}/ck5-draft and maps contentHtml', async () => {
    mockFetch.mockResolvedValue(ok({ contentHtml: '<p>loaded</p>', manifest: { fields: [] } }));
    const rec = await loadTemplate('tpl-2');
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/v1/templates/tpl-2/ck5-draft',
      expect.objectContaining({ credentials: 'include' }),
    );
    expect(rec?.contentHtml).toBe('<p>loaded</p>');
  });

  it('returns null on 404', async () => {
    mockFetch.mockResolvedValue(new Response('', { status: 404 }));
    const rec = await loadTemplate('missing');
    expect(rec).toBeNull();
  });
});

describe('saveDocument', () => {
  it('calls POST /api/v1/documents/{id}/content/ck5 with body field', async () => {
    mockFetch.mockResolvedValue(new Response('', { status: 201 }));
    await saveDocument('doc-1', '<p>doc</p>');
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/v1/documents/doc-1/content/ck5',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({ body: '<p>doc</p>' }),
      }),
    );
  });
});

describe('loadDocument', () => {
  it('calls GET /api/v1/documents/{id}/content/ck5 and returns body field', async () => {
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
