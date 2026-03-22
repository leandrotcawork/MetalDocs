import { useCallback } from "react";
import { api } from "../../lib.api";
import type { ManagedUserItem, UserRole } from "../../lib.types";
import { useUiStore } from "../../store/ui.store";
import { asMessage } from "../shared/errors";

export function useManagedUsers(onRefresh: () => Promise<void>) {
  const { userForm, managedUserForm, managedUsers, setUserForm, setManagedUserForm, setManagedUsers, setError, setMessage } = useUiStore();

  const selectManagedUser = useCallback((item: ManagedUserItem) => {
    setManagedUserForm({
      userId: item.userId,
      displayName: item.displayName,
      email: item.email ?? "",
      isActive: item.isActive,
      mustChangePassword: item.mustChangePassword,
      roles: Array.isArray(item.roles) && item.roles.length > 0 ? item.roles : ["viewer"],
      resetPassword: "",
    });
  }, [setManagedUserForm]);

  const toggleManagedUserRole = useCallback((role: UserRole) => {
    setManagedUserForm((current) => {
      const hasRole = current.roles.includes(role);
      const nextRoles = hasRole ? current.roles.filter((item) => item !== role) : [...current.roles, role];
      return {
        ...current,
        roles: nextRoles.length > 0 ? nextRoles : current.roles,
      };
    });
  }, [setManagedUserForm]);

  const handleCreateUser = useCallback(async () => {
    if (!userForm.username.trim() || !userForm.displayName.trim() || !userForm.password.trim()) {
      setError("Preencha username, display name e senha inicial.");
      return;
    }
    setError("");
    setMessage("");
    try {
      await api.createUser(userForm);
      setUserForm({
        userId: "",
        username: "",
        email: "",
        displayName: "",
        password: "",
        roles: ["viewer"],
      });
      await onRefresh();
      setMessage("Usuario criado. A senha inicial exigira troca no primeiro acesso.");
    } catch (err) {
      setError(asMessage(err));
    }
  }, [onRefresh, setError, setMessage, setUserForm, userForm]);

  const handleSaveManagedUser = useCallback(async () => {
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
      await onRefresh();
      setMessage("Usuario administrativo atualizado com sucesso.");
    } catch (err) {
      setError(asMessage(err));
    }
  }, [managedUserForm, onRefresh, setError, setMessage]);

  const handleAdminResetPassword = useCallback(async () => {
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
      await onRefresh();
      setMessage("Senha administrativa resetada. O usuario precisara trocar no proximo login.");
    } catch (err) {
      setError(asMessage(err));
    }
  }, [managedUserForm, onRefresh, setError, setManagedUserForm, setMessage]);

  const handleUnlockManagedUser = useCallback(async () => {
    if (!managedUserForm.userId) {
      setError("Selecione um usuario para desbloquear.");
      return;
    }
    try {
      setError("");
      setMessage("");
      await api.unlockUser(managedUserForm.userId);
      await onRefresh();
      setMessage("Usuario desbloqueado com sucesso.");
    } catch (err) {
      setError(asMessage(err));
    }
  }, [managedUserForm.userId, onRefresh, setError, setMessage]);

  return {
    userForm,
    managedUserForm,
    managedUsers,
    setUserForm,
    setManagedUserForm,
    setManagedUsers,
    selectManagedUser,
    toggleManagedUserRole,
    handleCreateUser,
    handleSaveManagedUser,
    handleAdminResetPassword,
    handleUnlockManagedUser,
  };
}
