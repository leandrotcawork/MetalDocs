import { z } from 'zod';

const EnvSchema = z.object({
  DOCGEN_V2_PORT: z.coerce.number().int().min(0).max(65535).default(3100),
  DOCGEN_V2_SERVICE_TOKEN: z.string().min(16, 'service token must be >= 16 chars'),
  LOG_LEVEL: z.enum(['fatal','error','warn','info','debug','trace']).default('info'),
  VERSION: z.string().default('dev'),
});

export type Env = z.infer<typeof EnvSchema>;

export function loadEnv(): Env {
  const parsed = EnvSchema.safeParse(process.env);
  if (!parsed.success) {
    const flat = parsed.error.flatten().fieldErrors;
    const safe = { ...flat, DOCGEN_V2_SERVICE_TOKEN: flat.DOCGEN_V2_SERVICE_TOKEN ? ['[redacted]'] : undefined };
    throw new Error(`invalid env: ${JSON.stringify(safe)}`);
  }
  return parsed.data;
}