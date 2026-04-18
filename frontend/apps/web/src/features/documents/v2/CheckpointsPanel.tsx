import { useCallback, useEffect, useState } from 'react';
import { listCheckpoints, createCheckpoint, restoreCheckpoint, type Checkpoint } from './api/documentsV2';

export type CheckpointsPanelProps = {
  documentID: string;
  onRestored: (newRevisionID: string) => void;
  disabled: boolean;
};

export function CheckpointsPanel({ documentID, onRestored, disabled }: CheckpointsPanelProps): React.ReactElement {
  const [items, setItems] = useState<Checkpoint[]>([]);
  const [label, setLabel] = useState('');
  const [error, setError] = useState('');
  const [busy, setBusy] = useState(false);

  const refresh = useCallback(async () => {
    try {
      const next = await listCheckpoints(documentID);
      setItems(next);
      setError('');
    } catch {
      setError('Failed to load checkpoints.');
    }
  }, [documentID]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  async function handleCreate() {
    if (disabled || !label.trim()) return;
    setBusy(true);
    setError('');
    try {
      await createCheckpoint(documentID, label.trim());
      setLabel('');
      await refresh();
    } catch (e: any) {
      if (e?.status === 403) {
        setError('Session error. Reacquire writer session.');
      } else if (e?.status === 404) {
        setError('Document not found.');
      } else {
        setError('Failed to create checkpoint.');
      }
    } finally {
      setBusy(false);
    }
  }

  async function handleRestore(c: Checkpoint) {
    if (disabled) return;
    if (!window.confirm(`Restore checkpoint v${c.VersionNum}?`)) return;
    setBusy(true);
    setError('');
    try {
      const res = await restoreCheckpoint(documentID, c.VersionNum);
      onRestored(res.new_revision_id);
      await refresh();
    } catch (e: any) {
      if (e?.status === 403) {
        setError('Session error. Reacquire writer session.');
      } else if (e?.status === 404) {
        setError('Checkpoint not found.');
      } else {
        setError('Failed to restore checkpoint.');
      }
    } finally {
      setBusy(false);
    }
  }

  return (
    <aside data-checkpoints-panel>
      <h3>Checkpoints</h3>
      <label>
        Label
        <input
          type="text"
          value={label}
          onChange={(event) => setLabel(event.target.value)}
          disabled={disabled || busy}
        />
      </label>
      <button type="button" onClick={() => void handleCreate()} disabled={disabled || busy || !label.trim()}>
        Create
      </button>
      {error && (
        <p role="alert" data-checkpoint-error>
          {error}
        </p>
      )}
      <ul>
        {items.map((checkpoint) => (
          <li key={checkpoint.ID}>
            <span>v{checkpoint.VersionNum}</span>
            <span>{checkpoint.Label}</span>
            <button
              type="button"
              data-restore-version={checkpoint.VersionNum}
              disabled={disabled || busy}
              onClick={() => void handleRestore(checkpoint)}
            >
              Restore
            </button>
          </li>
        ))}
      </ul>
    </aside>
  );
}
