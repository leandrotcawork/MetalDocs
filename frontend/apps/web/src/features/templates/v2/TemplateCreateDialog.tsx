import { type FormEvent, useState } from 'react';
import { createTemplate } from './api/templatesV2';
import styles from './TemplateCreateDialog.module.css';

export type TemplateCreateDialogProps = {
  onClose: () => void;
  onCreated: (templateId: string, versionNum: number) => void;
};

export function TemplateCreateDialog({ onClose, onCreated }: TemplateCreateDialogProps) {
  const [key, setKey] = useState('');
  const [docTypeCode, setDocTypeCode] = useState('');
  const [name, setName] = useState('');
  const [desc, setDesc] = useState('');
  const [visibility, setVisibility] = useState('public');
  const [approverRole, setApproverRole] = useState('');
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  async function submit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    setBusy(true);
    setErr(null);

    try {
      const { template, version } = await createTemplate({
        key,
        doc_type_code: docTypeCode,
        name,
        description: desc || undefined,
        visibility,
        approver_role: approverRole,
      });
      onCreated(template.id, version.version_number);
    } catch (error) {
      setErr(error instanceof Error ? error.message : String(error));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className={styles.overlay}>
      <div className={styles.modal}>
        <h3 className={styles.title}>New Template</h3>
        <form onSubmit={submit}>
          <div className={styles.field}>
            <label className={styles.label} htmlFor="template-key">
              Key*
            </label>
            <input
              id="template-key"
              className={styles.input}
              value={key}
              onChange={(e) => setKey(e.target.value)}
              required
              pattern="[a-z0-9_-]+"
              placeholder="e.g. nda-standard"
            />
          </div>

          <div className={styles.field}>
            <label className={styles.label} htmlFor="template-doc-type-code">
              Doc Type Code
            </label>
            <input
              id="template-doc-type-code"
              className={styles.input}
              value={docTypeCode}
              onChange={(e) => setDocTypeCode(e.target.value)}
            />
          </div>

          <div className={styles.field}>
            <label className={styles.label} htmlFor="template-name">
              Name*
            </label>
            <input
              id="template-name"
              className={styles.input}
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
            />
          </div>

          <div className={styles.field}>
            <label className={styles.label} htmlFor="template-description">
              Description
            </label>
            <textarea
              id="template-description"
              className={styles.textarea}
              value={desc}
              onChange={(e) => setDesc(e.target.value)}
              rows={3}
            />
          </div>

          <div className={styles.field}>
            <label className={styles.label} htmlFor="template-visibility">
              Visibility
            </label>
            <select
              id="template-visibility"
              className={styles.select}
              value={visibility}
              onChange={(e) => setVisibility(e.target.value)}
            >
              <option value="public">public</option>
              <option value="internal">internal</option>
              <option value="specific">specific</option>
            </select>
          </div>

          <div className={styles.field}>
            <label className={styles.label} htmlFor="template-approver-role">
              Approver Role*
            </label>
            <select
              id="template-approver-role"
              className={styles.select}
              value={approverRole}
              onChange={(e) => setApproverRole(e.target.value)}
              required
            >
              <option value="" disabled>Select a role</option>
              <option value="admin">admin</option>
              <option value="editor">editor</option>
              <option value="reviewer">reviewer</option>
              <option value="viewer">viewer</option>
            </select>
          </div>

          {err && (
            <div role="alert" className={styles.error}>
              {err}
            </div>
          )}

          <div className={styles.actions}>
            <button type="button" className={styles.cancelBtn} onClick={onClose} disabled={busy}>
              Cancel
            </button>
            <button
              type="submit"
              className={styles.submitBtn}
              disabled={busy || !key || !name || !approverRole}
            >
              {busy ? 'Creating…' : 'Create Template'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
