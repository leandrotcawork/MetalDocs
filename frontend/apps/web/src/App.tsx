import { Component, useEffect, useState } from "react";
import { api } from "./lib.api";
import { AppShellHeader } from "./components/AppShellHeader";
import { AuthShell } from "./components/AuthShell";
import { DocumentsWorkspace } from "./components/DocumentsWorkspace";
import { ManagedUsersPanel } from "./components/ManagedUsersPanel";
import { NotificationsPanel } from "./components/NotificationsPanel";
import { PasswordChangePanel } from "./components/PasswordChangePanel";
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

const emptyManagedUserForm = {
  userId: "",
  displayName: "",
  email: "",
  isActive: true,
  mustChangePassword: false,
  roles: ["viewer"] as UserRole[],
  resetPassword: "",
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
  const [policyScope] = useState<PolicyScope>("document");
  const [policyResourceId, setPolicyResourceId] = useState("");
  const [userForm, setUserForm] = useState(emptyUserForm);
  const [managedUserForm, setManagedUserForm] = useState(emptyManagedUserForm);

  const currentUserRoles = Array.isArray(user?.roles) ? user.roles : [];
  const isAdmin = currentUserRoles.includes("admin");
  const selectedManagedUser = managedUsers.find((item) => item.userId === managedUserForm.userId) ?? null;

  useEffect(() => {
    void bootstrap();
  }, []);

  useEffect(() => {
    if (!managedUserForm.userId) {
      return;
    }
    const current = managedUsers.find((item) => item.userId === managedUserForm.userId);
    if (!current) {
      return;
    }
    setManagedUserForm((previous) => ({
      ...previous,
      displayName: current.displayName,
      email: current.email ?? "",
      isActive: current.isActive,
      mustChangePassword: current.mustChangePassword,
      roles: Array.isArray(current.roles) && current.roles.length > 0 ? current.roles : previous.roles,
      resetPassword: "",
    }));
  }, [managedUsers, managedUserForm.userId]);

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

  function selectManagedUser(item: ManagedUserItem) {
    setManagedUserForm({
      userId: item.userId,
      displayName: item.displayName,
      email: item.email ?? "",
      isActive: item.isActive,
      mustChangePassword: item.mustChangePassword,
      roles: Array.isArray(item.roles) && item.roles.length > 0 ? item.roles : ["viewer"],
      resetPassword: "",
    });
  }

  function toggleManagedUserRole(role: UserRole) {
    setManagedUserForm((current) => {
      const hasRole = current.roles.includes(role);
      const nextRoles = hasRole ? current.roles.filter((item) => item !== role) : [...current.roles, role];
      return {
        ...current,
        roles: nextRoles.length > 0 ? nextRoles : current.roles,
      };
    });
  }

  async function handleSaveManagedUser() {
    if (!managedUserForm.userId) {
      setError("Selecione um usuario para editar.");
      return;
    }
    if (managedUserForm.roles.length === 0) {
      setError("Selecione pelo menos uma role.");
      return;
    }
    try {
      setError("");
      setMessage("");
      await api.updateUser(managedUserForm.userId, {
        displayName: managedUserForm.displayName,
        email: managedUserForm.email,
        isActive: managedUserForm.isActive,
        mustChangePassword: managedUserForm.mustChangePassword,
      });
      await api.replaceUserRoles(managedUserForm.userId, {
        displayName: managedUserForm.displayName,
        roles: managedUserForm.roles,
      });
      if (user) {
        await loadWorkspace(user);
      }
      setMessage("Usuario administrativo atualizado com sucesso.");
    } catch (err) {
      handleError(err);
    }
  }

  async function handleAdminResetPassword() {
    if (!managedUserForm.userId) {
      setError("Selecione um usuario para resetar a senha.");
      return;
    }
    if (!managedUserForm.resetPassword.trim()) {
      setError("Informe a nova senha temporaria.");
      return;
    }
    try {
      setError("");
      setMessage("");
      await api.adminResetPassword(managedUserForm.userId, {
        newPassword: managedUserForm.resetPassword,
      });
      setManagedUserForm((current) => ({
        ...current,
        resetPassword: "",
        mustChangePassword: true,
      }));
      if (user) {
        await loadWorkspace(user);
      }
      setMessage("Senha administrativa resetada. O usuario precisara trocar no proximo login.");
    } catch (err) {
      handleError(err);
    }
  }

  async function handleUnlockManagedUser() {
    if (!managedUserForm.userId) {
      setError("Selecione um usuario para desbloquear.");
      return;
    }
    try {
      setError("");
      setMessage("");
      await api.unlockUser(managedUserForm.userId);
      if (user) {
        await loadWorkspace(user);
      }
      setMessage("Usuario desbloqueado com sucesso.");
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
    return <AuthShell identifier={loginForm.identifier} password={loginForm.password} message={message} error={error} onIdentifierChange={(identifier) => setLoginForm({ ...loginForm, identifier })} onPasswordChange={(password) => setLoginForm({ ...loginForm, password })} onSubmit={handleLogin} />;
  }

  return (
    <div className="app-shell">
      <AppShellHeader user={user} apiBaseUrl={api.currentApiBaseUrl} currentUserRoles={currentUserRoles} notifications={notifications} onLogout={handleLogout} />

      {(message || error) && <section data-testid="app-banner" className={`banner ${error ? "banner-error" : "banner-success"}`}>{error || message}</section>}

      {user.mustChangePassword && (
        <PasswordChangePanel newPassword={passwordForm.newPassword} confirmPassword={passwordForm.confirmPassword} onNewPasswordChange={(newPassword) => setPasswordForm({ ...passwordForm, newPassword })} onConfirmPasswordChange={(confirmPassword) => setPasswordForm({ ...passwordForm, confirmPassword })} onSubmit={handleChangePassword} />
      )}

      {!user.mustChangePassword && (
        <main className="grid-layout wide-grid">
          <DocumentsWorkspace
            loadState={loadState}
            documentForm={documentForm}
            documentProfiles={documentProfiles}
            processAreas={processAreas}
            selectedProfileSchema={selectedProfileSchema}
            selectedProfileGovernance={selectedProfileGovernance}
            documents={documents}
            selectedDocument={selectedDocument}
            versions={versions}
            approvals={approvals}
            attachments={attachments}
            policies={policies}
            selectedFile={selectedFile}
            policyScope={policyScope}
            policyResourceId={policyResourceId}
            onDocumentFormChange={setDocumentForm}
            onSubmitCreateDocument={handleCreateDocument}
            onApplyProfile={applyDocumentProfile}
            onRefreshWorkspace={() => user && loadWorkspace(user)}
            onOpenDocument={openDocument}
            onFileChange={setSelectedFile}
            onUploadAttachment={handleUploadAttachment}
          />

          {isAdmin && (
            <ManagedUsersPanel
              userForm={userForm}
              managedUserForm={managedUserForm}
              managedUsers={managedUsers}
              selectedManagedUser={selectedManagedUser}
              formatDate={formatDate}
              onUserFormChange={setUserForm}
              onManagedUserFormChange={setManagedUserForm}
              onSubmitCreateUser={handleCreateUser}
              onSelectManagedUser={selectManagedUser}
              onToggleRole={toggleManagedUserRole}
              onSaveManagedUser={handleSaveManagedUser}
              onAdminResetPassword={handleAdminResetPassword}
              onUnlockManagedUser={handleUnlockManagedUser}
            />
          )}

          <NotificationsPanel notifications={notifications} formatDate={formatDate} onMarkRead={handleMarkNotificationRead} />
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
