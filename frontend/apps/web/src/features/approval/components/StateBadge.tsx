import type { ApprovalState } from '../api/approvalTypes';
import styles from './StateBadge.module.css';

interface Props {
  state: ApprovalState;
  size?: 'sm' | 'md';
}

const STATE_CONFIG: Record<ApprovalState, { label: string; color: string; icon: string }> = {
  draft: { label: 'Rascunho', color: 'gray', icon: '✏️' },
  under_review: { label: 'Em revisão', color: 'blue', icon: '🔍' },
  approved: { label: 'Aprovado', color: 'green', icon: '✅' },
  scheduled: { label: 'Agendado', color: 'purple', icon: '🕐' },
  published: { label: 'Publicado', color: 'teal', icon: '📢' },
  superseded: { label: 'Substituído', color: 'orange', icon: '🔄' },
  rejected: { label: 'Rejeitado', color: 'red', icon: '❌' },
  obsolete: { label: 'Obsoleto', color: 'brown', icon: '🗄️' },
  cancelled: { label: 'Cancelado', color: 'neutral', icon: '⛔' },
};

export function StateBadge({ state, size = 'md' }: Props) {
  const cfg = STATE_CONFIG[state];

  return (
    <span
      className={`${styles.badge} ${styles[cfg.color]} ${styles[size]}`}
      aria-label={`Estado: ${cfg.label}`}
      data-state={state}
    >
      <span aria-hidden="true">{cfg.icon}</span>
      {cfg.label}
    </span>
  );
}

export { STATE_CONFIG };
