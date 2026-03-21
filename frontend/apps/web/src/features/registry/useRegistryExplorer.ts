import { useCallback, useRef } from "react";
import { api, markUx, reportUxSequence, startApiTrace, stopApiTrace } from "../../lib.api";
import type {
  DocumentProfileGovernanceItem,
  DocumentProfileSchemaItem,
  MetadataFieldRuleItem,
} from "../../lib.types";
import { useDocumentsStore } from "../../store/documents.store";
import { useRegistryStore } from "../../store/registry.store";
import { useUiStore } from "../../store/ui.store";
import { asMessage } from "../shared/errors";

const profileCacheTtlMs = 5 * 60 * 1000;

export function useRegistryExplorer(onRefresh?: () => Promise<void> | void) {
  const {
    documentProfiles,
    processAreas,
    documentDepartments,
    subjects,
    selectedProfileSchema,
    selectedProfileSchemas,
    selectedProfileGovernance,
    setDocumentProfiles,
    setProcessAreas,
    setDocumentDepartments,
    setSubjects,
    setSelectedProfileSchema,
    setSelectedProfileSchemas,
    setSelectedProfileGovernance,
  } = useRegistryStore();
  const { documentForm, setDocumentForm } = useDocumentsStore();
  const { setError, setMessage } = useUiStore();

  const profileSchemaCacheRef = useRef(new Map<string, DocumentProfileSchemaItem[]>());
  const profileGovernanceCacheRef = useRef(new Map<string, DocumentProfileGovernanceItem>());
  const profileSchemaCacheMetaRef = useRef(new Map<string, number>());
  const profileGovernanceCacheMetaRef = useRef(new Map<string, number>());
  const profilePrefetchRef = useRef(new Set<string>());

  const applyDocumentProfile = useCallback(
    async (profileCode: string, preferredProcessArea = "") => {
      startApiTrace(`apply-profile:${profileCode}`);
      markUx(`profile-change-start:${profileCode}`);
      const now = Date.now();
      const schemaCacheAge = now - (profileSchemaCacheMetaRef.current.get(profileCode) ?? 0);
      const governanceCacheAge = now - (profileGovernanceCacheMetaRef.current.get(profileCode) ?? 0);
      const cachedSchemas = schemaCacheAge <= profileCacheTtlMs ? profileSchemaCacheRef.current.get(profileCode) : undefined;
      const cachedGovernance = governanceCacheAge <= profileCacheTtlMs ? profileGovernanceCacheRef.current.get(profileCode) : undefined;
      const cachedSchema = cachedSchemas?.find((item) => item.isActive) ?? cachedSchemas?.[0] ?? null;
      setSelectedProfileSchemas(cachedSchemas ?? []);
      setSelectedProfileSchema(cachedSchema);
      setSelectedProfileGovernance(cachedGovernance ?? null);
      setDocumentForm((current) => ({
        ...current,
        documentType: profileCode,
        documentProfile: profileCode,
        processArea: preferredProcessArea,
        metadata: metadataTextForProfileSchema(profileCode, cachedSchema),
      }));

      let schemaResponse: { items: DocumentProfileSchemaItem[] };
      let governance: DocumentProfileGovernanceItem | null = null;
      if (!cachedSchemas || !cachedGovernance) {
        try {
          const bundle = await api.getDocumentProfileBundle(profileCode);
          schemaResponse = { items: bundle.schema ? [bundle.schema] : [] };
          governance = bundle.governance ?? null;
          if (bundle.schema) {
            profileSchemaCacheRef.current.set(profileCode, schemaResponse.items);
            profileSchemaCacheMetaRef.current.set(profileCode, Date.now());
          }
          if (bundle.governance) {
            profileGovernanceCacheRef.current.set(profileCode, bundle.governance);
            profileGovernanceCacheMetaRef.current.set(profileCode, Date.now());
          }
          if (bundle.profile) {
            setDocumentProfiles((current) => current.map((item) => (item.code === bundle.profile.code ? bundle.profile : item)));
          }
          if (bundle.taxonomy.processAreas.length > 0 && processAreas.length === 0) {
            setProcessAreas(bundle.taxonomy.processAreas);
          }
          if (bundle.taxonomy.documentDepartments.length > 0 && documentDepartments.length === 0) {
            setDocumentDepartments(bundle.taxonomy.documentDepartments);
          }
          if (bundle.taxonomy.subjects.length > 0 && subjects.length === 0) {
            setSubjects(bundle.taxonomy.subjects);
          }
        } catch {
          const [fallbackSchemas, fallbackGovernance] = await Promise.all([
            api.listDocumentProfileSchemas(profileCode),
            api.getDocumentProfileGovernance(profileCode),
          ]);
          schemaResponse = fallbackSchemas;
          governance = fallbackGovernance;
        }
      } else {
        schemaResponse = { items: cachedSchemas };
        governance = cachedGovernance;
      }
      const schemas = Array.isArray(schemaResponse.items) ? schemaResponse.items : [];
      const schema = schemas.find((item) => item.isActive) ?? schemas[0] ?? null;
      if (schemas.length > 0 && !cachedSchemas) {
        profileSchemaCacheRef.current.set(profileCode, schemas);
        profileSchemaCacheMetaRef.current.set(profileCode, Date.now());
      }
      if (governance && !cachedGovernance) {
        profileGovernanceCacheRef.current.set(profileCode, governance);
        profileGovernanceCacheMetaRef.current.set(profileCode, Date.now());
      }
      setSelectedProfileSchemas(schemas);
      setSelectedProfileSchema(schema);
      setSelectedProfileGovernance(governance);
      markUx(`profile-schema-loaded:${profileCode}`);
      markUx(`profile-governance-loaded:${profileCode}`);
      setDocumentForm((current) => ({
        ...current,
        documentType: profileCode,
        documentProfile: profileCode,
        processArea: preferredProcessArea,
        metadata: metadataTextForProfileSchema(profileCode, schema),
      }));
      markUx(`profile-form-updated:${profileCode}`);
      if (typeof requestAnimationFrame === "function") {
        requestAnimationFrame(() => {
          requestAnimationFrame(() => {
            markUx(`profile-render-ready:${profileCode}`);
            reportUxSequence(`profile-change:${profileCode}`, [
              `profile-change-start:${profileCode}`,
              `profile-schema-loaded:${profileCode}`,
              `profile-governance-loaded:${profileCode}`,
              `profile-form-updated:${profileCode}`,
              `profile-render-ready:${profileCode}`,
            ]);
            stopApiTrace();
          });
        });
      } else {
        stopApiTrace();
      }
    },
    [
      documentDepartments.length,
      processAreas.length,
      setDocumentDepartments,
      setDocumentForm,
      setDocumentProfiles,
      setProcessAreas,
      setSelectedProfileGovernance,
      setSelectedProfileSchema,
      setSelectedProfileSchemas,
      setSubjects,
      subjects.length,
    ],
  );

  const prefetchProfile = useCallback(async (profileCode: string) => {
    if (profilePrefetchRef.current.has(profileCode)) return;
    profilePrefetchRef.current.add(profileCode);
    try {
      const bundle = await api.getDocumentProfileBundle(profileCode);
      const schemas = bundle.schema ? [bundle.schema] : [];
      if (schemas.length > 0) {
        profileSchemaCacheRef.current.set(profileCode, schemas);
        profileSchemaCacheMetaRef.current.set(profileCode, Date.now());
      }
      if (bundle.governance) {
        profileGovernanceCacheRef.current.set(profileCode, bundle.governance);
        profileGovernanceCacheMetaRef.current.set(profileCode, Date.now());
      }
    } catch {
      // Prefetch is best-effort.
    }
  }, []);

  const handleCreateProcessArea = useCallback(
    async (payload: { code: string; name: string; description: string }) => {
      try {
        setError("");
        await api.createProcessArea(payload);
        setMessage("Area de processo criada.");
        if (onRefresh) {
          await onRefresh();
        }
      } catch (err) {
        setError(asMessage(err));
      }
    },
    [onRefresh, setError, setMessage],
  );

  const handleUpdateProcessArea = useCallback(
    async (payload: { code: string; name: string; description: string }) => {
      try {
        setError("");
        await api.updateProcessArea(payload.code, payload);
        setMessage("Area de processo atualizada.");
        if (onRefresh) {
          await onRefresh();
        }
      } catch (err) {
        setError(asMessage(err));
      }
    },
    [onRefresh, setError, setMessage],
  );

  const handleDeleteProcessArea = useCallback(
    async (code: string) => {
      try {
        setError("");
        await api.deleteProcessArea(code);
        setMessage("Area de processo desativada.");
        if (onRefresh) {
          await onRefresh();
        }
      } catch (err) {
        setError(asMessage(err));
      }
    },
    [onRefresh, setError, setMessage],
  );

  const handleCreateSubject = useCallback(
    async (payload: { code: string; processAreaCode: string; name: string; description: string }) => {
      try {
        setError("");
        await api.createSubject(payload);
        setMessage("Subject criado.");
        if (onRefresh) {
          await onRefresh();
        }
      } catch (err) {
        setError(asMessage(err));
      }
    },
    [onRefresh, setError, setMessage],
  );

  const handleUpdateSubject = useCallback(
    async (payload: { code: string; processAreaCode: string; name: string; description: string }) => {
      try {
        setError("");
        await api.updateSubject(payload.code, payload);
        setMessage("Subject atualizado.");
        if (onRefresh) {
          await onRefresh();
        }
      } catch (err) {
        setError(asMessage(err));
      }
    },
    [onRefresh, setError, setMessage],
  );

  const handleDeleteSubject = useCallback(
    async (code: string) => {
      try {
        setError("");
        await api.deleteSubject(code);
        setMessage("Subject desativado.");
        if (onRefresh) {
          await onRefresh();
        }
      } catch (err) {
        setError(asMessage(err));
      }
    },
    [onRefresh, setError, setMessage],
  );

  const handleCreateDocumentProfile = useCallback(
    async (payload: { code: string; familyCode: string; name: string; alias: string; description: string; reviewIntervalDays: number }) => {
      try {
        setError("");
        await api.createDocumentProfile(payload);
        setMessage("Profile criado.");
        if (onRefresh) {
          await onRefresh();
        }
      } catch (err) {
        setError(asMessage(err));
      }
    },
    [onRefresh, setError, setMessage],
  );

  const handleUpdateDocumentProfile = useCallback(
    async (payload: { code: string; familyCode: string; name: string; alias: string; description: string; reviewIntervalDays: number }) => {
      try {
        setError("");
        await api.updateDocumentProfile(payload.code, payload);
        setMessage("Profile atualizado.");
        if (onRefresh) {
          await onRefresh();
        }
      } catch (err) {
        setError(asMessage(err));
      }
    },
    [onRefresh, setError, setMessage],
  );

  const handleDeleteDocumentProfile = useCallback(
    async (code: string) => {
      try {
        setError("");
        await api.deleteDocumentProfile(code);
        setMessage("Profile desativado.");
        if (onRefresh) {
          await onRefresh();
        }
      } catch (err) {
        setError(asMessage(err));
      }
    },
    [onRefresh, setError, setMessage],
  );

  const handleUpdateDocumentProfileGovernance = useCallback(
    async (payload: { profileCode: string; workflowProfile: string; reviewIntervalDays: number; approvalRequired: boolean; retentionDays: number; validityDays: number }) => {
      try {
        setError("");
        await api.updateDocumentProfileGovernance(payload.profileCode, payload);
        setMessage("Governanca atualizada.");
        profileGovernanceCacheRef.current.delete(payload.profileCode);
        profileGovernanceCacheMetaRef.current.delete(payload.profileCode);
        if (onRefresh) {
          await onRefresh();
        }
      } catch (err) {
        setError(asMessage(err));
      }
    },
    [onRefresh, setError, setMessage],
  );

  const handleUpsertDocumentProfileSchema = useCallback(
    async (payload: { profileCode: string; version: number; isActive: boolean; metadataRules: MetadataFieldRuleItem[] }) => {
      try {
        setError("");
        await api.upsertDocumentProfileSchema(payload.profileCode, payload);
        setMessage("Schema versionado atualizado.");
        profileSchemaCacheRef.current.delete(payload.profileCode);
        profileSchemaCacheMetaRef.current.delete(payload.profileCode);
        await applyDocumentProfile(payload.profileCode, documentForm.processArea);
      } catch (err) {
        setError(asMessage(err));
      }
    },
    [applyDocumentProfile, documentForm.processArea, setError, setMessage],
  );

  const handleActivateDocumentProfileSchema = useCallback(
    async (payload: { profileCode: string; version: number }) => {
      try {
        setError("");
        await api.activateDocumentProfileSchema(payload.profileCode, payload.version);
        setMessage("Schema ativo atualizado.");
        profileSchemaCacheRef.current.delete(payload.profileCode);
        profileSchemaCacheMetaRef.current.delete(payload.profileCode);
        await applyDocumentProfile(payload.profileCode, documentForm.processArea);
      } catch (err) {
        setError(asMessage(err));
      }
    },
    [applyDocumentProfile, documentForm.processArea, setError, setMessage],
  );

  return {
    applyDocumentProfile,
    prefetchProfile,
    handleCreateProcessArea,
    handleUpdateProcessArea,
    handleDeleteProcessArea,
    handleCreateSubject,
    handleUpdateSubject,
    handleDeleteSubject,
    handleCreateDocumentProfile,
    handleUpdateDocumentProfile,
    handleDeleteDocumentProfile,
    handleUpdateDocumentProfileGovernance,
    handleUpsertDocumentProfileSchema,
    handleActivateDocumentProfileSchema,
    documentProfiles,
    processAreas,
    documentDepartments,
    subjects,
    selectedProfileSchema,
    selectedProfileSchemas,
    selectedProfileGovernance,
  };
}

function metadataTextForProfileSchema(profileCode: string, schema?: DocumentProfileSchemaItem | null): string {
  const metadata: Record<string, string> = {};
  for (const rule of schema?.metadataRules ?? []) {
    const key = rule.name.trim();
    if (!key) continue;
    metadata[key] = "";
  }
  return JSON.stringify(metadata, null, 2);
}
