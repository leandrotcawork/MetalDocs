import type { FastifyInstance } from 'fastify';
import { z } from 'zod';
import type { Client } from 'minio';
import type { Env } from '../env.js';
import { getObjectBuffer, putObjectBuffer } from '../s3.js';
import { fanout } from '../render/fanout.js';

const BodySchema = z.object({
  body_docx_s3_key: z.string().min(1),
  tenant_id: z.string().min(1),
  revision_id: z.string().min(1),
  placeholder_values: z.record(z.string()),
  composition_config: z.object({
    header_sub_blocks: z.array(z.string()),
    footer_sub_blocks: z.array(z.string()),
    sub_block_params: z.record(z.record(z.unknown())),
  }),
  resolved_values: z.record(z.unknown()),
});

const DOCX_MIME =
  'application/vnd.openxmlformats-officedocument.wordprocessingml.document';

export function registerFanoutRoute(
  app: FastifyInstance,
  env: Env,
  s3Factory: () => Client,
): void {
  app.post('/render/fanout', async (req, reply) => {
    const parsed = BodySchema.safeParse(req.body);
    if (!parsed.success) {
      return reply
        .code(400)
        .send({ error: 'bad_request', details: parsed.error.format() });
    }
    const {
      body_docx_s3_key,
      tenant_id,
      revision_id,
      placeholder_values,
      composition_config,
      resolved_values,
    } = parsed.data;

    const output_key = `tenants/${tenant_id}/revisions/${revision_id}/frozen.docx`;

    const client = s3Factory();
    const bodyBuf = await getObjectBuffer(
      client,
      env.DOCGEN_V2_S3_BUCKET,
      body_docx_s3_key,
    );

    const result = await fanout({
      bodyDocx: new Uint8Array(bodyBuf),
      placeholderValues: placeholder_values,
      compositionConfig: composition_config,
      resolvedValues: resolved_values,
    });

    await putObjectBuffer(
      client,
      env.DOCGEN_V2_S3_BUCKET,
      output_key,
      Buffer.from(result.buffer),
      DOCX_MIME,
    );

    return reply.code(200).send({
      content_hash: result.contentHash,
      final_docx_s3_key: output_key,
      unreplaced_vars: result.unreplacedVars,
      size_bytes: result.buffer.byteLength,
    });
  });
}
