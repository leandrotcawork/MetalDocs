import { useEffect, useState } from 'react';
import { listTemplates, type TemplateListRow } from '../../templates/v2/api/templatesV2';
import { createDocument } from './api/documentsV2';

export type DocumentCreatePageProps = {
  onCreated: (documentID: string) => void;
};

export function DocumentCreatePage({ onCreated }: DocumentCreatePageProps): React.ReactElement {
  const [templates, setTemplates] = useState<TemplateListRow[]>([]);
  const [pick, setPick] = useState<TemplateListRow | null>(null);
  const [name, setName] = useState('');
  const [err, setErr] = useState('');
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      try {
        const rows = await listTemplates();
        if (!cancelled) {
          setTemplates(rows);
        }
      } catch {
        if (!cancelled) {
          setErr('Failed to load templates.');
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  async function handleCreate() {
    if (!pick || !name.trim()) return;
    setSubmitting(true);
    setErr('');
    try {
      const res = await createDocument({
        template_version_id: pick.latest_version_id,
        name: name.trim(),
        form_data: {},
      });
      onCreated(res.DocumentID);
    } catch {
      setErr('Failed to create document.');
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <div>
      <h1>New document</h1>
      <section>
        <h2>Pick template</h2>
        <ul>
          {templates.map((template) => (
            <li key={template.id}>
              <button
                type="button"
                aria-pressed={pick?.id === template.id}
                onClick={() => {
                  setPick(template);
                  setErr('');
                }}
              >
                {template.name}
              </button>
            </li>
          ))}
        </ul>
      </section>
      {pick && (
        <section>
          <label>
            Name
            <input
              type="text"
              value={name}
              onChange={(event) => setName(event.target.value)}
              disabled={submitting}
            />
          </label>
          <button type="button" onClick={() => void handleCreate()} disabled={submitting || !name.trim()}>
            Generate document
          </button>
        </section>
      )}
      {err && <div role="alert">{err}</div>}
    </div>
  );
}
