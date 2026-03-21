import type { WorkflowApprovalItem } from "../lib.types";
import { request } from "./client";

function normalizeApprovalItem(value: WorkflowApprovalItem): WorkflowApprovalItem {
  return {
    approvalId: value?.approvalId ?? "",
    documentId: value?.documentId ?? "",
    requestedBy: value?.requestedBy ?? "",
    assignedReviewer: value?.assignedReviewer ?? "",
    decisionBy: value?.decisionBy ?? "",
    status: value?.status ?? "PENDING",
    requestReason: value?.requestReason ?? "",
    decisionReason: value?.decisionReason ?? "",
    requestedAt: value?.requestedAt ?? "",
    decidedAt: value?.decidedAt ?? "",
  };
}

export async function listApprovals(documentId: string) {
  const response = await request<{ items: WorkflowApprovalItem[] }>(`/workflow/documents/${documentId}/approvals`);
  return { items: Array.isArray(response.items) ? response.items.map(normalizeApprovalItem) : [] };
}

export function transitionWorkflow(documentId: string, body: Record<string, unknown>) {
  return request<Record<string, unknown>>(`/workflow/documents/${documentId}/transitions`, {
    method: "POST",
    body: JSON.stringify(body),
  });
}
