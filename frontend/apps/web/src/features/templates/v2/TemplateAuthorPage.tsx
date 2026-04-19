import { useEffect, useState } from 'react';
import { useTemplateDraft } from './hooks/useTemplateDraft';
import { useTemplateAutosave } from './hooks/useTemplateAutosave';
import { publishVersion, type PublishError, type PublishSuccess } from './api/templatesV2';

export type TemplateAuthorPageProps = {
  templateId: string;
  versionNum: number;
  onNavigateToVersion?: (templateId: string, versionNum: number) => void;
};

export function TemplateAuthorPage({ templateId, versionNum, onNavigateToVersion }: TemplateAuthorPageProps) {
  const draft = useTemplateDraft(templateId, versionNum);
  const [schemaText, setSchemaText] = useState(draft.schemaText);
  const [publishErr, setPublishErr] = useState<PublishError | null>(null);

  useEffect(() => { setSchemaText(draft.schemaText); }, [draft.schemaText]);

  const autosave = useTemplateAutosave({
    templateId, versionNum,
    lockVersion: draft.lockVersion,
    docxStorageKey: draft.docxKey,
    schemaStorageKey: draft.schemaKey,
  });

  function handleFileChange(e: React.ChangeEvent<HTMLInputElement>) {
    const file = e.target.files?.[0];
    if (!file) return;
    file.arrayBuffer().then((buf) => autosave.queueDocx(buf));
  }

  async function handlePublish() {
    setPublishErr(null);
    if (autosave.hasPending()) await autosave.flush();
    const persisted = autosave.getPersisted();
    const result = await publishVersion(templateId, versionNum, persisted.docxStorageKey, persisted.schemaStorageKey);
    if ('parse_errors' in result) { setPublishErr(result as PublishError); return; }
    const ok = result as PublishSuccess;
    onNavigateToVersion?.(templateId, ok.next_draft_version_num);
  }

  if (draft.loading) return <div>Loading…</div>;
  if (draft.error) return <div role="alert">{draft.error}</div>;

  const schemaValid = (() => { try { JSON.parse(schemaText); return true; } catch { return false; } })();

  return (
    <div style={{ padding: '1.5rem', maxWidth: 800 }}>
      <h1 style={{ marginBottom: '1rem' }}>{draft.name} <small style={{ fontWeight: 400, fontSize: '0.75em', opacity: 0.6 }}>v{versionNum}</small></h1>

      <section style={{ marginBottom: '1.5rem' }}>
        <h2 style={{ fontSize: '1rem', marginBottom: '0.5rem' }}>DOCX template</h2>
        <p style={{ opacity: 0.6, fontSize: '0.85rem', marginBottom: '0.5rem' }}>
          {draft.docxKey ? `Saved: ${draft.docxKey}` : 'No file uploaded yet.'}
        </p>
        <input type="file" accept=".docx" onChange={handleFileChange} />
        <span style={{ marginLeft: '1rem', opacity: 0.6, fontSize: '0.85rem' }}>
          Autosave: {autosave.status}
        </span>
      </section>

      <section style={{ marginBottom: '1.5rem' }}>
        <h2 style={{ fontSize: '1rem', marginBottom: '0.5rem' }}>Schema JSON</h2>
        <textarea
          value={schemaText}
          onChange={(e) => { setSchemaText(e.target.value); autosave.queueSchema(e.target.value); }}
          rows={10}
          style={{ width: '100%', fontFamily: 'monospace', fontSize: '0.85rem' }}
        />
        {!schemaValid && <p style={{ color: 'red', fontSize: '0.8rem' }}>Invalid JSON</p>}
      </section>

      {publishErr && (
        <div role="alert" style={{ color: 'red', marginBottom: '1rem' }}>
          Publish rejected. Parse errors: {publishErr.parse_errors.length},
          missing: {publishErr.missing_tokens.join(', ')},
          orphans: {publishErr.orphan_tokens.join(', ')}
        </div>
      )}

      <button onClick={handlePublish} disabled={!schemaValid}>
        Publish version {versionNum}
      </button>
    </div>
  );
}
