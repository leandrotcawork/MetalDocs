import { create } from "zustand";
import type { WorkspaceView } from "../components/DocumentWorkspaceShell";
import type { ManagedUserItem, UserRole } from "../lib.types";

type UserFormState = {
  userId: string;
  username: string;
  email: string;
  displayName: string;
  password: string;
  roles: UserRole[];
};

type ManagedUserFormState = {
  userId: string;
  displayName: string;
  email: string;
  isActive: boolean;
  mustChangePassword: boolean;
  roles: UserRole[];
  resetPassword: string;
};

interface UiStore {
  message: string;
  error: string;
  isCreateSubmitting: boolean;
  activeView: WorkspaceView;
  pendingViewNavigation: WorkspaceView | null;
  searchQuery: string;
  userForm: UserFormState;
  managedUserForm: ManagedUserFormState;
  managedUsers: ManagedUserItem[];
  setMessage: (message: string) => void;
  setError: (error: string) => void;
  setIsCreateSubmitting: (isCreateSubmitting: boolean) => void;
  setActiveView: (activeView: WorkspaceView) => void;
  requestViewNavigation: (activeView: WorkspaceView) => void;
  clearPendingViewNavigation: () => void;
  setSearchQuery: (searchQuery: string) => void;
  setUserForm: (userForm: UserFormState | ((current: UserFormState) => UserFormState)) => void;
  setManagedUserForm: (
    managedUserForm: ManagedUserFormState | ((current: ManagedUserFormState) => ManagedUserFormState),
  ) => void;
  setManagedUsers: (managedUsers: ManagedUserItem[]) => void;
}

export const useUiStore = create<UiStore>((set) => ({
  message: "",
  error: "",
  isCreateSubmitting: false,
  activeView: "operations",
  pendingViewNavigation: null,
  searchQuery: "",
  userForm: {
    userId: "",
    username: "",
    email: "",
    displayName: "",
    password: "",
    roles: ["viewer"],
  },
  managedUserForm: {
    userId: "",
    displayName: "",
    email: "",
    isActive: true,
    mustChangePassword: false,
    roles: ["viewer"],
    resetPassword: "",
  },
  managedUsers: [],
  setMessage: (message) => set({ message }),
  setError: (error) => set({ error }),
  setIsCreateSubmitting: (isCreateSubmitting) => set({ isCreateSubmitting }),
  setActiveView: (activeView) => set({ activeView }),
  requestViewNavigation: (activeView) => set({ activeView, pendingViewNavigation: activeView }),
  clearPendingViewNavigation: () => set({ pendingViewNavigation: null }),
  setSearchQuery: (searchQuery) => set({ searchQuery }),
  setUserForm: (userForm) =>
    set((state) => ({
      userForm: typeof userForm === "function" ? userForm(state.userForm) : userForm,
    })),
  setManagedUserForm: (managedUserForm) =>
    set((state) => ({
      managedUserForm: typeof managedUserForm === "function" ? managedUserForm(state.managedUserForm) : managedUserForm,
    })),
  setManagedUsers: (managedUsers) => set({ managedUsers }),
}));

export type { ManagedUserFormState, UserFormState };
