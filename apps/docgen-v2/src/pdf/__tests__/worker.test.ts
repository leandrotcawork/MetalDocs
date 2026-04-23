import { describe, expect, test, vi, beforeEach, afterEach } from 'vitest';
import { runPdfJob, type PdfJobDeps, type PdfJobInput } from '../worker.js';

let fetchMock: ReturnType<typeof vi.fn>;

beforeEach(() => {
  fetchMock = vi.fn();
  vi.stubGlobal('fetch', fetchMock);
});

afterEach(() => {
  vi.unstubAllGlobals();
});

function makeDeps(overrides?: Partial<PdfJobDeps>): PdfJobDeps {
  return {
    gotenbergUrl: 'http://gotenberg.local',
    getObject: vi.fn(async () => Buffer.from('docx-bytes')),
    putObject: vi.fn(async () => {}),
    sleep: vi.fn(async () => {}),
    now: () => new Date('2026-04-23T18:00:00.000Z'),
    ...overrides,
  };
}

function makeInput(): PdfJobInput {
  return {
    tenant_id: 't',
    revision_id: 'r',
    final_docx_s3_key: 'final/r.docx',
  };
}

describe('runPdfJob', () => {
  test('reads DOCX, converts, writes PDF, returns metadata', async () => {
    fetchMock.mockResolvedValueOnce(
      new Response(Buffer.from('pdf-bytes'), { status: 200 }),
    );
    const deps = makeDeps();

    const result = await runPdfJob(makeInput(), deps);

    expect(deps.getObject).toHaveBeenCalledWith('final/r.docx');
    expect(fetchMock).toHaveBeenCalledWith(
      'http://gotenberg.local/forms/libreoffice/convert',
      expect.objectContaining({ method: 'POST' }),
    );
    expect(deps.putObject).toHaveBeenCalledWith(
      'final/r.docx.pdf',
      expect.any(Buffer),
      'application/pdf',
    );
    expect(result.final_pdf_s3_key).toBe('final/r.docx.pdf');
    expect(result.pdf_hash).toMatch(/^[0-9a-f]{64}$/);
    expect(result.pdf_generated_at).toBe('2026-04-23T18:00:00.000Z');
  });

  test('retries on 5xx up to 3 attempts, then succeeds', async () => {
    fetchMock
      .mockResolvedValueOnce(new Response('oops', { status: 502 }))
      .mockResolvedValueOnce(new Response('still', { status: 503 }))
      .mockResolvedValueOnce(
        new Response(Buffer.from('pdf-bytes'), { status: 200 }),
      );
    const deps = makeDeps();

    const result = await runPdfJob(makeInput(), deps);
    expect(fetchMock).toHaveBeenCalledTimes(3);
    expect(deps.sleep).toHaveBeenCalledTimes(2);
    expect(result.final_pdf_s3_key).toBe('final/r.docx.pdf');
  });

  test('throws after 3 failed attempts so message redelivers', async () => {
    fetchMock.mockResolvedValue(new Response('down', { status: 502 }));
    const deps = makeDeps();

    await expect(runPdfJob(makeInput(), deps)).rejects.toThrow(/gotenberg.*502/);
    expect(fetchMock).toHaveBeenCalledTimes(3);
    expect(deps.putObject).not.toHaveBeenCalled();
  });

  test('does not retry on 4xx', async () => {
    fetchMock.mockResolvedValueOnce(new Response('bad', { status: 400 }));
    const deps = makeDeps();

    await expect(runPdfJob(makeInput(), deps)).rejects.toThrow(/gotenberg.*400/);
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });
});
