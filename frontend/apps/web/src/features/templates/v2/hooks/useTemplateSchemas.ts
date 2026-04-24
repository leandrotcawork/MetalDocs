import { useCallback, useEffect, useState } from 'react';
import { getTemplateSchemas, putTemplateSchemas, type TemplateSchemas } from '../api/templatesV2';

interface UseTemplateSchemasResult {
  schemas: TemplateSchemas | null;
  loading: boolean;
  error: string | null;
  save: (s: TemplateSchemas) => Promise<void>;
  saving: boolean;
}

export function useTemplateSchemas(templateId: string, versionNum: number): UseTemplateSchemasResult {
  const [schemas, setSchemas] = useState<TemplateSchemas | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    let cancelled = false;
    setLoading(true);
    setError(null);
    getTemplateSchemas(templateId, versionNum)
      .then((s) => { if (!cancelled) setSchemas(s); })
      .catch((e) => { if (!cancelled) setError(e instanceof Error ? e.message : String(e)); })
      .finally(() => { if (!cancelled) setLoading(false); });
    return () => { cancelled = true; };
  }, [templateId, versionNum]);

  const save = useCallback(async (s: TemplateSchemas) => {
    setSaving(true);
    try {
      await putTemplateSchemas(templateId, versionNum, s);
      setSchemas(s);
    } finally {
      setSaving(false);
    }
  }, [templateId, versionNum]);

  return { schemas, loading, error, save, saving };
}
