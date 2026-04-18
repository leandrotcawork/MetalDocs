import type { FastifyInstance } from 'fastify';
import Ajv from 'ajv';
import { z } from 'zod';
import { parseDocxTokens, diffTokensVsSchema } from '@metaldocs/shared-tokens';
import type { Env } from '../env.js';
import { getObjectBuffer } from '../s3.js';
import type { Client } from 'minio';

const BodySchema = z.object({
  docx_key: z.string().min(1),
  schema_key: z.string().min(1),
});

export function registerValidateTemplate(
  app: FastifyInstance,
  env: Env,
  s3Factory: () => Client,
): void {
  app.post('/validate/template', async (req, reply) => {
    const parsed = BodySchema.safeParse(req.body);
    if (!parsed.success) {
      reply.code(400).send({ error: 'invalid_body', details: parsed.error.flatten() });
      return;
    }
    const { docx_key, schema_key } = parsed.data;

    const client = s3Factory();
    const [docxBuf, schemaBuf] = await Promise.all([
      getObjectBuffer(client, env.DOCGEN_V2_S3_BUCKET, docx_key),
      getObjectBuffer(client, env.DOCGEN_V2_S3_BUCKET, schema_key),
    ]);

    let schema: unknown;
    try { schema = JSON.parse(schemaBuf.toString('utf8')); }
    catch (e) {
      reply.code(422).send({
        valid: false,
        parse_errors: [{ type: 'malformed_schema', raw: String((e as Error).message) }],
        missing_tokens: [], orphan_tokens: []
      });
      return;
    }

    const ajv = new Ajv({ allErrors: true, strict: false });
    try { ajv.compile(schema as object); }
    catch (e) {
      reply.code(422).send({
        valid: false,
        parse_errors: [{ type: 'schema_invalid', raw: String((e as Error).message) }],
        missing_tokens: [], orphan_tokens: []
      });
      return;
    }

    const parse = await parseDocxTokens(docxBuf.buffer.slice(docxBuf.byteOffset, docxBuf.byteOffset + docxBuf.byteLength));
    const diff = diffTokensVsSchema(parse.tokens, schema);
    const valid = parse.errors.length === 0 && diff.missing.length === 0 && diff.orphans.length === 0;

    reply.code(valid ? 200 : 422).send({
      valid,
      parse_errors: parse.errors,
      missing_tokens: diff.missing,
      orphan_tokens: diff.orphans,
    });
  });
}
