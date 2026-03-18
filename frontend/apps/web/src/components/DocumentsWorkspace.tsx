import type {
  AccessPolicyItem,
  AttachmentItem,
  DocumentListItem,
  DocumentProfileGovernanceItem,
  DocumentProfileItem,
  DocumentProfileSchemaItem,
  ProcessAreaItem,
  SearchDocumentItem,
  VersionListItem,
  WorkflowApprovalItem,
} from "../lib.types";

type DocumentForm = {
  title: string;
  documentType: string;
  documentProfile: string;
  processArea: string;
  subject: string;
  ownerId: string;
  businessUnit: string;
  department: string;
  classification: string;
  tags: string;
  metadata: string;
  initialContent: string;
};

type PolicyScope = "document" | "document_type" | "area";

type DocumentsWorkspaceProps = {
  loadState: "idle" | "loading" | "ready" | "error";
  documentForm: DocumentForm;
  documentProfiles: DocumentProfileItem[];
  processAreas: ProcessAreaItem[];
  selectedProfileSchema: DocumentProfileSchemaItem | null;
  selectedProfileGovernance: DocumentProfileGovernanceItem | null;
  documents: SearchDocumentItem[];
  selectedDocument: DocumentListItem | null;
  versions: VersionListItem[];
  approvals: WorkflowApprovalItem[];
  attachments: AttachmentItem[];
  policies: AccessPolicyItem[];
  selectedFile: File | null;
  policyScope: PolicyScope;
  policyResourceId: string;
  onDocumentFormChange: (next: DocumentForm) => void;
  onSubmitCreateDocument: (event: React.FormEvent<HTMLFormElement>) => void | Promise<void>;
  onApplyProfile: (profileCode: string, preferredProcessArea?: string) => void | Promise<void>;
  onRefreshWorkspace: () => void | Promise<void>;
  onOpenDocument: (documentId: string) => void | Promise<void>;
  onFileChange: (file: File | null) => void;
  onUploadAttachment: (event: React.FormEvent<HTMLFormElement>) => void | Promise<void>;
};

