import { useMemo, useState } from "react";
import { buildDocumentProfileCountMap } from "../features/documents/adapters/catalogSummary";
import { metalNobreProcessAreaHint } from "../features/documents/adapters/metalNobreExperience";
import { FilterDropdown } from "./ui/FilterDropdown";
import { WorkspaceDataState } from "./WorkspaceDataState";
import type {
  AccessPolicyItem,
  AttachmentItem,
  AuditEventItem,
  CollaborationPresenceItem,
  DocumentListItem,
  DocumentEditLockItem,
  DocumentProfileGovernanceItem,
  DocumentProfileItem,
  ProcessAreaItem,
  SearchDocumentItem,
  VersionDiffResponse,
  VersionListItem,
  WorkflowApprovalItem,
} from "../lib.types";

type PolicyScope = "document" | "document_type" | "area";
type DocumentsWorkspaceView = "library" | "my-docs" | "recent";

type DocumentsWorkspaceProps = {
  view: DocumentsWorkspaceView;
  loadState: "idle" | "loading" | "ready" | "error";
  documentProfiles: DocumentProfileItem[];
  processAreas: ProcessAreaItem[];
  documents: SearchDocumentItem[];
  selectedDocument: DocumentListItem | null;
  selectedProfileGovernance: DocumentProfileGovernanceItem | null;
  versions: VersionListItem[];
  versionDiff: VersionDiffResponse | null;
  approvals: WorkflowApprovalItem[];
  attachments: AttachmentItem[];
  collaborationPresence: CollaborationPresenceItem[];
  documentEditLock: DocumentEditLockItem | null;
  policies: AccessPolicyItem[];
  auditEvents: AuditEventItem[];
  selectedFile: File | null;
  policyScope: PolicyScope;
  policyResourceId: string;
  searchQuery: string;
  formatDate: (value?: string) => string;
  onRefreshWorkspace: () => void | Promise<void>;
  onOpenDocument: (documentId: string) => void | Promise<void>;
  onFileChange: (file: File | null) => void;
  onUploadAttachment: (event: React.FormEvent<HTMLFormElement>) => void | Promise<void>;
};

type GroupedArea = {
  code: string;
  label: string;
  documents: SearchDocumentItem[];
};

type AreaSnapshot = {
  code: string;
  label: string;
  count: number;
  hint: string;
};

function statusClass(status: string): string {
  switch (status) {
    case "IN_REVIEW":
      return "status-pill is-review";
    case "APPROVED":
      return "status-pill is-approved";
    case "PUBLISHED":
      return "status-pill is-published";
    case "ARCHIVED":
      return "status-pill is-archived";
    default:
      return "status-pill is-draft";
  }
}

function profileLabel(code: string, profiles: DocumentProfileItem[]): string {
  return profiles.find((item) => item.code === code)?.name ?? code;
}

function profileAlias(code: string, profiles: DocumentProfileItem[]): string {
  const profile = profiles.find((item) => item.code === code);
  return profile?.alias || profile?.name || code;
}

function areaLabel(code: string, areas: ProcessAreaItem[]): string {
  return areas.find((item) => item.code === code)?.name ?? code ?? "Sem area";
}

function areaColor(index: number): string {
  const colors = ["#3A45A0", "#2A6B35", "#7A5010", "#6B2A7A", "#7A2020"];
  return colors[index % colors.length];
}

