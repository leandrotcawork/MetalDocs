import { useMemo } from "react";
import { buildDocumentProfileCountMap } from "./adapters/catalogSummary";
import { metalNobreProcessAreaHint } from "./adapters/metalNobreExperience";
import { useDocumentsStore } from "../../store/documents.store";
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
  formatDate: (value?: string) => string;
  onOpenDocument: (documentId: string, nextView?: "library" | "content-builder") => void | Promise<void>;
  onOpenDocumentForHub: (documentId: string) => void | Promise<void>;
};

type HubScope = "all" | "mine" | "recent";

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
    setDocumentsHubView,
    setDocumentsHubStatus,
    setDocumentsHubArea,
    setDocumentsHubProfile,
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
      return [...props.documents].sort((left, right) => new Date(right.createdAt).getTime() - new Date(left.createdAt).getTime());
    }
    return props.documents;
  }, [props.currentUserId, props.documents, scope]);

  const recentDocuments = useMemo(() => scopedDocuments.slice(0, 8), [scopedDocuments]);
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
          {recentDocuments.length === 0 ? (
            <div className={styles.emptyCard}>Nenhum documento recente.</div>
          ) : (
            recentDocuments.map((item) => (
              <button
                key={item.documentId}
                type="button"
                className={styles.recentRow}
                onClick={() => {
                  void props.onOpenDocumentForHub(item.documentId);
                  setDocumentsHubView("detail");
                }}
              >
                <span className={styles.recentBadge}>{item.documentProfile.toUpperCase()}</span>
                <div className={styles.recentMeta}>
                  <strong>{item.title || "Documento sem titulo"}</strong>
                  <small>{item.processArea ?? "Sem area"} · {props.formatDate(item.createdAt)}</small>
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
