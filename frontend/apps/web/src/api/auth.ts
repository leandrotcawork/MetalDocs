import type { CurrentUser, UserRole } from "../lib.types";
import { request } from "./client";

const allowedRoles = new Set<UserRole>(["admin", "editor", "reviewer", "viewer"]);

function normalizeRoles(value: unknown): UserRole[] {
  if (!Array.isArray(value)) {
    return [];
  }
  return value.filter((item): item is UserRole => typeof item === "string" && allowedRoles.has(item as UserRole));
}

function normalizeCurrentUser(value: CurrentUser): CurrentUser {
  return {
    userId: value?.userId ?? "",
    username: value?.username ?? "",
    email: value?.email ?? "",
    displayName: value?.displayName ?? value?.username ?? "",
    mustChangePassword: Boolean(value?.mustChangePassword),
    roles: normalizeRoles(value?.roles),
  };
}

export async function login(body: { identifier: string; password: string }) {
  const response = await request<{ user: CurrentUser; expiresAt: string }>("/auth/login", {
    method: "POST",
    body: JSON.stringify(body),
  });
  return { ...response, user: normalizeCurrentUser(response.user) };
}

export function logout() {
  return request<void>("/auth/logout", { method: "POST" });
}

export async function me() {
  return normalizeCurrentUser(await request<CurrentUser>("/auth/me"));
}

export async function changePassword(body: { currentPassword: string; newPassword: string }) {
  const response = await request<{ changed: boolean; user: CurrentUser }>("/auth/change-password", {
    method: "POST",
    body: JSON.stringify(body),
  });
  return { ...response, user: normalizeCurrentUser(response.user) };
}
