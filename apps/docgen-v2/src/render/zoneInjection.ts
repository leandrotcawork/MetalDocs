import JSZip from 'jszip';

export async function injectZones(
  docx: Uint8Array,
  zones: Record<string, string>,
): Promise<Uint8Array> {
  if (Object.keys(zones).length === 0) return docx;

  const zip = await JSZip.loadAsync(docx);
  const file = zip.file('word/document.xml');
  if (!file) throw new Error('malformed DOCX: missing word/document.xml');

  let xml = await file.async('string');
  for (const [zoneId, ooxml] of Object.entries(zones)) {
    const marker = new RegExp(
      `(<w:bookmarkStart[^/]*w:name="zone-start:${escapeRegex(zoneId)}"[^/]*/>)`,
    );
    xml = xml.replace(marker, `$1${ooxml}`);
  }
  zip.file('word/document.xml', xml);
  return zip.generateAsync({ type: 'uint8array' });
}

function escapeRegex(s: string): string {
  return s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&');
}
