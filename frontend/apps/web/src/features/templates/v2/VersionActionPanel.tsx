import { useState, type CSSProperties } from 'react';
import { type VersionDTO, approveVersion, reviewVersion } from './api/templatesV2';

type Props = {
  version: VersionDTO;
  onVersionUpdate: (v: VersionDTO) => void;
};

export function VersionActionPanel({ version, onVersionUpdate }: Props) {
  const [reason, setReason] = useState('');
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState<string | null>(null);
  const [success, setSuccess] = useState<string | null>(null);

  async function act(fn: () => Promise<VersionDTO>, successMsg: string) {
    setBusy(true);
    setErr(null);
    setSuccess(null);
    try {
      const v = await fn();
      setSuccess(successMsg);
      onVersionUpdate(v);
    } catch (e) {
      setErr(e instanceof Error ? e.message : String(e));
    } finally {
      setBusy(false);
    }
  }

  if (version.status === 'in_review') {
    return (
      <div style={panelStyle}>
        <strong style={{ fontSize: '0.875rem' }}>Reviewer actions</strong>
        <textarea
          placeholder="Reason (optional)"
          value={reason}
          onChange={(e) => setReason(e.target.value)}
          rows={2}
          style={textareaStyle}
          disabled={busy}
        />
        {err && <div role="alert" style={{ color: '#dc2626', fontSize: '0.8125rem' }}>{err}</div>}
        {success && <div style={{ color: '#065f46', fontSize: '0.8125rem' }}>{success}</div>}
        <div style={{ display: 'flex', gap: 8 }}>
          <button
            onClick={() => act(() => reviewVersion(version.template_id, version.version_number, true, reason), 'Review approved')}
            disabled={busy}
            style={{ ...btnStyle, background: '#16a34a', color: '#fff' }}
          >
            Approve Review
          </button>
          <button
            onClick={() => act(() => reviewVersion(version.template_id, version.version_number, false, reason), 'Review rejected')}
            disabled={busy}
            style={{ ...btnStyle, background: '#fff', color: '#dc2626', border: '1px solid #dc2626' }}
          >
            Reject
          </button>
        </div>
      </div>
    );
  }

  if (version.status === 'approved') {
    return (
      <div style={panelStyle}>
        <strong style={{ fontSize: '0.875rem' }}>Approver actions</strong>
        <textarea
          placeholder="Reason (optional)"
          value={reason}
          onChange={(e) => setReason(e.target.value)}
          rows={2}
          style={textareaStyle}
          disabled={busy}
        />
        {err && <div role="alert" style={{ color: '#dc2626', fontSize: '0.8125rem' }}>{err}</div>}
        {success && <div style={{ color: '#065f46', fontSize: '0.8125rem' }}>{success}</div>}
        <div style={{ display: 'flex', gap: 8 }}>
          <button
            onClick={() => act(() => approveVersion(version.template_id, version.version_number, true, reason), 'Published')}
            disabled={busy}
            style={{ ...btnStyle, background: '#2563eb', color: '#fff' }}
          >
            Publish
          </button>
          <button
            onClick={() => act(() => approveVersion(version.template_id, version.version_number, false, reason), 'Rejected - back to draft')}
            disabled={busy}
            style={{ ...btnStyle, background: '#fff', color: '#dc2626', border: '1px solid #dc2626' }}
          >
            Reject
          </button>
        </div>
      </div>
    );
  }

  if (version.status === 'published') {
    return (
      <div style={{ ...panelStyle, background: '#f0fdf4', borderColor: '#bbf7d0' }}>
        <span style={{ color: '#166534', fontSize: '0.875rem' }}>This version is published</span>
      </div>
    );
  }

  return null;
}

const panelStyle: CSSProperties = {
  padding: '16px 20px',
  borderTop: '1px solid #e5e7eb',
  background: '#f9fafb',
  display: 'flex',
  flexDirection: 'column',
  gap: 10,
  flexShrink: 0,
};

const textareaStyle: CSSProperties = {
  padding: '6px 10px',
  border: '1px solid #d1d5db',
  borderRadius: 6,
  fontSize: '0.875rem',
  fontFamily: 'inherit',
  resize: 'vertical',
};

const btnStyle: CSSProperties = {
  padding: '7px 16px',
  border: 'none',
  borderRadius: 6,
  cursor: 'pointer',
  fontSize: '0.875rem',
  fontWeight: 500,
};
