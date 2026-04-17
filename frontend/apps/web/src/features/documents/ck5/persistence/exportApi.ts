export class ExportError extends Error {
  constructor(
    public readonly status: number,
    message?: string,
  ) {
    super(message ?? `Export failed with status ${status}`);
    this.name = 'ExportError';
  }
}

export async function triggerExport(docId: string, fmt: 'docx' | 'pdf'): Promise<void> {
  const res = await fetch(`/api/v1/documents/${docId}/export/ck5/${fmt}`, {
    method: 'GET',
    credentials: 'include',
  });

  if (!res.ok) {
    throw new ExportError(res.status);
  }

  const blob = await res.blob();
  const url = URL.createObjectURL(blob);
  const a = document.createElement('a');
  a.href = url;
  a.download = fmt === 'docx' ? 'document.docx' : 'document.pdf';
  document.body.appendChild(a);
  a.click();
  document.body.removeChild(a);
  URL.revokeObjectURL(url);
}

export function clientPrint(html: string): void {
  const iframe = document.createElement('iframe');
  iframe.style.cssText = 'position:absolute;width:0;height:0;border:0;visibility:hidden';
  document.body.appendChild(iframe);

  const doc = iframe.contentWindow?.document;
  if (!doc) {
    document.body.removeChild(iframe);
    return;
  }

  doc.open();
  doc.write(`<!DOCTYPE html><html><body>${html}</body></html>`);
  doc.close();

  iframe.contentWindow?.focus();
  iframe.contentWindow?.print();

  const cleanup = () => {
    document.body.removeChild(iframe);
    iframe.contentWindow?.removeEventListener('afterprint', cleanup);
  };

  iframe.contentWindow?.addEventListener('afterprint', cleanup);
}
