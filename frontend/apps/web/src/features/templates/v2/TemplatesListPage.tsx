import { type CSSProperties, useCallback, useEffect, useState } from 'react';
import { listTemplates, type TemplateDTO } from './api/templatesV2';
import styles from './TemplatesListPage.module.css';

export type TemplatesListPageProps = {
  onOpenTemplate: (templateId: string, versionNum: number) => void;
  onCreate: () => void;
};

type LoadState = 'loading' | 'ready' | 'error';

function getTemplateStatus(template: TemplateDTO): 'Draft' | 'Published' | 'Archived' {
  if (template.archived_at) return 'Archived';
  if (template.published_version_id) return 'Published';
  return 'Draft';
}

function getBadgeStyle(status: 'Draft' | 'Published' | 'Archived'): CSSProperties {
  if (status === 'Published') return { background: '#d1fae5', color: '#065f46' };
  if (status === 'Archived') return { background: '#f3f4f6', color: '#6b7280' };
  return { background: '#fef3c7', color: '#92400e' };
}

export function TemplatesListPage({ onOpenTemplate, onCreate }: TemplatesListPageProps) {
  const [state, setState] = useState<LoadState>('loading');
  const [templates, setTemplates] = useState<TemplateDTO[]>([]);
  const [error, setError] = useState<string | null>(null);

  const reload = useCallback(async () => {
    setState('loading');
    setError(null);
    try {
      const { templates: rows } = await listTemplates();
      setTemplates(rows);
      setState('ready');
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
      setState('error');
    }
  }, []);

  useEffect(() => {
    void reload();
  }, [reload]);

  if (state === 'loading') {
    return <div style={{ padding: 24 }}>Loading templates…</div>;
  }

  if (state === 'error') {
    return (
      <div className={styles.page}>
        <div role="alert" className={styles.alert}>
          <span>{error || 'Failed to load templates.'}</span>
          <button type="button" className={styles.retryBtn} onClick={() => void reload()}>
            Retry
          </button>
        </div>
      </div>
    );
  }

  if (templates.length === 0) {
    return (
      <div className={styles.page}>
        <div className={styles.empty}>
          <p>No templates yet.</p>
          <button type="button" className={styles.newBtn} onClick={onCreate}>
            New Template
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className={styles.page}>
      <div className={styles.header}>
        <h2>Templates</h2>
        <button type="button" className={styles.newBtn} onClick={onCreate}>
          New Template
        </button>
      </div>

      <table className={styles.table}>
        <thead>
          <tr>
            <th>Name</th>
            <th>Key</th>
            <th>Doc Type</th>
            <th>Visibility</th>
            <th>Status</th>
            <th>Latest Version</th>
            <th>Actions</th>
          </tr>
        </thead>
        <tbody>
          {templates.map((t) => {
            const status = getTemplateStatus(t);
            return (
              <tr key={t.id}>
                <td>{t.name}</td>
                <td>{t.key}</td>
                <td>{t.doc_type_code || '—'}</td>
                <td>{t.visibility}</td>
                <td>
                  <span className={styles.badge} style={getBadgeStyle(status)}>
                    {status}
                  </span>
                </td>
                <td>v{t.latest_version}</td>
                <td>
                  <button
                    type="button"
                    className={styles.openBtn}
                    onClick={() => onOpenTemplate(t.id, t.latest_version)}
                  >
                    Open
                  </button>
                </td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
