import { useState } from 'react';
import {
  approveTemplate,
  submitTemplateForReview,
  type TemplateDraftStatus,
} from '../../persistence/templatePublishApi';
import styles from './ExportMenu.module.css';

interface PublishButtonProps {
  templateKey: string;
  draftStatus: TemplateDraftStatus;
  onStatusChange: (newStatus: TemplateDraftStatus) => void;
}

export function PublishButton({ templateKey, draftStatus, onStatusChange }: PublishButtonProps) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function handleClick() {
    if (loading || draftStatus === 'published') return;
    setError(null);
    setLoading(true);
    try {
      if (draftStatus === 'draft') {
        await submitTemplateForReview(templateKey);
        onStatusChange('pending_review');
      } else {
        await approveTemplate(templateKey);
        onStatusChange('published');
      }
    } catch {
      setError('Publish action failed');
    } finally {
      setLoading(false);
    }
  }

  const disabled = loading || draftStatus === 'published';
  const label = loading
    ? '...'
    : draftStatus === 'draft'
      ? 'Submit for Review'
      : draftStatus === 'pending_review'
        ? 'Approve & Publish'
        : 'Published';

  return (
    <div style={{ display: 'flex', alignItems: 'center', gap: 8 }}>
      <button className={styles.btn} onClick={handleClick} disabled={disabled}>
        {label}
      </button>
      {error && (
        <span role="alert" className={styles.error}>
          {error}
        </span>
      )}
    </div>
  );
}
