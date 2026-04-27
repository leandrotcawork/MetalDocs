import { useCallback, useEffect, useState } from 'react';
import { getVersion, getTemplate, getDocxURL, type VersionDTO, type TemplateDTO } from '../api/templatesV2';

type DraftState = {
  loading: boolean;
  error: string | null;
  template: TemplateDTO | null;
  version: VersionDTO | null;
  docxBytes: ArrayBuffer | null;
};

type TemplateDraft = DraftState & {
  refetch: () => void;
};

export function useTemplateDraft(templateId: string, versionNum: number): TemplateDraft {
  const [state, setState] = useState<DraftState>({
    loading: true,
    error: null,
    template: null,
    version: null,
    docxBytes: null,
  });
  const [tick, setTick] = useState(0);
  const refetch = useCallback(() => setTick((t) => t + 1), []);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const [template, version] = await Promise.all([getTemplate(templateId), getVersion(templateId, versionNum)]);

        let docxBytes: ArrayBuffer | null = null;
        if (version.docx_storage_key) {
          const url = await getDocxURL(templateId, versionNum);
          const res = await fetch(url);
          if (res.ok) {
            docxBytes = await res.arrayBuffer();
          }
        }

        if (!cancelled) {
          setState({
            loading: false,
            error: null,
            template: template.template,
            version,
            docxBytes,
          });
        }
      } catch (e) {
        if (!cancelled) {
          setState((s) => ({
            ...s,
            loading: false,
            error: e instanceof Error ? e.message : String(e),
          }));
        }
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [templateId, versionNum, tick]);

  return { ...state, refetch };
}
