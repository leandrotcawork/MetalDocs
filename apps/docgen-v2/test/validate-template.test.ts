import { describe, it, expect, beforeAll, afterAll, vi } from 'vitest';
import type { FastifyInstance } from 'fastify';
import { buildApp } from '../src/index';

const TOKEN = 'test-token-0123456789';

vi.mock('../src/s3', async () => {
  const happySchema = JSON.stringify({
    type: 'object',
    properties: {
      client_name: { type: 'string' },
      total_amount: { type: 'number' }
    },
    required: ['client_name']
  });
  const fixtures = await import('../../../packages/shared-tokens/test/fixtures');
  const docxBuf = await fixtures.makeDocx(fixtures.HAPPY_DOC);
  return {
    makeS3Client: () => ({} as any),
    getObjectBuffer: async (_c: any, _b: string, key: string) =>
      key.endsWith('.docx') ? Buffer.from(docxBuf) : Buffer.from(happySchema),
  };
});

let app: FastifyInstance;

beforeAll(async () => {
  process.env.DOCGEN_V2_SERVICE_TOKEN = TOKEN;
  process.env.DOCGEN_V2_S3_ACCESS_KEY = 'key';
  process.env.DOCGEN_V2_S3_SECRET_KEY = 'sec';
  app = await buildApp();
});
afterAll(async () => { await app.close(); });

describe('POST /validate/template', () => {
  it('returns valid=true for happy docx + matching schema', async () => {
    const res = await app.inject({
      method: 'POST', url: '/validate/template',
      headers: { 'x-service-token': TOKEN, 'content-type': 'application/json' },
      payload: { docx_key: 't/v1.docx', schema_key: 't/v1.schema.json' },
    });
    expect(res.statusCode).toBe(200);
    const body = res.json();
    expect(body.valid).toBe(true);
    expect(body.parse_errors).toEqual([]);
    expect(body.missing_tokens).toEqual([]);
    expect(body.orphan_tokens).toEqual([]);
  });

  it('rejects when X-Service-Token missing', async () => {
    const res = await app.inject({
      method: 'POST', url: '/validate/template',
      payload: { docx_key: 'x', schema_key: 'y' },
    });
    expect(res.statusCode).toBe(401);
  });

  it('rejects malformed body (missing docx_key)', async () => {
    const res = await app.inject({
      method: 'POST', url: '/validate/template',
      headers: { 'x-service-token': TOKEN, 'content-type': 'application/json' },
      payload: { schema_key: 'y' },
    });
    expect(res.statusCode).toBe(400);
  });
});
