import { useEffect, useState } from 'react';
import { listTemplates, type TemplateListRow } from './api/templatesV2';

export type TemplatesListPageProps = {
  onOpenTemplate: (templateId: string, versionNum: number) => void;
  onCreate: () => void;
};

export function TemplatesListPage({ onOpenTemplate, onCreate }: TemplatesListPageProps) {
  const [tpls, setTpls] = useState<TemplateListRow[]>([]);
  const [err, setErr] = useState<string | null>(null);

  useEffect(() => {
    listTemplates().then(setTpls).catch((e) => setErr(String(e)));
  }, []);

  if (err) return <div role="alert">{err}</div>;

  return (
    <div>
      <h1>Templates</h1>
      <button onClick={onCreate}>New template</button>
      <ul>
        {tpls.map((t) => (
          <li key={t.id}>
            <button onClick={() => onOpenTemplate(t.id, t.latest_version)}>
              {t.name} ({t.key}) — v{t.latest_version}
            </button>
          </li>
        ))}
      </ul>
    </div>
  );
}
