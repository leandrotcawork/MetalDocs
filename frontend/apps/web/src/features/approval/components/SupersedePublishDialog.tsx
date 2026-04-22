import { type FormEvent, useMemo, useState } from 'react';

import { publish, schedulePublish, supersede } from '../api/approvalApi';
import styles from './SupersedePublishDialog.module.css';

type PublishMode = 'now' | 'schedule';
type DialogState = 'idle' | 'submitting' | 'success' | 'error';

interface SupersedePublishDialogProps {
  documentId: string;
  contentHash: string;
  publishedDocumentId?: string;
  onClose: () => void;
  onSuccess: () => void;
}

function pad(value: number): string {
  return String(value).padStart(2, '0');
}

function toDateTimeLocalValue(date: Date): string {
  const year = date.getFullYear();
  const month = pad(date.getMonth() + 1);
  const day = pad(date.getDate());
  const hours = pad(date.getHours());
  const minutes = pad(date.getMinutes());
  return `${year}-${month}-${day}T${hours}:${minutes}`;
}

export function SupersedePublishDialog({
  documentId,
  contentHash,
  publishedDocumentId,
  onClose,
  onSuccess,
}: SupersedePublishDialogProps) {
  const [mode, setMode] = useState<PublishMode>('now');
  const [scheduledAt, setScheduledAt] = useState('');
  const [replacePublished, setReplacePublished] = useState(false);
  const [state, setState] = useState<DialogState>('idle');
  const [errorMessage, setErrorMessage] = useState<string | null>(null);

  const minScheduleDate = useMemo(() => {
    const date = new Date(Date.now() + 5 * 60 * 1000);
    return toDateTimeLocalValue(date);
  }, []);

  const isSubmitting = state === 'submitting';
  const isSuccess = state === 'success';
  const scheduleValidationMessage =
    mode === 'schedule' && scheduledAt
      ? (() => {
          const scheduledDate = new Date(scheduledAt);
          const minDateMs = Date.now() + 5 * 60 * 1000;
          if (Number.isNaN(scheduledDate.getTime()) || scheduledDate.getTime() < minDateMs) {
            return 'A data deve ser pelo menos 5 minutos no futuro.';
          }
          return null;
        })()
      : null;
  const submitDisabled = isSubmitting || (mode === 'schedule' && (!scheduledAt || scheduleValidationMessage !== null));

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (isSubmitting) {
      return;
    }

    setErrorMessage(null);
    setState('submitting');

    try {
      if (mode === 'schedule') {
        if (!scheduledAt || scheduleValidationMessage !== null) {
          setState('idle');
          return;
        }
        const scheduledDate = new Date(scheduledAt);
        await schedulePublish(documentId, {
          content_hash: contentHash,
          effective_from: scheduledDate.toISOString(),
        });
      } else if (publishedDocumentId && replacePublished) {
        await supersede(documentId, {
          content_hash: contentHash,
          supersedes_document_id: publishedDocumentId,
        });
      } else {
        await publish(documentId, {
          content_hash: contentHash,
        });
      }

      setState('success');
      onSuccess();
      onClose();
    } catch (_error) {
      setState('error');
      setErrorMessage('Não foi possível concluir a publicação. Tente novamente.');
    }
  };

  return (
    <div className={styles.overlay}>
      <div className={styles.dialog} role="dialog" aria-modal="true" aria-label="Publicação">
        <h2 className={styles.title}>Publicação</h2>

        {state === 'error' && errorMessage ? (
          <div className={styles.errorBox} role="alert">
            {errorMessage}
          </div>
        ) : null}

        {isSuccess ? (
          <div className={styles.success} role="status">
            Publicação concluída com sucesso.
          </div>
        ) : (
          <form onSubmit={handleSubmit}>
            <fieldset className={styles.fieldset}>
              <legend className={styles.legend}>Modo</legend>
              <label className={styles.radio}>
                <input
                  type="radio"
                  name="publish-mode"
                  value="now"
                  checked={mode === 'now'}
                  onChange={() => setMode('now')}
                  disabled={isSubmitting}
                />
                Publicar agora
              </label>
              <label className={styles.radio}>
                <input
                  type="radio"
                  name="publish-mode"
                  value="schedule"
                  checked={mode === 'schedule'}
                  onChange={() => setMode('schedule')}
                  disabled={isSubmitting}
                />
                Agendar publicação
              </label>
            </fieldset>

            {mode === 'schedule' ? (
              <div className={styles.field}>
                <label className={styles.label} htmlFor="publish-scheduled-at">
                  Data e hora da publicação
                </label>
                <input
                  id="publish-scheduled-at"
                  type="datetime-local"
                  className={styles.input}
                  min={minScheduleDate}
                  value={scheduledAt}
                  onChange={(event) => setScheduledAt(event.target.value)}
                  disabled={isSubmitting}
                />
                {scheduleValidationMessage ? (
                  <p className={styles.inlineError} role="alert">
                    {scheduleValidationMessage}
                  </p>
                ) : null}
              </div>
            ) : null}

            {mode === 'now' && publishedDocumentId ? (
              <label className={styles.checkbox}>
                <input
                  type="checkbox"
                  checked={replacePublished}
                  onChange={(event) => setReplacePublished(event.target.checked)}
                  disabled={isSubmitting}
                />
                Substituir versão publicada atual
              </label>
            ) : null}

            <div className={styles.actions}>
              <button
                type="button"
                className={`${styles.btn} ${styles.btnSecondary}`}
                onClick={onClose}
                disabled={isSubmitting}
              >
                Cancelar
              </button>
              <button type="submit" className={`${styles.btn} ${styles.btnPrimary}`} disabled={submitDisabled}>
                {isSubmitting ? 'Enviando...' : 'Confirmar publicação'}
              </button>
            </div>
          </form>
        )}
      </div>
    </div>
  );
}

