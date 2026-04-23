import { describe, expect, test } from 'vitest';
import JSZip from 'jszip';
import { injectZones } from '../zoneInjection.js';

async function buildDocxWithBookmarkPair(zoneId: string): Promise<Uint8Array> {
  const zip = new JSZip();
  const body = `<w:p><w:bookmarkStart w:id="1" w:name="zone-start:${zoneId}"/><w:bookmarkEnd w:id="1" w:name="zone-end:${zoneId}"/></w:p>`;
  const docXml = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>${body}<w:sectPr/></w:body>
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

  return zip.generateAsync({ type: 'uint8array' });
}

async function extractDocXml(docx: Uint8Array): Promise<string> {
  const zip = await JSZip.loadAsync(docx);
  const file = zip.file('word/document.xml');
  if (!file) throw new Error('missing document.xml');
  return file.async('string');
}

describe('injectZones', () => {
  test('inserts OOXML immediately after matching bookmarkStart', async () => {
    const docx = await buildDocxWithBookmarkPair('intro');
    const out = await injectZones(docx, { intro: '<w:p><w:r><w:t>X</w:t></w:r></w:p>' });
    const xml = await extractDocXml(out);

    expect(xml).toContain('<w:p><w:r><w:t>X</w:t></w:r></w:p>');
    expect(xml).toMatch(
      /zone-start:intro"\/><w:p><w:r><w:t>X<\/w:t><\/w:r><\/w:p>/,
    );
  });

  test('no-op when zones map empty', async () => {
    const docx = await buildDocxWithBookmarkPair('intro');
    const out = await injectZones(docx, {});
    expect(out).toBe(docx);
  });

  test('injects only matching bookmark, ignores unmatched zones', async () => {
    const docx = await buildDocxWithBookmarkPair('intro');
    const out = await injectZones(docx, {
      intro: '<w:p>IN</w:p>',
      unknown: '<w:p>NO</w:p>',
    });
    const xml = await extractDocXml(out);

    expect(xml).toContain('<w:p>IN</w:p>');
    expect(xml).not.toContain('<w:p>NO</w:p>');
  });

  test('throws when document.xml missing', async () => {
    const zip = new JSZip();
    zip.file('dummy', 'x');
    const bad = await zip.generateAsync({ type: 'uint8array' });
    await expect(injectZones(bad, { intro: 'x' })).rejects.toThrow(
      /malformed DOCX/,
    );
  });
});
