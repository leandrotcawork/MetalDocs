import { Component, useEffect, useRef, useState } from "react";
import { api } from "./lib.api";
import { AuthShell } from "./components/AuthShell";
import { DocumentCreateView } from "./components/DocumentCreateView";
import { DocumentWorkspaceShell, type WorkspaceView } from "./components/DocumentWorkspaceShell";
import { DocumentsWorkspace } from "./components/DocumentsWorkspace";
import { ManagedUsersPanel } from "./components/ManagedUsersPanel";
import { NotificationsPanel } from "./components/NotificationsPanel";
import { OperationsCenter } from "./components/OperationsCenter";
import { PasswordChangePanel } from "./components/PasswordChangePanel";
import { RegistryExplorer } from "./components/RegistryExplorer";
import { WorkspacePlaceholder } from "./components/WorkspacePlaceholder";
import type {
  AccessPolicyItem,
  AttachmentItem,
  AuditEventItem,
  CollaborationPresenceItem,
  CurrentUser,
  DocumentEditLockItem,
  DocumentProfileGovernanceItem,
  DocumentProfileItem,
  DocumentProfileSchemaItem,
  DocumentListItem,
  DocumentDepartmentItem,
  ManagedUserItem,
  MetadataFieldRuleItem,
  NotificationItem,
  ProcessAreaItem,
  SearchDocumentItem,
  SubjectItem,
  VersionDiffResponse,
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
    const key = rule.name.trim();
    if (!key) continue;
    metadata[key] = "";
  }
  return JSON.stringify(metadata, null, 2);
}

