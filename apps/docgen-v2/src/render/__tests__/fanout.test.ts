import { describe, expect, test } from 'vitest';
import JSZip from 'jszip';
import { fanout } from '../fanout.js';

async function buildTemplateDocx(bodyInnerXml: string): Promise<Uint8Array> {
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

  const buf = await zip.generateAsync({ type: 'uint8array' });
  return buf;
}

async function extractDocumentXml(docx: Uint8Array): Promise<string> {
  const zip = await JSZip.loadAsync(docx);
  const file = zip.file('word/document.xml');
  if (!file) throw new Error('missing document.xml');
  return file.async('string');
}

describe('fanout', () => {
  test('returns buffer with stable sha256 contentHash', async () => {
    const body = `<w:p><w:r><w:t>Doc code: {doc_code}</w:t></w:r></w:p>`;
    const tpl = await buildTemplateDocx(body);

    const result = await fanout({
      bodyDocx: tpl,
      placeholderValues: { doc_code: 'ABC-001' },

      compositionConfig: {
        header_sub_blocks: [],
        footer_sub_blocks: [],
        sub_block_params: {},
      },
      resolvedValues: {},
    });

    expect(result.contentHash).toMatch(/^[0-9a-f]{64}$/);
    expect(result.buffer.byteLength).toBeGreaterThan(0);

    const xml = await extractDocumentXml(result.buffer);
    expect(xml).toContain('ABC-001');
  });

  test('renders header sub-blocks via resolvedValues', async () => {
    const body = `<w:p><w:r><w:t>{doc_code}</w:t></w:r></w:p>`;
    const tpl = await buildTemplateDocx(body);

    const result = await fanout({
      bodyDocx: tpl,
      placeholderValues: { doc_code: 'ABC-001' },

      compositionConfig: {
        header_sub_blocks: ['doc_header_standard'],
        footer_sub_blocks: [],
        sub_block_params: {},
      },
      resolvedValues: {
        title: 'Test Doc',
        doc_code: 'ABC-001',
        effective_date: '2026-04-23',
        revision_number: '1',
      },
    });

    expect(result.contentHash).toMatch(/^[0-9a-f]{64}$/);
    expect(result.buffer.byteLength).toBeGreaterThan(0);
  });

  test('same inputs produce identical contentHash', async () => {
    const body = `<w:p><w:r><w:t>{doc_code}</w:t></w:r></w:p>`;
    const tpl = await buildTemplateDocx(body);
    const input = {
      bodyDocx: tpl,
      placeholderValues: { doc_code: 'STABLE-1' },

      compositionConfig: {
        header_sub_blocks: [],
        footer_sub_blocks: [],
        sub_block_params: {},
      },
      resolvedValues: {},
    };

    const r1 = await fanout(input);
    const r2 = await fanout(input);
    expect(r1.contentHash).toBe(r2.contentHash);
  });
});
