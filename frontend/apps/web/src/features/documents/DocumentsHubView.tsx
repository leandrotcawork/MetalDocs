import { useEffect, useMemo } from "react";
import { buildDocumentProfileCountMap } from "./adapters/catalogSummary";
import { metalNobreProcessAreaHint } from "./adapters/metalNobreExperience";
import { type RecentDocumentItem, useDocumentsStore } from "../../store/documents.store";
import styles from "./DocumentsHubView.module.css";
import type { DocumentListItem, DocumentProfileGovernanceItem, DocumentProfileItem, ProcessAreaItem, SearchDocumentItem } from "../../lib.types";

type DocumentsHubViewProps = {
  view: "library" | "my-docs" | "recent";
  loadState: "idle" | "loading" | "ready" | "error";
  currentUserId?: string;
  documents: SearchDocumentItem[];
  documentProfiles: DocumentProfileItem[];
  processAreas: ProcessAreaItem[];
  selectedDocument: DocumentListItem | null;
  selectedProfileGovernance: DocumentProfileGovernanceItem | null;
  searchQuery: string;
  formatDate: (value?: string) => string;
  onOpenDocument: (documentId: string, nextView?: "library" | "content-builder") => void | Promise<void>;
  onOpenDocumentForHub: (documentId: string) => void | Promise<void>;
};

type HubScope = "all" | "mine" | "recent";
const recentKeyPrefix = "metaldocs.recentDocuments";

function recentStorageKey(userId?: string) {
  if (!userId) return null;
  return `${recentKeyPrefix}.${userId}`;
}

function loadRecentDocuments(userId?: string): RecentDocumentItem[] {
  const key = recentStorageKey(userId);
  if (!key) return [];
  try {
    const raw = window.localStorage.getItem(key);
    if (!raw) return [];
    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) return [];
    return parsed.filter((item) => item && typeof item.documentId === "string");
  } catch {
    return [];
  }
}

function storeRecentDocuments(userId: string | undefined, items: RecentDocumentItem[]) {
  const key = recentStorageKey(userId);
  if (!key) return;
  window.localStorage.setItem(key, JSON.stringify(items));
}

function documentScope(view: DocumentsHubViewProps["view"]): HubScope {
  if (view === "my-docs") return "mine";
  if (view === "recent") return "recent";
  return "all";
}

function statusLabel(status: string): string {
  switch (status) {
    case "IN_REVIEW":
      return "Em revisao";
    case "APPROVED":
    case "PUBLISHED":
      return "Aprovado";
    case "ARCHIVED":
      return "Arquivado";
    default:
      return "Draft";
  }
}

