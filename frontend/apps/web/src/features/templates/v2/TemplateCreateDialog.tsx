import { useState } from 'react';
import { createTemplate } from './api/templatesV2';

export type TemplateCreateDialogProps = {
  onClose: () => void;
  onCreated: (templateId: string, versionNum: number) => void;
};

export function TemplateCreateDialog({ onClose, onCreated }: TemplateCreateDialogProps) {
  const [key, setKey] = useState('');
  const [name, setName] = useState('');
  const [desc, setDesc] = useState('');
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true); setErr(null);
    try {
      const r = await createTemplate(key, name, desc || undefined);
      onCreated(r.id, 1);
    } catch (e) {
      setErr(String(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <dialog open onClose={onClose}>
      <form onSubmit={submit}>
        <label>Key <input required value={key} onChange={(e) => setKey(e.target.value)} /></label>
        <label>Name <input required value={name} onChange={(e) => setName(e.target.value)} /></label>
        <label>Description <textarea value={desc} onChange={(e) => setDesc(e.target.value)} /></label>
        {err && <div role="alert">{err}</div>}
        <button type="button" onClick={onClose} disabled={busy}>Cancel</button>
        <button type="submit" disabled={busy}>Create</button>
      </form>
    </dialog>
  );
}
