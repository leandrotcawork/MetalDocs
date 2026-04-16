import { describe, it, expect, vi, beforeEach } from 'vitest';

describe('apiPersistence', () => {
  beforeEach(() => {
    vi.stubGlobal('fetch', vi.fn());
  });

  it('saveTemplate PUTs to /api/v1/templates/:key/draft', async () => {
    const mockFetch = vi.fn().mockResolvedValue(new Response('{}', { status: 200 }));
    vi.stubGlobal('fetch', mockFetch);
    const { saveTemplate } = await import('../apiPersistence');
    await saveTemplate('tpl-1', '<p>x</p>', { fields: [] });
    expect(mockFetch).toHaveBeenCalledWith(
      '/api/v1/templates/tpl-1/draft',
      expect.objectContaining({ method: 'PUT', credentials: 'include' }),
    );
  });

  it('loadTemplate returns null on 404', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('', { status: 404 })));
    const { loadTemplate } = await import('../apiPersistence');
    const result = await loadTemplate('missing');
    expect(result).toBeNull();
  });

  it('loadTemplate throws on 5xx', async () => {
    vi.stubGlobal('fetch', vi.fn().mockResolvedValue(new Response('err', { status: 500 })));
    const { loadTemplate } = await import('../apiPersistence');
    await expect(loadTemplate('tpl-1')).rejects.toThrow();
  });
});
