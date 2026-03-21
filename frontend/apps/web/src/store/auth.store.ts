import { create } from "zustand";
import type { CurrentUser } from "../lib.types";

export type LoadState = "idle" | "loading" | "ready" | "error";

type LoginForm = { identifier: string; password: string };
type PasswordForm = { currentPassword: string; newPassword: string; confirmPassword: string };

interface AuthStore {
  authState: LoadState;
  user: CurrentUser | null;
  loginForm: LoginForm;
  passwordForm: PasswordForm;
  setAuthState: (authState: LoadState) => void;
  setUser: (user: CurrentUser | null) => void;
  setLoginForm: (form: LoginForm | ((current: LoginForm) => LoginForm)) => void;
  setPasswordForm: (form: PasswordForm | ((current: PasswordForm) => PasswordForm)) => void;
}

export const useAuthStore = create<AuthStore>((set) => ({
  authState: "loading",
  user: null,
  loginForm: { identifier: "admin", password: "" },
  passwordForm: { currentPassword: "", newPassword: "", confirmPassword: "" },
  setAuthState: (authState) => set({ authState }),
  setUser: (user) => set({ user }),
  setLoginForm: (loginForm) =>
    set((state) => ({
      loginForm: typeof loginForm === "function" ? loginForm(state.loginForm) : loginForm,
    })),
  setPasswordForm: (passwordForm) =>
    set((state) => ({
      passwordForm: typeof passwordForm === "function" ? passwordForm(state.passwordForm) : passwordForm,
    })),
}));
