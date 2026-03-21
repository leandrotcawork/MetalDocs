import { useCallback, useRef } from "react";
import { api, markUx, reportUxSequence, startApiTrace, stopApiTrace } from "../../lib.api";
import type {
  AccessPolicyItem,
  AuditEventItem,
  CollaborationPresenceItem,
  CurrentUser,
  DocumentEditLockItem,
  DocumentListItem,
  ManagedUserItem,
  SearchDocumentItem,
  VersionDiffResponse,
  VersionListItem,
  WorkflowApprovalItem,
} from "../../lib.types";
import { emptyDocumentForm, useDocumentsStore } from "../../store/documents.store";
import { useNotificationsStore } from "../../store/notifications.store";
import { useRegistryStore } from "../../store/registry.store";
import { useUiStore } from "../../store/ui.store";
import { asMessage, statusOf } from "../shared/errors";
import type { ContentMode } from "../../components/create/documentCreateTypes";

type PolicyScope = "document" | "document_type" | "area";

export function useDocumentsWorkspace(applyDocumentProfile: (profileCode: string, preferredProcessArea?: string) => Promise<void>, prefetchProfile: (profileCode: string) => Promise<void>) {
  const {
    loadState,
    documentForm,
    contentMode,
    contentFile,
    contentPdfUrl,
    contentDocxUrl,
    contentStatus,
    contentError,
    documents,
    selectedDocument,
    versions,
    versionDiff,
    approvals,
    attachments,
    collaborationPresence,
    documentEditLock,
    policies,
    auditEvents,
    selectedFile,
    policyResourceId,
    setLoadState,
    setDocumentForm,
    setContentMode,
    setContentFile,
    setContentPdfUrl,
    setContentDocxUrl,
    setContentStatus,
    setContentError,
    setDocuments,
    setSelectedDocument,
    setVersions,
    setVersionDiff,
    setApprovals,
    setAttachments,
    setCollaborationPresence,
    setDocumentEditLock,
    setPolicies,
    setAuditEvents,
    setSelectedFile,
    setPolicyResourceId,
  } = useDocumentsStore();
  const { setNotifications } = useNotificationsStore();
  const {
    documentProfiles,
    processAreas,
    documentDepartments,
    subjects,
    setDocumentProfiles,
    setProcessAreas,
    setDocumentDepartments,
    setSubjects,
  } = useRegistryStore();
  const { setMessage, setError, setActiveView, setIsCreateSubmitting, setManagedUsers } = useUiStore();

  const streamRefreshInFlightRef = useRef(false);

  const loadWorkspace = useCallback(
    async (currentUser: CurrentUser) => {
      setLoadState("loading");
      try {
        const [profilesResponse, processAreasResponse, departmentsResponse, subjectsResponse, docsResponse, usersResponse, notificationsResponse] = await Promise.all([
          api.listDocumentProfiles(),
          api.listProcessAreas(),
          api.listDocumentDepartments(),
          api.listSubjects(),
          api.searchDocuments(new URLSearchParams({ limit: "25" })),
          (Array.isArray(currentUser.roles) ? currentUser.roles : []).includes("admin")
            ? api.listUsers()
            : Promise.resolve({ items: [] as ManagedUserItem[] }),
          api.listNotifications(new URLSearchParams({ limit: "10" })),
        ]);
        const profiles = Array.isArray(profilesResponse.items) ? profilesResponse.items : [];
        const areas = Array.isArray(processAreasResponse.items) ? processAreasResponse.items : [];
        const departments = Array.isArray(departmentsResponse.items) ? departmentsResponse.items : [];
        const nextSubjects = Array.isArray(subjectsResponse.items) ? subjectsResponse.items : [];
        const docs = Array.isArray(docsResponse.items) ? docsResponse.items : [];
        const users = Array.isArray(usersResponse.items) ? usersResponse.items : [];
        setDocumentProfiles(profiles);
        setProcessAreas(areas);
        setDocumentDepartments(departments);
        setSubjects(nextSubjects);
        setDocuments(docs);
        setManagedUsers(users);
        setNotifications(Array.isArray(notificationsResponse.items) ? notificationsResponse.items : []);
        if (profiles.length > 0) {
          const nextProfileCode = profiles.find((item) => item.code === documentForm.documentProfile)?.code ?? profiles[0]?.code ?? "";
          if (nextProfileCode) {
            await applyDocumentProfile(nextProfileCode, documentForm.processArea);
            await prefetchProfile(nextProfileCode);
          }
        }
        setLoadState("ready");
      } catch (err) {
        setError(asMessage(err));
        setLoadState("error");
      }
    },
    [applyDocumentProfile, documentForm.documentProfile, documentForm.processArea, prefetchProfile, setDocuments, setError, setLoadState, setNotifications],
  );

  const refreshOperationalSignals = useCallback(async () => {
    if (streamRefreshInFlightRef.current) {
      return;
    }
    streamRefreshInFlightRef.current = true;
    try {
      const [docsResponse, notificationsResponse] = await Promise.all([
        api.searchDocuments(new URLSearchParams({ limit: "25" })),
        api.listNotifications(new URLSearchParams({ limit: "10" })),
      ]);
      setDocuments(Array.isArray(docsResponse.items) ? docsResponse.items : []);
      setNotifications(Array.isArray(notificationsResponse.items) ? notificationsResponse.items : []);
    } finally {
      streamRefreshInFlightRef.current = false;
    }
  }, [setDocuments, setNotifications]);

  const refreshWorkspace = useCallback(
    async (currentUser: CurrentUser | null) => {
      if (!currentUser) return;
      await loadWorkspace(currentUser);
    },
    [loadWorkspace],
  );

  const openDocument = useCallback(
    async (documentId: string, nextView: "library" | "content-builder" = "library") => {
      startApiTrace(`open-document:${documentId}`);
      markUx(`open-document-start:${documentId}`);
      try {
        const [docResponse, versionsResponse, approvalsResponse, attachmentsResponse, auditResponse] = await Promise.all([
          api.getDocument(documentId),
          api.listVersions(documentId),
          api.listApprovals(documentId),
          api.listAttachments(documentId),
          api.listAuditEvents(new URLSearchParams({ resourceId: documentId })),
        ]);
        const orderedVersions = [...versionsResponse.items].sort((a, b) => b.version - a.version);
        setSelectedDocument(docResponse);
        setVersions(orderedVersions);
        setApprovals(approvalsResponse.items);
        setAttachments(attachmentsResponse.items);
        setCollaborationPresence([]);
        setDocumentEditLock(null);
        setAuditEvents(auditResponse.items);
        setPolicyResourceId(documentId);
        const policyResponse = await api.listAccessPolicies("document", documentId).catch((err) => {
          if (statusOf(err) === 403) {
            return { items: [] as AccessPolicyItem[] };
          }
          throw err;
        });
        const nextDiff = orderedVersions.length >= 2
          ? await api.getVersionDiff(documentId, orderedVersions[1].version, orderedVersions[0].version)
          : null;
        setPolicies(policyResponse.items);
        setVersionDiff(nextDiff);
        setActiveView(nextView);
        markUx(`open-document-ready:${documentId}`);
        stopApiTrace();
      } catch (err) {
        setError(asMessage(err));
        stopApiTrace();
      }
    },
    [setActiveView, setApprovals, setAttachments, setAuditEvents, setCollaborationPresence, setDocumentEditLock, setError, setPolicies, setPolicyResourceId, setSelectedDocument, setVersionDiff, setVersions],
  );

  const handleUploadAttachment = useCallback(
    async (event: React.FormEvent<HTMLFormElement>) => {
      event.preventDefault();
      if (!selectedDocument || !selectedFile) return;
      try {
        await api.uploadAttachment(selectedDocument.documentId, selectedFile);
        await openDocument(selectedDocument.documentId);
        setSelectedFile(null);
        setMessage("Anexo enviado.");
      } catch (err) {
        setError(asMessage(err));
      }
    },
    [openDocument, selectedDocument, selectedFile, setError, setMessage, setSelectedFile],
  );

  const handleCreateDocument = useCallback(
    async (event: React.FormEvent<HTMLFormElement>, currentUser: CurrentUser | null) => {
      event.preventDefault();
      setError("");
      setMessage("");
      const shouldOpenEditor = contentMode === "native";
      if (shouldOpenEditor) {
        setIsCreateSubmitting(true);
      }
      try {
        startApiTrace("create-document");
        const needsAudience = ["CONFIDENTIAL", "RESTRICTED"].includes(documentForm.classification);
        const audienceMode = documentForm.audienceMode || "DEPARTMENT";
        const audienceDepartments = documentForm.classification === "CONFIDENTIAL"
          ? documentForm.audienceDepartments
          : [documentForm.audienceDepartment || documentForm.department].filter(Boolean);
        const audience = needsAudience ? {
          mode: audienceMode,
          departmentCodes: audienceDepartments,
          processAreaCodes: audienceMode === "AREAS"
            ? [documentForm.audienceProcessArea || documentForm.processArea].filter(Boolean)
            : undefined,
        } : undefined;
        const created = await api.createDocument({
          ...documentForm,
          documentType: documentForm.documentProfile,
          documentProfile: documentForm.documentProfile,
          tags: documentForm.tags.split(",").map((item) => item.trim()).filter(Boolean),
          effectiveAt: documentForm.effectiveAt ? new Date(documentForm.effectiveAt).toISOString() : undefined,
          expiryAt: documentForm.expiryAt ? new Date(documentForm.expiryAt).toISOString() : undefined,
          metadata: documentForm.metadata.trim() ? JSON.parse(documentForm.metadata) : {},
          audience,
        });
        let handledContent = false;
        setContentError("");

        if (contentMode === "docx_upload" && contentFile) {
          handledContent = true;
          setContentStatus("saving");
          const response = await api.uploadDocumentContentDocx(created.documentId, contentFile);
          setContentPdfUrl(response.pdfUrl);
          setContentDocxUrl(response.docxUrl);
          setContentStatus("ready");
        }
        setDocumentForm({
          ...emptyDocumentForm,
          ownerId: currentUser?.userId ?? "",
          documentType: documentForm.documentProfile,
          documentProfile: documentForm.documentProfile,
          processArea: documentForm.processArea,
          metadata: documentForm.metadata,
        });
        if (!handledContent) {
          setContentMode("native");
          setContentFile(null);
          setContentPdfUrl("");
          setContentDocxUrl("");
          setContentStatus("idle");
          setContentError("");
        }
        setMessage(handledContent ? "Documento criado e conteudo processado." : "Documento criado com sucesso.");
        if (contentMode === "native") {
          await openDocument(created.documentId, "content-builder");
          setIsCreateSubmitting(false);
        } else if (!handledContent) {
          setActiveView("library");
          setIsCreateSubmitting(false);
        }
        if (currentUser) await loadWorkspace(currentUser);
        stopApiTrace();
      } catch (err) {
        setContentStatus("error");
        setContentError("Falha ao gerar o conteudo. O documento foi criado.");
        setError(asMessage(err));
        setIsCreateSubmitting(false);
        stopApiTrace();
      }
    },
    [contentFile, contentMode, documentForm, loadWorkspace, openDocument, setActiveView, setContentDocxUrl, setContentError, setContentFile, setContentMode, setContentPdfUrl, setContentStatus, setDocumentForm, setError, setIsCreateSubmitting, setMessage],
  );

  const handleContentModeChange = useCallback((mode: ContentMode) => {
    setContentMode(mode);
    setContentFile(null);
    setContentPdfUrl("");
    setContentDocxUrl("");
    setContentStatus("idle");
    setContentError("");
  }, [setContentDocxUrl, setContentError, setContentFile, setContentMode, setContentPdfUrl, setContentStatus]);

  const handleContentFileChange = useCallback((file: File | null) => {
    setContentFile(file);
  }, [setContentFile]);

  const handleDownloadTemplate = useCallback(async (profileCode: string) => {
    try {
      if (!profileCode.trim()) return;
      const blob = await api.downloadProfileTemplateDocx(profileCode);
      const url = window.URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = `template-${profileCode.toLowerCase()}.docx`;
      document.body.appendChild(link);
      link.click();
      link.remove();
      window.URL.revokeObjectURL(url);
    } catch (err) {
      setError(asMessage(err));
    }
  }, [setError]);

  return {
    loadState,
    documentForm,
    contentMode,
    contentFile,
    contentPdfUrl,
    contentDocxUrl,
    contentStatus,
    contentError,
    documents,
    selectedDocument,
    versions,
    versionDiff,
    approvals,
    attachments,
    collaborationPresence,
    documentEditLock,
    policies,
    auditEvents,
    selectedFile,
    policyResourceId,
    setDocumentForm,
    setSelectedFile,
    setContentMode,
    setContentFile,
    setContentPdfUrl,
    setContentDocxUrl,
    setContentStatus,
    setContentError,
    setSelectedDocument,
    setCollaborationPresence,
    setDocumentEditLock,
    loadWorkspace,
    refreshWorkspace,
    refreshOperationalSignals,
    openDocument,
    handleUploadAttachment,
    handleCreateDocument,
    handleContentModeChange,
    handleContentFileChange,
    handleDownloadTemplate,
  };
}
