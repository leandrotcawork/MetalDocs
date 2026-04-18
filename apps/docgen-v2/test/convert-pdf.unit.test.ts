import { afterAll, beforeAll, beforeEach, describe, expect, it, vi } from 'vitest';
import type { FastifyInstance } from 'fastify';
import { createHash } from 'node:crypto';
import { buildApp } from '../src/index';
import * as s3 from '../src/s3';

const TOKEN = 'test-token-0123456789';
const DOCX_BYTES = Buffer.from('fake-docx');
const PDF_BYTES = Buffer.from('%PDF-1.4\nfake-pdf');

vi.mock('../src/s3', () => {
  return {
    makeS3Client: () => ({} as any),
    getObjectBuffer: vi.fn(async () => DOCX_BYTES),
    putObjectBuffer: vi.fn(async () => {}),
  };
});

let app: FastifyInstance;

beforeAll(async () => {
  process.env.DOCGEN_V2_SERVICE_TOKEN = TOKEN;
  process.env.DOCGEN_V2_S3_ACCESS_KEY = 'key';
  process.env.DOCGEN_V2_S3_SECRET_KEY = 'sec';
  app = await buildApp();
});

afterAll(async () => {
  await app.close();
  vi.unstubAllGlobals();
});

beforeEach(() => {
  vi.mocked(s3.getObjectBuffer).mockResolvedValue(Buffer.from(DOCX_BYTES));
  vi.mocked(s3.putObjectBuffer).mockResolvedValue();
  vi.unstubAllGlobals();
});

describe('POST /convert/pdf', () => {
  it('returns 401 without token', async () => {
    const res = await app.inject({
      method: 'POST',
      url: '/convert/pdf',
      payload: { docx_key: 'in.docx', output_key: 'out.pdf', render_opts: {} },
    });

    expect(res.statusCode).toBe(401);
  });

  it('returns 400 for missing fields', async () => {
    const res = await app.inject({
      method: 'POST',
      url: '/convert/pdf',
      headers: { 'x-service-token': TOKEN, 'content-type': 'application/json' },
      payload: { docx_key: 'in.docx' },
    });

    expect(res.statusCode).toBe(400);
  });

  it('returns 502 when gotenberg is unreachable', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => {
      throw new Error('network down');
    }));

    const res = await app.inject({
      method: 'POST',
      url: '/convert/pdf',
      headers: { 'x-service-token': TOKEN, 'content-type': 'application/json' },
      payload: {
        docx_key: 'in.docx',
        output_key: 'out.pdf',
        render_opts: { paper: 'A4', landscape: false },
      },
    });

    expect(res.statusCode).toBe(502);
    expect(res.json()).toEqual({ error: 'gotenberg_failed' });
  });

  it('returns 502 when gotenberg responds non-2xx', async () => {
    vi.stubGlobal('fetch', vi.fn(async () => new Response('oops', { status: 500 })));

    const res = await app.inject({
      method: 'POST',
      url: '/convert/pdf',
      headers: { 'x-service-token': TOKEN, 'content-type': 'application/json' },
      payload: {
        docx_key: 'in.docx',
        output_key: 'out.pdf',
        render_opts: { paper: 'A4', landscape: false },
      },
    });

    expect(res.statusCode).toBe(502);
    expect(res.json()).toEqual({ error: 'gotenberg_failed' });
  });

  it('returns 200 with content hash for successful conversion', async () => {
    const fetchMock = vi.fn(async () => new Response(PDF_BYTES, { status: 200 }));
    vi.stubGlobal('fetch', fetchMock);

    const res = await app.inject({
      method: 'POST',
      url: '/convert/pdf',
      headers: { 'x-service-token': TOKEN, 'content-type': 'application/json' },
      payload: {
        docx_key: 'in.docx',
        output_key: 'out.pdf',
        render_opts: { paper: 'Letter', landscape: true },
      },
    });

    const expectedHash = createHash('sha256').update(PDF_BYTES).digest('hex');

    expect(res.statusCode).toBe(200);
    expect(res.json()).toEqual({
      output_key: 'out.pdf',
      content_hash: expectedHash,
      size_bytes: PDF_BYTES.byteLength,
      docgen_v2_version: 'docgen-v2@0.4.0',
    });

    expect(fetchMock).toHaveBeenCalledTimes(1);
    expect(s3.putObjectBuffer).toHaveBeenCalledTimes(1);
  });
});
