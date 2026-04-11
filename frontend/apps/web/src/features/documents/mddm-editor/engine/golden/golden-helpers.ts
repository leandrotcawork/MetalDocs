// Helpers for golden file tests. Normalizes DOCX document.xml and HTML
// output so tests compare semantics instead of engine-specific metadata.

const STRIP_ATTRIBUTES = new Set([
  "w:rsidR", "w:rsidRDefault", "w:rsidP", "w:rsidRPr",
  "w:paraId", "w:textId", "w:rsidTr",
]);

function stripRSIDs(xml: string): string {
  let result = xml;
  for (const attr of STRIP_ATTRIBUTES) {
    const re = new RegExp(`\\s${attr}="[^"]*"`, "g");
    result = result.replace(re, "");
  }
  return result;
}

function collapseWhitespace(xml: string): string {
  return xml.replace(/>\s+</g, "><").trim();
}

export function normalizeDocxXml(xml: string): string {
  return collapseWhitespace(stripRSIDs(xml));
}

export function normalizeHtml(html: string): string {
  return collapseWhitespace(
    html
      .replace(/<!--[\s\S]*?-->/g, "")
      .replace(/\s(data-reactroot|data-bn-key)(?:="[^"]*"|='[^']*'|(?=[\s>]))/g, ""),
  );
}

async function blobToArrayBuffer(blob: Blob): Promise<ArrayBuffer> {
  // blob.arrayBuffer() is unavailable in jsdom. Use FileReader as a
  // cross-environment fallback that works in both browser and jsdom.
  if (typeof blob.arrayBuffer === "function") {
    return blob.arrayBuffer();
  }
  return new Promise<ArrayBuffer>((resolve, reject) => {
    const reader = new FileReader();
    reader.onload = () => resolve(reader.result as ArrayBuffer);
    reader.onerror = () => reject(reader.error);
    reader.readAsArrayBuffer(blob);
  });
}

export async function unzipDocxDocumentXml(blob: Blob): Promise<string> {
  const JSZip = (await import("jszip")).default;
  const zip = await JSZip.loadAsync(await blobToArrayBuffer(blob));
  const documentXml = zip.file("word/document.xml");
  if (!documentXml) {
    throw new Error("word/document.xml not found in DOCX blob");
  }
  return await documentXml.async("string");
}
