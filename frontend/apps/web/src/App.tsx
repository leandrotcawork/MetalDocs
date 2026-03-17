import { useEffect, useMemo, useState } from "react";
import { api } from "./lib.api";
import type {
  AccessPolicyItem,
  AttachmentItem,
  DocumentListItem,
  DocumentTypeItem,
  SearchDocumentItem,
  VersionListItem,
  WorkflowApprovalItem,
} from "./lib.types";

type LoadState = "idle" | "loading" | "ready" | "error";
type PolicyScope = "document" | "document_type" | "area";

const emptyDocumentForm = {
  title: "",
  documentType: "policy",
  ownerId: "admin-local",
  businessUnit: "Quality",
  department: "Operations",
  classification: "INTERNAL",
  tags: "",
  effectiveAt: "",
  expiryAt: "",
  initialContent: "",
  metadata: '{"summary":""}',
};

const emptySearchForm = {
  q: "",
  documentType: "",
  businessUnit: "",
  department: "",
  classification: "",
  status: "",
  ownerId: "",
  tag: "",
};

const defaultPolicyForm = {
  resourceScope: "document" as PolicyScope,
  resourceId: "",
  subjectType: "user",
  subjectId: "",
  capability: "document.view",
  effect: "allow",
};

export default function App() {
  const [documentTypes, setDocumentTypes] = useState<DocumentTypeItem[]>([]);
  const [documents, setDocuments] = useState<SearchDocumentItem[]>([]);
  const [selectedDocument, setSelectedDocument] = useState<DocumentListItem | null>(null);
  const [versions, setVersions] = useState<VersionListItem[]>([]);
  const [approvals, setApprovals] = useState<WorkflowApprovalItem[]>([]);
  const [attachments, setAttachments] = useState<AttachmentItem[]>([]);
  const [policies, setPolicies] = useState<AccessPolicyItem[]>([]);
  const [documentForm, setDocumentForm] = useState(emptyDocumentForm);
  const [searchForm, setSearchForm] = useState(emptySearchForm);
  const [policyForm, setPolicyForm] = useState(defaultPolicyForm);
  const [versionContent, setVersionContent] = useState("");
  const [versionSummary, setVersionSummary] = useState("");
  const [workflowReason, setWorkflowReason] = useState("");
  const [workflowReviewer, setWorkflowReviewer] = useState("");
  const [workflowTarget, setWorkflowTarget] = useState("IN_REVIEW");
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [activeScope, setActiveScope] = useState<PolicyScope>("document");
  const [loadState, setLoadState] = useState<LoadState>("idle");
  const [detailState, setDetailState] = useState<LoadState>("idle");
  const [message, setMessage] = useState<string>("");
  const [error, setError] = useState<string>("");

  useEffect(() => {
    void bootstrap();
  }, []);

  async function bootstrap() {
    setLoadState("loading");
    setError("");
    try {
      const [typesResponse, docsResponse] = await Promise.all([
        api.listDocumentTypes(),
        api.searchDocuments(new URLSearchParams({ limit: "25" })),
      ]);
      setDocumentTypes(typesResponse.items);
      setDocuments(docsResponse.items);
      if (typesResponse.items[0]) {
        setDocumentForm((current) => ({ ...current, documentType: typesResponse.items[0].code }));
      }
      setLoadState("ready");
    } catch (err) {
      setLoadState("error");
      setError(asMessage(err));
    }
  }

  async function refreshSearch() {
    const params = new URLSearchParams();
    Object.entries(searchForm).forEach(([key, value]) => {
      if (value.trim()) params.set(key, value.trim());
    });
    params.set("limit", "50");
    const response = await api.searchDocuments(params);
    setDocuments(response.items);
  }

  async function openDocument(documentId: string) {
    setDetailState("loading");
    setError("");
    try {
      const [document, versionsResponse, approvalsResponse, attachmentsResponse] = await Promise.all([
        api.getDocument(documentId),
        api.listVersions(documentId),
        api.listApprovals(documentId),
        api.listAttachments(documentId),
      ]);
      setSelectedDocument(document);
      setVersions(versionsResponse.items);
      setApprovals(approvalsResponse.items);
      setAttachments(attachmentsResponse.items);
      setPolicyForm((current) => ({ ...current, resourceScope: "document", resourceId: documentId }));
      setActiveScope("document");
      await loadPolicies("document", documentId);
      setDetailState("ready");
    } catch (err) {
      setDetailState("error");
      setError(asMessage(err));
    }
  }

  async function loadPolicies(resourceScope: PolicyScope, resourceId: string) {
    if (!resourceId.trim()) {
      setPolicies([]);
      return;
    }
    const response = await api.listAccessPolicies(resourceScope, resourceId.trim());
    setPolicies(response.items);
  }

  async function handleCreateDocument(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setMessage("");
    setError("");
    try {
      const payload = {
        ...documentForm,
        tags: documentForm.tags.split(",").map((tag) => tag.trim()).filter(Boolean),
        metadata: documentForm.metadata.trim() ? JSON.parse(documentForm.metadata) : {},
      };
      await api.createDocument(payload);
      setDocumentForm(emptyDocumentForm);
      await refreshSearch();
      setMessage("Documento criado com sucesso.");
    } catch (err) {
      setError(asMessage(err));
    }
  }

  async function handleAddVersion(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selectedDocument) return;
    setMessage("");
    setError("");
    try {
      await api.addVersion(selectedDocument.documentId, { content: versionContent, changeSummary: versionSummary });
      setVersionContent("");
      setVersionSummary("");
      await openDocument(selectedDocument.documentId);
      setMessage("Nova versao registrada.");
    } catch (err) {
      setError(asMessage(err));
    }
  }

  async function handleTransition(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selectedDocument) return;
    setMessage("");
    setError("");
    try {
      await api.transitionWorkflow(selectedDocument.documentId, {
        toStatus: workflowTarget,
        reason: workflowReason || undefined,
        assignedReviewer: workflowReviewer || undefined,
      });
      setWorkflowReason("");
      setWorkflowReviewer("");
      await openDocument(selectedDocument.documentId);
      await refreshSearch();
      setMessage("Workflow atualizado.");
    } catch (err) {
      setError(asMessage(err));
    }
  }

  async function handleUploadAttachment(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selectedDocument || !selectedFile) return;
    setMessage("");
    setError("");
    try {
      await api.uploadAttachment(selectedDocument.documentId, selectedFile);
      setSelectedFile(null);
      await openDocument(selectedDocument.documentId);
      setMessage("Anexo enviado.");
    } catch (err) {
      setError(asMessage(err));
    }
  }

  async function handleReplacePolicies(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setMessage("");
    setError("");
    try {
      await api.replaceAccessPolicies({
        resourceScope: policyForm.resourceScope,
        resourceId: policyForm.resourceId,
        policies: [
          {
            subjectType: policyForm.subjectType,
            subjectId: policyForm.subjectId,
            capability: policyForm.capability,
            effect: policyForm.effect,
          },
        ],
      });
      await loadPolicies(policyForm.resourceScope, policyForm.resourceId);
      setMessage("Policies substituidas com sucesso.");
    } catch (err) {
      setError(asMessage(err));
    }
  }

  const timelineItems = useMemo(() => {
    const items = [
      ...versions.map((version) => ({
        id: `version-${version.version}`,
        title: `Versao ${version.version}`,
        subtitle: version.changeSummary || "Sem resumo de alteracao.",
        timestamp: version.createdAt,
        accent: "version",
      })),
      ...approvals.map((approval) => ({
        id: approval.approvalId,
        title: `${approval.status} - ${approval.assignedReviewer}`,
        subtitle: approval.decisionReason || approval.requestReason || "Fluxo de aprovacao registrado.",
        timestamp: approval.decidedAt || approval.requestedAt,
        accent: "approval",
      })),
      ...attachments.map((attachment) => ({
        id: attachment.attachmentId,
        title: `Anexo ${attachment.fileName}`,
        subtitle: `${attachment.contentType} • ${formatBytes(attachment.sizeBytes)}`,
        timestamp: attachment.createdAt,
        accent: "attachment",
      })),
    ];
    return items.sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime());
  }, [approvals, attachments, versions]);

  return (
    <div className="app-shell">
      <header className="hero">
        <div>
          <p className="eyebrow">MetalDocs Control Room</p>
          <h1>Operacao documental com controle fino de acesso, versao e aprovacao.</h1>
          <p className="hero-copy">
            UI minima operacional construida sobre o contrato real do backend. Nada aqui antecipa API inexistente.
          </p>
        </div>
        <div className="hero-panel">
          <span>Usuario atual</span>
          <strong>{api.currentUserId}</strong>
          <small>API base: {import.meta.env.VITE_API_BASE_URL ?? "http://192.168.0.3:8080/api/v1"}</small>
        </div>
      </header>

      {(message || error) && (
        <section className={`banner ${error ? "banner-error" : "banner-success"}`}>
          {error || message}
        </section>
      )}

      <main className="grid-layout">
        <section className="panel panel-form">
          <div className="panel-heading">
            <p className="kicker">Cadastro</p>
            <h2>Novo documento</h2>
          </div>
          <form className="stack" onSubmit={handleCreateDocument}>
            <label>
              <span>Titulo</span>
              <input value={documentForm.title} onChange={(event) => setDocumentForm({ ...documentForm, title: event.target.value })} required />
            </label>
            <div className="two-columns">
              <label>
                <span>Tipo documental</span>
                <select value={documentForm.documentType} onChange={(event) => setDocumentForm({ ...documentForm, documentType: event.target.value })}>
                  {documentTypes.map((item) => (
                    <option key={item.code} value={item.code}>{item.name}</option>
                  ))}
                </select>
              </label>
              <label>
                <span>Owner</span>
                <input value={documentForm.ownerId} onChange={(event) => setDocumentForm({ ...documentForm, ownerId: event.target.value })} required />
              </label>
            </div>
            <div className="two-columns">
              <label>
                <span>Business unit</span>
                <input value={documentForm.businessUnit} onChange={(event) => setDocumentForm({ ...documentForm, businessUnit: event.target.value })} required />
              </label>
              <label>
                <span>Departamento</span>
                <input value={documentForm.department} onChange={(event) => setDocumentForm({ ...documentForm, department: event.target.value })} required />
              </label>
            </div>
            <div className="two-columns">
              <label>
                <span>Classificacao</span>
                <select value={documentForm.classification} onChange={(event) => setDocumentForm({ ...documentForm, classification: event.target.value })}>
                  {['PUBLIC','INTERNAL','CONFIDENTIAL','RESTRICTED'].map((item) => <option key={item} value={item}>{item}</option>)}
                </select>
              </label>
              <label>
                <span>Tags</span>
                <input value={documentForm.tags} onChange={(event) => setDocumentForm({ ...documentForm, tags: event.target.value })} placeholder="iso, qualidade, processo" />
              </label>
            </div>
            <div className="two-columns">
              <label>
                <span>Vigencia</span>
                <input type="datetime-local" value={documentForm.effectiveAt} onChange={(event) => setDocumentForm({ ...documentForm, effectiveAt: event.target.value ? new Date(event.target.value).toISOString() : "" })} />
              </label>
              <label>
                <span>Expiracao</span>
                <input type="datetime-local" value={documentForm.expiryAt} onChange={(event) => setDocumentForm({ ...documentForm, expiryAt: event.target.value ? new Date(event.target.value).toISOString() : "" })} />
              </label>
            </div>
            <label>
              <span>Metadata JSON</span>
              <textarea rows={4} value={documentForm.metadata} onChange={(event) => setDocumentForm({ ...documentForm, metadata: event.target.value })} />
            </label>
            <label>
              <span>Conteudo inicial</span>
              <textarea rows={6} value={documentForm.initialContent} onChange={(event) => setDocumentForm({ ...documentForm, initialContent: event.target.value })} />
            </label>
            <button type="submit">Criar documento</button>
          </form>
        </section>

        <section className="panel panel-list">
          <div className="panel-heading split">
            <div>
              <p className="kicker">Busca</p>
              <h2>Acervo operacional</h2>
            </div>
            <button type="button" className="ghost-button" onClick={() => void refreshSearch()}>Atualizar</button>
          </div>
          <form className="filter-grid" onSubmit={(event) => { event.preventDefault(); void refreshSearch(); }}>
            <input placeholder="Busca livre" value={searchForm.q} onChange={(event) => setSearchForm({ ...searchForm, q: event.target.value })} />
            <input placeholder="Business unit" value={searchForm.businessUnit} onChange={(event) => setSearchForm({ ...searchForm, businessUnit: event.target.value })} />
            <input placeholder="Departamento" value={searchForm.department} onChange={(event) => setSearchForm({ ...searchForm, department: event.target.value })} />
            <input placeholder="Owner" value={searchForm.ownerId} onChange={(event) => setSearchForm({ ...searchForm, ownerId: event.target.value })} />
            <input placeholder="Tag" value={searchForm.tag} onChange={(event) => setSearchForm({ ...searchForm, tag: event.target.value })} />
            <select value={searchForm.documentType} onChange={(event) => setSearchForm({ ...searchForm, documentType: event.target.value })}>
              <option value="">Todos os tipos</option>
              {documentTypes.map((item) => <option key={item.code} value={item.code}>{item.name}</option>)}
            </select>
            <select value={searchForm.classification} onChange={(event) => setSearchForm({ ...searchForm, classification: event.target.value })}>
              <option value="">Todas as classificacoes</option>
              {['PUBLIC','INTERNAL','CONFIDENTIAL','RESTRICTED'].map((item) => <option key={item} value={item}>{item}</option>)}
            </select>
            <select value={searchForm.status} onChange={(event) => setSearchForm({ ...searchForm, status: event.target.value })}>
              <option value="">Todos os status</option>
              {['DRAFT','IN_REVIEW','APPROVED','PUBLISHED','ARCHIVED'].map((item) => <option key={item} value={item}>{item}</option>)}
            </select>
            <button type="submit">Aplicar filtros</button>
          </form>

          <div className="table-shell">
            <table>
              <thead>
                <tr>
                  <th>Titulo</th>
                  <th>Tipo</th>
                  <th>Area</th>
                  <th>Status</th>
                  <th>Owner</th>
                </tr>
              </thead>
              <tbody>
                {documents.map((document) => (
                  <tr key={document.documentId} onClick={() => void openDocument(document.documentId)} className={selectedDocument?.documentId === document.documentId ? "row-active" : ""}>
                    <td>
                      <strong>{document.title}</strong>
                      <small>{document.documentId}</small>
                    </td>
                    <td>{document.documentType}</td>
                    <td>{document.businessUnit} / {document.department}</td>
                    <td><span className={`status status-${document.status.toLowerCase()}`}>{document.status}</span></td>
                    <td>{document.ownerId}</td>
                  </tr>
                ))}
              </tbody>
            </table>
            {loadState === "loading" && <p className="hint">Carregando catalogo...</p>}
            {!documents.length && loadState === "ready" && <p className="hint">Nenhum documento encontrado com os filtros atuais.</p>}
          </div>
        </section>

        <section className="panel panel-detail">
          <div className="panel-heading">
            <p className="kicker">Detalhe</p>
            <h2>{selectedDocument ? selectedDocument.title : "Selecione um documento"}</h2>
          </div>
          {!selectedDocument ? (
            <p className="hint">Abra um documento da lista para operar permissao, versao, anexos e workflow.</p>
          ) : (
            <div className="detail-stack">
              <div className="detail-summary">
                <div>
                  <span>Tipo</span>
                  <strong>{selectedDocument.documentType}</strong>
                </div>
                <div>
                  <span>Classificacao</span>
                  <strong>{selectedDocument.classification}</strong>
                </div>
                <div>
                  <span>Area</span>
                  <strong>{selectedDocument.businessUnit} / {selectedDocument.department}</strong>
                </div>
                <div>
                  <span>Status</span>
                  <strong>{selectedDocument.status}</strong>
                </div>
              </div>

              <div className="subgrid">
                <form className="card stack" onSubmit={handleAddVersion}>
                  <h3>Nova versao</h3>
                  <textarea rows={5} value={versionContent} onChange={(event) => setVersionContent(event.target.value)} placeholder="Novo conteudo do documento" />
                  <input value={versionSummary} onChange={(event) => setVersionSummary(event.target.value)} placeholder="Resumo da alteracao" />
                  <button type="submit">Registrar versao</button>
                </form>

                <form className="card stack" onSubmit={handleTransition}>
                  <h3>Workflow</h3>
                  <select value={workflowTarget} onChange={(event) => setWorkflowTarget(event.target.value)}>
                    {['DRAFT','IN_REVIEW','APPROVED','PUBLISHED','ARCHIVED'].map((item) => <option key={item} value={item}>{item}</option>)}
                  </select>
                  <input value={workflowReviewer} onChange={(event) => setWorkflowReviewer(event.target.value)} placeholder="Reviewer designado" />
                  <textarea rows={3} value={workflowReason} onChange={(event) => setWorkflowReason(event.target.value)} placeholder="Motivo da transicao" />
                  <button type="submit">Aplicar transicao</button>
                </form>
              </div>

              <div className="subgrid">
                <form className="card stack" onSubmit={handleUploadAttachment}>
                  <h3>Anexos</h3>
                  <input type="file" onChange={(event) => setSelectedFile(event.target.files?.[0] ?? null)} />
                  <button type="submit" disabled={!selectedFile}>Enviar anexo</button>
                  <ul className="mini-list">
                    {attachments.map((attachment) => (
                      <li key={attachment.attachmentId}>
                        <span>{attachment.fileName}</span>
                        <button type="button" className="ghost-button" onClick={async () => {
                          try {
                            const response = await api.getAttachmentDownloadURL(selectedDocument.documentId, attachment.attachmentId);
                            window.open(response.downloadUrl, "_blank", "noopener,noreferrer");
                          } catch (err) {
                            setError(asMessage(err));
                          }
                        }}>Baixar</button>
                      </li>
                    ))}
                  </ul>
                </form>

                <form className="card stack" onSubmit={handleReplacePolicies}>
                  <h3>Permissoes</h3>
                  <div className="two-columns compact">
                    <select value={policyForm.resourceScope} onChange={(event) => {
                      const value = event.target.value as PolicyScope;
                      setPolicyForm({ ...policyForm, resourceScope: value, resourceId: value === "document" ? selectedDocument.documentId : "" });
                      setActiveScope(value);
                    }}>
                      <option value="document">Documento</option>
                      <option value="document_type">Tipo documental</option>
                      <option value="area">Area</option>
                    </select>
                    <input value={policyForm.resourceId} onChange={(event) => setPolicyForm({ ...policyForm, resourceId: event.target.value })} placeholder="resourceId" />
                  </div>
                  <div className="two-columns compact">
                    <select value={policyForm.subjectType} onChange={(event) => setPolicyForm({ ...policyForm, subjectType: event.target.value })}>
                      <option value="user">User</option>
                      <option value="role">Role</option>
                      <option value="group">Group</option>
                    </select>
                    <input value={policyForm.subjectId} onChange={(event) => setPolicyForm({ ...policyForm, subjectId: event.target.value })} placeholder="subjectId" />
                  </div>
                  <div className="two-columns compact">
                    <select value={policyForm.capability} onChange={(event) => setPolicyForm({ ...policyForm, capability: event.target.value })}>
                      {['document.create','document.view','document.edit','document.upload_attachment','document.change_workflow','document.manage_permissions'].map((item) => <option key={item} value={item}>{item}</option>)}
                    </select>
                    <select value={policyForm.effect} onChange={(event) => setPolicyForm({ ...policyForm, effect: event.target.value })}>
                      <option value="allow">allow</option>
                      <option value="deny">deny</option>
                    </select>
                  </div>
                  <div className="action-row">
                    <button type="submit">Substituir policy</button>
                    <button type="button" className="ghost-button" onClick={() => void loadPolicies(policyForm.resourceScope, policyForm.resourceId)}>Carregar</button>
                  </div>
                  <p className="hint">Escopo ativo: {activeScope}</p>
                  <ul className="mini-list">
                    {policies.map((policy, index) => (
                      <li key={`${policy.subjectId}-${policy.capability}-${index}`}>
                        <span>{policy.subjectType}:{policy.subjectId}</span>
                        <strong>{policy.capability}</strong>
                        <em>{policy.effect}</em>
                      </li>
                    ))}
                  </ul>
                </form>
              </div>

              <div className="subgrid wide">
                <div className="card">
                  <h3>Timeline operacional</h3>
                  <p className="hint">A timeline usa contratos reais de versoes, aprovacoes e anexos. Audit append-only ainda nao possui endpoint HTTP dedicado.</p>
                  <ul className="timeline">
                    {timelineItems.map((item) => (
                      <li key={item.id} className={`timeline-${item.accent}`}>
                        <div>
                          <strong>{item.title}</strong>
                          <p>{item.subtitle}</p>
                        </div>
                        <span>{formatDate(item.timestamp)}</span>
                      </li>
                    ))}
                  </ul>
                </div>
                <div className="card">
                  <h3>Aprovacoes</h3>
                  <ul className="mini-list approvals-list">
                    {approvals.map((approval) => (
                      <li key={approval.approvalId}>
                        <div>
                          <strong>{approval.status}</strong>
                          <p>{approval.assignedReviewer}</p>
                        </div>
                        <span>{formatDate(approval.decidedAt || approval.requestedAt)}</span>
                      </li>
                    ))}
                  </ul>
                </div>
              </div>
            </div>
          )}
          {detailState === "loading" && <p className="hint">Carregando detalhe...</p>}
        </section>
      </main>
    </div>
  );
}

function asMessage(error: unknown): string {
  if (error instanceof Error) {
    return error.message;
  }
  return "Falha inesperada.";
}

function formatDate(value?: string): string {
  if (!value) return "-";
  return new Intl.DateTimeFormat("pt-BR", { dateStyle: "short", timeStyle: "short" }).format(new Date(value));
}

function formatBytes(size: number): string {
  if (size < 1024) return `${size} B`;
  if (size < 1024 * 1024) return `${(size / 1024).toFixed(1)} KB`;
  return `${(size / (1024 * 1024)).toFixed(1)} MB`;
}
