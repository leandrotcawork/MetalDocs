import type { ApprovalInstance, Signoff, StageInstance } from '../api/approvalTypes';
import styles from './ApprovalTimelinePanel.module.css';

interface ApprovalTimelinePanelProps {
  instance: ApprovalInstance | null;
  loading: boolean;
  error?: string | null;
  onRetry?: () => void;
}

const STAGE_STATUS_LABEL: Record<StageInstance['status'], string> = {
  pending: 'Pendente',
  active: 'Em andamento',
  passed: 'Aprovado',
  failed: 'Reprovado',
  cancelled: 'Cancelado',
};

const DECISION_LABEL: Record<Signoff['decision'], string> = {
  approve: 'Aprovou',
  reject: 'Rejeitou',
};

function formatDateTime(iso?: string): string {
  if (!iso) {
    return '-';
  }
  return new Date(iso).toLocaleString('pt-BR');
}

function formatSignatureMethod(signatureMethod: Signoff['signature_method']): string {
  if (signatureMethod === 'password_reauth') {
    return 'Reautenticação por senha';
  }
  return 'ICP-Brasil';
}

export function ApprovalTimelinePanel({ instance, loading, error, onRetry }: ApprovalTimelinePanelProps) {
  if (loading) {
    return <div className={styles.state}>Carregando timeline...</div>;
  }

  if (error) {
    return (
      <div className={styles.state} role="alert">
        <p>{error}</p>
        <button type="button" onClick={onRetry} disabled={!onRetry}>
          Tentar novamente
        </button>
      </div>
    );
  }

  if (!instance) {
    return <div className={styles.state}>Nenhum evento de aprovação registrado.</div>;
  }

  return (
    <section className={styles.panel} aria-label="Timeline de aprovação">
      <ol className={styles.timeline}>
        <li className={styles.node}>
          <div className={styles.dot} aria-hidden="true" />
          <div className={styles.content}>
            <h3 className={styles.title}>Submetido</h3>
            <p className={styles.meta}>
              Por <strong>{instance.submitted_by}</strong> em {formatDateTime(instance.submitted_at)}
            </p>
          </div>
        </li>

        {instance.stages
          .slice()
          .sort((a, b) => a.stage_index - b.stage_index)
          .map((stage) => (
            <li className={styles.node} key={stage.id}>
              <div className={styles.dot} aria-hidden="true" />
              <div className={styles.content}>
                <div className={styles.stageHeader}>
                  <h3 className={styles.title}>{stage.label}</h3>
                  <span className={`${styles.stageStatus} ${styles[`stage_${stage.status}`]}`}>
                    {STAGE_STATUS_LABEL[stage.status]}
                  </span>
                </div>
                {stage.signoffs.length === 0 ? (
                  <p className={styles.meta}>Sem assinaturas registradas.</p>
                ) : (
                  <ul className={styles.signoffs}>
                    {stage.signoffs.map((signoff) => (
                      <li className={styles.signoff} key={signoff.id}>
                        <p>
                          <strong>{signoff.actor_user_id}</strong> - {DECISION_LABEL[signoff.decision]}
                        </p>
                        <p className={styles.meta}>
                          Assinatura: {formatSignatureMethod(signoff.signature_method)} | Em:{' '}
                          {formatDateTime(signoff.signed_at)}
                        </p>
                        {signoff.reason ? <p className={styles.meta}>Motivo: {signoff.reason}</p> : null}
                      </li>
                    ))}
                  </ul>
                )}
              </div>
            </li>
          ))}

        {instance.status === 'completed' || instance.status === 'cancelled' ? (
          <li className={styles.node}>
            <div className={styles.dot} aria-hidden="true" />
            <div className={styles.content}>
              <h3 className={styles.title}>Status final</h3>
              <p className={styles.meta}>
                {instance.status === 'completed' ? 'Concluído' : 'Cancelado'} em{' '}
                {formatDateTime(instance.completed_at)}
              </p>
            </div>
          </li>
        ) : null}
      </ol>
    </section>
  );
}
