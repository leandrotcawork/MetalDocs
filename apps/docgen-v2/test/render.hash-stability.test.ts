import { describe, it, expect, beforeAll, afterAll, vi } from 'vitest';
import type { FastifyInstance } from 'fastify';
import { buildApp } from '../src/index';
import { buildMinimalDocxFixture, buildMinimalSchemaFixture } from './fixtures';

const TOKEN = 'test-token-0123456789';

let docxBuf: Buffer;
let schemaBuf: Buffer;

vi.mock('../src/s3', async () => {
  return {
    makeS3Client: () => ({} as never),
    getObjectBuffer: async (_c: never, _b: string, key: string): Promise<Buffer> => {
      if (key.endsWith('.docx')) return docxBuf;
      return schemaBuf;
    },
    putObjectBuffer: async (): Promise<void> => {
      // no-op
    },
  };
});

let app: FastifyInstance;

beforeAll(async () => {
  docxBuf = await buildMinimalDocxFixture([
    { text: 'Invoice for {client_name} — amount {total_amount}' },
  ]);
  schemaBuf = Buffer.from(
    JSON.stringify(buildMinimalSchemaFixture(['client_name', 'total_amount'])),
  );

  process.env.DOCGEN_V2_SERVICE_TOKEN = TOKEN;
  process.env.DOCGEN_V2_S3_ACCESS_KEY = 'key';
  process.env.DOCGEN_V2_S3_SECRET_KEY = 'sec';
  app = await buildApp();
});

afterAll(async () => {
  await app.close();
});

const PAYLOAD = {
  template_docx_key: 'templates/stable.docx',
  schema_key: 'schemas/stable.schema.json',
  form_data: { client_name: 'Acme', total_amount: '42.00' },
  output_key: 'output/stable-001.docx',
};

describe('POST /render/docx — hash stability', () => {
  it('produces identical content_hash for two identical renders', async () => {
    const headers = { 'x-service-token': TOKEN, 'content-type': 'application/json' };

    const [res1, res2] = await Promise.all([
      app.inject({ method: 'POST', url: '/render/docx', headers, payload: PAYLOAD }),
      app.inject({ method: 'POST', url: '/render/docx', headers, payload: PAYLOAD }),
    ]);

    expect(res1.statusCode).toBe(200);
    expect(res2.statusCode).toBe(200);
    expect(res1.json().content_hash).toBe(res2.json().content_hash);
    expect(res1.json().size_bytes).toBe(res2.json().size_bytes);
  });

  it('produces different content_hash when form_data changes', async () => {
    const headers = { 'x-service-token': TOKEN, 'content-type': 'application/json' };

    const res1 = await app.inject({
      method: 'POST',
      url: '/render/docx',
      headers,
      payload: PAYLOAD,
    });
    const res2 = await app.inject({
      method: 'POST',
      url: '/render/docx',
      headers,
      payload: {
        ...PAYLOAD,
        form_data: { client_name: 'Different Corp', total_amount: '99.99' },
        output_key: 'output/stable-002.docx',
      },
    });

    expect(res1.statusCode).toBe(200);
    expect(res2.statusCode).toBe(200);
    expect(res1.json().content_hash).not.toBe(res2.json().content_hash);
  });
});
