import { useMemo } from "react";
import { useDocumentsStore } from "../../store/documents.store";

export function useDocumentDetail() {
  const {
    selectedDocument,
    versions,
    approvals,
    attachments,
    policies,
    auditEvents,
    collaborationPresence,
    documentEditLock,
  } = useDocumentsStore();

  const hasSelection = Boolean(selectedDocument?.documentId);
  const latestVersion = useMemo(() => versions[0] ?? null, [versions]);

  return {
    hasSelection,
    selectedDocument,
    latestVersion,
    versions,
    approvals,
    attachments,
    policies,
    auditEvents,
    collaborationPresence,
    documentEditLock,
  };
}

