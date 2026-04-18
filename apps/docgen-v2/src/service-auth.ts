import { timingSafeEqual } from 'node:crypto';
import type { FastifyInstance } from 'fastify';

function safeCompare(a: string, b: string): boolean {
  if (a.length !== b.length) return false;
  return timingSafeEqual(Buffer.from(a), Buffer.from(b));
}

export function registerServiceAuth(app: FastifyInstance, token: string): void {
  app.addHook('onRequest', async (req, reply) => {
    if (req.url === '/health') return;
    const header = req.headers['x-service-token'];
    if (typeof header !== 'string' || !safeCompare(header, token)) {
      return reply.code(401).send({ error: 'unauthorized' });
    }
  });
}