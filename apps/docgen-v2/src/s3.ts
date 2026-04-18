import { Client } from 'minio';
import type { Env } from './env';

export function makeS3Client(env: Env): Client {
  const url = new URL(env.DOCGEN_V2_S3_ENDPOINT);
  return new Client({
    endPoint: url.hostname,
    port: Number(url.port || (env.DOCGEN_V2_S3_USE_SSL ? 443 : 80)),
    useSSL: env.DOCGEN_V2_S3_USE_SSL,
    accessKey: env.DOCGEN_V2_S3_ACCESS_KEY,
    secretKey: env.DOCGEN_V2_S3_SECRET_KEY,
  });
}

export async function getObjectBuffer(client: Client, bucket: string, key: string): Promise<Buffer> {
  const stream = await client.getObject(bucket, key);
  const chunks: Buffer[] = [];
  for await (const c of stream) chunks.push(Buffer.isBuffer(c) ? c : Buffer.from(c as Uint8Array));
  return Buffer.concat(chunks);
}
