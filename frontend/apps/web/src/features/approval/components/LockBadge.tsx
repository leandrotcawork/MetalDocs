import { useCallback } from 'react';
import styles from './LockBadge.module.css';

interface Props {
  lockedByInstanceId?: string;
  lockedByActor?: string;
  lockAcquiredAt?: string;
  onBannerClick?: () => void;
}

function relativeTime(isoUTC: string): string {
  const diff = Date.now() - new Date(isoUTC).getTime();
  const mins = Math.floor(diff / 60000);

  if (mins < 1) return 'agora mesmo';
  if (mins < 60) return `há ${mins} min`;

  const hrs = Math.floor(mins / 60);
  if (hrs < 24) return `há ${hrs}h`;

  return `há ${Math.floor(hrs / 24)} dia(s)`;
}

export function LockBadge({ lockedByInstanceId, lockedByActor, lockAcquiredAt, onBannerClick }: Props) {
  const handleClick = useCallback(() => onBannerClick?.(), [onBannerClick]);

  if (!lockedByInstanceId) return null;

  const who = lockedByActor ?? 'outro usuário';
  const when = lockAcquiredAt ? relativeTime(lockAcquiredAt) : '';

  return (
    <button
      type="button"
      className={styles.banner}
      onClick={handleClick}
      aria-label="Documento em revisão — clique para ver detalhes de aprovação"
    >
      <span className={styles.icon} aria-hidden="true">
        🔒
      </span>
      <span>
        Documento em revisão por <strong>{who}</strong>
        {when && <> • {when}</>}
      </span>
    </button>
  );
}
