import { beforeEach, describe, expect, it, vi } from 'vitest';

describe('exportApi', () => {
  beforeEach(() => {
    vi.resetModules();
    vi.stubGlobal('fetch', vi.fn());
  });

  it('triggerExport creates an object URL for docx export', async () => {
    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      status: 200,
      blob: () => Promise.resolve(new Blob(['data'])),
    } as Response);
    vi.stubGlobal('fetch', mockFetch);

    const createObjectURL = vi.fn(() => 'blob:mock');
    const revokeObjectURL = vi.fn();
    vi.stubGlobal('URL', { ...URL, createObjectURL, revokeObjectURL });

    const { triggerExport } = await import('../exportApi');
    await expect(triggerExport('doc-1', 'docx')).resolves.toBeUndefined();
    expect(createObjectURL).toHaveBeenCalled();
  });

  it('triggerExport throws ExportError with status on failure', async () => {
    vi.stubGlobal(
      'fetch',
      vi.fn().mockResolvedValue({
        ok: false,
        status: 500,
      } as Response),
    );

    const { triggerExport, ExportError } = await import('../exportApi');

    await expect(triggerExport('doc-1', 'pdf')).rejects.toEqual(expect.any(ExportError));
    await expect(triggerExport('doc-1', 'pdf')).rejects.toMatchObject({ status: 500 });
  });

  it('clientPrint calls print on iframe window', async () => {
    const { clientPrint } = await import('../exportApi');

    const print = vi.fn();
    const focus = vi.fn();
    const open = vi.fn();
    const write = vi.fn();
    const close = vi.fn();
    const removeEventListener = vi.fn();
    const addEventListener = vi.fn((_type: string, cb: EventListenerOrEventListenerObject) => {
      if (typeof cb === 'function') cb(new Event('afterprint'));
    });

    const iframe = {
      style: { cssText: '' },
      contentWindow: {
        document: { open, write, close },
        print,
        focus,
        addEventListener,
        removeEventListener,
      },
    } as unknown as HTMLIFrameElement;

    const originalCreateElement = document.createElement.bind(document);
    vi.spyOn(document, 'createElement').mockImplementation(((tagName: string) => {
      if (tagName === 'iframe') return iframe;
      return originalCreateElement(tagName as keyof HTMLElementTagNameMap);
    }) as typeof document.createElement);

    const appendChildSpy = vi.spyOn(document.body, 'appendChild').mockImplementation((node: Node) => node);
    vi.spyOn(document.body, 'removeChild').mockImplementation((node: Node) => node);

    expect(() => clientPrint('<p>Hello</p>')).not.toThrow();
    expect(appendChildSpy).toHaveBeenCalled();
    expect(print).toHaveBeenCalled();
  });
});