export function DocumentsHubView(props: DocumentsHubViewProps) {
  const scope = documentScope(props.view);
  const {
    documentsHubView,
    documentsHubMode,
    documentsHubStatus,
    documentsHubArea,
    documentsHubProfile,
    setDocumentsHubView,
    setDocumentsHubMode,
    setDocumentsHubStatus,
    setDocumentsHubArea,
    setDocumentsHubProfile,
    recentDocuments,
    setRecentDocuments,
  } = useDocumentsStore();

  if (props.loadState === "loading") {
    return <section className={styles.state}>Carregando acervo...</section>;
  }

  if (props.loadState === "error") {
    return <section className={styles.state}>Falha ao carregar os documentos.</section>;
  }

  const scopedDocuments = useMemo(() => {
    if (scope === "mine") {
      return props.documents.filter((item) => item.ownerId === props.currentUserId);
    }
    if (scope === "recent") {
      if (recentDocuments.length > 0) {
        return recentDocuments;
      }
      return [...props.documents].sort((left, right) => new Date(right.createdAt).getTime() - new Date(left.createdAt).getTime());
    }
    return props.documents;
  }, [props.currentUserId, props.documents, recentDocuments, scope]);

  useEffect(() => {
    if (!props.currentUserId) return;
    setRecentDocuments(loadRecentDocuments(props.currentUserId));
  }, [props.currentUserId, setRecentDocuments]);

  const recentFallback = useMemo(
    () => scopedDocuments.slice(0, 8).map((item) => ({ ...item, openedAt: item.createdAt })),
    [scopedDocuments],
  );
  const recentItems = recentDocuments.length > 0 ? recentDocuments : recentFallback;
  const profileCounts = useMemo(() => buildDocumentProfileCountMap(scopedDocuments), [scopedDocuments]);

  const areaCounts = useMemo(() => {
    const counts: Record<string, number> = {};
    for (const document of scopedDocuments) {
      const key = (document.processArea ?? "sem-area").trim().toLowerCase();
      counts[key] = (counts[key] ?? 0) + 1;
    }
    return counts;
  }, [scopedDocuments]);

  const areaCards = useMemo(() => {
    const cards = props.processAreas.map((area) => ({
      code: area.code,
      label: area.name,
      count: areaCounts[area.code] ?? 0,
      hint: metalNobreProcessAreaHint(area.code),
    }));
    if (areaCounts["sem-area"]) {
      cards.unshift({
        code: "sem-area",
        label: "Sem area",
        count: areaCounts["sem-area"] ?? 0,
        hint: "Sem classificacao",
      });
    }
    return cards.filter((item) => item.count > 0);
  }, [areaCounts, props.processAreas]);

  const profileCards = useMemo(() => {
    return props.documentProfiles
      .map((profile) => ({
        code: profile.code,
        label: profile.name,
        alias: profile.alias ?? profile.code.toUpperCase(),
        count: profileCounts[profile.code] ?? 0,
      }))
      .filter((item) => item.count > 0);
  }, [profileCounts, props.documentProfiles]);

  const approvedCount = scopedDocuments.filter((item) => item.status === "APPROVED" || item.status === "PUBLISHED").length;
  const inReviewCount = scopedDocuments.filter((item) => item.status === "IN_REVIEW").length;
  const expiringSoon = scopedDocuments.filter((item) => {
    if (!item.expiryAt) return false;
    const expiry = new Date(item.expiryAt).getTime();
    const thirtyDays = 1000 * 60 * 60 * 24 * 30;
    return expiry - Date.now() <= thirtyDays && expiry > Date.now();
  }).length;

  const headerTitle = scope === "mine" ? "Meus documentos" : scope === "recent" ? "Recentes" : "Todos documentos";

  const normalizedQuery = props.searchQuery.trim().toLowerCase();
  const baseFilteredDocuments = useMemo(() => {
    return scopedDocuments.filter((item) => {
      if (documentsHubArea !== "all") {
        const area = item.processArea ?? "sem-area";
        if (area !== documentsHubArea) return false;
      }
      if (documentsHubProfile !== "all" && item.documentProfile !== documentsHubProfile) return false;
      if (!normalizedQuery) return true;
      const haystack = [
        item.title,
        item.documentId,
        item.documentProfile,
        item.processArea,
        item.ownerId,
      ].join(" ").toLowerCase();
      return haystack.includes(normalizedQuery);
    });
  }, [documentsHubArea, documentsHubProfile, normalizedQuery, scopedDocuments]);

  const tabCounts = useMemo(() => ({
    all: baseFilteredDocuments.length,
    draft: baseFilteredDocuments.filter((item) => item.status === "DRAFT").length,
    review: baseFilteredDocuments.filter((item) => item.status === "IN_REVIEW").length,
    approved: baseFilteredDocuments.filter((item) => item.status === "APPROVED" || item.status === "PUBLISHED").length,
  }), [baseFilteredDocuments]);

  const collectionDocuments = useMemo(() => {
    if (documentsHubStatus === "draft") {
      return baseFilteredDocuments.filter((item) => item.status === "DRAFT");
    }
    if (documentsHubStatus === "review") {
      return baseFilteredDocuments.filter((item) => item.status === "IN_REVIEW");
    }
    if (documentsHubStatus === "approved") {
      return baseFilteredDocuments.filter((item) => item.status === "APPROVED" || item.status === "PUBLISHED");
    }
    return baseFilteredDocuments;
  }, [baseFilteredDocuments, documentsHubStatus]);

  const collectionTitle = useMemo(() => {
    if (documentsHubProfile !== "all") {
      return props.documentProfiles.find((item) => item.code === documentsHubProfile)?.name ?? documentsHubProfile;
    }
    if (documentsHubArea !== "all") {
      if (documentsHubArea === "sem-area") return "Sem area";
      return props.processAreas.find((item) => item.code === documentsHubArea)?.name ?? documentsHubArea;
    }
    return headerTitle;
  }, [documentsHubArea, documentsHubProfile, headerTitle, props.documentProfiles, props.processAreas]);

  const handleRecentOpen = (item: SearchDocumentItem) => {
    const nextItems: RecentDocumentItem[] = [
      { ...item, openedAt: new Date().toISOString() },
      ...recentItems.filter((recent) => recent.documentId !== item.documentId),
    ].slice(0, 8);
    setRecentDocuments(nextItems);
    storeRecentDocuments(props.currentUserId, nextItems);
    void props.onOpenDocumentForHub(item.documentId);
    setDocumentsHubView("detail");
  };

  if (documentsHubView === "collection") {
    return (
      <section className={styles.collection}>
        <div className={styles.collectionHeader}>
          <div>
            <div className={styles.breadcrumb}>
              <button type="button" onClick={() => setDocumentsHubView("overview")}>Inicio</button>
              <span>/</span>
              <span>{collectionTitle}</span>
            </div>
            <h2>{collectionTitle} <span>({tabCounts.all})</span></h2>
          </div>
          <div className={styles.collectionActions}>
            <div className={styles.viewToggle}>
              <button
                type="button"
                className={documentsHubMode === "card" ? styles.isActive : ""}
                onClick={() => setDocumentsHubMode("card")}
                title="Exibir em cards"
              >
                <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.4">
                  <rect x="1" y="1" width="6" height="6" rx="1.5" />
                  <rect x="9" y="1" width="6" height="6" rx="1.5" />
                  <rect x="1" y="9" width="6" height="6" rx="1.5" />
                  <rect x="9" y="9" width="6" height="6" rx="1.5" />
                </svg>
              </button>
              <button
                type="button"
                className={documentsHubMode === "list" ? styles.isActive : ""}
                onClick={() => setDocumentsHubMode("list")}
                title="Exibir em lista"
              >
                <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.4">
                  <path d="M3 4h10M3 8h10M3 12h10" strokeLinecap="round" />
                </svg>
              </button>
            </div>
          </div>
        </div>

        <div className={styles.tabRow}>
          <button type="button" className={documentsHubStatus === "all" ? styles.tabActive : styles.tab} onClick={() => setDocumentsHubStatus("all")}>Todos ({tabCounts.all})</button>
          <button type="button" className={documentsHubStatus === "draft" ? styles.tabActive : styles.tab} onClick={() => setDocumentsHubStatus("draft")}>Draft ({tabCounts.draft})</button>
          <button type="button" className={documentsHubStatus === "review" ? styles.tabActive : styles.tab} onClick={() => setDocumentsHubStatus("review")}>Em revisao ({tabCounts.review})</button>
          <button type="button" className={documentsHubStatus === "approved" ? styles.tabActive : styles.tab} onClick={() => setDocumentsHubStatus("approved")}>Aprovados ({tabCounts.approved})</button>
        </div>

        {documentsHubMode === "card" ? (
          <div className={styles.cardGrid}>
            {collectionDocuments.map((item) => (
              <button
                key={item.documentId}
                type="button"
                className={styles.docCard}
                onClick={() => handleRecentOpen(item)}
              >
                <div className={styles.docCardHeader}>
                  <strong>{item.title || "Documento sem titulo"}</strong>
                  <span className={styles.statusChip}>{statusLabel(item.status)}</span>
                </div>
                <div className={styles.docCardMeta}>
                  <span>{item.documentProfile.toUpperCase()}</span>
                  <span>{item.processArea ?? "Sem area"}</span>
                </div>
                <span className={styles.docCardId}>{item.documentId}</span>
                <div className={styles.docCardFooter}>
                  <span>{item.ownerId}</span>
                  <span>{item.expiryAt ? `Revisao: ${props.formatDate(item.expiryAt)}` : "-"}</span>
                </div>
              </button>
            ))}
            {collectionDocuments.length === 0 && <div className={styles.emptyCard}>Nenhum documento encontrado.</div>}
          </div>
        ) : (
          <div className={styles.listTable}>
            <div className={styles.listHeader}>
              <span>Documento</span>
              <span>Tipo</span>
              <span>Status</span>
              <span>Owner</span>
              <span>Prox. revisao</span>
            </div>
            {collectionDocuments.map((item) => (
              <button
                key={item.documentId}
                type="button"
                className={styles.listRow}
                onClick={() => handleRecentOpen(item)}
              >
                <span className={styles.listTitle}>
                  <strong>{item.title || "Documento sem titulo"}</strong>
                  <small>{item.documentId} · {item.processArea ?? "Sem area"}</small>
                </span>
                <span>{item.documentProfile.toUpperCase()}</span>
                <span>{statusLabel(item.status)}</span>
                <span>{item.ownerId}</span>
                <span>{item.expiryAt ? props.formatDate(item.expiryAt) : "-"}</span>
              </button>
            ))}
            {collectionDocuments.length === 0 && <div className={styles.emptyCard}>Nenhum documento encontrado.</div>}
          </div>
        )}
      </section>
    );
  }

  return (
    <section className={styles.hub}>
      <header className={styles.hero}>
        <p className={styles.eyebrow}>MetalDocs</p>
        <h1 className={styles.title}>{headerTitle}</h1>
        <p className={styles.subtitle}>
          Acervo organizado por areas, tipos e status. Navegue pelos documentos mais relevantes.
        </p>
      </header>

      <section className={styles.kpiGrid}>
        <article className={styles.kpiCard}>
          <span>Total</span>
          <strong>{scopedDocuments.length}</strong>
          <small>documentos no recorte</small>
        </article>
        <article className={styles.kpiCard}>
          <span>Vigentes</span>
          <strong>{approvedCount}</strong>
          <small>prontos para referencia</small>
        </article>
        <article className={styles.kpiCard}>
          <span>Em andamento</span>
          <strong>{inReviewCount}</strong>
          <small>na fila de revisao</small>
        </article>
        <article className={styles.kpiCard}>
          <span>Atencao</span>
          <strong>{expiringSoon}</strong>
          <small>vencendo nos proximos 30 dias</small>
        </article>
      </section>

      <section className={styles.section}>
        <div className={styles.sectionHeader}>
          <h2>Areas</h2>
        </div>
        <div className={styles.areaGrid}>
          {areaCards.map((area) => (
            <article key={area.code} className={styles.areaCard}>
              <button
                type="button"
                className={styles.areaButton}
                onClick={() => {
                  setDocumentsHubArea(area.code);
                  setDocumentsHubProfile("all");
                  setDocumentsHubStatus("all");
                  setDocumentsHubView("collection");
                }}
              >
                <div>
                  <strong>{area.label}</strong>
                  <small>{area.count} documentos</small>
                </div>
                <span className={styles.areaHint}>{area.hint}</span>
              </button>
            </article>
          ))}
          {areaCards.length === 0 && (
            <article className={styles.emptyCard}>
              <span>Nenhuma area com documentos.</span>
            </article>
          )}
        </div>
      </section>

      <section className={styles.section}>
        <div className={styles.sectionHeader}>
          <h2>Tipos de documento</h2>
        </div>
        <div className={styles.typeGrid}>
          {profileCards.map((profile) => (
            <article key={profile.code} className={styles.typeCard}>
              <button
                type="button"
                className={styles.typeButton}
                onClick={() => {
                  setDocumentsHubProfile(profile.code);
                  setDocumentsHubArea("all");
                  setDocumentsHubStatus("all");
                  setDocumentsHubView("collection");
                }}
              >
                <span className={styles.typeBadge}>{profile.alias}</span>
                <div>
                  <strong>{profile.label}</strong>
                  <small>{profile.count} documentos</small>
                </div>
              </button>
            </article>
          ))}
          {profileCards.length === 0 && (
            <article className={styles.emptyCard}>
              <span>Nenhum tipo com documentos.</span>
            </article>
          )}
        </div>
      </section>

      <section className={styles.section}>
        <div className={styles.sectionHeader}>
          <h2>Abertos recentemente</h2>
          <button
            type="button"
            className={styles.linkButton}
            onClick={() => {
              setDocumentsHubView("collection");
              setDocumentsHubStatus("all");
            }}
          >
            Ver todos →
          </button>
        </div>
        <div className={styles.recentList}>
          {recentItems.length === 0 ? (
            <div className={styles.emptyCard}>Nenhum documento recente.</div>
          ) : (
            recentItems.map((item) => (
              <button
                key={item.documentId}
                type="button"
                className={styles.recentRow}
                onClick={() => handleRecentOpen(item)}
              >
                <span className={styles.recentBadge}>{item.documentProfile.toUpperCase()}</span>
                <div className={styles.recentMeta}>
                  <strong>{item.title || "Documento sem titulo"}</strong>
                  <small>{item.processArea ?? "Sem area"} · {props.formatDate(item.openedAt ?? item.createdAt)}</small>
                </div>
                <span className={styles.recentStatus}>{statusLabel(item.status)}</span>
              </button>
            ))
          )}
        </div>
      </section>
    </section>
  );
}
