import { createHash } from 'node:crypto';
import type { FastifyInstance } from 'fastify';
import type { Client } from 'minio';
import { z } from 'zod';
import type { Env } from '../env.js';
import { DOCGEN_V2_VERSION } from '../pdf/version.js';
import { getObjectBuffer, putObjectBuffer } from '../s3.js';

const BodySchema = z.object({
  docx_key: z.string().min(1),
  output_key: z.string().min(1),
  render_opts: z.object({
    paper: z.enum(['A4', 'Letter']).optional(),
    landscape: z.boolean().optional(),
  }).optional(),
});

const PAPER_SIZES: Record<'A4' | 'Letter', { width: number; height: number }> = {
  A4: { width: 8.27, height: 11.69 },
  Letter: { width: 8.5, height: 11.0 },
};

export function registerConvertPDF(
  app: FastifyInstance,
  env: Env,
  s3Factory: () => Client,
): void {
  app.post('/convert/pdf', async (req, reply) => {
    const token = req.headers['x-service-token'];
    if (typeof token !== 'string' || token !== env.DOCGEN_V2_SERVICE_TOKEN) {
      return reply.code(401).send({ error: 'unauthorized' });
    }

    const parsed = BodySchema.safeParse(req.body);
    if (!parsed.success) {
      return reply.code(400).send({ error: 'bad_request', details: parsed.error.format() });
    }

    const client = s3Factory();
    const { docx_key, output_key, render_opts } = parsed.data;

    const docxBuffer = await getObjectBuffer(client, env.DOCGEN_V2_S3_BUCKET, docx_key);
    const paper = render_opts?.paper ?? 'A4';
    const landscape = render_opts?.landscape ?? false;
    const dims = PAPER_SIZES[paper];

    const form = new FormData();
    form.append(
      'files',
      new Blob([docxBuffer], {
        type: 'application/vnd.openxmlformats-officedocument.wordprocessingml.document',
      }),
      'document.docx',
    );
    form.append('paperWidth', String(dims.width));
    form.append('paperHeight', String(dims.height));
    form.append('landscape', landscape ? 'true' : 'false');

    const gotenbergUrl = `${env.DOCGEN_V2_GOTENBERG_URL.replace(/\/+$/, '')}/forms/libreoffice/convert`;

    let gotenbergRes: Response;
    try {
      gotenbergRes = await fetch(gotenbergUrl, { method: 'POST', body: form });
    } catch {
      return reply.code(502).send({ error: 'gotenberg_failed' });
    }

    if (!gotenbergRes.ok) {
      return reply.code(502).send({ error: 'gotenberg_failed' });
    }

    const pdfBuffer = Buffer.from(await gotenbergRes.arrayBuffer());
    const contentHash = createHash('sha256').update(pdfBuffer).digest('hex');

    await putObjectBuffer(
      client,
      env.DOCGEN_V2_S3_BUCKET,
      output_key,
      pdfBuffer,
      'application/pdf',
    );

    return reply.code(200).send({
      output_key,
      content_hash: contentHash,
      size_bytes: pdfBuffer.byteLength,
      docgen_v2_version: DOCGEN_V2_VERSION,
    });
  });
}
