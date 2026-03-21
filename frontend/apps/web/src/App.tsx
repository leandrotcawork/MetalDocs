import { Component, useCallback, useEffect } from "react";
import { api } from "./lib.api";
import { AuthShell } from "./components/AuthShell";
import { DocumentCreateView } from "./components/DocumentCreateView";
import { ManagedUsersPanel } from "./components/ManagedUsersPanel";
import { NotificationsPanel } from "./components/NotificationsPanel";
import { OperationsCenter } from "./components/OperationsCenter";
import { PasswordChangePanel } from "./components/PasswordChangePanel";
import { ContentBuilderView } from "./components/content-builder/ContentBuilderView";
import { WorkspacePlaceholder } from "./components/WorkspacePlaceholder";
import type { UserRole } from "./lib.types";
import { useUiStore } from "./store/ui.store";
import { useAuthSession } from "./features/auth/useAuthSession";
import { useDocumentsWorkspace } from "./features/documents/useDocumentsWorkspace";
import { useRegistryExplorer } from "./features/registry/useRegistryExplorer";
import { useNotifications } from "./features/notifications/useNotifications";
import { useManagedUsers } from "./features/iam/useManagedUsers";
import { statusOf } from "./features/shared/errors";
import { DocumentsWorkspaceView } from "./features/documents/DocumentsWorkspaceView";
import { RegistryExplorerView } from "./features/registry/RegistryExplorerView";
import { WorkspaceShell } from "./features/shell/WorkspaceShell";