export function DocumentsWorkspace(props: DocumentsWorkspaceProps) {
  const selectedProfile = props.documentProfiles.find((item) => item.code === props.documentForm.documentProfile);

  return (
    <>
      <section className="panel">
        <div className="panel-heading"><p className="kicker">Cadastro</p><h2>Novo documento</h2></div>
        <form data-testid="document-create-form" className="stack" onSubmit={props.onSubmitCreateDocument}>
          <input data-testid="document-title" placeholder="Titulo" value={props.documentForm.title} onChange={(event) => props.onDocumentFormChange({ ...props.documentForm, title: event.target.value })} required />
          <select data-testid="document-profile" value={props.documentForm.documentProfile} onChange={(event) => void props.onApplyProfile(event.target.value, props.documentForm.processArea)}>
            {props.documentProfiles.map((item) => <option key={item.code} value={item.code}>{item.name} ({item.code})</option>)}
          </select>
          <div className="detail-summary">
            <div><span>Family</span><strong>{selectedProfile?.familyCode ?? "-"}</strong></div>
            <div><span>Workflow</span><strong>{props.selectedProfileGovernance?.workflowProfile ?? "-"}</strong></div>
            <div><span>Revisao</span><strong>{props.selectedProfileGovernance ? `${props.selectedProfileGovernance.reviewIntervalDays} dias` : "-"}</strong></div>
            <div><span>Aprovacao</span><strong>{props.selectedProfileGovernance?.approvalRequired ? "Obrigatoria" : "Opcional"}</strong></div>
          </div>
          <div className="two-columns">
            <select data-testid="document-process-area" value={props.documentForm.processArea} onChange={(event) => props.onDocumentFormChange({ ...props.documentForm, processArea: event.target.value })}>
              <option value="">Sem process area</option>
              {props.processAreas.map((item) => <option key={item.code} value={item.code}>{item.name}</option>)}
            </select>
            <input data-testid="document-subject" placeholder="Subject opcional" value={props.documentForm.subject} onChange={(event) => props.onDocumentFormChange({ ...props.documentForm, subject: event.target.value })} />
          </div>
          <div className="two-columns"><input data-testid="document-owner" placeholder="Owner" value={props.documentForm.ownerId} onChange={(event) => props.onDocumentFormChange({ ...props.documentForm, ownerId: event.target.value })} /><input data-testid="document-business-unit" placeholder="Business unit" value={props.documentForm.businessUnit} onChange={(event) => props.onDocumentFormChange({ ...props.documentForm, businessUnit: event.target.value })} /></div>
          <div className="two-columns"><input data-testid="document-department" placeholder="Departamento" value={props.documentForm.department} onChange={(event) => props.onDocumentFormChange({ ...props.documentForm, department: event.target.value })} /><input data-testid="document-tags" placeholder="Tags" value={props.documentForm.tags} onChange={(event) => props.onDocumentFormChange({ ...props.documentForm, tags: event.target.value })} /></div>
          <textarea data-testid="document-metadata" rows={4} value={props.documentForm.metadata} onChange={(event) => props.onDocumentFormChange({ ...props.documentForm, metadata: event.target.value })} />
          <p className="hint">O JSON acima nasce do schema ativo do profile selecionado. Ajuste os valores mantendo os campos obrigatorios exigidos pelo registry.</p>
          {props.selectedProfileSchema && <p className="hint">Schema ativo: v{props.selectedProfileSchema.version} com {(props.selectedProfileSchema.metadataRules ?? []).length} regra(s) de metadata.</p>}
          <textarea data-testid="document-initial-content" rows={5} value={props.documentForm.initialContent} onChange={(event) => props.onDocumentFormChange({ ...props.documentForm, initialContent: event.target.value })} placeholder="Conteudo inicial" />
          <button data-testid="document-submit" type="submit">Criar documento</button>
        </form>
      </section>

      <section data-testid="documents-panel" className="panel">
        <div className="panel-heading split"><div><p className="kicker">Acervo</p><h2>Documentos</h2></div><button type="button" className="ghost-button" onClick={() => void props.onRefreshWorkspace()}>Atualizar</button></div>
        <div className="table-shell">
          <table>
            <thead><tr><th>Titulo</th><th>Status</th><th>Owner</th></tr></thead>
            <tbody>{props.documents.map((item) => <tr key={item.documentId} onClick={() => void props.onOpenDocument(item.documentId)} className={props.selectedDocument?.documentId === item.documentId ? "row-active" : ""}><td><strong>{item.title}</strong><small>{item.documentProfile} / {item.documentFamily}{item.processArea ? ` / ${item.processArea}` : ""}</small></td><td>{item.status}</td><td>{item.ownerId}</td></tr>)}</tbody>
          </table>
        </div>
        {props.loadState === "loading" && <p className="hint">Carregando dados...</p>}
      </section>

      <section className="panel">
        <div className="panel-heading"><p className="kicker">Detalhe</p><h2>{props.selectedDocument?.title ?? "Selecione um documento"}</h2></div>
        {!props.selectedDocument ? <p className="hint">Abra um documento para validar versoes, anexos, approvals e policies.</p> : (
          <div className="stack">
            <div className="detail-summary"><div><span>Status</span><strong>{props.selectedDocument.status}</strong></div><div><span>Profile</span><strong>{props.selectedDocument.documentProfile}</strong></div><div><span>Family</span><strong>{props.selectedDocument.documentFamily}</strong></div><div><span>Owner</span><strong>{props.selectedDocument.ownerId}</strong></div></div>
            <div className="detail-summary"><div><span>Processo</span><strong>{props.selectedDocument.processArea || "-"}</strong></div><div><span>Subject</span><strong>{props.selectedDocument.subject || "-"}</strong></div><div><span>Area</span><strong>{props.selectedDocument.businessUnit} / {props.selectedDocument.department}</strong></div><div><span>Schema</span><strong>v{props.selectedDocument.profileSchemaVersion ?? 1}</strong></div></div>
            <form className="card stack" onSubmit={props.onUploadAttachment}><h3>Anexos</h3><input type="file" onChange={(event) => props.onFileChange(event.target.files?.[0] ?? null)} /><button type="submit" disabled={!props.selectedFile}>Enviar anexo</button><ul className="mini-list">{props.attachments.map((item) => <li key={item.attachmentId}><span>{item.fileName}</span><small>{item.createdAt}</small></li>)}</ul></form>
            <div className="card"><h3>Versoes</h3><ul className="mini-list">{props.versions.map((item) => <li key={item.version}><span>Versao {item.version}</span><small>{item.changeSummary || item.contentHash}</small></li>)}</ul></div>
            <div className="card"><h3>Aprovacoes</h3><ul className="mini-list">{props.approvals.map((item) => <li key={item.approvalId}><span>{item.status}</span><small>{item.assignedReviewer}</small></li>)}</ul></div>
            <div className="card"><h3>Policies ({props.policyScope}:{props.policyResourceId})</h3><ul className="mini-list">{props.policies.map((item, index) => <li key={`${item.subjectId}-${index}`}><span>{item.subjectType}:{item.subjectId}</span><small>{item.capability} / {item.effect}</small></li>)}</ul></div>
          </div>
        )}
      </section>
    </>
  );
}
