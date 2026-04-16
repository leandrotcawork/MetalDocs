import { useState } from 'react';
import styles from './ExportMenu.module.css';
import { triggerExport, clientPrint, ExportError } from '../../persistence/exportApi';

interface ExportMenuProps {
  docId: string;
  editorHtml: string | null;
  disabled?: boolean;
}

export function ExportMenu({ docId, editorHtml, disabled }: ExportMenuProps) {
  const [error, setError] = useState<string | null>(null);
  const isDisabled = !editorHtml || disabled;

  async function handleExport(fmt: 'docx' | 'pdf') {
    setError(null);
    try {
      await triggerExport(docId, fmt);
    } catch (e) {
      setError(e instanceof ExportError ? `Export failed (${e.status})` : 'Export failed');
    }
  }

  function handlePrint() {
    if (editorHtml) clientPrint(editorHtml);
  }

  return (
    <div className={styles.menu}>
      <button className={styles.btn} onClick={() => handleExport('docx')} disabled={isDisabled}>
        Export DOCX
      </button>
      <button className={styles.btn} onClick={() => handleExport('pdf')} disabled={isDisabled}>
        Export PDF
      </button>
      <button className={styles.btn} onClick={handlePrint} disabled={isDisabled}>
        Print Preview
      </button>
      {error && (
        <span role="alert" className={styles.error}>
          {error}
        </span>
      )}
    </div>
  );
}
