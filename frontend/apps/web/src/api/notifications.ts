import type { AuditEventItem, NotificationItem } from "../lib.types";
import { API_BASE_URL, request } from "./client";

export type OperationsStreamSnapshot = {
  pendingNotifications: number;
  pendingApprovals: number;
  documentsInReview: number;
  totalDocuments: number;
  generatedAt: string;
};

function normalizeNotificationItem(value: NotificationItem): NotificationItem {
  return {
    id: value?.id ?? "",
    recipientUserId: value?.recipientUserId ?? "",
    eventType: value?.eventType ?? "",
    resourceType: value?.resourceType ?? "",
    resourceId: value?.resourceId ?? "",
    title: value?.title ?? "",
    message: value?.message ?? "",
    status: value?.status ?? "PENDING",
    createdAt: value?.createdAt ?? "",
    readAt: value?.readAt ?? "",
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

export async function listNotifications(params?: URLSearchParams) {
  const query = params?.toString();
  const response = await request<{ items: NotificationItem[] }>(`/notifications${query ? `?${query}` : ""}`);
  return { items: Array.isArray(response.items) ? response.items.map(normalizeNotificationItem) : [] };
}

export async function listAuditEvents(params?: URLSearchParams) {
  const query = params?.toString();
  const response = await request<{ items: AuditEventItem[] }>(`/audit/events${query ? `?${query}` : ""}`);
  return { items: Array.isArray(response.items) ? response.items.map(normalizeAuditEventItem) : [] };
}

export function markNotificationRead(notificationId: string) {
  return request<{ id: string; status: string; readAt: string }>(
    `/notifications/${encodeURIComponent(notificationId)}/read`,
    { method: "POST" },
  );
}

export function subscribeOperationsStream(
  onSnapshot: (snapshot: OperationsStreamSnapshot) => void,
  onError?: (error: Event) => void,
) {
  const stream = new EventSource(`${API_BASE_URL}/operations/stream`, { withCredentials: true });
  const listener = (event: MessageEvent<string>) => {
    try {
      const payload = JSON.parse(event.data) as OperationsStreamSnapshot;
      onSnapshot({
        pendingNotifications: Number(payload?.pendingNotifications ?? 0),
        pendingApprovals: Number(payload?.pendingApprovals ?? 0),
        documentsInReview: Number(payload?.documentsInReview ?? 0),
        totalDocuments: Number(payload?.totalDocuments ?? 0),
        generatedAt: payload?.generatedAt ?? "",
      });
    } catch {
      return;
    }
  };

  stream.addEventListener("snapshot", listener as EventListener);
  stream.onerror = (error) => {
    if (onError) {
      onError(error);
    }
  };

  return () => {
    stream.removeEventListener("snapshot", listener as EventListener);
    stream.close();
  };
}
