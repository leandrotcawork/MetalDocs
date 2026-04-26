import { fileURLToPath } from 'node:url';
import { resolve } from 'node:path';
import Fastify, { type FastifyInstance } from 'fastify';
import { loadEnv } from './env.js';
import { registerServiceAuth } from './service-auth.js';
import { registerRoutes } from './routes/index.js';
import { makeS3Client } from './s3.js';

export async function buildApp(): Promise<FastifyInstance> {
  const env = loadEnv();
  const app = Fastify({ logger: { level: env.LOG_LEVEL } });

  registerServiceAuth(app, env.DOCGEN_V2_SERVICE_TOKEN);

  app.get('/health', async () => ({ status: 'ok', version: env.VERSION }));

  let cachedClient: ReturnType<typeof makeS3Client> | null = null;
  const s3Factory = () => (cachedClient ??= makeS3Client(env));

  registerRoutes(app, env, s3Factory);
  return app;
}

if (resolve(fileURLToPath(import.meta.url)) === resolve(process.argv[1])) {
  const env = loadEnv();
  buildApp().then((app) => {
    app.listen({ port: env.DOCGEN_V2_PORT, host: '0.0.0.0' })
       .catch((err) => { app.log.fatal(err); process.exit(1); });
  });
}