import { z } from 'zod';
const EnvSchema = z.object({
    DOCGEN_V2_PORT: z.coerce.number().int().min(0).max(65535).default(3100),
    DOCGEN_V2_SERVICE_TOKEN: z.string().min(16, 'service token must be >= 16 chars'),
    DOCGEN_V2_LOG_LEVEL: z.enum(['fatal', 'error', 'warn', 'info', 'debug', 'trace']).default('info'),
    DOCGEN_V2_VERSION: z.string().default('0.0.0-dev'),
});
export function loadEnv() {
    const parsed = EnvSchema.safeParse(process.env);
    if (!parsed.success) {
        const flat = parsed.error.flatten().fieldErrors;
        throw new Error(`invalid env: ${JSON.stringify(flat)}`);
    }
    return parsed.data;
}
