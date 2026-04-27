import type {
  DocumentDepartmentItem,
  DocumentFamilyItem,
  DocumentProfileBundleResponse,
  DocumentProfileGovernanceItem,
  DocumentProfileItem,
  DocumentProfileSchemaItem,
  DocumentTypeItem,
  ProcessAreaItem,
  SubjectItem,
} from "../lib.types";
import { request, requestBlob } from "./client";

function normalizeDocumentProfile(value: DocumentProfileItem): DocumentProfileItem {
  const fallbackName = value?.name ?? value?.code ?? "";
  return {
    code: value?.code ?? "",
    familyCode: value?.familyCode ?? "",
    name: fallbackName,
    alias: value?.alias?.trim?.() || fallbackName,
    description: value?.description ?? "",
    reviewIntervalDays: Number(value?.reviewIntervalDays ?? 0),
    activeSchemaVersion: Number(value?.activeSchemaVersion ?? 0),
    workflowProfile: value?.workflowProfile ?? "",
    approvalRequired: Boolean(value?.approvalRequired),
    retentionDays: Number(value?.retentionDays ?? 0),
    validityDays: Number(value?.validityDays ?? 0),
  };
}

function normalizeProcessArea(value: ProcessAreaItem): ProcessAreaItem {
  return {
    code: value?.code ?? "",
    name: value?.name ?? value?.code ?? "",
    description: value?.description ?? "",
  };
}

function normalizeDocumentDepartment(value: DocumentDepartmentItem): DocumentDepartmentItem {
  return {
    code: value?.code ?? "",
    name: value?.name ?? value?.code ?? "",
    description: value?.description ?? "",
  };
}

function normalizeSubject(value: SubjectItem): SubjectItem {
  return {
    code: value?.code ?? "",
    processAreaCode: value?.processAreaCode ?? "",
    name: value?.name ?? value?.code ?? "",
    description: value?.description ?? "",
  };
}

function normalizeMetadataRule(value: DocumentProfileSchemaItem["metadataRules"][number]) {
  return {
    name: value?.name ?? "",
    type: value?.type ?? "text",
    required: Boolean(value?.required),
  };
}

function normalizeDocumentProfileSchema(value: DocumentProfileSchemaItem): DocumentProfileSchemaItem {
  return {
    profileCode: value?.profileCode ?? "",
    version: Number(value?.version ?? 0),
    isActive: Boolean(value?.isActive),
    metadataRules: Array.isArray(value?.metadataRules) ? value.metadataRules.map(normalizeMetadataRule) : [],
    contentSchema: value?.contentSchema && typeof value.contentSchema === "object" ? value.contentSchema : {},
  };
}

function normalizeDocumentProfileGovernance(value: DocumentProfileGovernanceItem): DocumentProfileGovernanceItem {
  return {
    profileCode: value?.profileCode ?? "",
    workflowProfile: value?.workflowProfile ?? "",
    reviewIntervalDays: Number(value?.reviewIntervalDays ?? 0),
    approvalRequired: Boolean(value?.approvalRequired),
    retentionDays: Number(value?.retentionDays ?? 0),
    validityDays: Number(value?.validityDays ?? 0),
  };
}

function normalizeDocumentProfileBundle(value: DocumentProfileBundleResponse): DocumentProfileBundleResponse {
  return {
    profile: normalizeDocumentProfile(value?.profile),
    schema: normalizeDocumentProfileSchema(value?.schema),
    governance: normalizeDocumentProfileGovernance(value?.governance),
    taxonomy: {
      processAreas: Array.isArray(value?.taxonomy?.processAreas)
        ? value.taxonomy.processAreas.map(normalizeProcessArea)
        : [],
      documentDepartments: Array.isArray(value?.taxonomy?.documentDepartments)
        ? value.taxonomy.documentDepartments.map(normalizeDocumentDepartment)
        : [],
      subjects: Array.isArray(value?.taxonomy?.subjects) ? value.taxonomy.subjects.map(normalizeSubject) : [],
    },
  };
}

export function listDocumentTypes() {
  return request<{ items: DocumentTypeItem[] }>("/document-types");
}

export function listDocumentFamilies() {
  return request<{ items: DocumentFamilyItem[] }>("/document-families");
}

export async function listDocumentProfiles() {
  const response = await request<{ items: DocumentProfileItem[] }>("/document-profiles");
  return { items: Array.isArray(response.items) ? response.items.map(normalizeDocumentProfile) : [] };
}

export function createDocumentProfile(body: Record<string, unknown>) {
  return request<{ code: string }>("/document-profiles", { method: "POST", body: JSON.stringify(body) });
}

export function updateDocumentProfile(code: string, body: Record<string, unknown>) {
  return request<{ code: string }>(`/document-profiles/${encodeURIComponent(code)}`, {
    method: "PUT",
    body: JSON.stringify(body),
  });
}

export function deleteDocumentProfile(code: string) {
  return request<void>(`/document-profiles/${encodeURIComponent(code)}`, { method: "DELETE" });
}

export async function getDocumentProfileBundle(profileCode: string) {
  return normalizeDocumentProfileBundle(
    await request<DocumentProfileBundleResponse>(`/document-profiles/${encodeURIComponent(profileCode)}/bundle`),
  );
}

