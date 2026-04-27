import { useCallback, useRef, useState } from 'react';
import { presignAutosave, commitAutosave } from '../api/templatesV2';

const DEBOUNCE_MS = 15_000;

async function sha256Hex(buf: ArrayBuffer): Promise<string> {
  const digest = await crypto.subtle.digest('SHA-256', buf);
  return Array.from(new Uint8Array(digest)).map((b) => b.toString(16).padStart(2, '0')).join('');
}

export function useTemplateAutosave(templateId: string, versionNum: number) {
  const pendingDocx = useRef<ArrayBuffer | null>(null);
  const timer = useRef<number | null>(null);
  const [status, setStatus] = useState<'idle' | 'saving' | 'saved' | 'error'>('idle');

  const flush = useCallback(async () => {
    if (!pendingDocx.current) return;
    const buf = pendingDocx.current;
    setStatus('saving');
    try {
      const { upload_url, storage_key: _key } = await presignAutosave(templateId, versionNum);
      await fetch(upload_url, {
        method: 'PUT',
        headers: { 'content-type': 'application/vnd.openxmlformats-officedocument.wordprocessingml.document' },
        body: buf,
      });
      const hash = await sha256Hex(buf);
      await commitAutosave(templateId, versionNum, hash);
      pendingDocx.current = null;
      setStatus('saved');
    } catch {
      setStatus('error');
    }
  }, [templateId, versionNum]);

  const schedule = useCallback(() => {
    if (timer.current) window.clearTimeout(timer.current);
    timer.current = window.setTimeout(() => void flush(), DEBOUNCE_MS);
  }, [flush]);

  const queueDocx = useCallback(
    (buf: ArrayBuffer) => {
      pendingDocx.current = buf;
      schedule();
    },
    [schedule],
  );

  const hasPending = useCallback(() => pendingDocx.current !== null, []);

  const importDocx = useCallback(async (buf: ArrayBuffer): Promise<void> => {
    setStatus('saving');
    try {
      const { upload_url } = await presignAutosave(templateId, versionNum);
      await fetch(upload_url, {
        method: 'PUT',
        headers: { 'content-type': 'application/vnd.openxmlformats-officedocument.wordprocessingml.document' },
        body: buf,
      });
      const hash = await sha256Hex(buf);
      await commitAutosave(templateId, versionNum, hash);
      setStatus('saved');
    } catch (err) {
      setStatus('error');
      throw err;
    }
  }, [templateId, versionNum]);

  return { queueDocx, flush, status, hasPending, importDocx };
}
