import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { MddmUploadAdapter } from '../MddmUploadAdapter';

describe('MddmUploadAdapter', () => {
  beforeEach(() => {
    globalThis.fetch = vi.fn(async () =>
      new Response(JSON.stringify({ url: 'https://cdn.example/abc.png' }), {
        status: 200,
        headers: { 'Content-Type': 'application/json' },
      }),
    );
  });

  afterEach(() => {
    vi.restoreAllMocks();
  });

  it('uploads a file via POST /assets and returns default URL', async () => {
    const file = new File(['data'], 'a.png', { type: 'image/png' });
    const adapter = new MddmUploadAdapter({
      loader: { file: Promise.resolve(file) } as never,
      endpoint: '/assets',
      getAuthHeader: () => 'Bearer x',
    });
    const result = await adapter.upload();
    expect(result).toEqual({ default: 'https://cdn.example/abc.png' });
    expect(globalThis.fetch).toHaveBeenCalledWith(
      '/assets',
      expect.objectContaining({
        method: 'POST',
        headers: expect.objectContaining({ Authorization: 'Bearer x' }),
      }),
    );
  });

  it('rejects when server returns non-OK', async () => {
    globalThis.fetch = vi.fn(async () => new Response('bad', { status: 500 }));
    const adapter = new MddmUploadAdapter({
      loader: { file: Promise.resolve(new File([], 'x')) } as never,
      endpoint: '/assets',
      getAuthHeader: () => null,
    });
    await expect(adapter.upload()).rejects.toThrow(/upload failed/i);
  });
});
