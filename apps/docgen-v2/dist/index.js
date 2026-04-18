import Fastify from 'fastify';
import { loadEnv } from './env.js';
import { registerServiceAuth } from './service-auth.js';
export async function buildApp() {
    const env = loadEnv();
    const app = Fastify({ logger: { level: env.DOCGEN_V2_LOG_LEVEL } });
    registerServiceAuth(app, env.DOCGEN_V2_SERVICE_TOKEN);
    app.get('/health', async () => ({ status: 'ok', version: env.DOCGEN_V2_VERSION }));
    return app;
}
if (import.meta.url === `file://${process.argv[1]}`) {
    const env = loadEnv();
    buildApp().then((app) => {
        app.listen({ port: env.DOCGEN_V2_PORT, host: '0.0.0.0' })
            .catch((err) => { app.log.fatal(err); process.exit(1); });
    });
}
