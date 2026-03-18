import { useEffect, useState } from "react";
import { api } from "./lib.api";
import type {
  AccessPolicyItem,
  AttachmentItem,
  CurrentUser,
  DocumentListItem,
  DocumentTypeItem,
  ManagedUserItem,
  SearchDocumentItem,
  UserRole,
  VersionListItem,
  WorkflowApprovalItem,
} from "./lib.types";

type LoadState = "idle" | "loading" | "ready" | "error";
type PolicyScope = "document" | "document_type" | "area";

const metadataTemplates: Record<string, Record<string, string>> = {
  policy: { policy_code: "POL-001" },
  procedure: { procedure_code: "PROC-001" },
  work_instruction: { instruction_code: "WI-001" },
  contract: {
    counterparty: "Fornecedor Exemplo",
    contract_number: "CTR-001",
    start_date: "2026-01-01",
    end_date: "2026-12-31",
  },
  supplier_document: {
    supplier_name: "Fornecedor Exemplo",
    supplier_document_code: "SUP-001",
  },
  technical_drawing: {
    drawing_code: "DWG-001",
    revision_code: "A",
    plant: "Matriz",
  },
  certificate: {
    issuer: "Organismo Certificador",
    issue_date: "2026-01-01",
    expiry_date: "2027-01-01",
  },
  report: { report_period: "2026-Q1" },
  form: { form_code: "FRM-001" },
  manual: { manual_code: "MAN-001" },
};

function metadataTextForDocumentType(documentType: string): string {
  return JSON.stringify(metadataTemplates[documentType] ?? {}, null, 2);
}

const emptyDocumentForm = {
  title: "",
  documentType: "policy",
  ownerId: "",
  businessUnit: "Quality",
  department: "Operations",
  classification: "INTERNAL",
  tags: "",
  metadata: metadataTextForDocumentType("policy"),
  initialContent: "",
};

const emptyUserForm = {
  userId: "",
  username: "",
  email: "",
  displayName: "",
  password: "",
  roles: ["viewer"] as UserRole[],
};

