import type { AdminOverviewResponse, AuditEventItem, ManagedUserItem, OnlineUserItem, UserRole } from "../lib.types";
import { request } from "./client";

const allowedRoles = new Set<UserRole>(["admin", "editor", "reviewer", "viewer"]);

function normalizeRoles(value: unknown): UserRole[] {
  if (!Array.isArray(value)) {
    return [];
  }
  return value.filter((item): item is UserRole => typeof item === "string" && allowedRoles.has(item as UserRole));
}

function normalizeManagedUser(value: ManagedUserItem): ManagedUserItem {
  return {
    userId: value?.userId ?? "",
    username: value?.username ?? "",
    email: value?.email ?? "",
    displayName: value?.displayName ?? value?.username ?? "",
    isActive: Boolean(value?.isActive),
    mustChangePassword: Boolean(value?.mustChangePassword),
    failedLoginAttempts: Number(value?.failedLoginAttempts ?? 0),
    lockedUntil: value?.lockedUntil ?? "",
    lastLoginAt: value?.lastLoginAt ?? "",
    createdAt: value?.createdAt ?? "",
    updatedAt: value?.updatedAt ?? "",
    roles: normalizeRoles(value?.roles),
  };
}

function normalizeOnlineUser(value: OnlineUserItem): OnlineUserItem {
  return {
    userId: value?.userId ?? "",
    username: value?.username ?? "",
    displayName: value?.displayName ?? value?.username ?? "",
    lastSeenAt: value?.lastSeenAt ?? "",
  };
}

function normalizeAuditEventItem(value: AuditEventItem): AuditEventItem {
  return {
    id: value?.id ?? "",
    occurredAt: value?.occurredAt ?? "",
    actorId: value?.actorId ?? "",
    action: value?.action ?? "",
    resourceType: value?.resourceType ?? "",
    resourceId: value?.resourceId ?? "",
    payload: typeof value?.payload === "object" && value?.payload !== null ? value.payload : {},
    traceId: value?.traceId ?? "",
  };
}

export async function listUsers() {
  const response = await request<{ items: ManagedUserItem[] }>("/iam/users");
  return { items: Array.isArray(response.items) ? response.items.map(normalizeManagedUser) : [] };
}

export async function getAdminOverview() {
  const response = await request<AdminOverviewResponse>("/iam/admin/overview");
  return {
    users: Array.isArray(response.users) ? response.users.map(normalizeManagedUser) : [],
    onlineUsers: Array.isArray(response.onlineUsers) ? response.onlineUsers.map(normalizeOnlineUser) : [],
    recentActivities: Array.isArray(response.recentActivities) ? response.recentActivities.map(normalizeAuditEventItem) : [],
  };
}

export function createUser(body: Record<string, unknown>) {
  return request<{ userId: string }>("/iam/users", { method: "POST", body: JSON.stringify(body) });
}

export function updateUser(userId: string, body: Record<string, unknown>) {
  return request<{ userId: string; updated: boolean }>(`/iam/users/${userId}`, {
    method: "PATCH",
    body: JSON.stringify(body),
  });
}

export function assignRole(userId: string, body: Record<string, unknown>) {
  return request<{ userId: string; role: string; displayName: string }>(`/iam/users/${userId}/roles`, {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export function replaceUserRoles(userId: string, body: Record<string, unknown>) {
  return request<{ userId: string; displayName: string; roles: string[] }>(`/iam/users/${userId}/roles`, {
    method: "PUT",
    body: JSON.stringify(body),
  });
}

export function adminResetPassword(userId: string, body: Record<string, unknown>) {
  return request<{ userId: string; reset: boolean; mustChangePassword: boolean }>(
    `/iam/users/${userId}/reset-password`,
    { method: "POST", body: JSON.stringify(body) },
  );
}

export function unlockUser(userId: string) {
  return request<{ userId: string; unlocked: boolean }>(`/iam/users/${userId}/unlock`, { method: "POST" });
}
