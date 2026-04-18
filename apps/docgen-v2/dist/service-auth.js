export function registerServiceAuth(app, token) {
    app.addHook('onRequest', async (req, reply) => {
        if (req.url === '/health')
            return;
        const header = req.headers['x-service-token'];
        if (typeof header !== 'string' || header !== token) {
            reply.code(401).send({ error: 'unauthorized' });
        }
    });
}
