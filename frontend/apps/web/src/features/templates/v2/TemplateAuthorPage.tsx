import '@eigenpal/docx-js-editor/styles.css';
import { useEffect, useMemo, useRef, useState } from 'react';
import { DocxEditor, type DocxEditorRef } from '@eigenpal/docx-js-editor/react';
import { createEmptyDocument } from '@eigenpal/docx-js-editor/core';
import { type VersionDTO, submitForReview } from './api/templatesV2';
import { useTemplateDraft } from './hooks/useTemplateDraft';
import { useTemplateAutosave } from './hooks/useTemplateAutosave';
import { VersionActionPanel } from './VersionActionPanel';
import styles from './TemplateAuthorPage.module.css';

export type TemplateAuthorPageProps = {
  templateId: string;
  versionNum: number;
  onNavigateToVersion?: (templateId: string, versionNum: number) => void;
};

export function TemplateAuthorPage({ templateId, versionNum, onNavigateToVersion: _nav }: TemplateAuthorPageProps) {
  const draft = useTemplateDraft(templateId, versionNum);
  const autosave = useTemplateAutosave(templateId, versionNum);
  const editorRef = useRef<DocxEditorRef>(null);
  const blankDoc = useMemo(() => createEmptyDocument(), []);
  const [submitting, setSubmitting] = useState(false);
  const [submitErr, setSubmitErr] = useState<string | null>(null);
  const [liveVersion, setLiveVersion] = useState<VersionDTO | null>(null);

  useEffect(() => {
    setLiveVersion(draft.version ?? null);
  }, [draft.version]);

  async function handleSubmitForReview() {
    setSubmitErr(null);
    setSubmitting(true);
    try {
      if (autosave.hasPending()) await autosave.flush();
      const updated = await submitForReview(templateId, versionNum);
      setLiveVersion(updated);
      setSubmitErr('Submitted for review.');
    } catch (e) {
      setSubmitErr(e instanceof Error ? e.message : String(e));
    } finally {
      setSubmitting(false);
    }
  }

  if (draft.loading) return <div className={styles.loading}>Loading template...</div>;
  if (draft.error) return <div role="alert" className={styles.error}>{draft.error}</div>;

  const currentVersion = liveVersion ?? draft.version ?? null;
  const isDraft = currentVersion?.status === 'draft';

  return (
    <div className={styles.page}>
      <div className={styles.header}>
        <div>
          <h2 className={styles.title}>
            {draft.template?.name}
            <span className={styles.versionBadge}>v{versionNum}</span>
          </h2>
          <span className={styles.statusLabel}>{currentVersion?.status}</span>
        </div>
        <div className={styles.actions}>
          <span className={styles.autosaveStatus}>{autosave.status === 'saving' ? 'Saving...' : autosave.status === 'saved' ? 'Saved' : autosave.status === 'error' ? 'Save failed' : ''}</span>
          {isDraft && (
            <button
              className={styles.submitBtn}
              onClick={() => void handleSubmitForReview()}
              disabled={submitting}
            >
              {submitting ? 'Submitting...' : 'Submit for Review'}
            </button>
          )}
        </div>
      </div>

      {submitErr && (
        <div role="alert" className={styles.submitAlert} style={{ color: submitErr === 'Submitted for review.' ? '#065f46' : '#dc2626' }}>
          {submitErr}
        </div>
      )}

      <div className={styles.editorWrapper}>
        <DocxEditor
          ref={editorRef}
          documentBuffer={draft.docxBytes ?? undefined}
          document={draft.docxBytes ? undefined : blankDoc}
          readOnly={!isDraft}
          onChange={() => {
            editorRef.current?.save().then((buffer) => {
              if (buffer) {
                autosave.queueDocx(buffer);
              }
            }).catch(() => {
              // ignore autosave buffer serialization errors
            });
          }}
        />
      </div>
      {currentVersion && ['in_review', 'approved', 'published'].includes(currentVersion.status) && (
        <VersionActionPanel
          version={currentVersion}
          onVersionUpdate={(v) => setLiveVersion(v)}
        />
      )}
    </div>
  );
}
