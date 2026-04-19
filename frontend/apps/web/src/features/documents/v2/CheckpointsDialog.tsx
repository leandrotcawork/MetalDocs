import { useEffect, useRef } from 'react';
import { CheckpointsPanel } from './CheckpointsPanel';
import styles from './styles/DocumentEditorPage.module.css';

export function CheckpointsDialog({
  open,
  onClose,
  documentID,
  disabled,
  onRestored,
}: {
  open: boolean;
  onClose: () => void;
  documentID: string;
  disabled: boolean;
  onRestored: (revisionID: string) => void;
}) {
  const ref = useRef<HTMLDialogElement>(null);
  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    if (open && !el.open) el.showModal();
    if (!open && el.open) el.close();
  }, [open]);
  return (
    <dialog
      ref={ref}
      className={styles.checkpointsDialog}
      onClose={onClose}
      onClick={(e) => {
        if (e.target === ref.current) onClose();
      }}
    >
      <CheckpointsPanel documentID={documentID} disabled={disabled} onRestored={onRestored} />
      <form method="dialog" style={{ marginTop: 12, textAlign: 'right' }}>
        <button>Close</button>
      </form>
    </dialog>
  );
}
