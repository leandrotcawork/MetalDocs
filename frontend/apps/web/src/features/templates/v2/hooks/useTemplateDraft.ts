import { useEffect, useState } from 'react';

export function useTemplateDraft(templateId: string, versionNum: number) {
  const [state, setState] = useState({
    loading: true, error: null as string | null,
    name: '', docxBuffer: undefined as ArrayBuffer | undefined,
    schemaText: '{}', docxKey: '', schemaKey: '', lockVersion: 0, userId: '',
  });

  useEffect(() => {
    (async () => {
      try {
        const meta = await fetch(`/api/v2/templates/${templateId}/versions/${versionNum}`).then((r) => r.json());
        const [docxRes, schemaRes] = await Promise.all([
          meta.docx_storage_key ? fetch(`/api/v2/signed?key=${encodeURIComponent(meta.docx_storage_key)}`).then(r => r.arrayBuffer()) : Promise.resolve(undefined),
          meta.schema_storage_key ? fetch(`/api/v2/signed?key=${encodeURIComponent(meta.schema_storage_key)}`).then(r => r.text()) : Promise.resolve('{}'),
        ]);
        setState({
          loading: false, error: null, name: meta.name,
          docxBuffer: docxRes, schemaText: schemaRes,
          docxKey: meta.docx_storage_key, schemaKey: meta.schema_storage_key,
          lockVersion: meta.lock_version, userId: meta.viewer_user_id,
        });
      } catch (e) {
        setState((s) => ({ ...s, loading: false, error: String(e) }));
      }
    })();
  }, [templateId, versionNum]);

  return state;
}
