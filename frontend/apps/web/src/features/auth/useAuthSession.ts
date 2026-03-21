import { useCallback } from "react";
import { api } from "../../lib.api";
import type { CurrentUser } from "../../lib.types";
import { useAuthStore } from "../../store/auth.store";
import { useDocumentsStore } from "../../store/documents.store";
import { useNotificationsStore } from "../../store/notifications.store";
import { useRegistryStore } from "../../store/registry.store";
import { useUiStore } from "../../store/ui.store";
import { asMessage, statusOf } from "../shared/errors";

type UseAuthSessionOptions = {
  onAuthenticated: (user: CurrentUser) => Promise<void>;
};

export function useAuthSession({ onAuthenticated }: UseAuthSessionOptions) {
  const { authState, user, loginForm, passwordForm, setAuthState, setUser, setLoginForm, setPasswordForm } = useAuthStore();
  const {
    setLoadState,
    setDocuments,
    setSelectedDocument,
    setVersions,
    setVersionDiff,
    setApprovals,
    setAttachments,
    setCollaborationPresence,
    setDocumentEditLock,
    setPolicies,
    setAuditEvents,
    setDocumentForm,
    setContentMode,
    setContentFile,
    setContentPdfUrl,
    setContentDocxUrl,
    setContentStatus,
    setContentError,
  } = useDocumentsStore();
  const { setNotifications } = useNotificationsStore();
  const { setSelectedProfileSchema, setSelectedProfileSchemas, setSelectedProfileGovernance, setSubjects } = useRegistryStore();
  const { setMessage, setError, setManagedUsers } = useUiStore();

  const clearWorkspaceAfterPasswordChange = useCallback(() => {
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
  }, [
    setApprovals,
    setAttachments,
    setAuditEvents,
    setCollaborationPresence,
    setDocumentEditLock,
    setDocuments,
    setLoadState,
    setManagedUsers,
    setNotifications,
    setPolicies,
    setSelectedDocument,
    setSubjects,
    setVersionDiff,
    setVersions,
  ]);

  const clearWorkspaceOnLogout = useCallback(() => {
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
    setContentMode("native");
    setContentFile(null);
    setContentPdfUrl("");
    setContentDocxUrl("");
    setContentStatus("idle");
    setContentError("");
    setMessage("");
    setError("");
  }, [
    setApprovals,
    setAttachments,
    setAuditEvents,
    setCollaborationPresence,
    setContentDocxUrl,
    setContentError,
    setContentFile,
    setContentMode,
    setContentPdfUrl,
    setContentStatus,
    setDocumentEditLock,
    setDocuments,
    setError,
    setManagedUsers,
    setMessage,
    setNotifications,
    setPolicies,
    setSelectedDocument,
    setSelectedProfileGovernance,
    setSelectedProfileSchema,
    setSelectedProfileSchemas,
    setSubjects,
    setVersionDiff,
    setVersions,
  ]);

  const bootstrap = useCallback(async () => {
    try {
      const currentUser = await api.me();
      setUser(currentUser);
      setDocumentForm((current) => ({ ...current, ownerId: currentUser.userId }));
      setAuthState("ready");
      if (!currentUser.mustChangePassword) {
        await onAuthenticated(currentUser);
      }
    } catch (err) {
      if (statusOf(err) === 401) {
        setAuthState("idle");
        return;
      }
      setAuthState("error");
      setError(asMessage(err));
    }
  }, [onAuthenticated, setAuthState, setDocumentForm, setError, setUser]);

  const handleLogin = useCallback(
    async (event: React.FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      setError("");
      setMessage("");
      try {
        setAuthState("loading");
        const response = await api.login(loginForm);
        setUser(response.user);
        setDocumentForm((current) => ({ ...current, ownerId: response.user.userId }));
        setAuthState("ready");
        if (!response.user.mustChangePassword) {
          await onAuthenticated(response.user);
        } else {
          clearWorkspaceAfterPasswordChange();
        }
      } catch (err) {
        setUser(null);
        setManagedUsers([]);
        setDocuments([]);
        setSelectedDocument(null);
        if (statusOf(err) === 401) {
          setError("Usuario ou senha invalidos.");
          setAuthState("idle");
          return;
        }
        setError(asMessage(err));
        setAuthState("error");
      }
    },
    [
      clearWorkspaceAfterPasswordChange,
      loginForm,
      onAuthenticated,
      setAuthState,
      setDocumentForm,
      setDocuments,
      setError,
      setManagedUsers,
      setMessage,
      setSelectedDocument,
      setUser,
    ],
  );

  const handleLogout = useCallback(async () => {
    setError("");
    setMessage("");
    try {
      await api.logout();
    } catch {
      // Best-effort logout; still clear local state.
    } finally {
      clearWorkspaceOnLogout();
      setUser(null);
      setAuthState("idle");
    }
  }, [clearWorkspaceOnLogout, setAuthState, setError, setMessage, setUser]);

  const handleChangePassword = useCallback(
    async (event: React.FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      setError("");
      setMessage("");
      try {
        if (passwordForm.newPassword !== passwordForm.confirmPassword) {
          setError("A confirmacao da nova senha nao confere.");
          return;
        }
        const response = await api.changePassword(passwordForm);
        setPasswordForm({ currentPassword: "", newPassword: "", confirmPassword: "" });
        setUser(response.user);
        setLoginForm((current) => ({ ...current, identifier: response.user.username, password: "" }));
        setDocumentForm((current) => ({ ...current, ownerId: response.user.userId }));
        await onAuthenticated(response.user);
        setAuthState("ready");
        setMessage("Senha alterada com sucesso.");
      } catch (err) {
        setError(asMessage(err));
      }
    },
    [onAuthenticated, passwordForm, setAuthState, setDocumentForm, setError, setLoginForm, setMessage, setPasswordForm, setUser],
  );

  return {
    authState,
    user,
    loginForm,
    passwordForm,
    setLoginForm,
    setPasswordForm,
    bootstrap,
    handleLogin,
    handleLogout,
    handleChangePassword,
  };
}
