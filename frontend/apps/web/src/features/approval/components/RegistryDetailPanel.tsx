import { useCallback, useEffect, useMemo, useState } from 'react';

import { cancel, getInstance, listRoutes, submit } from '../api/approvalApi';
import { etagCache } from '../api/etagCache';
import type { ApprovalInstance, ApprovalState, Route } from '../api/approvalTypes';
import { ApprovalTimelinePanel } from './ApprovalTimelinePanel';
import { LockBadge } from './LockBadge';
import { SignoffDialog } from './SignoffDialog';
import { StateBadge } from './StateBadge';
import { SupersedePublishDialog } from './SupersedePublishDialog';
import styles from './RegistryDetailPanel.module.css';

interface RegistryDetailPanelProps {
  documentId: string;
  approvalState: string;
  contentHash: string;
  revisionVersion: number;
  lockedByInstanceId?: string;
  lockedByActor?: string;
  lockAcquiredAt?: string;
  effectiveFrom?: string;
  effectiveTo?: string;
  publishedDocumentId?: string;
}

interface TransitionPolicy {
  disabledReason?: string;
  readOnly?: boolean;
  actions: {
    submit: boolean;
    signoff: boolean;
    cancelInstance: boolean;
    publishOrSchedule: boolean;
  };
}

const TRANSITION_POLICY: Record<ApprovalState, TransitionPolicy> = {
  draft: {
    actions: { submit: true, signoff: false, cancelInstance: false, publishOrSchedule: false },
  },
  under_review: {
    disabledReason: 'Documento em revisão — edição bloqueada',
    actions: { submit: false, signoff: true, cancelInstance: true, publishOrSchedule: false },
  },
  approved: {
    actions: { submit: false, signoff: false, cancelInstance: false, publishOrSchedule: true },
  },
  scheduled: {
    disabledReason: 'Aguardando data de vigência agendada',
    readOnly: true,
    actions: { submit: false, signoff: false, cancelInstance: false, publishOrSchedule: false },
  },
  published: {
    actions: { submit: false, signoff: false, cancelInstance: false, publishOrSchedule: true },
  },
  superseded: {
    disabledReason: 'Versão substituída — somente leitura',
    readOnly: true,
    actions: { submit: false, signoff: false, cancelInstance: false, publishOrSchedule: false },
  },
  rejected: {
    disabledReason: 'Documento rejeitado — edite e submeta novamente',
    actions: { submit: false, signoff: false, cancelInstance: false, publishOrSchedule: false },
  },
  obsolete: {
    disabledReason: 'Documento obsoleto — somente leitura',
    readOnly: true,
    actions: { submit: false, signoff: false, cancelInstance: false, publishOrSchedule: false },
  },
  cancelled: {
    disabledReason: 'Aprovação cancelada',
    readOnly: true,
    actions: { submit: false, signoff: false, cancelInstance: false, publishOrSchedule: false },
  },
};

function formatDate(iso?: string): string {
  if (!iso) {
    return '';
  }
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) {
    return '-';
  }
  return date.toLocaleDateString('pt-BR');
}

function toApprovalState(status: string): ApprovalState {
  const allowed: ApprovalState[] = [
    'draft',
    'under_review',
    'approved',
    'scheduled',
    'published',
    'superseded',
    'rejected',
    'obsolete',
    'cancelled',
  ];
  if (allowed.includes(status as ApprovalState)) {
    return status as ApprovalState;
  }
  return 'draft';
}

