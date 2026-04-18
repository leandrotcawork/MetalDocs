import { useState } from 'react';
import { exportPDF, getDocxSignedURL, type ExportPDFResult } from './api/exportsV2';

type Status =
  | { kind: 'idle' }
  | { kind: 'pending' }
  | { kind: 'done'; cached: boolean; url: string; sizeBytes: number }
  | { kind: 'error'; message: string }
  | { kind: 'rate_limited'; retryAfterSec: number };

export function ExportMenu({ documentID, canExport }: { documentID: string; canExport: boolean }) {
  const [status, setStatus] = useState<Status>({ kind: 'idle' });

  async function handleDOCX() {
    setStatus({ kind: 'pending' });
    try {
      const { signed_url } = await getDocxSignedURL(documentID);
      window.open(signed_url, '_blank', 'noopener');
      setStatus({ kind: 'done', cached: true, url: signed_url, sizeBytes: 0 });
    } catch (e: any) {
      if (e?.status === 429) {
        setStatus({ kind: 'rate_limited', retryAfterSec: e.body?.retry_after_seconds ?? 60 });
        return;
      }
      setStatus({ kind: 'error', message: String(e?.message ?? e) });
    }
  }

  async function handlePDF() {
    setStatus({ kind: 'pending' });
    try {
      const res: ExportPDFResult = await exportPDF(documentID, { paper_size: 'A4' });
      window.open(res.signed_url, '_blank', 'noopener');
      setStatus({ kind: 'done', cached: res.cached, url: res.signed_url, sizeBytes: res.size_bytes });
    } catch (e: any) {
      if (e?.status === 429) {
        setStatus({ kind: 'rate_limited', retryAfterSec: e.body?.retry_after_seconds ?? 60 });
        return;
      }
      if (e?.status === 502) {
        setStatus({ kind: 'error', message: 'PDF service unavailable — retry in a moment.' });
        return;
      }
      if (e?.status === 409) {
        setStatus({ kind: 'error', message: 'Document missing. Save and retry.' });
        return;
      }
      setStatus({ kind: 'error', message: String(e?.message ?? e) });
    }
  }

  return (
    <div data-export-menu>
      <button onClick={handleDOCX} disabled={!canExport || status.kind === 'pending'} data-export-docx>
        Download .docx
      </button>
      <button onClick={handlePDF} disabled={!canExport || status.kind === 'pending'} data-export-pdf>
        Export PDF
      </button>
      {status.kind === 'pending' && <span data-export-status="pending">Working…</span>}
      {status.kind === 'done' && (
        <span data-export-status="done" data-export-cached={String(status.cached)}>
          {status.cached ? 'Cached' : 'Generated'} ({(status.sizeBytes / 1024).toFixed(0)} KB)
        </span>
      )}
      {status.kind === 'rate_limited' && (
        <span role="alert" data-export-status="rate_limited">
          Rate limited — retry in {status.retryAfterSec}s
        </span>
      )}
      {status.kind === 'error' && (
        <span role="alert" data-export-status="error">{status.message}</span>
      )}
    </div>
  );
}
