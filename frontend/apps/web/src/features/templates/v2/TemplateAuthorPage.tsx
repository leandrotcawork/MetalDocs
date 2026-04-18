import { useEffect, useRef, useState } from 'react';
import { MetalDocsEditor, computeSidebarModel, type MetalDocsEditorRef } from '@metaldocs/editor-ui';
import { SchemaEditor, FormRenderer, validateJsonSchema } from '@metaldocs/form-ui';
import { parseDocxTokens } from '@metaldocs/shared-tokens';
import { useTemplateDraft } from './hooks/useTemplateDraft';
import { useTemplateAutosave } from './hooks/useTemplateAutosave';
import { publishVersion, type PublishError, type PublishSuccess } from './api/templatesV2';
import styles from './TemplateAuthorPage.module.css';

export type TemplateAuthorPageProps = {
  templateId: string;
  versionNum: number;
  onNavigateToVersion?: (templateId: string, versionNum: number) => void;
};

export function TemplateAuthorPage({ templateId, versionNum, onNavigateToVersion }: TemplateAuthorPageProps) {
  const draft = useTemplateDraft(templateId, versionNum);
  const editorRef = useRef<MetalDocsEditorRef>(null);
  const [tab, setTab] = useState<'schema' | 'preview'>('schema');
  const [schemaText, setSchemaText] = useState(draft.schemaText);
  const [tokens, setTokens] = useState<any[]>([]);
  const [parseErrors, setParseErrors] = useState<any[]>([]);
  const [publishErr, setPublishErr] = useState<PublishError | null>(null);

  useEffect(() => { setSchemaText(draft.schemaText); }, [draft.schemaText]);

  const autosave = useTemplateAutosave({
    templateId, versionNum,
    lockVersion: draft.lockVersion,
    docxStorageKey: draft.docxKey,
    schemaStorageKey: draft.schemaKey,
  });

  async function handleDocxChange(buf: ArrayBuffer) {
    const r = await parseDocxTokens(buf);
    setTokens(r.tokens);
    setParseErrors(r.errors);
    autosave.queueDocx(buf);
  }

  const schemaValidation = validateJsonSchema(schemaText);
  const schemaObj = schemaValidation.valid ? JSON.parse(schemaText) : {};
  const sidebar = computeSidebarModel(tokens, parseErrors, schemaObj);

  async function handlePublish() {
    setPublishErr(null);
    if (autosave.hasPending()) {
      await autosave.flush();
    }
    const persisted = autosave.getPersisted();
    const result = await publishVersion(templateId, versionNum, persisted.docxStorageKey, persisted.schemaStorageKey);
    if ('parse_errors' in result) { setPublishErr(result as PublishError); return; }
    const ok = result as PublishSuccess;
    onNavigateToVersion?.(templateId, ok.next_draft_version_num);
  }

  if (draft.loading) return <div>Loading…</div>;
  if (draft.error) return <div role="alert">{draft.error}</div>;

  return (
    <div className={styles.page}>
      <header className={styles.header}>
        <h1>{draft.name}</h1>
        <button onClick={handlePublish} disabled={sidebar.bannerError || sidebar.missing.length > 0 || !schemaValidation.valid}>
          Publish
        </button>
      </header>
      {sidebar.bannerError && (
        <div role="alert" className={styles.banner}>
          Template contains unsupported OOXML: {sidebar.errorCategories.join(', ')}
        </div>
      )}
      {publishErr && (
        <div role="alert" className={styles.banner}>
          Publish rejected. Parse errors: {publishErr.parse_errors.length}, missing: {publishErr.missing_tokens.join(', ')}, orphans: {publishErr.orphan_tokens.join(', ')}
        </div>
      )}
      <div className={styles.split}>
        <div className={styles.editor}>
          <MetalDocsEditor ref={editorRef} mode="template-draft" documentBuffer={draft.docxBuffer} userId={draft.userId} onAutoSave={handleDocxChange} />
        </div>
        <aside className={styles.sidebar}>
          <div className={styles.tabs}>
            <button data-active={tab==='schema'} onClick={() => setTab('schema')}>Schema</button>
            <button data-active={tab==='preview'} onClick={() => setTab('preview')}>Preview</button>
          </div>
          {tab === 'schema' ? (
            <SchemaEditor value={schemaText} onChange={(v) => { setSchemaText(v); autosave.queueSchema(v); }} height={500} />
          ) : (
            <FormRenderer schema={schemaObj} formData={{}} onChange={() => {}} />
          )}
          <section className={styles.fieldsSidebar}>
            <h3>Fields</h3>
            <ul>
              {sidebar.used.map((i) => <li key={i} data-state="used">{i}</li>)}
              {sidebar.missing.map((i) => <li key={i} data-state="missing">missing: {i}</li>)}
              {sidebar.orphans.map((i) => <li key={i} data-state="orphan">orphan: {i}</li>)}
            </ul>
          </section>
        </aside>
      </div>
    </div>
  );
}
