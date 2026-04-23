import type {
  CreateAreaRequest,
  CreateProfileRequest,
  DocumentProfile,
  ProcessArea,
  SetDefaultTemplateRequest,
  UpdateAreaRequest,
  UpdateProfileRequest,
} from "./types";

const BASE = "/api/v2/taxonomy";

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
    throw new Error((data as { message?: string; code?: string }).message ?? (data as { code?: string }).code ?? "Request failed");
  }
  return data as T;
}

export async function fetchProfiles(includeArchived?: boolean): Promise<DocumentProfile[]> {
  const qs = includeArchived ? "?includeArchived=true" : "";
  const res = await request<{ items: DocumentProfile[] }>(`/profiles${qs}`);
  return res.items;
}

export async function fetchProfile(code: string): Promise<DocumentProfile> {
  return request<DocumentProfile>(`/profiles/${encodeURIComponent(code)}`);
}

export async function createProfile(req: CreateProfileRequest): Promise<DocumentProfile> {
  return request<DocumentProfile>("/profiles", {
    method: "POST",
    body: JSON.stringify(req),
  });
}

export async function updateProfile(code: string, req: UpdateProfileRequest): Promise<DocumentProfile> {
  return request<DocumentProfile>(`/profiles/${encodeURIComponent(code)}`, {
    method: "PATCH",
    body: JSON.stringify(req),
  });
}

export async function setDefaultTemplate(code: string, req: SetDefaultTemplateRequest): Promise<void> {
  return request<void>(`/profiles/${encodeURIComponent(code)}/default-template`, {
    method: "PUT",
    body: JSON.stringify(req),
  });
}

export async function archiveProfile(code: string): Promise<void> {
  return request<void>(`/profiles/${encodeURIComponent(code)}/archive`, {
    method: "POST",
  });
}

export async function fetchAreas(includeArchived?: boolean): Promise<ProcessArea[]> {
  const qs = includeArchived ? "?includeArchived=true" : "";
  const res = await request<{ items: ProcessArea[] }>(`/areas${qs}`);
  return res.items;
}

export async function fetchArea(code: string): Promise<ProcessArea> {
  return request<ProcessArea>(`/areas/${encodeURIComponent(code)}`);
}

export async function createArea(req: CreateAreaRequest): Promise<ProcessArea> {
  return request<ProcessArea>("/areas", {
    method: "POST",
    body: JSON.stringify(req),
  });
}

export async function updateArea(code: string, req: UpdateAreaRequest): Promise<ProcessArea> {
  return request<ProcessArea>(`/areas/${encodeURIComponent(code)}`, {
    method: "PATCH",
    body: JSON.stringify(req),
  });
}

export async function archiveArea(code: string): Promise<void> {
  return request<void>(`/areas/${encodeURIComponent(code)}/archive`, {
    method: "POST",
  });
}
