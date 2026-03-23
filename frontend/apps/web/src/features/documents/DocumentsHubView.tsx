import { useEffect, useMemo } from "react";
import { buildDocumentProfileCountMap } from "./adapters/catalogSummary";
import { metalNobreProcessAreaHint } from "./adapters/metalNobreExperience";
import { formatDocumentDisplayName } from "../shared/documentDisplay";
import { type RecentDocumentItem, useDocumentsStore } from "../../store/documents.store";
import { DocumentsHubHeader } from "./DocumentsHubHeader";
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
  onSearchQueryChange: (value: string) => void;
  onCreateDocument: () => void;
  onOpenDocument: (documentId: string, nextView?: "library" | "content-builder") => void | Promise<void>;
  onOpenDocumentForHub: (documentId: string) => void | Promise<void>;
};

type HubScope = "all" | "mine" | "recent";
const recentKeyPrefix = "metaldocs.recentDocuments";

function recentStorageKey(userId?: string) {
  if (!userId) return null;
  return `${recentKeyPrefix}.${userId}`;
}

function hydrateRecentDocuments(items: RecentDocumentItem[], documents: SearchDocumentItem[]): RecentDocumentItem[] {
  if (items.length === 0 || documents.length === 0) return items;
  const byId = new Map(documents.map((doc) => [doc.documentId, doc] as const));
  return items.map((item) => {
    const match = byId.get(item.documentId);
    if (!match) return item;
    return {
      ...match,
      openedAt: item.openedAt || item.createdAt || match.createdAt || new Date().toISOString(),
    };
  });
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

function normalizeAreaCode(value?: string): string {
  return (value ?? "sem-area").trim().toLowerCase();
}

function profileBadgeText(profile: DocumentProfileItem): string {
  const alias = profile.alias?.trim?.();
  const code = profile.code?.trim?.();
  const name = profile.name?.trim?.();

  const shortAlias = alias && alias.length <= 3 ? alias.toUpperCase() : "";
  if (shortAlias) return shortAlias;

  const shortCode = code && code.length <= 3 ? code.toUpperCase() : "";
  if (shortCode) return shortCode;

  const source = (alias || code || name || "").toUpperCase();
  const words = source.split(/[^A-Z0-9]+/).filter(Boolean);
  if (words.length >= 2) return `${words[0][0]}${words[1][0]}`;
  if (words.length === 1 && words[0].length >= 2) return words[0].slice(0, 2);
  return source.slice(0, 2) || "--";
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

  const headerTitle = scope === "mine" ? "Meus documentos" : scope === "recent" ? "Recentes" : "Todos documentos";
  const headerVariant = documentsHubView === "collection" || documentsHubView === "detail" ? "compact" : "default";
  const headerShell = (
    <DocumentsHubHeader
      title={headerTitle}
      searchQuery={props.searchQuery}
      onSearchQueryChange={props.onSearchQueryChange}
      variant={headerVariant}
    />
  );

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

  useEffect(() => {
    if (!props.currentUserId) return;
    if (recentDocuments.length === 0) return;
    if (props.documents.length === 0) return;

    const hydrated = hydrateRecentDocuments(recentDocuments, props.documents);
    const needsUpgrade = hydrated.some((item, index) => {
      const previous = recentDocuments[index];
      if (!previous) return true;
      return (
        item.documentCode !== previous.documentCode ||
        item.documentSequence !== previous.documentSequence ||
        item.title !== previous.title ||
        item.openedAt !== previous.openedAt
      );
    });

    if (!needsUpgrade) return;
    setRecentDocuments(hydrated);
    storeRecentDocuments(props.currentUserId, hydrated);
  }, [props.currentUserId, props.documents, recentDocuments, setRecentDocuments]);

  useEffect(() => {
    setDocumentsHubView("overview");
    setDocumentsHubStatus("all");
    setDocumentsHubArea("all");
    setDocumentsHubProfile("all");
  }, [props.view, setDocumentsHubArea, setDocumentsHubProfile, setDocumentsHubStatus, setDocumentsHubView]);

  const recentFallback = useMemo(
    () => scopedDocuments.slice(0, 8).map((item) => ({ ...item, openedAt: item.createdAt })),
    [scopedDocuments],
  );
  const recentItems = useMemo(() => {
    const base = recentDocuments.length > 0 ? recentDocuments : recentFallback;
    return hydrateRecentDocuments(base, props.documents);
  }, [props.documents, recentDocuments, recentFallback]);
  const profileCounts = useMemo(() => buildDocumentProfileCountMap(scopedDocuments), [scopedDocuments]);

  const areaCounts = useMemo(() => {
    const counts: Record<string, number> = {};
    for (const document of scopedDocuments) {
      const key = normalizeAreaCode(document.processArea);
      counts[key] = (counts[key] ?? 0) + 1;
    }
    return counts;
  }, [scopedDocuments]);

  const totalDocuments = scopedDocuments.length;
  const areaColors = ["#9D2335", "#1F5A3F", "#6B3A9C", "#B4541A", "#2B5C8A", "#A32B6B"];
  const areaCards = useMemo(() => {
    const cards = props.processAreas.map((area, index) => ({
      code: normalizeAreaCode(area.code),
      label: area.name,
      count: areaCounts[normalizeAreaCode(area.code)] ?? 0,
      description: metalNobreProcessAreaHint(area.code),
      color: areaColors[index % areaColors.length],
    }));
    cards.unshift({
      code: "sem-area",
      label: "Sem area",
      count: areaCounts["sem-area"] ?? 0,
      description: "Sem classificacao atribuida.",
      color: areaColors[0],
    });
    return cards;
  }, [areaCounts, props.processAreas]);

  const profileCards = useMemo(() => {
    return props.documentProfiles
      .map((profile) => ({
        code: profile.code,
        label: profile.name,
        description: profile.description || "Perfil documental configurado no registry.",
        badge: profileBadgeText(profile),
        count: profileCounts[profile.code] ?? 0,
        color: areaColors[props.documentProfiles.findIndex((item) => item.code === profile.code) % areaColors.length],
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


  const normalizedQuery = props.searchQuery.trim().toLowerCase();
  const baseFilteredDocuments = useMemo(() => {
    return scopedDocuments.filter((item) => {
      if (documentsHubArea !== "all") {
        const area = normalizeAreaCode(item.processArea);
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
      return props.processAreas.find((item) => normalizeAreaCode(item.code) === documentsHubArea)?.name ?? documentsHubArea;
    }
    return headerTitle;
  }, [documentsHubArea, documentsHubProfile, headerTitle, props.documentProfiles, props.processAreas]);

  if (props.loadState === "loading") {
    return (
      <div className={styles.page}>
        {headerShell}
        <section className={styles.hub}>
          <div className={styles.state}>Carregando acervo...</div>
        </section>
      </div>
    );
  }

  if (props.loadState === "error") {
    return (
      <div className={styles.page}>
        {headerShell}
        <section className={styles.hub}>
          <div className={styles.state}>Falha ao carregar os documentos.</div>
        </section>
      </div>
    );
  }

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

  if (documentsHubView === "detail") {
    if (!props.selectedDocument) {
      return (
        <div className={styles.page}>
          {headerShell}
          <section className={styles.hub}>
            <div className={styles.state}>Selecione um documento para ver os detalhes.</div>
          </section>
        </div>
      );
    }

    const doc = props.selectedDocument;
    const profileLabel = props.documentProfiles.find((item) => item.code === doc.documentProfile)?.name ?? doc.documentProfile;
    const areaLabel = doc.processArea
      ? props.processAreas.find((item) => item.code === doc.processArea)?.name ?? doc.processArea
      : "Sem area";
    const governance = props.selectedProfileGovernance?.profileCode === doc.documentProfile
      ? props.selectedProfileGovernance
      : null;

    return (
      <div className={styles.page}>
        {headerShell}
        <section className={styles.detail}>
        <div className={styles.breadcrumb}>
          <button type="button" onClick={() => setDocumentsHubView("overview")}>Inicio</button>
          <span>/</span>
          <button type="button" onClick={() => setDocumentsHubView("collection")}>{collectionTitle}</button>
          <span>/</span>
          <span>{formatDocumentDisplayName(doc, props.documentProfiles)}</span>
        </div>

        <article className={styles.detailHero}>
          <div className={styles.detailHeroHeader}>
            <div className={styles.detailHeroBadge}>{doc.documentProfile.toUpperCase()}</div>
            <div>
              <h2>{formatDocumentDisplayName(doc, props.documentProfiles)}</h2>
              <small>{doc.documentId}</small>
            </div>
            <span className={styles.statusChip}>{statusLabel(doc.status)}</span>
          </div>
          <div className={styles.detailMeta}>
            <span><strong>Area</strong>{areaLabel}</span>
            <span><strong>Processo</strong>{doc.businessUnit || "-"}</span>
            <span><strong>Versao</strong>{doc.profileSchemaVersion ?? "-"}</span>
            <span><strong>Owner</strong>{doc.ownerId}</span>
            <span><strong>Prox. revisao</strong>{doc.expiryAt ? props.formatDate(doc.expiryAt) : "-"}</span>
          </div>
          <div className={styles.detailActions}>
            <button type="button" className={styles.primaryButton} onClick={() => props.onOpenDocument(doc.documentId, "content-builder")}>
              Abrir documento
            </button>
            <button type="button" className={styles.ghostButton} disabled>
              Enviar para revisao
            </button>
            <button type="button" className={styles.ghostButton} disabled>
              Duplicar
            </button>
            <button type="button" className={styles.ghostButton} disabled>
              Historico de versoes
            </button>
          </div>
        </article>

        <div className={styles.detailGrid}>
          <article className={styles.detailCard}>
            <h3>Classificacao</h3>
            <div className={styles.detailRows}>
              <span><strong>Familia</strong>{doc.documentFamily}</span>
              <span><strong>Profile</strong>{profileLabel}</span>
              <span><strong>Departamento</strong>{doc.department}</span>
              <span><strong>Subject</strong>{doc.subject ?? "-"}</span>
            </div>
          </article>
          <article className={styles.detailCard}>
            <h3>Governanca</h3>
            <div className={styles.detailRows}>
              <span><strong>Workflow</strong>{governance?.workflowProfile ?? "-"}</span>
              <span><strong>Revisao</strong>{governance ? `${governance.reviewIntervalDays} dias` : "-"}</span>
              <span><strong>Aprovacao</strong>{governance ? (governance.approvalRequired ? "Obrigatoria" : "Opcional") : "-"}</span>
              <span><strong>Validade</strong>{governance ? `${governance.validityDays} dias` : "-"}</span>
            </div>
          </article>
          <article className={styles.detailCard}>
            <h3>Colaboracao</h3>
            <div className={styles.detailRows}>
              <span><strong>Lock de edicao</strong>Sem lock ativo</span>
              <span><strong>Ativo</strong>{props.formatDate(new Date().toISOString())}</span>
            </div>
          </article>
          <article className={styles.detailCard}>
            <h3>Diff da versao atual</h3>
            <p className={styles.detailMuted}>Nenhuma alteracao registrada nesta versao.</p>
          </article>
        </div>
        </section>
      </div>
    );
  }

  if (documentsHubView === "collection") {
    return (
      <div className={styles.page}>
        {headerShell}
        <section className={styles.collection}>
          <div className={styles.collectionShell}>
            <div className={styles.collectionHeader}>
              <h2>{collectionTitle} <span>({tabCounts.all})</span></h2>
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
                <button type="button" className={styles.collectionCreateButton} onClick={props.onCreateDocument}>
                  + Novo documento
                </button>
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
                      <div className={styles.docCardTitleBlock}>
                        <strong>{formatDocumentDisplayName(item, props.documentProfiles)}</strong>
                        <span className={styles.docCardId}>{item.documentId}</span>
                      </div>
                      <span className={styles.statusChip}>{statusLabel(item.status)}</span>
                    </div>
                    <div className={styles.docCardMeta}>
                      <span className={styles.profilePill}>{item.documentProfile.toUpperCase()}</span>
                      <span>{item.processArea ?? "Sem area"} · {item.ownerId}</span>
                    </div>
                    <div className={styles.docCardFooter}>
                      <span>{item.businessUnit || item.department || "-"}</span>
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
                      <strong>{formatDocumentDisplayName(item, props.documentProfiles)}</strong>
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
          </div>
        </section>
      </div>
    );
  }

  return (
    <div className={styles.page}>
      {headerShell}

      <section className={styles.hub}>

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
                style={{ ["--area-color" as string]: area.color } as React.CSSProperties}
              >
                <span className={styles.areaStripe} />
                <div className={styles.areaMeta}>
                  <strong>{area.label}</strong>
                  <small>{area.count} documentos</small>
                  <div className={styles.areaBar}>
                    <span
                      className={styles.areaBarFill}
                      style={{ width: totalDocuments > 0 ? `${Math.round((area.count / totalDocuments) * 100)}%` : "0%" }}
                    />
                  </div>
                </div>
                <span className={styles.areaDescription}>{area.description}</span>
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
                style={{ ["--type-color" as string]: profile.color } as React.CSSProperties}
              >
                <span className={styles.typeStripe} />
                <span className={styles.typeBadge}>{profile.badge}</span>
                <div className={styles.typeMeta}>
                  <strong>{profile.label}</strong>
                  <small>{profile.count} documentos</small>
                  <div className={styles.typeBar}>
                    <span
                      className={styles.typeBarFill}
                      style={{ width: totalDocuments > 0 ? `${Math.round((profile.count / totalDocuments) * 100)}%` : "0%" }}
                    />
                  </div>
                </div>
                <span className={styles.typeDescription}>{profile.description}</span>
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
        <div className={styles.recentPanel}>
          <div className={styles.recentHeader}>
            <h2>Abertos recentemente</h2>
            <button
              type="button"
              className={styles.linkButton}
              onClick={() => {
                setDocumentsHubView("collection");
                setDocumentsHubStatus("all");
              }}
            >
              Ver todos â†’
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
                    <strong>{formatDocumentDisplayName(item, props.documentProfiles)}</strong>
                    <div className={styles.recentSubline}>
                      <span>{item.processArea ?? "Sem area"}</span>
                      <span>Aberto por {item.ownerId}</span>
                      <span>{props.formatDate(item.openedAt ?? item.createdAt)}</span>
                    </div>
                  </div>
                  <div className={styles.recentAside}>
                    <span className={styles.recentStatus}>{statusLabel(item.status)}</span>
                    <small className={styles.recentCode}>{item.documentId}</small>
                  </div>
                </button>
              ))
            )}
          </div>
        </div>
      </section>
      </section>
    </div>
  );
}

