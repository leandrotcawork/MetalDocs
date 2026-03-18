import { Component, useEffect, useState } from "react";
import { api } from "./lib.api";
import type {
  AccessPolicyItem,
  AttachmentItem,
  CurrentUser,
  DocumentProfileGovernanceItem,
  DocumentProfileItem,
  DocumentProfileSchemaItem,
  DocumentListItem,
  ManagedUserItem,
  NotificationItem,
  ProcessAreaItem,
  SearchDocumentItem,
  UserRole,
  VersionListItem,
  WorkflowApprovalItem,
} from "./lib.types";

type LoadState = "idle" | "loading" | "ready" | "error";
type PolicyScope = "document" | "document_type" | "area";

function metadataValueExample(rule: { name: string; type: string }, profileCode: string): string {
  if (rule.type === "date") {
    return rule.name.includes("end") || rule.name.includes("expiry") ? "2026-12-31" : "2026-01-01";
  }
  if (rule.name.endsWith("_code") || rule.name.endsWith("_number")) {
    return `${profileCode.toUpperCase()}-001`;
  }
  if (rule.name.includes("issuer")) {
    return "Organizacao Exemplo";
  }
  if (rule.name.includes("counterparty") || rule.name.includes("supplier")) {
    return "Metal Nobre";
  }
  if (rule.name.includes("plant")) {
    return "Matriz";
  }
  if (rule.name.includes("revision")) {
    return "A";
  }
  if (rule.name.includes("period")) {
    return "2026-Q1";
  }
  return "preencher";
}

function metadataTextForProfileSchema(profileCode: string, schema?: DocumentProfileSchemaItem | null): string {
  const metadata: Record<string, string> = {};
  for (const rule of schema?.metadataRules ?? []) {
    metadata[rule.name] = metadataValueExample(rule, profileCode);
  }
  return JSON.stringify(metadata, null, 2);
}

