export type ExportPDFResult = {
  storage_key: string;
  signed_url: string;
  composite_hash: string;
  size_bytes: number;
  cached: boolean;
  revision_id: string;
};

export type DocxURLResult = {
  signed_url: string;
  revision_id: string;
};

async function json<T>(res: Response): Promise<T> {
  if (!res.ok) {
    let body: unknown;
    try { body = await res.json(); } catch { body = await res.text(); }
    throw Object.assign(new Error(`http_${res.status}`), { status: res.status, body });
  }
  return res.json() as Promise<T>;
}

export async function exportPDF(
  documentID: string,
  opts: { paper_size?: 'A4' | 'Letter'; landscape?: boolean } = {},
): Promise<ExportPDFResult> {
  return json(await fetch(`/api/v2/documents/${documentID}/export/pdf`, {
    method: 'POST',
    headers: { 'content-type': 'application/json' },
    body: JSON.stringify(opts),
  }));
}

export async function getDocxSignedURL(documentID: string): Promise<DocxURLResult> {
  return json(await fetch(`/api/v2/documents/${documentID}/export/docx-url`));
}
