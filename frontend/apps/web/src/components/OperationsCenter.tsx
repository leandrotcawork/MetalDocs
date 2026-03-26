import { useMemo, useState } from "react";
import { metalNobreProcessAreaHint } from "../features/documents/adapters/metalNobreExperience";
import { formatDocumentDisplayName } from "../features/shared/documentDisplay";
import type { DocumentProfileItem, NotificationItem, ProcessAreaItem, SearchDocumentItem } from "../lib.types";
import { WorkspaceDataState } from "./WorkspaceDataState";
import styles from "./OperationsCenter.module.css";
import { TimelineRail } from "./ui/TimelineRail";
import { WorkspaceHeroHeader } from "./ui/WorkspaceHeroHeader";

type LoadState = "idle" | "loading" | "ready" | "error";

type OperationsCenterProps = {
  loadState: LoadState;
  documents: SearchDocumentItem[];
  notifications: NotificationItem[];
  documentProfiles: DocumentProfileItem[];
  processAreas: ProcessAreaItem[];
  formatDate: (value?: string) => string;
  onRefreshWorkspace: () => void | Promise<void>;
  onOpenDocument: (documentId: string) => void | Promise<void>;
};

export function OperationsCenter(props: OperationsCenterProps) {
  const [searchQuery, setSearchQuery] = useState("");
  const normalizedQuery = searchQuery.trim().toLowerCase();
  const filteredDocuments = useMemo(() => {
    if (!normalizedQuery) return props.documents;
    return props.documents.filter((item) => {
      const haystack = [
        item.title,
        item.documentCode,
        item.documentId,
        item.documentProfile,
        item.processArea,
        item.department,
        item.ownerId,
      ].join(" ").toLowerCase();
      return haystack.includes(normalizedQuery);
    });
  }, [normalizedQuery, props.documents]);

  const hasOperationalData = filteredDocuments.length > 0 || props.notifications.length > 0;
  const profileNameByCode = new Map(props.documentProfiles.map((item) => [item.code, item.name]));
  const areaNameByCode = new Map(props.processAreas.map((item) => [item.code, item.name]));
  const pendingReviews = filteredDocuments.filter((item) => item.status === "IN_REVIEW");
  const recentDocuments = [...filteredDocuments]
    .sort((left, right) => new Date(right.createdAt).getTime() - new Date(left.createdAt).getTime())
    .slice(0, 5);
  const unreadNotifications = props.notifications.filter((item) => item.status !== "READ");
  const expiringSoon = filteredDocuments
    .filter((item) => {
      if (!item.expiryAt) return false;
      const diff = new Date(item.expiryAt).getTime() - Date.now();
      return diff > 0 && diff <= 1000 * 60 * 60 * 24 * 30;
    })
    .slice(0, 5);
  const nextExpiryDays = expiringSoon
    .map((item) => item.expiryAt)
    .filter((value): value is string => Boolean(value))
    .map((value) => Math.ceil((new Date(value).getTime() - Date.now()) / (1000 * 60 * 60 * 24)))
    .filter((value) => value >= 0)
    .sort((left, right) => left - right)[0] ?? 0;
  const areaCountByCode = new Map<string, number>();
  for (const document of filteredDocuments) {
    const areaCode = (document.processArea ?? "").trim().toLowerCase();
    if (areaCode === "") {
      continue;
    }
    areaCountByCode.set(areaCode, (areaCountByCode.get(areaCode) ?? 0) + 1);
  }
  const processAreaSnapshot = Array.from(areaCountByCode.entries())
    .map(([code, count]) => ({ code, label: areaNameByCode.get(code) ?? code, count, hint: metalNobreProcessAreaHint(code) }))
    .sort((left, right) => right.count - left.count)
    .slice(0, 5);
  const focusArea = processAreaSnapshot[0] ?? null;

  return (
    <>
      <WorkspaceHeroHeader
        title="Painel documental"
        subtitle="Visao executiva do que pede atencao agora, sem depender de realtime obrigatorio para ser util no dia a dia."
        searchQuery={searchQuery}
        onSearchQueryChange={setSearchQuery}
      />

      <section className={styles.content}>
      <WorkspaceDataState
        loadState={props.loadState}
        isEmpty={!hasOperationalData}
        emptyTitle={normalizedQuery ? "Nenhum documento para a busca atual" : "Sem sinais operacionais no momento"}
        emptyDescription={normalizedQuery
          ? "A busca nao retornou documentos no dashboard. Ajuste os termos e tente novamente."
          : "Ainda nao ha documentos ou notificacoes para compor o centro operacional."}
        loadingLabel="Atualizando centro operacional"
        errorDescription="Nao foi possivel sincronizar os indicadores operacionais agora."
        onRetry={props.onRefreshWorkspace}
      />

      {props.loadState === "ready" && hasOperationalData && (
        <>
          <section className={styles.kpiStrip}>
            <article className={styles.kpiItem}>
              <strong>{filteredDocuments.length}</strong>
              <small>Documentos Ativos</small>
            </article>
            <article className={styles.kpiItem}>
              <strong>{pendingReviews.length}</strong>
              <small>Em Revisao</small>
            </article>
            <article className={styles.kpiItem}>
              <strong>{unreadNotifications.length}</strong>
              <small>Notificacoes Pendentes</small>
            </article>
            <article className={styles.kpiItem}>
              <strong>{props.documentProfiles.length}</strong>
              <small>Profiles Disponiveis</small>
            </article>
          </section>

          <section className={styles.mainGrid}>
            <article className={`${styles.card} ${styles.cardBlue}`}>
              <header className={styles.cardHeader}>
                <h2>Ultimos Documentos</h2>
              </header>
              <div className={styles.timelineBody}>
                <TimelineRail
                  accent="blue"
                  ariaLabel="Ultimos documentos"
                  emptyState="Sem documentos recentes."
                  items={recentDocuments.map((item, index) => ({
                    id: item.documentId,
                    title: formatDocumentDisplayName(item, props.documentProfiles),
                    subtitle: props.formatDate(item.createdAt),
                    aside: profileNameByCode.get(item.documentProfile) ?? item.documentProfile,
                    active: index === 0,
                    onClick: () => void props.onOpenDocument(item.documentId),
                  }))}
                />
              </div>
            </article>

            <div className={styles.rightStack}>
              <article className={`${styles.card} ${styles.cardOrange}`}>
                <header className={styles.cardHeader}>
                  <h2>Pendencias de revisao</h2>
                </header>
                <div className={styles.pendingTable}>
                  <div className={styles.pendingHead}>
                    <span>Prioridade</span>
                    <span>Revisao</span>
                  </div>
                  {["Prioritario", "Medio", "Represente", "Politico"].map((priority) => (
                    <div key={priority} className={styles.pendingRow}>
                      <span className={styles.priorityTag}>{priority}</span>
                      <span>{pendingReviews.length > 0 ? `${pendingReviews.length} documento(s) aguardando revisao.` : "Nenhum documento aguardando revisao."}</span>
                    </div>
                  ))}
                </div>
              </article>

              <article className={`${styles.card} ${styles.cardRed}`}>
                <header className={styles.cardHeader}>
                  <h2>Expiracoes Proximas</h2>
                </header>
                <div className={styles.expiry}>
                  <div className={styles.expiryCount}>
                    <strong>{nextExpiryDays}</strong>
                    <small>Dias Restantes</small>
                  </div>
                  <p>{expiringSoon.length > 0 ? `${expiringSoon.length} documento(s) vencendo nos proximos 30 dias.` : "Nenhuma expiracao nos proximos 30 dias."}</p>
                </div>
              </article>
            </div>
          </section>

          <section className={styles.bottomGrid}>
            <article className={`${styles.card} ${styles.cardGreen}`}>
              <header className={styles.cardHeader}>
                <h2>Snapshot Operacional</h2>
              </header>
              <div className={styles.snapshot}>
                <span>{unreadNotifications.length > 0 ? `${unreadNotifications.length} notificacao(oes) nao lidas.` : "Sem notificacoes nao lidas."}</span>
              </div>
            </article>

            <article className={`${styles.card} ${styles.cardPurple}`}>
              <header className={styles.cardHeader}>
                <h2>Foco Metal Nobre</h2>
              </header>
              <div className={styles.focus}>
                <strong>{focusArea ? `${focusArea.label} (${focusArea.count})` : "Sem area (0)"}</strong>
                <span>{focusArea?.hint ?? "Sem processos com documentos no recorte atual."}</span>
              </div>
            </article>
          </section>
        </>
      )}
      </section>
    </>
  );
}