const emptyDocumentForm = {
  title: "",
  documentType: "policy",
  documentProfile: "policy",
  processArea: "",
  subject: "",
  ownerId: "",
  businessUnit: "Quality",
  department: "Operations",
  classification: "INTERNAL",
  tags: "",
  metadata: "{}",
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

type AppErrorBoundaryState = {
  hasError: boolean;
  message: string;
};

class AppErrorBoundary extends Component<{ children: React.ReactNode }, AppErrorBoundaryState> {
  state: AppErrorBoundaryState = {
    hasError: false,
    message: "",
  };

  static getDerivedStateFromError(error: Error): AppErrorBoundaryState {
    return {
      hasError: true,
      message: error.message || "Falha inesperada ao renderizar a interface.",
    };
  }

  componentDidCatch(error: Error): void {
    console.error("MetalDocs UI render error", error);
  }

  render() {
    if (this.state.hasError) {
      return (
        <div className="app-shell">
          <section className="hero-panel stack">
            <strong>Falha ao montar a interface.</strong>
            <p className="hint">{this.state.message}</p>
            <p className="hint">A API respondeu, mas a interface encontrou um dado inesperado durante o render. Recarregue a pagina apos atualizar o frontend local.</p>
          </section>
        </div>
      );
    }
    return this.props.children;
  }
}

export default function App() {
  return (
    <AppErrorBoundary>
      <AppContent />
    </AppErrorBoundary>
  );
}

function AppContent() {
  const [authState, setAuthState] = useState<LoadState>("loading");
  const [loadState, setLoadState] = useState<LoadState>("idle");
  const [user, setUser] = useState<CurrentUser | null>(null);
  const [message, setMessage] = useState("");
  const [error, setError] = useState("");
  const [loginForm, setLoginForm] = useState({ identifier: "admin", password: "" });
  const [passwordForm, setPasswordForm] = useState({ currentPassword: "", newPassword: "", confirmPassword: "" });
  const [documentForm, setDocumentForm] = useState(emptyDocumentForm);
  const [documentProfiles, setDocumentProfiles] = useState<DocumentProfileItem[]>([]);
  const [processAreas, setProcessAreas] = useState<ProcessAreaItem[]>([]);
  const [selectedProfileSchema, setSelectedProfileSchema] = useState<DocumentProfileSchemaItem | null>(null);
  const [selectedProfileGovernance, setSelectedProfileGovernance] = useState<DocumentProfileGovernanceItem | null>(null);
  const [documents, setDocuments] = useState<SearchDocumentItem[]>([]);
  const [selectedDocument, setSelectedDocument] = useState<DocumentListItem | null>(null);
  const [versions, setVersions] = useState<VersionListItem[]>([]);
  const [approvals, setApprovals] = useState<WorkflowApprovalItem[]>([]);
  const [attachments, setAttachments] = useState<AttachmentItem[]>([]);
  const [policies, setPolicies] = useState<AccessPolicyItem[]>([]);
  const [managedUsers, setManagedUsers] = useState<ManagedUserItem[]>([]);
  const [notifications, setNotifications] = useState<NotificationItem[]>([]);
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [policyScope, setPolicyScope] = useState<PolicyScope>("document");
  const [policyResourceId, setPolicyResourceId] = useState("");
  const [userForm, setUserForm] = useState(emptyUserForm);

  const currentUserRoles = Array.isArray(user?.roles) ? user.roles : [];
  const isAdmin = currentUserRoles.includes("admin");

  useEffect(() => {
    void bootstrap();
  }, []);

  async function bootstrap() {
    try {
      const currentUser = await api.me();
      setUser(currentUser);
      setDocumentForm((current) => ({ ...current, ownerId: currentUser.userId }));
      if (!currentUser.mustChangePassword) {
        await loadWorkspace(currentUser);
      }
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
      const [profilesResponse, processAreasResponse, docsResponse, usersResponse, notificationsResponse] = await Promise.all([
        api.listDocumentProfiles(),
        api.listProcessAreas(),
        api.searchDocuments(new URLSearchParams({ limit: "25" })),
        (Array.isArray(currentUser.roles) ? currentUser.roles : []).includes("admin") ? api.listUsers() : Promise.resolve({ items: [] as ManagedUserItem[] }),
        api.listNotifications(new URLSearchParams({ limit: "10" })),
      ]);
      const profiles = Array.isArray(profilesResponse.items) ? profilesResponse.items : [];
      const areas = Array.isArray(processAreasResponse.items) ? processAreasResponse.items : [];
      const docs = Array.isArray(docsResponse.items) ? docsResponse.items : [];
      const users = Array.isArray(usersResponse.items) ? usersResponse.items : [];
      setDocumentProfiles(profiles);
      setProcessAreas(areas);
      setDocuments(docs);
      setManagedUsers(users);
      setNotifications(Array.isArray(notificationsResponse.items) ? notificationsResponse.items : []);
      const nextProfileCode = profiles.find((item) => item.code === documentForm.documentProfile)?.code ?? profiles[0]?.code ?? "";
      if (nextProfileCode) {
        await applyDocumentProfile(nextProfileCode, documentForm.processArea);
      }
      setLoadState("ready");
    } catch (err) {
      handleError(err);
      setLoadState("error");
    }
  }

  async function applyDocumentProfile(profileCode: string, preferredProcessArea = "") {
    const [schema, governance] = await Promise.all([
      api.getDocumentProfileSchema(profileCode),
      api.getDocumentProfileGovernance(profileCode),
    ]);
    setSelectedProfileSchema(schema);
    setSelectedProfileGovernance(governance);
    setDocumentForm((current) => ({
      ...current,
      documentType: profileCode,
      documentProfile: profileCode,
      processArea: preferredProcessArea,
      metadata: metadataTextForProfileSchema(profileCode, schema),
    }));
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
      setNotifications([]);
      setSelectedDocument(null);
      setLoadState("idle");
    }
    setAuthState("ready");
  }

  async function handleLogout() {
    await api.logout().catch(() => undefined);
    setUser(null);
    setSelectedProfileSchema(null);
    setSelectedProfileGovernance(null);
    setDocuments([]);
    setVersions([]);
    setApprovals([]);
    setAttachments([]);
    setPolicies([]);
    setManagedUsers([]);
    setNotifications([]);
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
        documentType: documentForm.documentProfile,
        documentProfile: documentForm.documentProfile,
        tags: documentForm.tags.split(",").map((item) => item.trim()).filter(Boolean),
        metadata: documentForm.metadata.trim() ? JSON.parse(documentForm.metadata) : {},
      });
      setDocumentForm({
        ...emptyDocumentForm,
        ownerId: user?.userId ?? "",
        documentType: documentForm.documentProfile,
        documentProfile: documentForm.documentProfile,
        processArea: documentForm.processArea,
        metadata: metadataTextForProfileSchema(documentForm.documentProfile, selectedProfileSchema),
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

  async function handleMarkNotificationRead(notificationId: string) {
    try {
      await api.markNotificationRead(notificationId);
      setNotifications((current) => current.map((item) => item.id === notificationId ? { ...item, status: "READ", readAt: new Date().toISOString() } : item));
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
          <p className="hero-copy">Usuario atual: {user.displayName} ({user.username}) - roles: {currentUserRoles.join(", ") || "sem role"}.</p>
        </div>
        <div className="hero-panel">
          <span>Runtime</span>
          <strong>{api.currentApiBaseUrl}</strong>
          <small>{notifications.filter((item) => item.status !== "READ").length} notificacao(oes) pendentes</small>
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
              <select
                data-testid="document-profile"
                value={documentForm.documentProfile}
                onChange={(event) => void applyDocumentProfile(event.target.value, documentForm.processArea)}
              >
                {documentProfiles.map((item) => <option key={item.code} value={item.code}>{item.name} ({item.code})</option>)}
              </select>
              <div className="detail-summary">
                <div><span>Family</span><strong>{documentProfiles.find((item) => item.code === documentForm.documentProfile)?.familyCode ?? "-"}</strong></div>
                <div><span>Workflow</span><strong>{selectedProfileGovernance?.workflowProfile ?? "-"}</strong></div>
                <div><span>Revisao</span><strong>{selectedProfileGovernance ? `${selectedProfileGovernance.reviewIntervalDays} dias` : "-"}</strong></div>
                <div><span>Aprovacao</span><strong>{selectedProfileGovernance?.approvalRequired ? "Obrigatoria" : "Opcional"}</strong></div>
              </div>
              <div className="two-columns">
                <select data-testid="document-process-area" value={documentForm.processArea} onChange={(event) => setDocumentForm({ ...documentForm, processArea: event.target.value })}>
                  <option value="">Sem process area</option>
                  {processAreas.map((item) => <option key={item.code} value={item.code}>{item.name}</option>)}
                </select>
                <input data-testid="document-subject" placeholder="Subject opcional" value={documentForm.subject} onChange={(event) => setDocumentForm({ ...documentForm, subject: event.target.value })} />
              </div>
              <div className="two-columns"><input data-testid="document-owner" placeholder="Owner" value={documentForm.ownerId} onChange={(event) => setDocumentForm({ ...documentForm, ownerId: event.target.value })} /><input data-testid="document-business-unit" placeholder="Business unit" value={documentForm.businessUnit} onChange={(event) => setDocumentForm({ ...documentForm, businessUnit: event.target.value })} /></div>
              <div className="two-columns"><input data-testid="document-department" placeholder="Departamento" value={documentForm.department} onChange={(event) => setDocumentForm({ ...documentForm, department: event.target.value })} /><input data-testid="document-tags" placeholder="Tags" value={documentForm.tags} onChange={(event) => setDocumentForm({ ...documentForm, tags: event.target.value })} /></div>
              <textarea data-testid="document-metadata" rows={4} value={documentForm.metadata} onChange={(event) => setDocumentForm({ ...documentForm, metadata: event.target.value })} />
              <p className="hint">O JSON acima nasce do schema ativo do profile selecionado. Ajuste os valores mantendo os campos obrigatorios exigidos pelo registry.</p>
              {selectedProfileSchema && <p className="hint">Schema ativo: v{selectedProfileSchema.version} com {(selectedProfileSchema.metadataRules ?? []).length} regra(s) de metadata.</p>}
              <textarea data-testid="document-initial-content" rows={5} value={documentForm.initialContent} onChange={(event) => setDocumentForm({ ...documentForm, initialContent: event.target.value })} placeholder="Conteudo inicial" />
              <button data-testid="document-submit" type="submit">Criar documento</button>
            </form>
          </section>

          <section data-testid="documents-panel" className="panel">
            <div className="panel-heading split"><div><p className="kicker">Acervo</p><h2>Documentos</h2></div><button type="button" className="ghost-button" onClick={() => user && void loadWorkspace(user)}>Atualizar</button></div>
            <div className="table-shell">
              <table>
                <thead><tr><th>Titulo</th><th>Status</th><th>Owner</th></tr></thead>
                <tbody>{documents.map((item) => <tr key={item.documentId} onClick={() => void openDocument(item.documentId)} className={selectedDocument?.documentId === item.documentId ? "row-active" : ""}><td><strong>{item.title}</strong><small>{item.documentProfile} / {item.documentFamily}{item.processArea ? ` / ${item.processArea}` : ""}</small></td><td>{item.status}</td><td>{item.ownerId}</td></tr>)}</tbody>
              </table>
            </div>
            {loadState === "loading" && <p className="hint">Carregando dados...</p>}
          </section>

          <section className="panel">
            <div className="panel-heading"><p className="kicker">Detalhe</p><h2>{selectedDocument?.title ?? "Selecione um documento"}</h2></div>
            {!selectedDocument ? <p className="hint">Abra um documento para validar versoes, anexos, approvals e policies.</p> : (
              <div className="stack">
                <div className="detail-summary"><div><span>Status</span><strong>{selectedDocument.status}</strong></div><div><span>Profile</span><strong>{selectedDocument.documentProfile}</strong></div><div><span>Family</span><strong>{selectedDocument.documentFamily}</strong></div><div><span>Owner</span><strong>{selectedDocument.ownerId}</strong></div></div>
                <div className="detail-summary"><div><span>Processo</span><strong>{selectedDocument.processArea || "-"}</strong></div><div><span>Subject</span><strong>{selectedDocument.subject || "-"}</strong></div><div><span>Area</span><strong>{selectedDocument.businessUnit} / {selectedDocument.department}</strong></div><div><span>Schema</span><strong>v{selectedDocument.profileSchemaVersion ?? 1}</strong></div></div>
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
                  <ul className="mini-list">{managedUsers.map((item) => <li key={item.userId}><div><strong>{item.displayName}</strong><p>{item.username} - {(Array.isArray(item.roles) ? item.roles : []).join(", ") || "sem role"}</p></div><span>{item.isActive ? "Ativo" : "Inativo"}</span></li>)}</ul>
                </div>
              </div>
            </section>
          )}

          <section className="panel">
            <div className="panel-heading"><p className="kicker">Operacao</p><h2>Notificacoes</h2></div>
            <ul className="mini-list">
              {notifications.map((item) => <li key={item.id}><div><strong>{item.title}</strong><p>{item.message}</p><small>{item.eventType} / {formatDate(item.createdAt)}</small></div><div className="stack"><span>{item.status}</span>{item.status !== "READ" && <button type="button" className="ghost-button" onClick={() => void handleMarkNotificationRead(item.id)}>Marcar como lida</button>}</div></li>)}
              {notifications.length === 0 && <li><span>Nenhuma notificacao disponivel.</span></li>}
            </ul>
          </section>
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

