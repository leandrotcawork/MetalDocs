import { type FormEvent, useEffect, useId, useRef, useState } from 'react';

import { signoff } from '../api/approvalApi';
import { ApprovalError } from '../api/mutationClient';
import styles from './SignoffDialog.module.css';

type DialogState =
  | 'idle'
  | 'submitting'
  | 'success'
  | 'error_bad_password'
  | 'error_session_expired'
  | 'error_rate_limited'
  | 'error_network'
  | 'error_server';

type Decision = 'approve' | 'reject';

const ERROR_MESSAGES: Record<Exclude<DialogState, 'idle' | 'submitting' | 'success'>, string> = {
  error_bad_password: 'Senha incorreta. Verifique e tente novamente.',
  error_session_expired: 'Sessão expirada. Autentique novamente para assinar.',
  error_rate_limited: 'Muitas tentativas. Aguarde 30 segundos antes de tentar novamente.',
  error_network: 'Erro de conexão. Verifique sua internet e tente novamente.',
  error_server: 'Erro interno do servidor. Tente novamente em instantes.',
};

interface SignoffDialogProps {
  documentId: string;
  contentHash: string;
  instanceId: string;
  onClose: () => void;
  onSuccess: () => void;
}

const FOCUSABLE_SELECTOR =
  'button:not([disabled]), [href], input:not([disabled]), textarea:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])';

