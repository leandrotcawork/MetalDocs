import { describe, it, expect } from 'vitest';
import { makeS3Client } from '../src/s3';

describe('makeS3Client', () => {
  it('parses endpoint URL into host/port', () => {
    const c = makeS3Client({
      DOCGEN_V2_PORT: 0,
      DOCGEN_V2_SERVICE_TOKEN: 'test-token-0123456789',
      LOG_LEVEL: 'info',
      VERSION: 'dev',
      DOCGEN_V2_S3_ENDPOINT: 'http://minio:9000',
      DOCGEN_V2_S3_ACCESS_KEY: 'k',
      DOCGEN_V2_S3_SECRET_KEY: 's',
      DOCGEN_V2_S3_BUCKET: 'b',
      DOCGEN_V2_S3_USE_SSL: false,
    });
    expect(c).toBeDefined();
  });
});