export default function App() {
  const [authState, setAuthState] = useState<LoadState>("loading");
  const [loadState, setLoadState] = useState<LoadState>("idle");
  const [user, setUser] = useState<CurrentUser | null>(null);
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");
  const [loginForm, setLoginForm] = useState({ identifier: "admin", password: "" });
  const [passwordForm, setPasswordForm] = useState({ currentPassword: "", newPassword: "", confirmPassword: "" });
  const [documentForm, setDocumentForm] = useState(emptyDocumentForm);
  const [documentTypes, setDocumentTypes] = useState<DocumentTypeItem[]>([]);
  const [documents, setDocuments] = useState<SearchDocumentItem[]>([]);
  const [selectedDocument, setSelectedDocument] = useState<DocumentListItem | null>(null);
  const [versions, setVersions] = useState<VersionListItem[]>([]);
  const [approvals, setApprovals] = useState<WorkflowApprovalItem[]>([]);
  const [attachments, setAttachments] = useState<AttachmentItem[]>([]);
  const [policies, setPolicies] = useState<AccessPolicyItem[]>([]);
  const [managedUsers, setManagedUsers] = useState<ManagedUserItem[]>([]);
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [policyScope, setPolicyScope] = useState<PolicyScope>("document");
  const [policyResourceId, setPolicyResourceId] = useState("");
  const [userForm, setUserForm] = useState(emptyUserForm);

  const isAdmin = user?.roles.includes("admin") ?? false;

  useEffect(() => {
    void bootstrap();
  }, []);

  async function bootstrap() {
    try {
      const currentUser = await api.me();
      setUser(currentUser);
      setDocumentForm((current) => ({ ...current, ownerId: currentUser.userId }));
      await loadWorkspace(currentUser);
      setAuthState("ready");
    } catch (err) {
      if (statusOf(err) === 401) {
        setAuthState("idle");
        return;
      }
      setAuthState("error");
      setError(asMessage(err));
    }
  }

  async function loadWorkspace(currentUser: CurrentUser) {
    setLoadState("loading");
    try {
      const [typesResponse, docsResponse, usersResponse] = await Promise.all([
        api.listDocumentTypes(),
        api.searchDocuments(new URLSearchParams({ limit: "25" })),
        currentUser.roles.includes("admin") ? api.listUsers() : Promise.resolve({ items: [] as ManagedUserItem[] }),
      ]);
      setDocumentTypes(typesResponse.items);
      setDocuments(docsResponse.items);
      setManagedUsers(usersResponse.items);
      if (typesResponse.items[0]) {
        setDocumentForm((current) => ({ ...current, documentType: typesResponse.items[0].code, metadata: metadataTextForDocumentType(typesResponse.items[0].code) }));
      }
      setLoadState("ready");
    } catch (err) {
      handleError(err);
      setLoadState("error");
    }
  }

  async function handleLogin(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setMessage("");
    const response = await api.login(loginForm).catch((err) => {
      setUser(null);
      setManagedUsers([]);
      setDocuments([]);
      setSelectedDocument(null);
      setAuthState("idle");
      setError(asMessage(err));
      return null;
    });
    if (!response) return;
    setUser(response.user);
    setDocumentForm((current) => ({ ...current, ownerId: response.user.userId }));
    if (!response.user.mustChangePassword) {
      await loadWorkspace(response.user);
    } else {
      setDocuments([]);
      setVersions([]);
      setApprovals([]);
      setAttachments([]);
      setPolicies([]);
      setManagedUsers([]);
      setSelectedDocument(null);
      setLoadState("idle");
    }
    setAuthState("ready");
  }

  async function handleLogout() {
    await api.logout().catch(() => undefined);
    setUser(null);
    setDocuments([]);
    setVersions([]);
    setApprovals([]);
    setAttachments([]);
    setPolicies([]);
    setManagedUsers([]);
    setSelectedDocument(null);
    setMessage("");
    setError("");
    setAuthState("idle");
  }

  async function handleChangePassword(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    try {
      setError("");
      setMessage("");
      if (passwordForm.newPassword !== passwordForm.confirmPassword) {
        setError("A confirmacao da nova senha nao confere.");
        return;
      }
      const response = await api.changePassword(passwordForm);
      setPasswordForm({ currentPassword: "", newPassword: "", confirmPassword: "" });
      setUser(response.user);
      setLoginForm((current) => ({ ...current, identifier: response.user.username, password: "" }));
      setDocumentForm((current) => ({ ...current, ownerId: response.user.userId }));
      await loadWorkspace(response.user);
      setAuthState("ready");
      setMessage("Senha alterada com sucesso.");
    } catch (err) {
      handleError(err);
    }
  }

  async function handleCreateDocument(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setMessage("");
    try {
      await api.createDocument({
        ...documentForm,
        tags: documentForm.tags.split(",").map((item) => item.trim()).filter(Boolean),
        metadata: documentForm.metadata.trim() ? JSON.parse(documentForm.metadata) : {},
      });
      setDocumentForm({
        ...emptyDocumentForm,
        ownerId: user?.userId ?? "",
        documentType: documentForm.documentType,
        metadata: metadataTextForDocumentType(documentForm.documentType),
      });
      setMessage("Documento criado com sucesso.");
      if (user) await loadWorkspace(user);
    } catch (err) {
      handleError(err);
    }
  }

  async function openDocument(documentId: string) {
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
      setPolicyResourceId(documentId);
      const policyResponse = await api.listAccessPolicies("document", documentId);
      setPolicies(policyResponse.items);
    } catch (err) {
      handleError(err);
    }
  }

  async function handleUploadAttachment(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!selectedDocument || !selectedFile) return;
    try {
      await api.uploadAttachment(selectedDocument.documentId, selectedFile);
      await openDocument(selectedDocument.documentId);
      setMessage("Anexo enviado.");
    } catch (err) {
      handleError(err);
    }
  }

  async function handleCreateUser(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError("");
    setMessage("");
    try {
      await api.createUser(userForm);
      setUserForm(emptyUserForm);
      if (user) await loadWorkspace(user);
      setMessage("Usuario criado. A senha inicial exigira troca no primeiro acesso.");
    } catch (err) {
      handleError(err);
    }
  }

  function handleError(err: unknown) {
    if (statusOf(err) === 401) {
      setUser(null);
      setAuthState("idle");
      setError("Sua sessao expirou. Faca login novamente.");
      return;
    }
    setError(asMessage(err));
  }

  if (authState === "loading") {
    return <div className="app-shell"><section className="hero-panel"><strong>Validando sessao...</strong></section></div>;
  }

  if (!user) {
    return (
      <div className="app-shell auth-shell">
        <section className="hero auth-hero">
          <div>
            <p className="eyebrow">MetalDocs Access</p>
            <h1>Login real com sessao segura e banco persistente em Docker.</h1>
            <p className="hero-copy">O fluxo oficial agora usa cookie HTTP-only e IAM backend-first. O header tecnico deixou de ser o caminho principal.</p>
          </div>
          <form className="hero-panel stack" onSubmit={handleLogin} data-testid="login-form">
            <label><span>Username ou e-mail</span><input data-testid="login-identifier" value={loginForm.identifier} onChange={(event) => setLoginForm({ ...loginForm, identifier: event.target.value })} required /></label>
            <label><span>Senha</span><input data-testid="login-password" type="password" value={loginForm.password} onChange={(event) => setLoginForm({ ...loginForm, password: event.target.value })} required /></label>
            <button data-testid="login-submit" type="submit">Entrar</button>
            {message && <p className="hint">{message}</p>}
            {error && <p className="hint">{error}</p>}
          </form>
        </section>
      </div>
    );
  }

  return (
    <div className="app-shell">
      <header className="hero">
        <div>
          <p className="eyebrow">MetalDocs Control Room</p>
          <h1>Operacao documental profissional com identidade real.</h1>
          <p className="hero-copy">Usuario atual: {user.displayName} ({user.username}) � roles: {user.roles.join(", ")}.</p>
        </div>
        <div className="hero-panel">
          <span>Runtime</span>
          <strong>{api.currentApiBaseUrl}</strong>
          <button data-testid="logout-button" type="button" className="ghost-button" onClick={() => void handleLogout()}>Logout</button>
        </div>
      </header>

      {(message || error) && <section data-testid="app-banner" className={`banner ${error ? "banner-error" : "banner-success"}`}>{error || message}</section>}

      {user.mustChangePassword && (
        <section className="panel auth-panel">
          <div className="panel-heading"><p className="kicker">Seguranca</p><h2>Troca obrigatoria de senha</h2></div>
          <form data-testid="password-change-form" className="stack" onSubmit={handleChangePassword}>
            <p className="hint">No primeiro acesso, a sessao atual ja comprova a senha temporaria. Defina apenas a nova senha para concluir a ativacao.</p>
            <input data-testid="password-new" type="password" placeholder="Nova senha" value={passwordForm.newPassword} onChange={(event) => setPasswordForm({ ...passwordForm, newPassword: event.target.value })} required />
            <input data-testid="password-confirm" type="password" placeholder="Confirmar nova senha" value={passwordForm.confirmPassword} onChange={(event) => setPasswordForm({ ...passwordForm, confirmPassword: event.target.value })} required />
            <button data-testid="password-submit" type="submit">Atualizar senha</button>
          </form>
        </section>
      )}

      {!user.mustChangePassword && (
        <main className="grid-layout wide-grid">
          <section className="panel">
            <div className="panel-heading"><p className="kicker">Cadastro</p><h2>Novo documento</h2></div>
            <form data-testid="document-create-form" className="stack" onSubmit={handleCreateDocument}>
              <input data-testid="document-title" placeholder="Titulo" value={documentForm.title} onChange={(event) => setDocumentForm({ ...documentForm, title: event.target.value })} required />
              <select data-testid="document-type" value={documentForm.documentType} onChange={(event) => setDocumentForm({ ...documentForm, documentType: event.target.value, metadata: metadataTextForDocumentType(event.target.value) })}>{documentTypes.map((item) => <option key={item.code} value={item.code}>{item.name}</option>)}</select>
              <div className="two-columns"><input data-testid="document-owner" placeholder="Owner" value={documentForm.ownerId} onChange={(event) => setDocumentForm({ ...documentForm, ownerId: event.target.value })} /><input data-testid="document-business-unit" placeholder="Business unit" value={documentForm.businessUnit} onChange={(event) => setDocumentForm({ ...documentForm, businessUnit: event.target.value })} /></div>
              <div className="two-columns"><input data-testid="document-department" placeholder="Departamento" value={documentForm.department} onChange={(event) => setDocumentForm({ ...documentForm, department: event.target.value })} /><input data-testid="document-tags" placeholder="Tags" value={documentForm.tags} onChange={(event) => setDocumentForm({ ...documentForm, tags: event.target.value })} /></div>
              <textarea data-testid="document-metadata" rows={4} value={documentForm.metadata} onChange={(event) => setDocumentForm({ ...documentForm, metadata: event.target.value })} />
              <p className="hint">O JSON acima ja nasce com os campos obrigatorios do tipo documental selecionado. Ajuste os valores antes de salvar, se precisar.</p>
              <textarea data-testid="document-initial-content" rows={5} value={documentForm.initialContent} onChange={(event) => setDocumentForm({ ...documentForm, initialContent: event.target.value })} placeholder="Conteudo inicial" />
              <button data-testid="document-submit" type="submit">Criar documento</button>
            </form>
          </section>

          <section data-testid="documents-panel" className="panel">
            <div className="panel-heading split"><div><p className="kicker">Acervo</p><h2>Documentos</h2></div><button type="button" className="ghost-button" onClick={() => user && void loadWorkspace(user)}>Atualizar</button></div>
            <div className="table-shell">
              <table>
                <thead><tr><th>Titulo</th><th>Status</th><th>Owner</th></tr></thead>
                <tbody>{documents.map((item) => <tr key={item.documentId} onClick={() => void openDocument(item.documentId)} className={selectedDocument?.documentId === item.documentId ? "row-active" : ""}><td><strong>{item.title}</strong><small>{item.documentType}</small></td><td>{item.status}</td><td>{item.ownerId}</td></tr>)}</tbody>
              </table>
            </div>
            {loadState === "loading" && <p className="hint">Carregando dados...</p>}
          </section>

          <section className="panel">
            <div className="panel-heading"><p className="kicker">Detalhe</p><h2>{selectedDocument?.title ?? "Selecione um documento"}</h2></div>
            {!selectedDocument ? <p className="hint">Abra um documento para validar versoes, anexos, approvals e policies.</p> : (
              <div className="stack">
                <div className="detail-summary"><div><span>Status</span><strong>{selectedDocument.status}</strong></div><div><span>Tipo</span><strong>{selectedDocument.documentType}</strong></div><div><span>Area</span><strong>{selectedDocument.businessUnit} / {selectedDocument.department}</strong></div><div><span>Owner</span><strong>{selectedDocument.ownerId}</strong></div></div>
                <form className="card stack" onSubmit={handleUploadAttachment}><h3>Anexos</h3><input type="file" onChange={(event) => setSelectedFile(event.target.files?.[0] ?? null)} /><button type="submit" disabled={!selectedFile}>Enviar anexo</button><ul className="mini-list">{attachments.map((item) => <li key={item.attachmentId}><span>{item.fileName}</span><small>{formatDate(item.createdAt)}</small></li>)}</ul></form>
                <div className="card"><h3>Versoes</h3><ul className="mini-list">{versions.map((item) => <li key={item.version}><span>Versao {item.version}</span><small>{item.changeSummary || item.contentHash}</small></li>)}</ul></div>
                <div className="card"><h3>Aprovacoes</h3><ul className="mini-list">{approvals.map((item) => <li key={item.approvalId}><span>{item.status}</span><small>{item.assignedReviewer}</small></li>)}</ul></div>
                <div className="card"><h3>Policies ({policyScope}:{policyResourceId})</h3><ul className="mini-list">{policies.map((item, index) => <li key={`${item.subjectId}-${index}`}><span>{item.subjectType}:{item.subjectId}</span><small>{item.capability} / {item.effect}</small></li>)}</ul></div>
              </div>
            )}
          </section>

          {isAdmin && (
            <section data-testid="managed-users-panel" className="panel panel-admin">
              <div className="panel-heading"><p className="kicker">IAM + Auth</p><h2>Usuarios internos</h2></div>
              <div className="subgrid wide">
                <form data-testid="user-create-form" className="card stack" onSubmit={handleCreateUser}>
                  <h3>Criar usuario</h3>
                  <input data-testid="user-id" placeholder="userId opcional" value={userForm.userId} onChange={(event) => setUserForm({ ...userForm, userId: event.target.value })} />
                  <input data-testid="user-username" placeholder="username" value={userForm.username} onChange={(event) => setUserForm({ ...userForm, username: event.target.value })} required />
                  <input data-testid="user-email" placeholder="email" value={userForm.email} onChange={(event) => setUserForm({ ...userForm, email: event.target.value })} />
                  <input data-testid="user-display-name" placeholder="display name" value={userForm.displayName} onChange={(event) => setUserForm({ ...userForm, displayName: event.target.value })} required />
                  <input data-testid="user-password" type="password" placeholder="senha inicial" value={userForm.password} onChange={(event) => setUserForm({ ...userForm, password: event.target.value })} required />
                  <select data-testid="user-role" value={userForm.roles[0]} onChange={(event) => setUserForm({ ...userForm, roles: [event.target.value as UserRole] })}>{["admin", "editor", "reviewer", "viewer"].map((role) => <option key={role} value={role}>{role}</option>)}</select>
                  <button data-testid="user-submit" type="submit">Criar usuario</button>
                </form>
                <div className="card">
                  <h3>Base de usuarios</h3>
                  <ul className="mini-list">{managedUsers.map((item) => <li key={item.userId}><div><strong>{item.displayName}</strong><p>{item.username} � {item.roles.join(", ")}</p></div><span>{item.isActive ? "Ativo" : "Inativo"}</span></li>)}</ul>
                </div>
              </div>
            </section>
          )}
        </main>
      )}
    </div>
  );
}

function asMessage(error: unknown): string {
  return error instanceof Error ? error.message : "Falha inesperada.";
}

function statusOf(error: unknown): number | undefined {
  if (error && typeof error === "object" && "status" in error && typeof (error as { status?: unknown }).status === "number") {
    return (error as { status: number }).status;
  }
  return undefined;
}

function formatDate(value?: string): string {
  if (!value) return "-";
  return new Intl.DateTimeFormat("pt-BR", { dateStyle: "short", timeStyle: "short" }).format(new Date(value));
}
