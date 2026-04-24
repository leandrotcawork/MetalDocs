import { useCallback, useEffect, useRef, useState } from 'react';
import { putPlaceholderValue } from '../api/documentsV2';

export interface PlaceholderValueState {
  value: string;
  setValue: (v: string) => void;
  error: string | null;
  saving: boolean;
}

const DEBOUNCE_MS = 400;

export function usePlaceholderValue(
  docId: string,
  placeholderId: string,
  initialValue: string,
): PlaceholderValueState {
  const [value, setValueState] = useState(initialValue);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const latestValue = useRef(initialValue);

  useEffect(() => {
    latestValue.current = value;
  }, [value]);

  const setValue = useCallback(
    (v: string) => {
      setValueState(v);
      setError(null);
      if (timer.current !== null) clearTimeout(timer.current);
      timer.current = setTimeout(async () => {
        setSaving(true);
        try {
          await putPlaceholderValue(docId, placeholderId, v);
          setError(null);
        } catch (err: unknown) {
          if (err instanceof Error && 'status' in err && (err as any).status === 422) {
            try {
              const body = JSON.parse((err as any).body as string) as {
                error?: { message?: string; code?: string };
              };
              setError(body?.error?.message ?? 'Validation error');
            } catch {
              setError('Validation error');
            }
          } else {
            setError(err instanceof Error ? err.message : String(err));
          }
        } finally {
          setSaving(false);
        }
      }, DEBOUNCE_MS);
    },
    [docId, placeholderId],
  );

  return { value, setValue, error, saving };
}
