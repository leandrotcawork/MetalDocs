import { describe, it, expect, beforeAll, afterAll, vi } from 'vitest';
import type { FastifyInstance } from 'fastify';
import { buildApp } from '../src/index';
import { buildMinimalDocxFixture, buildMinimalSchemaFixture } from './fixtures';

const TOKEN = 'test-token-0123456789';
const FIELDS = ['client_name', 'total_amount'];

// vi.hoisted runs before the mock factory is evaluated
const { uploadedBuffers } = vi.hoisted(() => ({
  uploadedBuffers: new Map<string, Buffer>(),
}));

let docxBuf: Buffer;
let schemaBuf: Buffer;

vi.mock('../src/s3', () => ({
  makeS3Client: () => ({} as never),
  getObjectBuffer: async (_c: never, _b: string, key: string): Promise<Buffer> => {
    if (uploadedBuffers.has(key)) return uploadedBuffers.get(key)!;
    if (key.endsWith('.docx')) return docxBuf;
    return schemaBuf;
  },
  putObjectBuffer: async (_c: never, _b: string, key: string, data: Buffer): Promise<void> => {
    uploadedBuffers.set(key, data);
  },
}));

let app: FastifyInstance;

beforeAll(async () => {
  docxBuf = await buildMinimalDocxFixture([
    { text: 'Hello {client_name}, your total is {total_amount}.' },
  ]);
  schemaBuf = Buffer.from(JSON.stringify(buildMinimalSchemaFixture(FIELDS)));

  process.env.DOCGEN_V2_SERVICE_TOKEN = TOKEN;
  process.env.DOCGEN_V2_S3_ACCESS_KEY = 'key';
  process.env.DOCGEN_V2_S3_SECRET_KEY = 'sec';
  app = await buildApp();
});

afterAll(async () => {
  await app.close();
});

describe('POST /render/docx', () => {
  it('returns 200 with output_key and content_hash for valid input', async () => {
    const res = await app.inject({
      method: 'POST',
      url: '/render/docx',
      headers: { 'x-service-token': TOKEN, 'content-type': 'application/json' },
      payload: {
        template_docx_key: 'templates/v1.docx',
        schema_key: 'schemas/v1.schema.json',
        form_data: { client_name: 'Acme Corp', total_amount: '1500.00' },
        output_key: 'output/doc-001.docx',
      },
    });

    expect(res.statusCode).toBe(200);
    const body = res.json();
    expect(body.output_key).toBe('output/doc-001.docx');
    expect(typeof body.content_hash).toBe('string');
    expect(body.content_hash).toHaveLength(64); // sha256 hex
    expect(typeof body.size_bytes).toBe('number');
    expect(body.size_bytes).toBeGreaterThan(0);
    expect(Array.isArray(body.warnings)).toBe(true);
    expect(Array.isArray(body.unreplaced_vars)).toBe(true);
  });

  it('returns 400 when required fields are missing', async () => {
    const res = await app.inject({
      method: 'POST',
      url: '/render/docx',
      headers: { 'x-service-token': TOKEN, 'content-type': 'application/json' },
      payload: { template_docx_key: 'templates/v1.docx' },
    });
    expect(res.statusCode).toBe(400);
    const body = res.json();
    expect(body.error).toBe('bad_request');
  });

  it('returns 422 when form_data fails schema validation', async () => {
    const res = await app.inject({
      method: 'POST',
      url: '/render/docx',
      headers: { 'x-service-token': TOKEN, 'content-type': 'application/json' },
      payload: {
        template_docx_key: 'templates/v1.docx',
        schema_key: 'schemas/v1.schema.json',
        form_data: { total_amount: '999' }, // missing required client_name
        output_key: 'output/doc-002.docx',
      },
    });
    expect(res.statusCode).toBe(422);
    const body = res.json();
    expect(body.error).toBe('form_data_invalid');
  });

  it('returns 401 when X-Service-Token is missing', async () => {
    const res = await app.inject({
      method: 'POST',
      url: '/render/docx',
      payload: {
        template_docx_key: 'templates/v1.docx',
        schema_key: 'schemas/v1.schema.json',
        form_data: { client_name: 'X', total_amount: '0' },
        output_key: 'output/doc-003.docx',
      },
    });
    expect(res.statusCode).toBe(401);
  });

  it('returns 422 when schema_key resolves to invalid JSON', async () => {
    const badSchemaKey = '__bad_schema__.json';
    uploadedBuffers.set(badSchemaKey, Buffer.from('not json at all'));

    const res = await app.inject({
      method: 'POST',
      url: '/render/docx',
      headers: { 'x-service-token': TOKEN, 'content-type': 'application/json' },
      payload: {
        template_docx_key: 'templates/v1.docx',
        schema_key: badSchemaKey,
        form_data: { client_name: 'X', total_amount: '0' },
        output_key: 'output/doc-004.docx',
      },
    });
    expect(res.statusCode).toBe(422);
    const body = res.json();
    expect(body.error).toBe('schema_invalid');
  });
});
