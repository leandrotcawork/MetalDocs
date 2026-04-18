import JSZip from 'jszip';

export async function buildMinimalDocxFixture(
  paragraphs: Array<{ text: string }>,
): Promise<Buffer> {
  const zip = new JSZip();
  const paras = paragraphs
    .map((p) => `<w:p><w:r><w:t>${p.text}</w:t></w:r></w:p>`)
    .join('\n');
  const docXml = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
  <w:body>${paras}<w:sectPr/></w:body>
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

export function buildMinimalSchemaFixture(fields: string[]): object {
  const props: Record<string, object> = {};
  for (const f of fields) props[f] = { type: 'string' };
  return {
    $schema: 'http://json-schema.org/draft-07/schema#',
    type: 'object',
    required: fields,
    properties: props,
  };
}
