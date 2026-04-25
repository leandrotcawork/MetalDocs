import type { ControlledDocument, CreateControlledDocumentRequest } from "./types";

const BASE = "/api/v2/controlled-documents";

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE}${path}`, {
    headers: { "Content-Type": "application/json" },
    ...options,
  });
  if (res.status === 204) {
    return undefined as T;
  }
  const data = await res.json() as { message?: string; code?: string };
  if (!res.ok) {
    throw new Error(data.message ?? data.code ?? "Request failed");
  }
  return data as T;
}

export async function fetchControlledDocuments(filter?: {
  profileCode?: string;
  processAreaCode?: string;
  status?: string;
  limit?: number;
  offset?: number;
}): Promise<ControlledDocument[]> {
  const params = new URLSearchParams();
  if (filter?.profileCode) params.set("profileCode", filter.profileCode);
  if (filter?.processAreaCode) params.set("processAreaCode", filter.processAreaCode);
  if (filter?.status) params.set("status", filter.status);
  if (filter?.limit != null) params.set("limit", String(filter.limit));
  if (filter?.offset != null) params.set("offset", String(filter.offset));
  const qs = params.toString() ? `?${params.toString()}` : "";
  const res = await request<{ items: ControlledDocument[] }>(`${qs}`);
  return res.items;
}

export async function fetchControlledDocument(id: string): Promise<ControlledDocument> {
  return request<ControlledDocument>(`/${encodeURIComponent(id)}`);
}

export async function createControlledDocument(req: CreateControlledDocumentRequest): Promise<ControlledDocument> {
  return request<ControlledDocument>("", {
    method: "POST",
    body: JSON.stringify(req),
  });
}

export async function obsoleteControlledDocument(id: string): Promise<void> {
  return request<void>(`/${encodeURIComponent(id)}/obsolete`, {
    method: "PUT",
  });
}

export async function supersedeControlledDocument(id: string): Promise<void> {
  return request<void>(`/${encodeURIComponent(id)}/supersede`, {
    method: "PUT",
  });
}