export function SignoffDialog({
  documentId,
  contentHash,
  instanceId,
  onClose,
  onSuccess,
}: SignoffDialogProps) {
  const [state, setState] = useState<DialogState>('idle');
  const [decision, setDecision] = useState<Decision>('approve');
  const [reason, setReason] = useState('');
  const [password, setPassword] = useState('');
  const [reasonError, setReasonError] = useState<string | null>(null);
  const [showStaleBanner, setShowStaleBanner] = useState(false);

  const dialogRef = useRef<HTMLDivElement | null>(null);
  const previousFocusRef = useRef<HTMLElement | null>(null);
  const successTimeoutRef = useRef<number | null>(null);
  const titleId = useId();
  const isSubmitting = state === 'submitting';
  const isSuccess = state === 'success';
  const hasError = state.startsWith('error_');

  useEffect(() => {
    previousFocusRef.current = document.activeElement instanceof HTMLElement ? document.activeElement : null;

    const focusables = dialogRef.current?.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR);
    focusables?.[0]?.focus();

    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        event.preventDefault();
        onClose();
        return;
      }

      if (event.key !== 'Tab') {
        return;
      }

      const dialogNode = dialogRef.current;
      if (!dialogNode) {
        return;
      }

      const elements = Array.from(dialogNode.querySelectorAll<HTMLElement>(FOCUSABLE_SELECTOR));
      if (elements.length === 0) {
        event.preventDefault();
        return;
      }

      const first = elements[0];
      const last = elements[elements.length - 1];
      const active = document.activeElement instanceof HTMLElement ? document.activeElement : null;
      const activeInsideDialog = active ? dialogNode.contains(active) : false;

      if (event.shiftKey) {
        if (!activeInsideDialog || active === first) {
          event.preventDefault();
          last.focus();
        }
        return;
      }

      if (!activeInsideDialog || active === last) {
        event.preventDefault();
        first.focus();
      }
    };

    document.addEventListener('keydown', handleKeyDown);

    return () => {
      document.removeEventListener('keydown', handleKeyDown);
      if (successTimeoutRef.current !== null) {
        window.clearTimeout(successTimeoutRef.current);
      }
      previousFocusRef.current?.focus();
    };
  }, [onClose]);

  const mapErrorToState = (error: unknown): DialogState => {
    if (error instanceof ApprovalError) {
      if (error.status === 412 || error.code === 'conflict.stale') {
        setShowStaleBanner(true);
        return 'idle';
      }
      if (error.code === 'authn.signature_invalid') return 'error_bad_password';
      if (error.status === 401) return 'error_session_expired';
      if (error.status === 429 || error.code === 'authn.rate_limited') return 'error_rate_limited';
      return 'error_server';
    }

    if (error instanceof TypeError) {
      return 'error_network';
    }

    if (error instanceof Error && /network|failed to fetch|fetch/i.test(error.message)) {
      return 'error_network';
    }

    return 'error_server';
  };

  const handleSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (isSubmitting || isSuccess) {
      return;
    }

    setReasonError(null);
    setShowStaleBanner(false);

    const normalizedReason = reason.trim();
    if (decision === 'reject' && normalizedReason.length === 0) {
      setReasonError('Informe o motivo da rejeição.');
      return;
    }

    setState('submitting');
    try {
      await signoff(documentId, {
        decision,
        reason: decision === 'reject' ? normalizedReason : undefined,
        password,
        content_hash: contentHash,
      });

      setState('success');
      successTimeoutRef.current = window.setTimeout(() => {
        onSuccess();
        onClose();
      }, 1500);
    } catch (error) {
      setState(mapErrorToState(error));
    } finally {
      setPassword('');
    }
  };

  return (
    <div className={styles.overlay}>
      <div
        ref={dialogRef}
        className={styles.dialog}
        role="dialog"
        aria-modal="true"
        aria-labelledby={titleId}
        data-instance-id={instanceId}
      >
        <h2 id={titleId} className={styles.title}>
          Assinar aprovação
        </h2>

        {showStaleBanner ? (
          <div className={styles.staleBanner} role="status">
            Documento foi alterado. Atualize a página antes de tentar novamente.
          </div>
        ) : null}

        {hasError ? (
          <div className={styles.errorBox} role="alert">
            {ERROR_MESSAGES[state as keyof typeof ERROR_MESSAGES]}
          </div>
        ) : null}

        {isSuccess ? (
          <div className={styles.success} role="status">
            Assinatura registrada com sucesso.
          </div>
        ) : (
          <form onSubmit={handleSubmit}>
            <fieldset className={styles.fieldset}>
              <legend className={styles.legend}>Decisão</legend>
              <div className={styles.radioGroup}>
                <label className={styles.radio}>
                  <input
                    type="radio"
                    name="decision"
                    value="approve"
                    checked={decision === 'approve'}
                    onChange={() => setDecision('approve')}
                    disabled={isSubmitting}
                  />
                  Aprovado
                </label>
                <label className={styles.radio}>
                  <input
                    type="radio"
                    name="decision"
                    value="reject"
                    checked={decision === 'reject'}
                    onChange={() => setDecision('reject')}
                    disabled={isSubmitting}
                  />
                  Rejeitado
                </label>
              </div>
            </fieldset>

            <div className={styles.field}>
              <label className={styles.label} htmlFor="signoff-reason">
                Motivo
              </label>
              <textarea
                id="signoff-reason"
                className={styles.textarea}
                value={reason}
                onChange={(event) => {
                  setReason(event.target.value);
                  if (reasonError) {
                    setReasonError(null);
                  }
                }}
                disabled={isSubmitting}
                aria-invalid={reasonError ? 'true' : 'false'}
              />
              {reasonError ? (
                <div className={styles.errorBox} role="alert">
                  {reasonError}
                </div>
              ) : null}
            </div>

            <div className={styles.field}>
              <label className={styles.label} htmlFor="signoff-password">
                Senha
              </label>
              <input
                id="signoff-password"
                className={styles.input}
                type="password"
                value={password}
                onChange={(event) => setPassword(event.target.value)}
                required
                disabled={isSubmitting}
              />
            </div>

            <div className={styles.actions}>
              <button
                type="button"
                className={`${styles.btn} ${styles.btnSecondary}`}
                onClick={onClose}
                disabled={isSubmitting}
              >
                Cancelar
              </button>
              <button
                type="submit"
                className={`${styles.btn} ${styles.btnPrimary}`}
                disabled={isSubmitting}
              >
                {isSubmitting ? 'Enviando...' : 'Confirmar assinatura'}
              </button>
            </div>
          </form>
        )}
      </div>
    </div>
  );
}
