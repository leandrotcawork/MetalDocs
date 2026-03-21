import type { ApiErrorEnvelope } from "../lib.types";

export const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? "/api/v1";

type RequestTrace = {
  id: number;
  method: string;
  path: string;
  startedAt: number;
};

let traceId = 0;
let activeTrace: { name: string; startedAt: number; items: RequestTrace[] } | null = null;
const uxMarks = new Map<string, number>();

function isTraceEnabled() {
  if (typeof window === "undefined") return false;
  if (window.location.search.includes("trace=1")) return true;
  return localStorage.getItem("md_trace") === "1";
}

function traceStart(name: string) {
  if (!isTraceEnabled()) return;
  activeTrace = { name, startedAt: performance.now(), items: [] };
}

function traceStop() {
  if (!activeTrace) return;
  const total = performance.now() - activeTrace.startedAt;
  console.groupCollapsed(`[md-trace] ${activeTrace.name} (${total.toFixed(0)}ms)`);
  activeTrace.items
    .sort((a, b) => a.startedAt - b.startedAt)
    .forEach((item) => {
      console.log(`${item.method} ${item.path}`, `+${(item.startedAt - activeTrace!.startedAt).toFixed(0)}ms`);
    });
  console.groupEnd();
  activeTrace = null;
}

function traceRequestStart(method: string, path: string) {
  if (!activeTrace) return;
  const item: RequestTrace = {
    id: traceId++,
    method,
    path,
    startedAt: performance.now(),
  };
  activeTrace.items.push(item);
  return item.id;
}

function traceRequestEnd(id?: number) {
  if (!activeTrace || id === undefined) return;
  const item = activeTrace.items.find((it) => it.id === id);
  if (item) {
    item.startedAt = item.startedAt;
  }
}

export function startApiTrace(name: string) {
  traceStart(name);
}

export function stopApiTrace() {
  traceStop();
}

export function markUx(label: string) {
  if (!isTraceEnabled()) return;
  const now = performance.now();
  uxMarks.set(label, now);
  console.log(`[md-ux] ${label} @ ${now.toFixed(0)}ms`);
}

export function reportUxSequence(title: string, labels: string[]) {
  if (!isTraceEnabled()) return;
  const base = uxMarks.get(labels[0] ?? "");
  if (base === undefined) return;
  console.groupCollapsed(`[md-ux] ${title}`);
  labels.forEach((label) => {
    const stamp = uxMarks.get(label);
    if (stamp === undefined) return;
    console.log(`${label}`, `+${(stamp - base).toFixed(0)}ms`);
  });
  console.groupEnd();
}

export async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const method = (init?.method ?? "GET").toUpperCase();
  const traceItemId = traceRequestStart(method, path);
  const response = await fetch(`${API_BASE_URL}${path}`, {
    credentials: "include",
    ...init,
    headers: {
      ...(init?.body instanceof FormData ? {} : { "Content-Type": "application/json" }),
      ...(init?.headers ?? {}),
    },
  });
  traceRequestEnd(traceItemId);

  if (!response.ok) {
    const errorPayload = (await response.json().catch(() => null)) as ApiErrorEnvelope | null;
    const error = new Error(errorPayload?.error.message ?? `HTTP ${response.status}`);
    (error as Error & { status?: number }).status = response.status;
    throw error;
  }

  if (response.status === 204) {
    return undefined as T;
  }

  return (await response.json()) as T;
}

export async function requestBlob(path: string, init?: RequestInit): Promise<Blob> {
  const method = (init?.method ?? "GET").toUpperCase();
  const traceItemId = traceRequestStart(method, path);
  const response = await fetch(`${API_BASE_URL}${path}`, {
    credentials: "include",
    ...init,
    headers: {
      ...(init?.headers ?? {}),
    },
  });
  traceRequestEnd(traceItemId);

  if (!response.ok) {
    const errorPayload = (await response.json().catch(() => null)) as ApiErrorEnvelope | null;
    const error = new Error(errorPayload?.error.message ?? `HTTP ${response.status}`);
    (error as Error & { status?: number }).status = response.status;
    throw error;
  }

  return response.blob();
}
