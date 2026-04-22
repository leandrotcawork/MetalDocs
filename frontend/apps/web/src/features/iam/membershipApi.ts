export interface AreaMembership {
  userId: string;
  tenantId: string;
  areaCode: string;
  role: string;
  effectiveFrom: string;
  effectiveTo: string | null;
  grantedBy: string | null;
}

export interface GrantMembershipRequest {
  userId: string;
  areaCode: string;
  role: 'viewer' | 'editor' | 'reviewer' | 'approver';
}

const BASE = "/api/v2/iam/area-memberships";

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

export async function fetchMemberships(userId: string): Promise<AreaMembership[]> {
  return request<AreaMembership[]>(`?userId=${encodeURIComponent(userId)}`);
}

export async function grantMembership(req: GrantMembershipRequest): Promise<void> {
  return request<void>("", {
    method: "POST",
    body: JSON.stringify(req),
  });
}

export async function revokeMembership(userId: string, areaCode: string): Promise<void> {
  return request<void>(`?userId=${encodeURIComponent(userId)}&areaCode=${encodeURIComponent(areaCode)}`, {
    method: "DELETE",
  });
}