const emptyDocumentForm = {
  title: "",
  documentType: "po",
  documentProfile: "po",
  processArea: "",
  subject: "",
  ownerId: "",
  businessUnit: "Quality",
  department: "operacoes",
  classification: "INTERNAL",
  audienceMode: "DEPARTMENT",
  audienceDepartment: "operacoes",
  audienceProcessArea: "",
  tags: "",
  effectiveAt: "",
  expiryAt: "",
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
  const [activeView, setActiveView] = useState<WorkspaceView>("operations");
  const [searchQuery, setSearchQuery] = useState("");
  const [loginForm, setLoginForm] = useState({ identifier: "admin", password: "" });
  const [passwordForm, setPasswordForm] = useState({ currentPassword: "", newPassword: "", confirmPassword: "" });
  const [documentForm, setDocumentForm] = useState(emptyDocumentForm);
  const [documentProfiles, setDocumentProfiles] = useState<DocumentProfileItem[]>([]);
  const [processAreas, setProcessAreas] = useState<ProcessAreaItem[]>([]);
  const [documentDepartments, setDocumentDepartments] = useState<DocumentDepartmentItem[]>([]);
  const [subjects, setSubjects] = useState<SubjectItem[]>([]);
  const [selectedProfileSchema, setSelectedProfileSchema] = useState<DocumentProfileSchemaItem | null>(null);
  const [selectedProfileSchemas, setSelectedProfileSchemas] = useState<DocumentProfileSchemaItem[]>([]);
  const [selectedProfileGovernance, setSelectedProfileGovernance] = useState<DocumentProfileGovernanceItem | null>(null);
  const [documents, setDocuments] = useState<SearchDocumentItem[]>([]);
  const [selectedDocument, setSelectedDocument] = useState<DocumentListItem | null>(null);
  const [versions, setVersions] = useState<VersionListItem[]>([]);
  const [versionDiff, setVersionDiff] = useState<VersionDiffResponse | null>(null);
  const [approvals, setApprovals] = useState<WorkflowApprovalItem[]>([]);
  const [attachments, setAttachments] = useState<AttachmentItem[]>([]);
  const [collaborationPresence, setCollaborationPresence] = useState<CollaborationPresenceItem[]>([]);
  const [documentEditLock, setDocumentEditLock] = useState<DocumentEditLockItem | null>(null);
  const [policies, setPolicies] = useState<AccessPolicyItem[]>([]);
  const [auditEvents, setAuditEvents] = useState<AuditEventItem[]>([]);
  const [managedUsers, setManagedUsers] = useState<ManagedUserItem[]>([]);
  const [notifications, setNotifications] = useState<NotificationItem[]>([]);
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [policyScope] = useState<PolicyScope>("document");
  const [policyResourceId, setPolicyResourceId] = useState("");
  const [userForm, setUserForm] = useState(emptyUserForm);
  const [managedUserForm, setManagedUserForm] = useState(emptyManagedUserForm);
  const streamRefreshInFlightRef = useRef(false);

  const currentUserRoles = Array.isArray(user?.roles) ? user.roles : [];
  const isAdmin = currentUserRoles.includes("admin");
  const userRoleLabel = roleLabelFromRoles(currentUserRoles);
  const selectedManagedUser = managedUsers.find((item) => item.userId === managedUserForm.userId) ?? null;
  const visibleDocuments = activeView === "my-docs"
    ? documents.filter((item) => item.ownerId === user?.userId)
    : activeView === "recent"
      ? [...documents].sort((left, right) => new Date(right.createdAt).getTime() - new Date(left.createdAt).getTime())
      : documents;

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

  useEffect(() => {
    if (authState !== "ready" || !user || user.mustChangePassword) {
      return;
    }
    return api.subscribeOperationsStream(
      () => {
        void refreshOperationalSignals();
      },
      () => {
        // Stream keeps retrying in browser; UI fallback remains available.
      },
    );
  }, [authState, user?.mustChangePassword, user?.userId]);

  useEffect(() => {
    if (authState !== "ready" || !selectedDocument?.documentId) {
      return;
    }

    const emitHeartbeat = async () => {
      try {
        await api.heartbeatDocumentPresence(selectedDocument.documentId, { displayName: user?.displayName ?? "" });
        const [presenceResponse, lockResponse] = await Promise.all([
          api.listDocumentPresence(selectedDocument.documentId),
          api.getDocumentEditLock(selectedDocument.documentId).catch((err) => {
            if (statusOf(err) === 404) {
              return null;
            }
            throw err;
          }),
        ]);
        setCollaborationPresence(presenceResponse.items);
        setDocumentEditLock(lockResponse);
      } catch {
        // Collaboration refresh is best-effort and must not block normal workspace usage.
      }
    };

    void emitHeartbeat();
    const timer = window.setInterval(() => {
      void emitHeartbeat();
    }, 30000);
    return () => {
      window.clearInterval(timer);
    };
  }, [authState, selectedDocument?.documentId, user?.displayName]);

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
      const [profilesResponse, processAreasResponse, departmentsResponse, subjectsResponse, docsResponse, usersResponse, notificationsResponse] = await Promise.all([
        api.listDocumentProfiles(),
        api.listProcessAreas(),
        api.listDocumentDepartments(),
        api.listSubjects(),
        api.searchDocuments(new URLSearchParams({ limit: "25" })),
        (Array.isArray(currentUser.roles) ? currentUser.roles : []).includes("admin") ? api.listUsers() : Promise.resolve({ items: [] as ManagedUserItem[] }),
        api.listNotifications(new URLSearchParams({ limit: "10" })),
      ]);
      const profiles = Array.isArray(profilesResponse.items) ? profilesResponse.items : [];
      const areas = Array.isArray(processAreasResponse.items) ? processAreasResponse.items : [];
      const departments = Array.isArray(departmentsResponse.items) ? departmentsResponse.items : [];
      const nextSubjects = Array.isArray(subjectsResponse.items) ? subjectsResponse.items : [];
      const docs = Array.isArray(docsResponse.items) ? docsResponse.items : [];
      const users = Array.isArray(usersResponse.items) ? usersResponse.items : [];
      setDocumentProfiles(profiles);
      setProcessAreas(areas);
      setDocumentDepartments(departments);
      setSubjects(nextSubjects);
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

  async function refreshOperationalSignals() {
    if (streamRefreshInFlightRef.current) {
      return;
    }
    streamRefreshInFlightRef.current = true;
    try {
      const [docsResponse, notificationsResponse] = await Promise.all([
        api.searchDocuments(new URLSearchParams({ limit: "25" })),
        api.listNotifications(new URLSearchParams({ limit: "10" })),
      ]);
      setDocuments(Array.isArray(docsResponse.items) ? docsResponse.items : []);
      setNotifications(Array.isArray(notificationsResponse.items) ? notificationsResponse.items : []);
    } catch {
      // Keep stream resilient: realtime refresh failure should not break UI session.
    } finally {
      streamRefreshInFlightRef.current = false;
    }
  }

  async function applyDocumentProfile(profileCode: string, preferredProcessArea = "") {
    const [schemaResponse, governance] = await Promise.all([
      api.listDocumentProfileSchemas(profileCode),
      api.getDocumentProfileGovernance(profileCode),
    ]);
    const schemas = Array.isArray(schemaResponse.items) ? schemaResponse.items : [];
    const schema = schemas.find((item) => item.isActive) ?? schemas[0] ?? null;
    setSelectedProfileSchemas(schemas);
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
      setSubjects([]);
      setDocuments([]);
      setVersions([]);
      setVersionDiff(null);
      setApprovals([]);
      setAttachments([]);
      setCollaborationPresence([]);
      setDocumentEditLock(null);
      setPolicies([]);
      setAuditEvents([]);
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
    setSelectedProfileSchemas([]);
    setSelectedProfileGovernance(null);
    setSubjects([]);
    setDocuments([]);
    setVersions([]);
    setVersionDiff(null);
    setApprovals([]);
    setAttachments([]);
    setCollaborationPresence([]);
    setDocumentEditLock(null);
    setPolicies([]);
    setAuditEvents([]);
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
      const needsAudience = ["CONFIDENTIAL", "RESTRICTED"].includes(documentForm.classification);
      const audienceMode = documentForm.audienceMode || "DEPARTMENT";
      const audience = needsAudience ? {
        mode: audienceMode,
        departmentCodes: [documentForm.audienceDepartment || documentForm.department].filter(Boolean),
        processAreaCodes: audienceMode === "AREAS"
          ? [documentForm.audienceProcessArea || documentForm.processArea].filter(Boolean)
          : undefined,
      } : undefined;
      await api.createDocument({
        ...documentForm,
        documentType: documentForm.documentProfile,
        documentProfile: documentForm.documentProfile,
        tags: documentForm.tags.split(",").map((item) => item.trim()).filter(Boolean),
        effectiveAt: documentForm.effectiveAt ? new Date(documentForm.effectiveAt).toISOString() : undefined,
        expiryAt: documentForm.expiryAt ? new Date(documentForm.expiryAt).toISOString() : undefined,
        metadata: documentForm.metadata.trim() ? JSON.parse(documentForm.metadata) : {},
        audience,
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
      setActiveView("library");
      if (user) await loadWorkspace(user);
    } catch (err) {
      handleError(err);
    }
  }

  async function openDocument(documentId: string) {
    try {
      const [document, versionsResponse, approvalsResponse, attachmentsResponse, auditResponse, presenceResponse, editLockResponse] = await Promise.all([
        api.getDocument(documentId),
        api.listVersions(documentId),
        api.listApprovals(documentId),
        api.listAttachments(documentId),
        api.listAuditEvents(new URLSearchParams({ resourceType: "document", resourceId: documentId, limit: "10" })),
        api.listDocumentPresence(documentId),
        api.getDocumentEditLock(documentId).catch((err) => {
          if (statusOf(err) === 404) {
            return null;
          }
          throw err;
        }),
      ]);
      const [schema, governance] = await Promise.all([
        api.listDocumentProfileSchemas(document.documentProfile),
        api.getDocumentProfileGovernance(document.documentProfile),
      ]);
      const orderedVersions = [...versionsResponse.items].sort((left, right) => right.version - left.version);
      const schemas = Array.isArray(schema.items) ? schema.items : [];
      setSelectedProfileSchemas(schemas);
      setSelectedProfileSchema(schemas.find((item) => item.isActive) ?? schemas[0] ?? null);
      setSelectedProfileGovernance(governance);
      setSelectedDocument(document);
      setVersions(orderedVersions);
      setApprovals(approvalsResponse.items);
      setAttachments(attachmentsResponse.items);
      setCollaborationPresence(presenceResponse.items);
      setDocumentEditLock(editLockResponse);
      setAuditEvents(auditResponse.items);
      setPolicyResourceId(documentId);
      setActiveView("library");
      const [policyResponse, nextDiff] = await Promise.all([
        api.listAccessPolicies("document", documentId),
        orderedVersions.length >= 2
          ? api.getVersionDiff(documentId, orderedVersions[1].version, orderedVersions[0].version)
          : Promise.resolve(null),
      ]);
      setPolicies(policyResponse.items);
      setVersionDiff(nextDiff);
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
      setSelectedFile(null);
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

  async function handleCreateProcessArea(payload: { code: string; name: string; description: string }) {
    try {
      setError("");
      await api.createProcessArea(payload);
      setMessage("Area de processo criada.");
      await refreshWorkspace();
    } catch (err) {
      handleError(err);
    }
  }

  async function handleCreateDocumentProfile(payload: { code: string; familyCode: string; name: string; alias: string; description: string; reviewIntervalDays: number }) {
    try {
      setError("");
      await api.createDocumentProfile(payload);
      setMessage("Profile criado.");
      await refreshWorkspace();
    } catch (err) {
      handleError(err);
    }
  }

  async function handleUpdateDocumentProfile(payload: { code: string; familyCode: string; name: string; alias: string; description: string; reviewIntervalDays: number }) {
    try {
      setError("");
      await api.updateDocumentProfile(payload.code, payload);
      setMessage("Profile atualizado.");
      await refreshWorkspace();
    } catch (err) {
      handleError(err);
    }
  }

  async function handleDeleteDocumentProfile(code: string) {
    try {
      setError("");
      await api.deleteDocumentProfile(code);
      setMessage("Profile desativado.");
      await refreshWorkspace();
    } catch (err) {
      handleError(err);
    }
  }

  async function handleUpdateDocumentProfileGovernance(payload: { profileCode: string; workflowProfile: string; reviewIntervalDays: number; approvalRequired: boolean; retentionDays: number; validityDays: number }) {
    try {
      setError("");
      await api.updateDocumentProfileGovernance(payload.profileCode, payload);
      setMessage("Governanca atualizada.");
      await refreshWorkspace();
    } catch (err) {
      handleError(err);
    }
  }

  async function handleUpsertDocumentProfileSchema(payload: { profileCode: string; version: number; isActive: boolean; metadataRules: MetadataFieldRuleItem[] }) {
    try {
      setError("");
      await api.upsertDocumentProfileSchema(payload.profileCode, payload);
      setMessage("Schema versionado atualizado.");
      await applyDocumentProfile(payload.profileCode, documentForm.processArea);
    } catch (err) {
      handleError(err);
    }
  }

  async function handleActivateDocumentProfileSchema(payload: { profileCode: string; version: number }) {
    try {
      setError("");
      await api.activateDocumentProfileSchema(payload.profileCode, payload.version);
      setMessage("Schema ativo atualizado.");
      await applyDocumentProfile(payload.profileCode, documentForm.processArea);
    } catch (err) {
      handleError(err);
    }
  }

  async function handleUpdateProcessArea(payload: { code: string; name: string; description: string }) {
    try {
      setError("");
      await api.updateProcessArea(payload.code, payload);
      setMessage("Area de processo atualizada.");
      await refreshWorkspace();
    } catch (err) {
      handleError(err);
    }
  }

  async function handleDeleteProcessArea(code: string) {
    try {
      setError("");
      await api.deleteProcessArea(code);
      setMessage("Area de processo desativada.");
      await refreshWorkspace();
    } catch (err) {
      handleError(err);
    }
  }

  async function handleCreateSubject(payload: { code: string; processAreaCode: string; name: string; description: string }) {
    try {
      setError("");
      await api.createSubject(payload);
      setMessage("Subject criado.");
      await refreshWorkspace();
    } catch (err) {
      handleError(err);
    }
  }

  async function handleUpdateSubject(payload: { code: string; processAreaCode: string; name: string; description: string }) {
    try {
      setError("");
      await api.updateSubject(payload.code, payload);
      setMessage("Subject atualizado.");
      await refreshWorkspace();
    } catch (err) {
      handleError(err);
    }
  }

  async function handleDeleteSubject(code: string) {
    try {
      setError("");
      await api.deleteSubject(code);
      setMessage("Subject desativado.");
      await refreshWorkspace();
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

  const currentUser = user;

  function refreshWorkspace() {
    return loadWorkspace(currentUser);
  }

  function renderWorkspaceView() {
    if (activeView === "operations" || activeView === "approvals" || activeView === "audit") {
      return (
        <OperationsCenter
          loadState={loadState}
          documents={activeView === "approvals" ? documents.filter((item) => item.status === "IN_REVIEW") : documents}
          notifications={notifications}
          documentProfiles={documentProfiles}
          processAreas={processAreas}
          formatDate={formatDate}
          onCreateDocument={() => setActiveView("create")}
          onRefreshWorkspace={refreshWorkspace}
          onOpenDocument={openDocument}
        />
      );
    }

    if (activeView === "library" || activeView === "my-docs" || activeView === "recent") {
      return (
        <DocumentsWorkspace
          view={activeView}
          loadState={loadState}
          documentProfiles={documentProfiles}
          processAreas={processAreas}
          documents={visibleDocuments}
          selectedDocument={selectedDocument}
          selectedProfileGovernance={selectedProfileGovernance}
          versions={versions}
          versionDiff={versionDiff}
          approvals={approvals}
          attachments={attachments}
          collaborationPresence={collaborationPresence}
          documentEditLock={documentEditLock}
          policies={policies}
          auditEvents={auditEvents}
          selectedFile={selectedFile}
          policyScope={policyScope}
          policyResourceId={policyResourceId}
          searchQuery={searchQuery}
          formatDate={formatDate}
          onRefreshWorkspace={refreshWorkspace}
          onOpenDocument={openDocument}
          onFileChange={setSelectedFile}
          onUploadAttachment={handleUploadAttachment}
        />
      );
    }

    if (activeView === "create") {
      return (
        <DocumentCreateView
          documentForm={documentForm}
          documentProfiles={documentProfiles}
          processAreas={processAreas}
          documentDepartments={documentDepartments}
          subjects={subjects}
          selectedProfileSchema={selectedProfileSchema}
          selectedProfileGovernance={selectedProfileGovernance}
          onDocumentFormChange={setDocumentForm}
          onApplyProfile={applyDocumentProfile}
          onSubmitCreateDocument={handleCreateDocument}
        />
      );
    }

    if (activeView === "registry") {
      return (
        <RegistryExplorer
          loadState={loadState}
          documentProfiles={documentProfiles}
          processAreas={processAreas}
          subjects={subjects}
          selectedProfileCode={documentForm.documentProfile}
          selectedProfileSchema={selectedProfileSchema}
          selectedProfileSchemas={selectedProfileSchemas}
          selectedProfileGovernance={selectedProfileGovernance}
          showAdmin={isAdmin}
          onRefreshWorkspace={refreshWorkspace}
          onSelectProfile={(profileCode) => applyDocumentProfile(profileCode, documentForm.processArea)}
          onCreateProcessArea={handleCreateProcessArea}
          onUpdateProcessArea={handleUpdateProcessArea}
          onDeleteProcessArea={handleDeleteProcessArea}
          onCreateSubject={handleCreateSubject}
          onUpdateSubject={handleUpdateSubject}
          onDeleteSubject={handleDeleteSubject}
          onCreateDocumentProfile={handleCreateDocumentProfile}
          onUpdateDocumentProfile={handleUpdateDocumentProfile}
          onDeleteDocumentProfile={handleDeleteDocumentProfile}
          onUpdateDocumentProfileGovernance={handleUpdateDocumentProfileGovernance}
          onUpsertDocumentProfileSchema={handleUpsertDocumentProfileSchema}
          onActivateDocumentProfileSchema={handleActivateDocumentProfileSchema}
        />
      );
    }

    if (activeView === "notifications") {
      return (
        <NotificationsPanel
          loadState={loadState}
          notifications={notifications}
          formatDate={formatDate}
          onRefreshWorkspace={refreshWorkspace}
          onMarkRead={handleMarkNotificationRead}
        />
      );
    }

    if (activeView === "admin" && isAdmin) {
      return (
        <ManagedUsersPanel
          loadState={loadState}
          userForm={userForm}
          managedUserForm={managedUserForm}
          managedUsers={managedUsers}
          selectedManagedUser={selectedManagedUser}
          formatDate={formatDate}
          onRefreshWorkspace={refreshWorkspace}
          onUserFormChange={setUserForm}
          onManagedUserFormChange={setManagedUserForm}
          onSubmitCreateUser={handleCreateUser}
          onSelectManagedUser={selectManagedUser}
          onToggleRole={toggleManagedUserRole}
          onSaveManagedUser={handleSaveManagedUser}
          onAdminResetPassword={handleAdminResetPassword}
          onUnlockManagedUser={handleUnlockManagedUser}
        />
      );
    }

    return (
      <WorkspacePlaceholder
        kicker="Workspace"
        title="Workspace"
        description="Selecione uma visao operacional na barra lateral para continuar."
        bullets={[
          "Acesse Documentos para explorar o acervo e revisar detalhes.",
          "Use Novo documento para iniciar o fluxo Profile -> Metadata -> Content -> Review.",
          "Abra Tipos documentais para consultar regras e governanca por perfil.",
        ]}
      />
    );
  }

  const workspaceView = renderWorkspaceView();

  return (
    <div className={`app-shell ${!user.mustChangePassword ? "is-workspace" : ""}`}>
      {(message || error) && <section data-testid="app-banner" className={`banner ${error ? "banner-error" : "banner-success"}`}>{error || message}</section>}

      {user.mustChangePassword && (
        <PasswordChangePanel newPassword={passwordForm.newPassword} confirmPassword={passwordForm.confirmPassword} onNewPasswordChange={(newPassword) => setPasswordForm({ ...passwordForm, newPassword })} onConfirmPasswordChange={(confirmPassword) => setPasswordForm({ ...passwordForm, confirmPassword })} onSubmit={handleChangePassword} />
      )}

      {!user.mustChangePassword && (
        <DocumentWorkspaceShell
          userDisplayName={user.displayName}
          userRoleLabel={userRoleLabel}
          organizationLabel="Metal Nobre"
          activeView={activeView}
          searchValue={searchQuery}
          notificationsPending={notifications.filter((item) => item.status !== "READ").length}
          documentCount={documents.length}
          reviewCount={documents.filter((item) => item.status === "IN_REVIEW").length}
          registryCount={documentProfiles.length}
          showAdmin={isAdmin}
          documentProfiles={documentProfiles}
          processAreas={processAreas}
          documents={documents}
          onSearchChange={setSearchQuery}
          onNavigate={setActiveView}
          onPrimaryAction={() => setActiveView("create")}
          onRefreshWorkspace={refreshWorkspace}
          isRefreshing={loadState === "loading"}
          onLogout={handleLogout}
        >
          {workspaceView}
        </DocumentWorkspaceShell>
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

function roleLabelFromRoles(roles: UserRole[]): string {
  if (roles.includes("admin")) return "Administrador";
  if (roles.includes("reviewer")) return "Revisor";
  if (roles.includes("editor")) return "Editor";
  return "Visualizador";
}