export function RegistryDetailPanel({
  documentId,
  approvalState,
  contentHash,
  revisionVersion,
  lockedByInstanceId,
  lockedByActor,
  lockAcquiredAt,
  effectiveFrom,
  effectiveTo,
  publishedDocumentId,
}: RegistryDetailPanelProps) {
  const [instance, setInstance] = useState<ApprovalInstance | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [lastFetchedAt, setLastFetchedAt] = useState<number | null>(null);
  const [now, setNow] = useState(() => Date.now());

  const [showSubmitSection, setShowSubmitSection] = useState(false);
  const [routes, setRoutes] = useState<Route[]>([]);
  const [routesLoading, setRoutesLoading] = useState(false);
  const [routesError, setRoutesError] = useState<string | null>(null);
  const [selectedRouteId, setSelectedRouteId] = useState('');
  const [submitting, setSubmitting] = useState(false);
  const [submitError, setSubmitError] = useState<string | null>(null);

  const [showSignoffDialog, setShowSignoffDialog] = useState(false);
  const [showPublishDialog, setShowPublishDialog] = useState(false);
  const [copied, setCopied] = useState(false);

  const fetchInstance = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const next = await getInstance(documentId);
      setInstance(next);
      setLastFetchedAt(Date.now());
    } catch (err) {
      if ((err as { status?: number }).status === 404) {
        setInstance(null);
        setLastFetchedAt(Date.now());
        if (!etagCache.get(documentId)) etagCache.set(documentId, '"v0"');
      } else {
        setError('Erro ao carregar dados de aprovação.');
        setInstance(null);
      }
    } finally {
      setLoading(false);
    }
  }, [documentId]);

  useEffect(() => {
    void fetchInstance();
  }, [fetchInstance]);

  useEffect(() => {
    const timer = window.setInterval(() => {
      setNow(Date.now());
    }, 1000);
    return () => window.clearInterval(timer);
  }, []);

  const isStale = useMemo(() => {
    if (lastFetchedAt == null) {
      return false;
    }
    return now - lastFetchedAt > 30_000;
  }, [lastFetchedAt, now]);

  const status = toApprovalState(approvalState);
  const policy = TRANSITION_POLICY[status];
  const etag = etagCache.get(documentId) ?? '-';
  const shortHash = contentHash.length > 8 ? `${contentHash.slice(0, 8)}…` : contentHash;

  const handleCopyHash = async () => {
    try {
      await navigator.clipboard.writeText(contentHash);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1200);
    } catch (_error) {
      setCopied(false);
    }
  };

  const scrollToTimeline = () => {
    document.getElementById('approval-timeline')?.scrollIntoView({ behavior: 'smooth', block: 'start' });
  };

  const openSubmitSection = async () => {
    setShowSubmitSection(true);
    setRoutesLoading(true);
    setRoutesError(null);
    setSubmitError(null);
    try {
      const response = await listRoutes();
      const activeRoutes = response.routes.filter((route) => route.active);
      setRoutes(activeRoutes);
      setSelectedRouteId((prev) => prev || activeRoutes[0]?.id || '');
    } catch (_error) {
      setRoutesError('Erro ao carregar rotas.');
    } finally {
      setRoutesLoading(false);
    }
  };

  const handleSubmitForReview = async () => {
    if (!selectedRouteId || submitting) {
      return;
    }
    setSubmitting(true);
    setSubmitError(null);
    try {
      await submit(documentId, {
        route_id: selectedRouteId,
        content_hash: contentHash,
      });
      setShowSubmitSection(false);
      await fetchInstance();
    } catch (_error) {
      setSubmitError('Erro ao submeter para revisão.');
    } finally {
      setSubmitting(false);
    }
  };

  const handleCancelInstance = async () => {
    const reason = window.prompt('Motivo do cancelamento da instância:');
    if (!reason || !reason.trim()) {
      return;
    }
    try {
      await cancel(documentId, { reason: reason.trim() });
      await fetchInstance();
    } catch (_error) {
      setError('Erro ao cancelar instância.');
    }
  };

  if (loading) {
    return <div className={styles.state}>Carregando painel de aprovação...</div>;
  }

  if (error) {
    return (
      <div className={styles.state} role="alert">
        <p>{error}</p>
        <button type="button" className={styles.actionButton} onClick={() => void fetchInstance()}>
          Tentar novamente
        </button>
      </div>
    );
  }

  return (
    <section className={styles.panel} aria-label="Painel de detalhes de aprovação">
      <div className={styles.topRow}>
        <LockBadge
          lockedByInstanceId={lockedByInstanceId}
          lockedByActor={lockedByActor}
          lockAcquiredAt={lockAcquiredAt}
          onBannerClick={scrollToTimeline}
        />
        <StateBadge state={status} />
      </div>

      {(effectiveFrom || effectiveTo) && (
        <div className={styles.section}>
          <strong>Vigência:</strong> {formatDate(effectiveFrom)} → {formatDate(effectiveTo)}
        </div>
      )}

      <div className={styles.section}>
        <h3>Integridade</h3>
        <div className={styles.integrityGrid}>
          <div>
            <span className={styles.label}>content_hash</span>
            <div className={styles.hashRow}>
              <code title={contentHash}>{shortHash}</code>
              <button type="button" className={styles.smallButton} onClick={() => void handleCopyHash()}>
                {copied ? 'Copiado' : 'Copiar'}
              </button>
            </div>
          </div>
          <div>
            <span className={styles.label}>revision_version</span>
            <code>{revisionVersion}</code>
          </div>
          <div>
            <span className={styles.label}>ETag</span>
            <code>{etag}</code>
          </div>
        </div>
      </div>

      {isStale ? (
        <div className={styles.staleBanner} role="status">
          <span>Dados podem estar desatualizados.</span>
          <button type="button" className={styles.smallButton} onClick={() => void fetchInstance()}>
            Atualizar
          </button>
        </div>
      ) : null}

      <div className={styles.section}>
        <h3>Ações</h3>
        <div className={styles.actions}>
          {policy.actions.submit ? (
            <button type="button" className={styles.actionButton} onClick={() => void openSubmitSection()}>
              Submeter para revisão
            </button>
          ) : null}

          {policy.actions.signoff && lockedByInstanceId ? (
            <button
              type="button"
              className={styles.actionButton}
              onClick={() => setShowSignoffDialog(true)}
              disabled={!instance}
            >
              Assinar
            </button>
          ) : null}

          {policy.actions.cancelInstance ? (
            <button type="button" className={styles.actionButtonSecondary} onClick={() => void handleCancelInstance()}>
              Cancelar instância
            </button>
          ) : null}

          {policy.actions.publishOrSchedule ? (
            <button type="button" className={styles.actionButton} onClick={() => setShowPublishDialog(true)}>
              Publicar / Agendar
            </button>
          ) : null}

          {policy.disabledReason ? <p className={styles.disabledReason}>{policy.disabledReason}</p> : null}

          {policy.readOnly ? <p className={styles.readOnlyTag}>Somente leitura</p> : null}
        </div>

        {showSubmitSection ? (
          <div className={styles.submitSection}>
            <h4>Submeter para revisão</h4>
            {routesLoading ? <p>Carregando rotas...</p> : null}
            {routesError ? <p role="alert">{routesError}</p> : null}
            {!routesLoading && !routesError ? (
              <>
                <label htmlFor="route-select" className={styles.label}>
                  Rota
                </label>
                <select
                  id="route-select"
                  className={styles.select}
                  value={selectedRouteId}
                  onChange={(event) => setSelectedRouteId(event.target.value)}
                >
                  {routes.map((route) => (
                    <option key={route.id} value={route.id}>
                      {route.name}
                    </option>
                  ))}
                </select>
                {routes.length === 0 ? <p>Nenhuma rota configurada.</p> : null}
                {submitError ? <p role="alert">{submitError}</p> : null}
                <div className={styles.inlineActions}>
                  <button
                    type="button"
                    className={styles.actionButton}
                    onClick={() => void handleSubmitForReview()}
                    disabled={!selectedRouteId || submitting}
                  >
                    {submitting ? 'Enviando...' : 'Submeter'}
                  </button>
                  <button
                    type="button"
                    className={styles.actionButtonSecondary}
                    onClick={() => setShowSubmitSection(false)}
                  >
                    Cancelar
                  </button>
                </div>
              </>
            ) : null}
          </div>
        ) : null}
      </div>

      {instance ? (
        <section id="approval-timeline" className={styles.section}>
          <ApprovalTimelinePanel instance={instance} loading={false} />
        </section>
      ) : null}

      {showSignoffDialog && instance ? (
        <SignoffDialog
          documentId={documentId}
          contentHash={contentHash}
          instanceId={instance.id}
          onClose={() => setShowSignoffDialog(false)}
          onSuccess={() => void fetchInstance()}
        />
      ) : null}

      {showPublishDialog ? (
        <SupersedePublishDialog
          documentId={documentId}
          contentHash={contentHash}
          publishedDocumentId={publishedDocumentId}
          onClose={() => setShowPublishDialog(false)}
          onSuccess={() => void fetchInstance()}
        />
      ) : null}
    </section>
  );
}

export { TRANSITION_POLICY };