type PolicyScope = "document" | "document_type" | "area";

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
  const {
    message,
    error,
    isCreateSubmitting,
    activeView,
    searchQuery,
    setMessage,
    setError,
    setActiveView,
    setSearchQuery,
  } = useUiStore();

  const registry = useRegistryExplorer(() => documentsWorkspace.refreshWorkspace(user));
  const documentsWorkspace = useDocumentsWorkspace(registry.applyDocumentProfile, registry.prefetchProfile);
  const notificationsApi = useNotifications();
  const authSession = useAuthSession({ onAuthenticated: documentsWorkspace.loadWorkspace });

  const { authState, user, loginForm, passwordForm, setLoginForm, setPasswordForm, bootstrap, handleLogin, handleLogout, handleChangePassword } = authSession;
  const refreshWorkspace = useCallback(() => documentsWorkspace.refreshWorkspace(user), [documentsWorkspace, user]);
  const managedUsersApi = useManagedUsers(refreshWorkspace);
  const {
    loadState,
    documentForm,
    contentMode,
    contentFile,
    contentPdfUrl,
    contentDocxUrl,
    contentStatus,
    contentError,
    documents,
    selectedDocument,
    versions,
    versionDiff,
    approvals,
    attachments,
    collaborationPresence,
    documentEditLock,
    policies,
    auditEvents,
    selectedFile,
    policyResourceId,
    setDocumentForm,
    setCollaborationPresence,
    setDocumentEditLock,
    setSelectedFile,
    openDocument,
    refreshOperationalSignals,
    handleCreateDocument: handleCreateDocumentInternal,
    handleContentModeChange,
    handleContentFileChange,
    handleDownloadTemplate,
    handleUploadAttachment,
  } = documentsWorkspace;
  const handleCreateDocument = useCallback(
    (event: React.FormEvent<HTMLFormElement>) => handleCreateDocumentInternal(event, user),
    [handleCreateDocumentInternal, user],
  );
  const {
    applyDocumentProfile,
    handleCreateProcessArea,
    handleUpdateProcessArea,
    handleDeleteProcessArea,
    handleCreateSubject,
    handleUpdateSubject,
    handleDeleteSubject,
    handleCreateDocumentProfile,
    handleUpdateDocumentProfile,
    handleDeleteDocumentProfile,
    handleUpdateDocumentProfileGovernance,
    handleUpsertDocumentProfileSchema,
    handleActivateDocumentProfileSchema,
    documentProfiles,
    processAreas,
    documentDepartments,
    subjects,
    selectedProfileSchema,
    selectedProfileSchemas,
    selectedProfileGovernance,
  } = registry;
  const { notifications, handleMarkNotificationRead, subscribeOperations } = notificationsApi;
  const {
    userForm,
    managedUserForm,
    managedUsers,
    setUserForm,
    setManagedUserForm,
    handleCreateUser,
    selectManagedUser,
    toggleManagedUserRole,
    handleSaveManagedUser,
    handleAdminResetPassword,
    handleUnlockManagedUser,
  } = managedUsersApi;
  const policyScope: PolicyScope = "document";

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
  }, [bootstrap]);

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
    return subscribeOperations(() => {
      void refreshOperationalSignals();
    });
  }, [authState, refreshOperationalSignals, subscribeOperations, user?.mustChangePassword, user?.userId]);

  useEffect(() => {
    if (!message) {
      return;
    }
    const timer = window.setTimeout(() => {
      setMessage("");
    }, 2000);
    return () => {
      window.clearTimeout(timer);
    };
  }, [message]);

  useEffect(() => {
    if (!error) {
      return;
    }
    const timer = window.setTimeout(() => {
      setError("");
    }, 6000);
    return () => {
      window.clearTimeout(timer);
    };
  }, [error]);

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

  if (authState === "loading") {
    return <div className="app-shell"><section className="hero-panel"><strong>Validando sessao...</strong></section></div>;
  }

  if (!user) {
    return <AuthShell identifier={loginForm.identifier} password={loginForm.password} message={message} error={error} onIdentifierChange={(identifier) => setLoginForm({ ...loginForm, identifier })} onPasswordChange={(password) => setLoginForm({ ...loginForm, password })} onSubmit={handleLogin} />;
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
        <DocumentsWorkspaceView
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
          contentMode={contentMode}
          contentFile={contentFile}
          contentPdfUrl={contentPdfUrl}
          contentDocxUrl={contentDocxUrl}
          contentStatus={contentStatus}
          contentError={contentError}
          isSubmitting={isCreateSubmitting}
          onDocumentFormChange={setDocumentForm}
          onApplyProfile={applyDocumentProfile}
          onSubmitCreateDocument={handleCreateDocument}
          onContentModeChange={handleContentModeChange}
          onContentFileChange={handleContentFileChange}
          onDownloadTemplate={handleDownloadTemplate}
        />
      );
    }

    if (activeView === "content-builder") {
      return (
        <ContentBuilderView
          document={selectedDocument}
          onBack={() => setActiveView("library")}
        />
      );
    }

    if (activeView === "registry") {
      return (
        <RegistryExplorerView
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
      {(error || message) && (
        <div className="toast-container" aria-live="polite" aria-atomic="false">
          {error && (
            <div data-testid="app-toast-error" className="toast toast-error" role="alert">
              <span className="toast-icon" aria-hidden="true">!</span>
              <div className="toast-body">
                <strong>Falha</strong>
                <span>{error}</span>
              </div>
              <button type="button" className="toast-close" aria-label="Fechar alerta" onClick={() => setError("")}>x</button>
            </div>
          )}
          {message && (
            <div data-testid="app-toast" className="toast toast-success" role="status">
              <span className="toast-icon" aria-hidden="true">v</span>
              <div className="toast-body">
                <strong>Pronto</strong>
                <span>{message}</span>
              </div>
              <button type="button" className="toast-close" aria-label="Fechar mensagem" onClick={() => setMessage("")}>x</button>
            </div>
          )}
        </div>
      )}

      {user.mustChangePassword && (
        <PasswordChangePanel newPassword={passwordForm.newPassword} confirmPassword={passwordForm.confirmPassword} onNewPasswordChange={(newPassword) => setPasswordForm({ ...passwordForm, newPassword })} onConfirmPasswordChange={(confirmPassword) => setPasswordForm({ ...passwordForm, confirmPassword })} onSubmit={handleChangePassword} />
      )}

      {!user.mustChangePassword && (
        <WorkspaceShell
          userDisplayName={user.displayName}
          userRoleLabel={userRoleLabel}
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
        </WorkspaceShell>
      )}
    </div>
  );
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

