import { useCallback, useEffect, useRef, useState } from 'react';
import { presignDocxUpload, presignSchemaUpload, saveDraft } from '../api/templatesV2';

interface AutosaveArgs {
  templateId: string;
  versionNum: number;
  lockVersion: number;
  docxStorageKey: string;
  schemaStorageKey: string;
}

const DEBOUNCE_MS = 15_000;

async function sha256Hex(buf: ArrayBuffer | string): Promise<string> {
  const data = typeof buf === 'string' ? new TextEncoder().encode(buf) : new Uint8Array(buf);
  const digest = await crypto.subtle.digest('SHA-256', data);
  return Array.from(new Uint8Array(digest)).map((b) => b.toString(16).padStart(2, '0')).join('');
}

export type PersistedDraftState = {
  docxStorageKey: string;
  schemaStorageKey: string;
  docxContentHash: string;
  schemaContentHash: string;
  lockVersion: number;
};

export function useTemplateAutosave(args: AutosaveArgs) {
  const pendingDocx = useRef<ArrayBuffer | null>(null);
  const pendingSchema = useRef<string | null>(null);
  const timer = useRef<number | null>(null);
  const persistedDocxKey = useRef(args.docxStorageKey);
  const persistedSchemaKey = useRef(args.schemaStorageKey);
  const persistedDocxHash = useRef('');
  const persistedSchemaHash = useRef('');
  const lockRef = useRef(args.lockVersion);
  const [status, setStatus] = useState<'idle' | 'saving' | 'saved' | 'stale' | 'error'>('idle');

  useEffect(() => {
    persistedDocxKey.current = args.docxStorageKey;
    persistedSchemaKey.current = args.schemaStorageKey;
    persistedDocxHash.current = '';
    persistedSchemaHash.current = '';
    lockRef.current = args.lockVersion;
  }, [args.templateId, args.versionNum, args.docxStorageKey, args.schemaStorageKey, args.lockVersion]);

  const flush = useCallback(async () => {
    if (!pendingDocx.current && pendingSchema.current === null) return;
    setStatus('saving');
    try {
      let docxKey = persistedDocxKey.current;
      let docxHash = persistedDocxHash.current;
      if (pendingDocx.current) {
        const up = await presignDocxUpload(args.templateId, args.versionNum);
        await fetch(up.url, {
          method: 'PUT',
          headers: { 'content-type': 'application/vnd.openxmlformats-officedocument.wordprocessingml.document' },
          body: pendingDocx.current,
        });
        docxKey = up.storage_key;
        docxHash = await sha256Hex(pendingDocx.current);
      }
      let schemaKey = persistedSchemaKey.current;
      let schemaHash = persistedSchemaHash.current;
      if (pendingSchema.current !== null) {
        const up = await presignSchemaUpload(args.templateId, args.versionNum);
        await fetch(up.url, {
          method: 'PUT',
          headers: { 'content-type': 'application/json' },
          body: pendingSchema.current,
        });
        schemaKey = up.storage_key;
        schemaHash = await sha256Hex(pendingSchema.current);
      }
      await saveDraft(args.templateId, args.versionNum, {
        expected_lock_version: lockRef.current,
        docx_storage_key: docxKey,
        schema_storage_key: schemaKey,
        docx_content_hash: docxHash,
        schema_content_hash: schemaHash,
      });
      persistedDocxKey.current = docxKey;
      persistedSchemaKey.current = schemaKey;
      persistedDocxHash.current = docxHash;
      persistedSchemaHash.current = schemaHash;
      lockRef.current += 1;
      pendingDocx.current = null;
      pendingSchema.current = null;
      setStatus('saved');
    } catch (e) {
      if (String(e).includes('template_draft_stale')) { setStatus('stale'); return; }
      setStatus('error');
    }
  }, [args.templateId, args.versionNum]);

  const schedule = useCallback(() => {
    if (timer.current) window.clearTimeout(timer.current);
    timer.current = window.setTimeout(flush, DEBOUNCE_MS);
  }, [flush]);

  const queueDocx = useCallback((buf: ArrayBuffer) => { pendingDocx.current = buf; schedule(); }, [schedule]);
  const queueSchema = useCallback((txt: string) => { pendingSchema.current = txt; schedule(); }, [schedule]);

  const getPersisted = useCallback((): PersistedDraftState => ({
    docxStorageKey: persistedDocxKey.current,
    schemaStorageKey: persistedSchemaKey.current,
    docxContentHash: persistedDocxHash.current,
    schemaContentHash: persistedSchemaHash.current,
    lockVersion: lockRef.current,
  }), []);

  const hasPending = useCallback(() => (pendingDocx.current !== null || pendingSchema.current !== null), []);

  useEffect(() => () => { if (timer.current) window.clearTimeout(timer.current); }, []);

  return { queueDocx, queueSchema, flush, status, getPersisted, hasPending };
}
