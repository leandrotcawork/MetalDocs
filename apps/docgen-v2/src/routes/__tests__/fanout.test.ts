import { describe, expect, test, beforeAll, afterAll, vi } from 'vitest';
import type { FastifyInstance } from 'fastify';
import JSZip from 'jszip';

const TOKEN = 'test-token-fanout';

let uploadedKey: string | null = null;
let uploadedSize = 0;

async function buildTemplateDocx(bodyInnerXml: string): Promise<Buffer> {
  const zip = new JSZip();
  const docXml = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>${bodyInnerXml}<w:sectPr/></w:body>
</w:document>`;
  const contentTypes = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
  <Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
  <Default Extension="xml" ContentType="application/xml"/>
  <Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`;
  const rels = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
  <Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`;
  const wordRels = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"></Relationships>`;

  zip.file('[Content_Types].xml', contentTypes);
  zip.file('_rels/.rels', rels);
  zip.file('word/document.xml', docXml);
  zip.file('word/_rels/document.xml.rels', wordRels);

  return zip.generateAsync({ type: 'nodebuffer' }) as Promise<Buffer>;
}

let templateBuf: Buffer;

vi.mock('../../s3', async () => {
  return {
    makeS3Client: () => ({}) as never,
    getObjectBuffer: async (_c: never, _b: string, _key: string): Promise<Buffer> =>
      templateBuf,
    putObjectBuffer: async (
      _c: never,
      _b: string,
      key: string,
      data: Buffer,
    ): Promise<void> => {
      uploadedKey = key;
      uploadedSize = data.byteLength;
    },
  };
});

let app: FastifyInstance;

beforeAll(async () => {
  templateBuf = await buildTemplateDocx(
    `<w:p><w:r><w:t>{doc_code}</w:t></w:r></w:p>`,
  );
  process.env.DOCGEN_V2_SERVICE_TOKEN = TOKEN;
  process.env.DOCGEN_V2_S3_ACCESS_KEY = 'key';
  process.env.DOCGEN_V2_S3_SECRET_KEY = 'sec';
  const { buildApp } = await import('../../index.js');
  app = await buildApp();
});

afterAll(async () => {
  if (app) await app.close();
});

describe('POST /render/fanout', () => {
  test('returns content_hash + final_docx_s3_key + unreplaced_vars', async () => {
    const headers = { 'x-service-token': TOKEN, 'content-type': 'application/json' };
    const res = await app.inject({
      method: 'POST',
      url: '/render/fanout',
      headers,
      payload: {
        body_docx_key: 'templates/x.docx',
        placeholder_values: { doc_code: 'ABC-001' },
        composition_config: {
          header_sub_blocks: [],
          footer_sub_blocks: [],
          sub_block_params: {},
        },
        resolved_values: {},
        output_key: 'output/fanout-001.docx',
      },
    });

    expect(res.statusCode).toBe(200);
    const body = res.json();
    expect(body.content_hash).toMatch(/^[0-9a-f]{64}$/);
    expect(body.final_docx_s3_key).toBe('output/fanout-001.docx');
    expect(Array.isArray(body.unreplaced_vars)).toBe(true);
    expect(uploadedKey).toBe('output/fanout-001.docx');
    expect(uploadedSize).toBeGreaterThan(0);
  });

  test('returns 400 on malformed body', async () => {
    const headers = { 'x-service-token': TOKEN, 'content-type': 'application/json' };
    const res = await app.inject({
      method: 'POST',
      url: '/render/fanout',
      headers,
      payload: { body_docx_key: '' },
    });
    expect(res.statusCode).toBe(400);
    expect(res.json().error).toBe('bad_request');
  });
});
