import { useEffect, useState } from 'react';
import { createDocument } from './api/documentsV2';
import { fetchControlledDocuments } from '../../registry/api';
import type { ControlledDocument } from '../../registry/types';

export type DocumentCreatePageProps = {
  onCreated: (documentID: string) => void;
};

export function DocumentCreatePage({ onCreated }: DocumentCreatePageProps): React.ReactElement {
  const [cds, setCds] = useState<ControlledDocument[]>([]);
  const [cdPick, setCdPick] = useState<ControlledDocument | null>(null);
  const [name, setName] = useState('');
  const [err, setErr] = useState('');
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    let cancelled = false;
    void (async () => {
      try {
        const rows = await fetchControlledDocuments({ status: 'active' });
        if (!cancelled) {
          setCds(rows);
        }
      } catch {
        if (!cancelled) {
          setErr('Failed to load controlled documents.');
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  async function handleCreate() {
    if (!cdPick || !name.trim()) return;
    setSubmitting(true);
    setErr('');
    try {
      const res = await createDocument({
        template_version_id: cdPick.overrideTemplateVersionId ?? '<resolved-server-side>',
        name: name.trim(),
        form_data: {},
        controlled_document_id: cdPick.id,
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
        <h2>Step 1: Pick Controlled Document</h2>
        <table style={{ borderCollapse: 'collapse', fontSize: 13, width: '100%' }}>
          <thead>
            <tr style={{ borderBottom: '2px solid #e0e0e0', textAlign: 'left' }}>
              <th style={{ padding: '6px 8px' }}>Code</th>
              <th style={{ padding: '6px 8px' }}>Title</th>
              <th style={{ padding: '6px 8px' }}>Profile</th>
              <th style={{ padding: '6px 8px' }}>Area</th>
            </tr>
          </thead>
          <tbody>
            {cds.length === 0 && (
              <tr>
                <td colSpan={4} style={{ padding: '12px 8px', color: '#888', textAlign: 'center' }}>
                  No active controlled documents found.
                </td>
              </tr>
            )}
            {cds.map((cd) => (
              <tr
                key={cd.id}
                onClick={() => { setCdPick(cd); setErr(''); }}
                style={{
                  borderBottom: '1px solid #f0f0f0',
                  cursor: 'pointer',
                  background: cdPick?.id === cd.id ? '#e8f0fe' : undefined,
                }}
              >
                <td style={{ padding: '6px 8px', fontFamily: 'monospace' }}>{cd.code}</td>
                <td style={{ padding: '6px 8px' }}>{cd.title}</td>
                <td style={{ padding: '6px 8px', fontFamily: 'monospace' }}>{cd.profileCode}</td>
                <td style={{ padding: '6px 8px', fontFamily: 'monospace' }}>{cd.processAreaCode}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>
      {cdPick && (
        <section style={{ marginTop: 16 }}>
          <p style={{ fontSize: 12, color: '#555' }}>
            Template: resolved server-side from profile <strong>{cdPick.profileCode}</strong>
            {cdPick.overrideTemplateVersionId ? ` (override: ${cdPick.overrideTemplateVersionId})` : ''}
          </p>
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
