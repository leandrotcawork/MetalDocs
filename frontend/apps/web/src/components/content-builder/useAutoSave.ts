import { useCallback, useEffect, useRef, useState } from "react";

type AutoSaveOptions = {
  documentId: string;
  contentDraft: Record<string, unknown>;
  saveFn: (documentId: string, body: { content: Record<string, unknown> }) => Promise<{ pdfUrl: string; version: number | null }>;
  debounceMs?: number;
  enabled?: boolean;
};

type AutoSaveResult = {
  isSaving: boolean;
  lastSavedPdfUrl: string;
  lastSavedAt: Date | null;
  error: string;
  saveNow: () => void;
  acknowledgeSave: (content: Record<string, unknown>, pdfUrl: string) => void;
};

export function useAutoSave(options: AutoSaveOptions): AutoSaveResult {
  const { documentId, contentDraft, saveFn, debounceMs = 3000, enabled = true } = options;

  const [isSaving, setIsSaving] = useState(false);
  const [lastSavedPdfUrl, setLastSavedPdfUrl] = useState("");
  const [lastSavedAt, setLastSavedAt] = useState<Date | null>(null);
  const [error, setError] = useState("");

  const timerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const abortRef = useRef<AbortController | null>(null);
  const lastSavedJsonRef = useRef<string>("");
  const contentRef = useRef(contentDraft);
  const saveVersionRef = useRef(0);

  contentRef.current = contentDraft;

  const doSave = useCallback(async () => {
    if (!documentId || !enabled) return;

    const currentJson = JSON.stringify(contentRef.current);
    if (currentJson === lastSavedJsonRef.current) return;

    abortRef.current?.abort();
    const controller = new AbortController();
    abortRef.current = controller;

    const version = ++saveVersionRef.current;

    setIsSaving(true);
    setError("");

    try {
      const response = await saveFn(documentId, { content: contentRef.current });

      if (controller.signal.aborted || version !== saveVersionRef.current) return;

      lastSavedJsonRef.current = currentJson;
      setLastSavedPdfUrl(response.pdfUrl);
      setLastSavedAt(new Date());
    } catch (err) {
      if (controller.signal.aborted || version !== saveVersionRef.current) return;
      setError(err instanceof Error ? err.message : "Falha ao salvar automaticamente.");
    } finally {
      if (!controller.signal.aborted && version === saveVersionRef.current) {
        setIsSaving(false);
      }
    }
  }, [documentId, saveFn, enabled]);

  const saveNow = useCallback(() => {
    if (timerRef.current) {
      clearTimeout(timerRef.current);
      timerRef.current = null;
    }
    void doSave();
  }, [doSave]);

  const acknowledgeSave = useCallback((content: Record<string, unknown>, pdfUrl: string) => {
    if (timerRef.current) {
      clearTimeout(timerRef.current);
      timerRef.current = null;
    }
    const currentJson = JSON.stringify(content);
    lastSavedJsonRef.current = currentJson;
    setLastSavedPdfUrl(pdfUrl);
    setLastSavedAt(new Date());
    setError("");
  }, []);

  useEffect(() => {
    if (!documentId || !enabled) return;

    const currentJson = JSON.stringify(contentDraft);
    if (currentJson === lastSavedJsonRef.current) return;

    if (timerRef.current) {
      clearTimeout(timerRef.current);
    }

    timerRef.current = setTimeout(() => {
      timerRef.current = null;
      void doSave();
    }, debounceMs);

    return () => {
      if (timerRef.current) {
        clearTimeout(timerRef.current);
        timerRef.current = null;
      }
    };
  }, [contentDraft, documentId, debounceMs, doSave, enabled]);

  useEffect(() => {
    return () => {
      abortRef.current?.abort();
      if (timerRef.current) {
        clearTimeout(timerRef.current);
      }
    };
  }, []);

  return { isSaving, lastSavedPdfUrl, lastSavedAt, error, saveNow, acknowledgeSave };
}