export async function getDocumentProfileSchema(profileCode: string) {
  const response = await request<{ items: DocumentProfileSchemaItem[] }>(
    `/document-profiles/${encodeURIComponent(profileCode)}/schema`,
  );
  const items = Array.isArray(response.items) ? response.items.map(normalizeDocumentProfileSchema) : [];
  return items.find((item) => item.isActive) ?? items[0] ?? null;
}

export async function listDocumentProfileSchemas(profileCode: string) {
  const response = await request<{ items: DocumentProfileSchemaItem[] }>(
    `/document-profiles/${encodeURIComponent(profileCode)}/schema`,
  );
  return { items: Array.isArray(response.items) ? response.items.map(normalizeDocumentProfileSchema) : [] };
}

export function upsertDocumentProfileSchema(profileCode: string, body: Record<string, unknown>) {
  return request<{ code: string }>(`/document-profiles/${encodeURIComponent(profileCode)}/schema`, {
    method: "POST",
    body: JSON.stringify(body),
  });
}

export function activateDocumentProfileSchema(profileCode: string, version: number) {
  return request<{ code: string }>(
    `/document-profiles/${encodeURIComponent(profileCode)}/schema/${encodeURIComponent(String(version))}/activate`,
    { method: "PUT" },
  );
}

export async function getDocumentProfileGovernance(profileCode: string) {
  return normalizeDocumentProfileGovernance(
    await request<DocumentProfileGovernanceItem>(`/document-profiles/${encodeURIComponent(profileCode)}/governance`),
  );
}

export function updateDocumentProfileGovernance(profileCode: string, body: Record<string, unknown>) {
  return request<{ code: string }>(`/document-profiles/${encodeURIComponent(profileCode)}/governance`, {
    method: "PUT",
    body: JSON.stringify(body),
  });
}

export async function listProcessAreas() {
  const response = await request<{ items: ProcessAreaItem[] }>("/process-areas");
  return { items: Array.isArray(response.items) ? response.items.map(normalizeProcessArea) : [] };
}

export async function listDocumentDepartments() {
  const response = await request<{ items: DocumentDepartmentItem[] }>("/document-departments");
  return { items: Array.isArray(response.items) ? response.items.map(normalizeDocumentDepartment) : [] };
}

export function createProcessArea(body: Record<string, unknown>) {
  return request<{ code: string }>("/process-areas", { method: "POST", body: JSON.stringify(body) });
}

export function updateProcessArea(code: string, body: Record<string, unknown>) {
  return request<{ code: string }>(`/process-areas/${encodeURIComponent(code)}`, {
    method: "PUT",
    body: JSON.stringify(body),
  });
}

export function deleteProcessArea(code: string) {
  return request<void>(`/process-areas/${encodeURIComponent(code)}`, { method: "DELETE" });
}

export async function listSubjects(params?: URLSearchParams) {
  const query = params?.toString();
  const response = await request<{ items: SubjectItem[] }>(`/document-subjects${query ? `?${query}` : ""}`);
  return { items: Array.isArray(response.items) ? response.items.map(normalizeSubject) : [] };
}

export function createSubject(body: Record<string, unknown>) {
  return request<{ code: string }>("/document-subjects", { method: "POST", body: JSON.stringify(body) });
}

export function updateSubject(code: string, body: Record<string, unknown>) {
  return request<{ code: string }>(`/document-subjects/${encodeURIComponent(code)}`, {
    method: "PUT",
    body: JSON.stringify(body),
  });
}

export function deleteSubject(code: string) {
  return request<void>(`/document-subjects/${encodeURIComponent(code)}`, { method: "DELETE" });
}

export function downloadProfileTemplateDocx(profileCode: string) {
  return requestBlob(`/document-profiles/${encodeURIComponent(profileCode)}/template/docx`);
}

type TaxonomyProfileItem = Pick<DocumentProfileItem, "code" | "name" | "description" | "familyCode"> & {
  archived?: boolean;
};

type TaxonomyAreaItem = Pick<ProcessAreaItem, "code" | "name" | "description"> & {
  archived?: boolean;
};

async function fetchV2<T>(path: string): Promise<T> {
  const res = await fetch(`/api/v2${path}`, { credentials: "include", headers: { "Content-Type": "application/json" } });
  if (!res.ok) throw new Error(`${path} failed: ${res.status}`);
  return res.json() as Promise<T>;
}

export async function listTaxonomyProfiles(): Promise<{ items: DocumentProfileItem[] }> {
  const response = await fetchV2<{ items: TaxonomyProfileItem[] }>("/taxonomy/profiles");
  return {
    items: Array.isArray(response.items)
      ? response.items
          .filter((item) => item.archived !== true)
          .map((item) =>
            normalizeDocumentProfile({
              code: item.code,
              familyCode: item.familyCode,
              name: item.name,
              description: item.description,
            } as DocumentProfileItem),
          )
      : [],
  };
}

export async function listTaxonomyAreas(): Promise<{ items: ProcessAreaItem[] }> {
  const response = await fetchV2<{ items: TaxonomyAreaItem[] }>("/taxonomy/areas");
  return {
    items: Array.isArray(response.items)
      ? response.items.filter((item) => item.archived !== true).map(normalizeProcessArea)
      : [],
  };
}
