import { useCallback, useEffect, useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { listInbox } from '../api/approvalApi';
import type { InboxItem } from '../api/approvalTypes';
import styles from './InboxPage.module.css';

const PAGE_SIZE = 20;
const AREA_OPTIONS = ['', 'JUR', 'RH', 'FIN', 'TI', 'COM', 'ENG'];

function formatRelativeTime(iso: string): string {
  const diffMs = Date.now() - new Date(iso).getTime();
  const minutes = Math.max(0, Math.floor(diffMs / 60000));

  if (minutes < 1) return 'há instantes';
  if (minutes < 60) return `há ${minutes} min`;

  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `há ${hours}h`;

  const days = Math.floor(hours / 24);
  return `há ${days} dia(s)`;
}

function isOlderThanSevenDays(iso: string): boolean {
  const submittedAt = new Date(iso).getTime();
  const sevenDaysMs = 7 * 24 * 60 * 60 * 1000;
  return Date.now() - submittedAt > sevenDaysMs;
}

export function InboxPage() {
  const navigate = useNavigate();
  const [items, setItems] = useState<InboxItem[]>([]);
  const [total, setTotal] = useState(0);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [areaFilter, setAreaFilter] = useState('');
  const [onlyOverdue, setOnlyOverdue] = useState(false);
  const [page, setPage] = useState(0);

  const reload = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const response = await listInbox({
        area_code: areaFilter || undefined,
        offset: page * PAGE_SIZE,
        limit: PAGE_SIZE,
      });
      setItems(response.items);
      setTotal(response.total);
    } catch (_error) {
      setError('Erro ao carregar caixa de entrada. Tente novamente.');
    } finally {
      setLoading(false);
    }
  }, [areaFilter, page]);

  useEffect(() => {
    void reload();
  }, [reload, onlyOverdue]);

  const rows = useMemo(
    () => (onlyOverdue ? items.filter((item) => isOlderThanSevenDays(item.submitted_at)) : items),
    [items, onlyOverdue],
  );

  const canGoPrev = page > 0;
  const canGoNext = (page + 1) * PAGE_SIZE < total;

  return (
    <section className={styles.page}>
      <header className={styles.header}>
        <h1>Caixa de Entrada de Aprovação</h1>
        <div className={styles.actions}>
          <button type="button" onClick={() => void reload()}>
            Atualizar
          </button>
        </div>
      </header>

      <div className={styles.filters}>
        <label htmlFor="area-filter">
          Área
          <select
            id="area-filter"
            value={areaFilter}
            onChange={(event) => {
              setAreaFilter(event.target.value);
              setPage(0);
            }}
          >
            {AREA_OPTIONS.map((area) => (
              <option key={area || 'all'} value={area}>
                {area || 'Todas'}
              </option>
            ))}
          </select>
        </label>

        <button
          type="button"
          className={onlyOverdue ? styles.toggleOn : styles.toggleOff}
          onClick={() => {
            setOnlyOverdue((prev) => !prev);
            setPage(0);
          }}
        >
          Apenas atrasados
        </button>
      </div>

      {loading ? <div className={styles.state}>Carregando...</div> : null}

      {!loading && error ? (
        <div className={styles.state} role="alert">
          <p>{error}</p>
          <button type="button" onClick={() => void reload()}>
            Tentar novamente
          </button>
        </div>
      ) : null}

      {!loading && !error && rows.length === 0 ? (
        <div className={styles.state}>Nada pendente para revisão.</div>
      ) : null}

      {!loading && !error && rows.length > 0 ? (
        <div className={styles.tableWrap}>
          <table className={styles.table}>
            <thead>
              <tr>
                <th>Título do Documento</th>
                <th>Área</th>
                <th>Submetido por</th>
                <th>Há quanto tempo</th>
                <th>Estágio</th>
                <th>Progresso de Quórum</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((item) => (
                <tr
                  key={item.instance_id}
                  className={styles.clickableRow}
                  onClick={() => navigate(`/documents/${item.document_id}`)}
                  onKeyDown={(event) => {
                    if (event.key === 'Enter' || event.key === ' ') {
                      event.preventDefault();
                      navigate(`/documents/${item.document_id}`);
                    }
                  }}
                  tabIndex={0}
                >
                  <td>{item.document_title}</td>
                  <td>{item.area_code}</td>
                  <td>{item.submitted_by}</td>
                  <td>{formatRelativeTime(item.submitted_at)}</td>
                  <td>{item.stage_label}</td>
                  <td>{item.quorum_progress}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      ) : null}

      <footer className={styles.pagination}>
        <button type="button" disabled={!canGoPrev} onClick={() => setPage((prev) => prev - 1)}>
          Anterior
        </button>
        <span>Página {page + 1}</span>
        <button type="button" disabled={!canGoNext} onClick={() => setPage((prev) => prev + 1)}>
          Próxima
        </button>
      </footer>
    </section>
  );
}