export function DocumentsWorkspace(props: DocumentsWorkspaceProps) {
  const [statusFilter, setStatusFilter] = useState<string>("all");
  const [profileFilter, setProfileFilter] = useState<string>("all");
  const [areaFilter, setAreaFilter] = useState<string>("all");
  const [openGroups, setOpenGroups] = useState<Record<string, boolean>>({});

  const filteredDocuments = useMemo(() => {
    const normalizedQuery = props.searchQuery.trim().toLowerCase();
    return props.documents.filter((item) => {
      if (statusFilter !== "all" && item.status !== statusFilter) return false;
      if (profileFilter !== "all" && item.documentProfile !== profileFilter) return false;
      if (areaFilter !== "all" && (item.processArea ?? "") !== areaFilter) return false;
      if (!normalizedQuery) return true;
      const haystack = [
        item.title,
        item.documentProfile,
        item.documentFamily,
        item.ownerId,
        item.processArea,
        item.businessUnit,
        item.department,
      ].join(" ").toLowerCase();
      return haystack.includes(normalizedQuery);
    });
  }, [areaFilter, profileFilter, props.documents, props.searchQuery, statusFilter]);

  const groupedByArea = useMemo<GroupedArea[]>(() => {
    const map = new Map<string, SearchDocumentItem[]>();
    for (const document of filteredDocuments) {
      const key = document.processArea || "sem-area";
      const current = map.get(key) ?? [];
      current.push(document);
      map.set(key, current);
    }

    return Array.from(map.entries()).map(([code, documents]) => ({
      code,
      label: code === "sem-area" ? "Sem area" : areaLabel(code, props.processAreas),
      documents,
    }));
  }, [filteredDocuments, props.processAreas]);

  const profileCountByCode = useMemo(
    () => buildDocumentProfileCountMap(props.documents),
    [props.documents],
  );
  const areaSnapshots = useMemo<AreaSnapshot[]>(() => {
    const countByCode: Record<string, number> = {};
    for (const document of filteredDocuments) {
      const key = (document.processArea ?? "").trim().toLowerCase();
      if (key === "") {
        continue;
      }
      countByCode[key] = (countByCode[key] ?? 0) + 1;
    }
    return props.processAreas
      .map((area) => ({
        code: area.code,
        label: area.name,
        count: countByCode[area.code] ?? 0,
        hint: metalNobreProcessAreaHint(area.code),
      }))
      .filter((area) => area.count > 0)
      .sort((left, right) => right.count - left.count)
      .slice(0, 4);
  }, [filteredDocuments, props.processAreas]);

  const inReviewCount = props.documents.filter((item) => item.status === "IN_REVIEW").length;
  const approvedCount = props.documents.filter((item) => item.status === "APPROVED" || item.status === "PUBLISHED").length;
  const expiringSoonDocuments = props.documents.filter((item) => {
    if (!item.expiryAt) return false;
    const expiry = new Date(item.expiryAt).getTime();
    const now = Date.now();
    const thirtyDays = 1000 * 60 * 60 * 24 * 30;
    return expiry > now && expiry - now <= thirtyDays;
  });

  function toggleGroup(code: string) {
    setOpenGroups((current) => ({
      ...current,
      [code]: !current[code],
    }));
  }

  const viewTitle = props.view === "my-docs" ? "Meus Documentos" : props.view === "recent" ? "Recentes" : "Todos Documentos";
  const scopeLabel = props.view === "my-docs" ? "Meus documentos" : props.view === "recent" ? "Documentos recentes" : "Documentos";

  return (
    <section className="catalog-shell catalog-shell-dense">
      <div className="catalog-toolbar-panel">
        <div className="catalog-toolbar-top">
          <div className="workspace-breadcrumb">
            <span>MetalDocs</span>
            <span>/</span>
            <span>{scopeLabel}</span>
            <span>/</span>
            <strong>{viewTitle}</strong>
          </div>
          <div className="catalog-toolbar-spacer" />
          <div className="catalog-view-toggle">
            <button type="button" className="catalog-view-toggle-button is-active">
              <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.4">
                <rect x="1" y="1" width="12" height="12" rx="2" />
                <path d="M1 5h12M5 5v8" />
              </svg>
            </button>
            <button type="button" className="catalog-view-toggle-button">
              <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.4">
                <rect x="1" y="1" width="5" height="5" rx="1.5" />
                <rect x="8" y="1" width="5" height="5" rx="1.5" />
                <rect x="1" y="8" width="5" height="5" rx="1.5" />
                <rect x="8" y="8" width="5" height="5" rx="1.5" />
              </svg>
            </button>
          </div>
          <FilterDropdown
            id="catalog-area-filter"
            value={areaFilter}
            onSelect={setAreaFilter}
            options={[
              { value: "all", label: "Agrupar: por area" },
              ...props.processAreas.map((item) => ({ value: item.code, label: item.name })),
            ]}
          />
          <FilterDropdown
            id="catalog-profile-filter"
            value={profileFilter}
            onSelect={setProfileFilter}
            options={[
              { value: "all", label: "Todos os profiles" },
              ...props.documentProfiles.map((item) => ({ value: item.code, label: item.alias || item.name })),
            ]}
          />
        </div>

        <div className="catalog-chip-row">
          <button type="button" className={`catalog-chip ${statusFilter === "all" ? "is-active" : ""}`} onClick={() => setStatusFilter("all")}><span>Todos</span><span className="catalog-chip-count">{props.documents.length}</span></button>
          {props.documentProfiles.map((item) => (
            <button key={item.code} type="button" className={`catalog-chip ${profileFilter === item.code ? "is-active" : ""}`} onClick={() => setProfileFilter((current) => current === item.code ? "all" : item.code)}>
              <span className={`catalog-chip-dot profile-${item.code}`} />
              {profileAlias(item.code, props.documentProfiles)}
              <span className="catalog-chip-count">{profileCountByCode[item.code] ?? 0}</span>
            </button>
          ))}
          <span className="catalog-chip-separator" />
          <button type="button" className={`catalog-chip ${statusFilter === "DRAFT" ? "is-active" : ""}`} onClick={() => setStatusFilter("DRAFT")}>Rascunho</button>
          <button type="button" className={`catalog-chip ${statusFilter === "IN_REVIEW" ? "is-active" : ""}`} onClick={() => setStatusFilter("IN_REVIEW")}>Em revisao</button>
          <button type="button" className={`catalog-chip ${statusFilter === "APPROVED" ? "is-active" : ""}`} onClick={() => setStatusFilter("APPROVED")}>Aprovados</button>
          <span className="catalog-chip-separator" />
          <button type="button" className="catalog-chip" onClick={() => void props.onRefreshWorkspace()}>
            <svg width="11" height="11" viewBox="0 0 11 11" fill="none" stroke="currentColor" strokeWidth="1.4">
              <path d="M1 3h9M2.5 5.5h6M4 8h3" strokeLinecap="round" />
            </svg>
            Atualizar
          </button>
        </div>
      </div>

      <div className={`catalog-content-grid ${props.selectedDocument ? "" : "is-detail-hidden"}`}>
        <section className="catalog-primary-column">
          <WorkspaceDataState
            loadState={props.loadState}
            isEmpty={props.documents.length === 0}
            emptyTitle="Nenhum documento no recorte"
            emptyDescription="Nao ha documentos disponiveis para os filtros e visao selecionados."
            loadingLabel="Atualizando acervo documental"
            onRetry={props.onRefreshWorkspace}
          />

          {expiringSoonDocuments.length > 0 && (
            <div className="catalog-alert">
              <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="#C89020" strokeWidth="1.4">
                <path d="M7 1L1 12.5h12L7 1z" />
                <path d="M7 5.5v3" />
                <circle cx="7" cy="10.5" r=".6" fill="#C89020" stroke="none" />
              </svg>
              <span><strong>{expiringSoonDocuments.length} documentos</strong> com revisao vencendo nos proximos 30 dias.</span>
              <span className="catalog-alert-link">Ver documentos →</span>
            </div>
          )}

          <div className="catalog-stats">
            <article className="catalog-stat">
              <span>Total no recorte</span>
              <strong>{props.documents.length}</strong>
              <small>{groupedByArea.length} grupo(s) por area</small>
              <div className="catalog-stat-bar"><div className="catalog-stat-fill vinho" style={{ width: "100%" }} /></div>
            </article>
            <article className="catalog-stat">
              <span>Vigentes</span>
              <strong>{approvedCount}</strong>
              <small>Base pronta para referencia</small>
              <div className="catalog-stat-bar"><div className="catalog-stat-fill success" style={{ width: `${props.documents.length === 0 ? 0 : Math.round((approvedCount / props.documents.length) * 100)}%` }} /></div>
            </article>
            <article className="catalog-stat">
              <span>Em andamento</span>
              <strong>{inReviewCount}</strong>
              <small>Fila de revisao operacional</small>
              <div className="catalog-stat-bar"><div className="catalog-stat-fill warning" style={{ width: `${props.documents.length === 0 ? 0 : Math.round((inReviewCount / props.documents.length) * 100)}%` }} /></div>
            </article>
            <article className="catalog-stat">
              <span>Recorte atual</span>
              <strong>{filteredDocuments.length}</strong>
              <small>Resultados apos busca e filtros</small>
              <div className="catalog-stat-bar"><div className="catalog-stat-fill info" style={{ width: `${props.documents.length === 0 ? 0 : Math.round((filteredDocuments.length / props.documents.length) * 100)}%` }} /></div>
            </article>
          </div>

          {areaSnapshots.length > 0 && (
            <div className="catalog-card">
              <h3>Prioridades por processo</h3>
              <ul className="catalog-mini-list">
                {areaSnapshots.map((item) => (
                  <li key={`snapshot-${item.code}`}>
                    <span>{item.label} ({item.count})</span>
                    <small>{item.hint}</small>
                  </li>
                ))}
              </ul>
            </div>
          )}

          <div className="catalog-group-list">
            {groupedByArea.map((group, index) => {
              const isOpen = openGroups[group.code] ?? index === 0;
              return (
                <div key={group.code} className="catalog-group-section">
                  <button type="button" className="catalog-group-header" onClick={() => toggleGroup(group.code)}>
                    <span className={`catalog-group-chevron ${isOpen ? "is-open" : ""}`}>
                      <svg width="13" height="13" viewBox="0 0 13 13" fill="none" stroke="currentColor" strokeWidth="1.5">
                        <path d="M2.5 4.5l4 4 4-4" strokeLinecap="round" />
                      </svg>
                    </span>
                    <span className="catalog-group-dot" style={{ background: areaColor(index) }} />
                    <span className="catalog-group-label">{group.label}</span>
                    <span className="catalog-group-count">{group.documents.length} documentos</span>
                    <span className="catalog-group-line" />
                  </button>

                  {isOpen && (
                    <div className="catalog-table-shell catalog-table-shell-rich">
                      <div className="catalog-table-head-rich">
                        <span />
                        <span>Documento</span>
                        <span>Tipo / family</span>
                        <span>Status</span>
                        <span>Owner</span>
                        <span>Versao</span>
                        <span>Prox. revisao</span>
                        <span />
                      </div>
                      {group.documents.map((item) => (
                        <button key={item.documentId} type="button" className={`catalog-row-rich ${props.selectedDocument?.documentId === item.documentId ? "is-selected" : ""}`} onClick={() => void props.onOpenDocument(item.documentId)}>
                          <span className="catalog-row-checkbox" />
                          <span className="catalog-row-document">
                            <span className={`document-icon profile-${item.documentProfile}`}>{item.documentProfile.toUpperCase().slice(0, 2)}</span>
                            <span className="catalog-row-document-copy">
                              <strong>{item.title}</strong>
                              <small>{item.documentId}</small>
                            </span>
                          </span>
                          <span className="catalog-row-muted">{profileLabel(item.documentProfile, props.documentProfiles)}</span>
                          <span><span className={statusClass(item.status)}>{item.status}</span></span>
                          <span className="catalog-row-muted">{item.ownerId}</span>
                          <span className="catalog-row-mono">v{item.profileSchemaVersion ?? 1}</span>
                          <span className={`catalog-row-review ${item.expiryAt ? "is-warning" : ""}`}>{item.expiryAt ? formatShortDate(item.expiryAt) : "-"}</span>
                          <span className="catalog-row-actions">
                            <span className="catalog-row-action-dot">⋮</span>
                          </span>
                        </button>
                      ))}
                    </div>
                  )}
                </div>
              );
            })}
          </div>

          {filteredDocuments.length === 0 && (
            <div className="catalog-empty-state">
              <svg width="36" height="36" viewBox="0 0 36 36" fill="none" stroke="currentColor" strokeWidth="1">
                <path d="M8 4h13l7 7v21H8V4z" strokeLinejoin="round" />
                <path d="M21 4v7h7" strokeLinejoin="round" />
                <path d="M12 16h12M12 20h12M12 24h8" strokeLinecap="round" />
              </svg>
              <p>Nenhum documento encontrado</p>
            </div>
          )}

          <div className="catalog-pagination">
            <span>Mostrando {filteredDocuments.length} de {props.documents.length} documentos</span>
            <div className="catalog-pagination-buttons">
              {["‹", "1", "2", "3", "›"].map((item, index) => (
                <button key={`${item}-${index}`} type="button" className={`catalog-pagination-button ${item === "1" ? "is-active" : ""}`}>{item}</button>
              ))}
            </div>
          </div>
        </section>

        {props.selectedDocument && (
        <aside className="catalog-panel catalog-detail-panel">
          <div className="catalog-panel-head">
            <div>
              <p className="catalog-kicker">Detalhe</p>
              <h2>{props.selectedDocument.title}</h2>
            </div>
          </div>

          <div className="catalog-detail-stack">
              <div className="catalog-info-grid">
                <div><span>Status</span><strong>{props.selectedDocument.status}</strong></div>
                <div><span>Profile</span><strong>{profileLabel(props.selectedDocument.documentProfile, props.documentProfiles)}</strong></div>
                <div><span>Family</span><strong>{props.selectedDocument.documentFamily}</strong></div>
                <div><span>Schema</span><strong>v{props.selectedDocument.profileSchemaVersion ?? 1}</strong></div>
              </div>
              <div className="catalog-info-grid">
                <div><span>Processo</span><strong>{props.selectedDocument.processArea ? areaLabel(props.selectedDocument.processArea, props.processAreas) : "-"}</strong></div>
                <div><span>Subject</span><strong>{props.selectedDocument.subject || "-"}</strong></div>
                <div><span>Area</span><strong>{props.selectedDocument.businessUnit}</strong></div>
                <div><span>Departamento</span><strong>{props.selectedDocument.department}</strong></div>
              </div>

              <div className="catalog-card">
                <h3>Governanca</h3>
                <ul className="catalog-mini-list">
                  <li><span>Workflow</span><small>{props.selectedProfileGovernance?.workflowProfile ?? "-"}</small></li>
                  <li><span>Revisao</span><small>{props.selectedProfileGovernance ? `${props.selectedProfileGovernance.reviewIntervalDays} dias` : "-"}</small></li>
                  <li><span>Aprovacao</span><small>{props.selectedProfileGovernance?.approvalRequired ? "Obrigatoria" : "Opcional"}</small></li>
                  <li><span>Validade</span><small>{props.selectedProfileGovernance?.validityDays ? `${props.selectedProfileGovernance.validityDays} dias` : "-"}</small></li>
                </ul>
              </div>

              <div className="catalog-card">
                <h3>Colaboracao</h3>
                <ul className="catalog-mini-list">
                  <li>
                    <span>Lock de edicao</span>
                    <small>
                      {props.documentEditLock
                        ? `${props.documentEditLock.displayName} ate ${props.formatDate(props.documentEditLock.expiresAt)}`
                        : "Sem lock ativo"}
                    </small>
                  </li>
                  {props.collaborationPresence.map((item) => (
                    <li key={`${item.documentId}-${item.userId}`}>
                      <span>{item.displayName}</span>
                      <small>Ativo em {props.formatDate(item.lastSeenAt)}</small>
                    </li>
                  ))}
                  {props.collaborationPresence.length === 0 && (
                    <li><span>Nenhum colaborador ativo no momento.</span></li>
                  )}
                </ul>
              </div>

              <div className="catalog-card">
                <h3>Diff da versao atual</h3>
                {!props.versionDiff ? <p className="catalog-muted">O diff aparece quando houver pelo menos duas versoes para comparar.</p> : (
                  <div className="diff-grid">
                    <div><span>Comparacao</span><strong>v{props.versionDiff.fromVersion} para v{props.versionDiff.toVersion}</strong></div>
                    <div><span>Conteudo</span><strong>{props.versionDiff.contentChanged ? "Alterado" : "Sem mudanca"}</strong></div>
                    <div><span>Classificacao</span><strong>{props.versionDiff.classificationChanged ? "Alterada" : "Igual"}</strong></div>
                    <div><span>Effective / expiry</span><strong>{props.versionDiff.effectiveAtChanged || props.versionDiff.expiryAtChanged ? "Alterado" : "Igual"}</strong></div>
                    <div className="diff-grid-full"><span>Metadata alterada</span><strong>{props.versionDiff.metadataChanged.length > 0 ? props.versionDiff.metadataChanged.join(", ") : "Nenhum campo alterado"}</strong></div>
                  </div>
                )}
              </div>

              <form className="catalog-card stack" onSubmit={props.onUploadAttachment}>
                <h3>Anexos</h3>
                <input type="file" onChange={(event) => props.onFileChange(event.target.files?.[0] ?? null)} />
                <button type="submit" disabled={!props.selectedFile}>Enviar anexo</button>
                <ul className="catalog-mini-list">
                  {props.attachments.map((item) => <li key={item.attachmentId}><span>{item.fileName}</span><small>{props.formatDate(item.createdAt)}</small></li>)}
                  {props.attachments.length === 0 && <li><span>Nenhum anexo enviado.</span></li>}
                </ul>
              </form>

              <div className="catalog-card">
                <h3>Versoes</h3>
                <ul className="catalog-mini-list">
                  {props.versions.map((item) => <li key={item.version}><span>Versao {item.version}</span><small>{item.changeSummary || item.contentHash}</small></li>)}
                  {props.versions.length === 0 && <li><span>Sem versoes adicionais.</span></li>}
                </ul>
              </div>

              <div className="catalog-card">
                <h3>Aprovacoes</h3>
                <ul className="catalog-mini-list">
                  {props.approvals.map((item) => <li key={item.approvalId}><span>{item.status}</span><small>{item.assignedReviewer}</small></li>)}
                  {props.approvals.length === 0 && <li><span>Nenhuma aprovacao registrada.</span></li>}
                </ul>
              </div>

              <div className="catalog-card">
                <h3>Policies ({props.policyScope}:{props.policyResourceId || "-"})</h3>
                <ul className="catalog-mini-list">
                  {props.policies.map((item, index) => <li key={`${item.subjectId}-${index}`}><span>{item.subjectType}:{item.subjectId}</span><small>{item.capability} / {item.effect}</small></li>)}
                  {props.policies.length === 0 && <li><span>Sem policies especificas.</span></li>}
                </ul>
              </div>

              <div className="catalog-card">
                <h3>Audit timeline</h3>
                <ul className="catalog-mini-list">
                  {props.auditEvents.map((item: AuditEventItem) => <li key={item.id}><span>{item.action}</span><small>{item.actorId} em {props.formatDate(item.occurredAt)}</small></li>)}
                  {props.auditEvents.length === 0 && <li><span>Sem eventos de auditoria retornados.</span></li>}
                </ul>
              </div>
            </div>
        </aside>
        )}
      </div>
    </section>
  );
}

function formatShortDate(value: string): string {
  const date = new Date(value);
  return new Intl.DateTimeFormat("pt-BR", { day: "2-digit", month: "short", year: "numeric" }).format(date);
}
