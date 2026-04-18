import { describe, it, expect, beforeAll, afterAll } from 'vitest';
import { buildApp } from '../src/index';
import type { FastifyInstance } from 'fastify';

let app: FastifyInstance;

beforeAll(async () => {
  process.env.DOCGEN_V2_SERVICE_TOKEN = 'test-token-0123456789';
  process.env.DOCGEN_V2_PORT = '0';
  process.env.DOCGEN_V2_S3_ACCESS_KEY = 'minioadmin';
  process.env.DOCGEN_V2_S3_SECRET_KEY = 'minioadmin';
  app = await buildApp();
});

afterAll(async () => {
  await app.close();
});

describe('GET /health', () => {
  it('returns 200 and version/status payload', async () => {
    const res = await app.inject({ method: 'GET', url: '/health' });
    expect(res.statusCode).toBe(200);
    const body = res.json();
    expect(body.status).toBe('ok');
    expect(typeof body.version).toBe('string');
  });

  it('does NOT require X-Service-Token on /health', async () => {
    const res = await app.inject({ method: 'GET', url: '/health' });
    expect(res.statusCode).toBe(200);
  });

  it('rejects requests to other paths without X-Service-Token', async () => {
    const res = await app.inject({ method: 'POST', url: '/render/docx', payload: {} });
    expect(res.statusCode).toBe(401);
  });
});