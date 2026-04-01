import { useCallback, useEffect, useMemo, useReducer, useRef, useState } from "react";
import { api } from "../../lib.api";
import type { DocumentListItem, DocumentProfileSchemaItem } from "../../lib.types";
import { formatDocumentDisplayName } from "../../features/shared/documentDisplay";
import { ProgressSidebar } from "../create/widgets/ProgressSidebar";
import type { StepStatus } from "../create/documentCreateTypes";
import { DynamicEditor } from "../../features/documents/runtime/DynamicEditor";
import type { SchemaSection } from "./contentSchemaTypes";
import { hasAnyValue, isFieldComplete, sectionAnchorId } from "./contentBuilderUtils";
import { useAutoSave } from "./useAutoSave";
import { PreviewPanel } from "./preview/PreviewPanel";
import { ResizableSplitPane } from "./ResizableSplitPane";

type ContentBuilderViewProps = {
  document: DocumentListItem | null;
  onBack: () => void;
  onCreateFromDraft?: (contentDraft: Record<string, unknown>) => Promise<{ documentId: string; pdfUrl: string; version: number | null }>;
};

type BuilderStatus = "loading" | "idle" | "dirty" | "saving" | "rendering" | "error";

type BuilderState = {
  status: BuilderStatus;
  error: string;
  pdfUrl: string;
  version: number | null;
  contentDraft: Record<string, unknown>;
  schema: DocumentProfileSchemaItem | null;
  previewCollapsed: boolean;
  sidebarCollapsed: boolean;
};

type BuilderAction =
  | { type: "load_start" }
  | { type: "load_success"; payload: { contentDraft: Record<string, unknown>; schema: DocumentProfileSchemaItem | null; version: number | null; pdfUrl: string } }
  | { type: "load_error"; payload: { message: string } }
  | { type: "set_draft"; payload: { contentDraft: Record<string, unknown> } }
  | { type: "set_status"; payload: { status: BuilderStatus } }
  | { type: "set_error"; payload: { message: string } }
  | { type: "set_pdf"; payload: { pdfUrl: string } }
  | { type: "set_version"; payload: { version: number | null } }
  | { type: "set_preview"; payload: { collapsed: boolean } }
  | { type: "set_sidebar"; payload: { collapsed: boolean } };

const initialState: BuilderState = {
  status: "loading",
  error: "",
  pdfUrl: "",
  version: null,
  contentDraft: {},
  schema: null,
  previewCollapsed: false,
  sidebarCollapsed: false,
};

function reducer(state: BuilderState, action: BuilderAction): BuilderState {
  switch (action.type) {
    case "load_start":
      return { ...state, status: "loading", error: "" };
    case "load_success":
      return {
        ...state,
        status: "idle",
        error: "",
        contentDraft: action.payload.contentDraft,
        schema: action.payload.schema,
        version: action.payload.version,
        pdfUrl: action.payload.pdfUrl,
      };
    case "load_error":
      return { ...state, status: "error", error: action.payload.message };
    case "set_draft":
      return { ...state, contentDraft: action.payload.contentDraft };
    case "set_status":
      return { ...state, status: action.payload.status };
    case "set_error":
      return { ...state, error: action.payload.message };
    case "set_pdf":
      return { ...state, pdfUrl: action.payload.pdfUrl };
    case "set_version":
      return { ...state, version: action.payload.version };
    case "set_preview":
      return { ...state, previewCollapsed: action.payload.collapsed };
    case "set_sidebar":
      return { ...state, sidebarCollapsed: action.payload.collapsed };
    default:
      return state;
  }
}

export function ContentBuilderView(props: ContentBuilderViewProps) {
  const documentId = props.document?.documentId ?? "";
  const [state, dispatch] = useReducer(reducer, initialState);
  const { status, error, pdfUrl, version, contentDraft, schema, previewCollapsed, sidebarCollapsed } = state;
  const editorRef = useRef<HTMLDivElement | null>(null);
  const [activeSectionKey, setActiveSectionKey] = useState<string | null>(null);
  const [autosaveLabel, setAutosaveLabel] = useState("Nao salvo");
  const [isExporting, setIsExporting] = useState(false);

  const profileCode = props.document?.documentProfile ?? "";
  const documentCode = useMemo(() => {
    if (!props.document?.documentId) return "--";
    return props.document.documentCode ?? formatDocumentDisplayName(props.document);
  }, [props.document]);

  const documentTitle = useMemo(() => {
    return props.document?.title ?? "";
  }, [props.document]);

  const sections = useMemo(() => {
    const raw = schema?.contentSchema as { sections?: SchemaSection[] } | undefined;
    return Array.isArray(raw?.sections) ? raw?.sections : [];
  }, [schema]);
  const currentSectionKey = activeSectionKey ?? sections[0]?.key ?? null;

  const sectionCompletion = useMemo(() => {
    const completion: Record<string, boolean> = {};
    sections.forEach((section) => {
      const sectionValue = (contentDraft?.[section.key] as Record<string, unknown>) ?? {};
      const fields = section.fields ?? [];
      const requiredFields = fields.filter((field) => field.required);
      const hasRequired = requiredFields.length > 0;
      const requiredOk = requiredFields.every((field) => isFieldComplete(field, sectionValue[field.key]));
      const anyValue = fields.some((field) => hasAnyValue(field, sectionValue[field.key]));
      completion[section.key] = hasRequired ? requiredOk : anyValue;
    });
    return completion;
  }, [sections, contentDraft]);

  // --- Auto-save integration ---
  const saveFn = useCallback(
    async (docId: string, body: { content: Record<string, unknown> }) => {
      const response = await api.saveDocumentContentNative(docId, body);
      return { pdfUrl: response.pdfUrl, version: response.version ?? null };
    },
    [],
  );

  const autoSave = useAutoSave({
    documentId,
    contentDraft,
    saveFn,
    debounceMs: 3000,
    enabled: !!documentId && status !== "loading",
  });
  const lastSavedAt = autoSave.lastSavedAt;

  // Sync auto-save results back into reducer
  useEffect(() => {
    if (autoSave.lastSavedPdfUrl && autoSave.lastSavedPdfUrl !== pdfUrl) {
      dispatch({ type: "set_pdf", payload: { pdfUrl: autoSave.lastSavedPdfUrl } });
    }
  }, [autoSave.lastSavedPdfUrl, pdfUrl]);

  useEffect(() => {
    if (lastSavedAt && status === "dirty") {
      dispatch({ type: "set_status", payload: { status: "idle" } });
    }
  }, [lastSavedAt, status]);

  // --- Data loading ---
  useEffect(() => {
    if (!documentId) {
      const pCode = props.document?.documentProfile;
      if (!pCode) {
        dispatch({ type: "set_status", payload: { status: "idle" } });
        dispatch({ type: "set_draft", payload: { contentDraft: {} } });
        return;
      }
      let isActive = true;
      async function loadDraftSchema() {
        dispatch({ type: "load_start" });
        try {
          const schemasResponse = await api.listDocumentProfileSchemas(pCode!);
          if (!isActive) return;
          const items = Array.isArray(schemasResponse.items) ? schemasResponse.items : [];
          const activeSchema = items.find((item) => item.isActive) ?? items[0] ?? null;
          dispatch({
            type: "load_success",
            payload: { contentDraft: {}, schema: activeSchema, version: null, pdfUrl: "" },
          });
        } catch {
          if (!isActive) return;
          dispatch({ type: "load_error", payload: { message: "Falha ao carregar o schema." } });
        }
      }
      void loadDraftSchema();
      return () => { isActive = false; };
    }
    let isActive = true;
    async function loadContent() {
      dispatch({ type: "load_start" });
      try {
        const [contentResponse, schemasResponse, pdfResponse] = await Promise.all([
          api.getDocumentContentNative(documentId),
          props.document?.documentProfile
            ? api.listDocumentProfileSchemas(props.document.documentProfile)
            : Promise.resolve({ items: [] as DocumentProfileSchemaItem[] }),
          api.getDocumentContentPdf(documentId).catch((err) => {
            if (statusOf(err) === 404) return null;
            throw err;
          }),
        ]);
        if (!isActive) return;
        const items = Array.isArray(schemasResponse.items) ? schemasResponse.items : [];
        const activeSchema = items.find((item) => item.isActive) ?? items[0] ?? null;
        dispatch({
          type: "load_success",
          payload: {
            contentDraft: (contentResponse.content ?? {}) as Record<string, unknown>,
            schema: activeSchema,
            version: contentResponse.version ?? null,
            pdfUrl: pdfResponse?.pdfUrl ?? "",
          },
        });
      } catch (err) {
        if (!isActive) return;
        if (statusOf(err) === 404) {
          dispatch({
            type: "load_success",
            payload: { contentDraft: {}, schema: null, version: null, pdfUrl: "" },
          });
          return;
        }
        dispatch({ type: "load_error", payload: { message: "Falha ao carregar o conteudo nativo." } });
      }
    }
    void loadContent();
    return () => { isActive = false; };
  }, [documentId, props.document?.documentProfile]);

  useEffect(() => {
    if (sections.length === 0) {
      setActiveSectionKey(null);
      return;
    }
    setActiveSectionKey((current) => current ?? sections[0].key);
  }, [sections]);

  useEffect(() => {
    const root = editorRef.current;
    if (!root) return;
    const nodes = Array.from(root.querySelectorAll<HTMLElement>("[data-section-key]"));
    if (nodes.length === 0) return;

    const observer = new IntersectionObserver(
      (entries) => {
        const visible = entries
          .filter((entry) => entry.isIntersecting)
          .sort((a, b) => b.intersectionRatio - a.intersectionRatio);
        if (visible.length === 0) return;
        const key = (visible[0].target as HTMLElement).dataset.sectionKey;
        if (key) setActiveSectionKey(key);
      },
      { root, threshold: [0.35, 0.6, 0.85] },
    );

    nodes.forEach((node) => observer.observe(node));
    return () => { observer.disconnect(); };
  }, [sections]);

  function handleSectionNav(sectionKey: string) {
    const anchorId = sectionAnchorId(sectionKey);
    const target = editorRef.current?.querySelector<HTMLElement>(`#${anchorId}`);
    if (target) {
      target.scrollIntoView({ behavior: "smooth", block: "start" });
      setActiveSectionKey(sectionKey);
    }
  }

  async function handleSave() {
    if (!documentId) return false;
    const savedContent = contentDraft ?? {};
    dispatch({ type: "set_error", payload: { message: "" } });
    dispatch({ type: "set_status", payload: { status: "saving" } });
    try {
      const response = await api.saveDocumentContentNative(documentId, { content: savedContent });
      autoSave.acknowledgeSave(savedContent, response.pdfUrl);
      dispatch({ type: "set_pdf", payload: { pdfUrl: response.pdfUrl } });
      dispatch({ type: "load_success", payload: { contentDraft: savedContent, schema, version: response.version ?? null, pdfUrl: response.pdfUrl } });
      return true;
    } catch {
      dispatch({ type: "load_error", payload: { message: "Falha ao salvar o conteudo." } });
      return false;
    }
  }

  async function handleExportDocx() {
    if (isExporting) return;
    setIsExporting(true);
    dispatch({ type: "set_error", payload: { message: "" } });
    try {
      let exportId = documentId;
      if (!exportId) {
        if (!props.onCreateFromDraft) {
          dispatch({
            type: "set_error",
            payload: { message: "Salve o rascunho antes de exportar." },
          });
          return;
        }
        dispatch({ type: "set_status", payload: { status: "saving" } });
        const created = await props.onCreateFromDraft(contentDraft ?? {});
        autoSave.acknowledgeSave(contentDraft ?? {}, created.pdfUrl);
        dispatch({
          type: "load_success",
          payload: {
            contentDraft: contentDraft ?? {},
            schema,
            version: created.version ?? null,
            pdfUrl: created.pdfUrl,
          },
        });
        exportId = created.documentId;
      } else if (status === "dirty") {
        const saved = await handleSave();
        if (!saved) return;
      }
      const blob = await api.exportDocumentDocx(exportId);
      const url = window.URL.createObjectURL(blob);
      const link = document.createElement("a");
      const nameBase = documentCode && documentCode !== "--" ? documentCode : exportId;
      const downloadName = `documento-${(nameBase || "documento")
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, "-")
        .replace(/^-+|-+$/g, "")}.docx`;
      link.href = url;
      link.download = downloadName;
      document.body.appendChild(link);
      link.click();
      link.remove();
      window.URL.revokeObjectURL(url);
    } catch {
      dispatch({ type: "set_error", payload: { message: "Nao foi possivel exportar o DOCX." } });
      dispatch({ type: "set_status", payload: { status: "idle" } });
    } finally {
      setIsExporting(false);
    }
  }

  async function handleRenderPdf() {
    if (!documentId) {
      if (!props.onCreateFromDraft) return;
      dispatch({ type: "set_error", payload: { message: "" } });
      dispatch({ type: "set_status", payload: { status: "rendering" } });
      try {
        const created = await props.onCreateFromDraft(contentDraft ?? {});
        autoSave.acknowledgeSave(contentDraft ?? {}, created.pdfUrl);
        dispatch({
          type: "load_success",
          payload: { contentDraft: contentDraft ?? {}, schema, version: created.version ?? null, pdfUrl: created.pdfUrl },
        });
      } catch {
        dispatch({ type: "load_error", payload: { message: "Falha ao criar o documento." } });
      }
      return;
    }
    if (status === "dirty") {
      await handleSave();
      return;
    }
    dispatch({ type: "set_error", payload: { message: "" } });
    dispatch({ type: "set_status", payload: { status: "rendering" } });
    try {
      const response = await api.renderDocumentContentPdf(documentId);
      dispatch({ type: "set_pdf", payload: { pdfUrl: response.pdfUrl } });
      dispatch({ type: "set_status", payload: { status: "idle" } });
    } catch {
      dispatch({ type: "load_error", payload: { message: "Nao foi possivel gerar o PDF." } });
    }
  }

  // --- Status label ---
  const effectiveStatus = autoSave.isSaving ? "saving" : status;

  useEffect(() => {
    if (autoSave.isSaving) {
      setAutosaveLabel("Salvando...");
      return;
    }
    if (effectiveStatus === "saving") {
      setAutosaveLabel("Salvando...");
      return;
    }
    if (effectiveStatus === "rendering") {
      setAutosaveLabel("Gerando PDF...");
      return;
    }
    if (effectiveStatus === "dirty") {
      setAutosaveLabel("Editando...");
      return;
    }
    if (effectiveStatus === "idle" && lastSavedAt) {
      setAutosaveLabel("Salvo agora");
      const timer = window.setTimeout(() => setAutosaveLabel("Salvo ha pouco"), 3000);
      return () => window.clearTimeout(timer);
    }
    setAutosaveLabel("Salvo");
  }, [effectiveStatus, lastSavedAt, autoSave.isSaving]);

  const statusLabel = autoSave.isSaving
    ? "Salvando..."
    : status === "dirty"
      ? "Editando..."
      : status === "saving"
        ? "Salvando..."
        : status === "rendering"
          ? "Gerando PDF..."
          : "Salvo";

  if (!props.document) {
    return (
      <section className="content-builder-empty">
        <strong>Nenhum documento selecionado.</strong>
        <p>Abra um documento antes de editar o conteudo.</p>
        <button type="button" className="ghost-button" onClick={props.onBack}>
          Voltar ao acervo
        </button>
      </section>
    );
  }

  const editorPane = (
    <main className="content-builder-editor" ref={editorRef}>
      <div className="content-builder-editor-inner">
        <DynamicEditor
          schema={schema}
          value={contentDraft}
          activeSectionKey={activeSectionKey}
          onChange={(next) => {
            dispatch({ type: "set_draft", payload: { contentDraft: next } });
            dispatch({ type: "set_status", payload: { status: "dirty" } });
          }}
        />
        {error && <div className="content-builder-error">{error}</div>}
        {autoSave.error && <div className="content-builder-error">{autoSave.error}</div>}
      </div>
    </main>
  );

  const previewPane = (
    <PreviewPanel
      pdfUrl={pdfUrl}
      isDirty={autoSave.hasPendingChanges}
      isBusy={autoSave.isSaving || status === "saving" || status === "rendering"}
      collapsed={previewCollapsed}
      onToggleCollapse={(collapsed) => dispatch({ type: "set_preview", payload: { collapsed } })}
    />
  );

  return (
    <section className="content-builder">
      <header className="content-builder-topbar">
        <div className="content-builder-topbar-brand">
          <div className="content-builder-topbar-mark" aria-hidden="true">
            <svg width="15" height="15" viewBox="0 0 15 15" fill="none" stroke="currentColor" strokeWidth="1.3">
              <path d="M3 2h6.5L12 4.5V13H3V2z" strokeLinejoin="round" />
              <path d="M9.5 2v2.5H12" strokeLinejoin="round" />
              <path d="M5 7h5M5 9.5h5M5 12h3" strokeLinecap="round" />
            </svg>
          </div>
        </div>
        <div className="content-builder-breadcrumb">
          <span className="content-builder-breadcrumb-link">MetalDocs</span>
          <span className="content-builder-breadcrumb-sep">&rsaquo;</span>
          <span className="content-builder-breadcrumb-link">Acervo</span>
          <span className="content-builder-breadcrumb-sep">&rsaquo;</span>
          <span className="content-builder-breadcrumb-link">{documentCode}</span>
          <span className="content-builder-breadcrumb-sep">&rsaquo;</span>
          <strong className="content-builder-breadcrumb-current">Editor de conteudo</strong>
        </div>
        <div className="content-builder-topbar-actions">
          <span className={`content-builder-status ${status === "dirty" ? "is-warning" : ""}`}>{statusLabel}</span>
        </div>
      </header>

      <section className="content-builder-docbar">
        <div className="content-builder-docbar-left">
          <div className="content-builder-meta">
            <span className="content-builder-pill">Profile / {profileCode.toUpperCase()}</span>
            <span className="content-builder-pill">Status / {props.document.status}</span>
            <span className="content-builder-pill">Versao / {version ?? "-"}</span>
          </div>
        </div>
        <div className="content-builder-docbar-center">
          <div className="content-builder-title">
            <strong className="content-builder-doccode">
              {props.document ? formatDocumentDisplayName(props.document, []) : "Documento"}
            </strong>
          </div>
        </div>
        <div className="content-builder-docbar-right">
          <button type="button" className="content-builder-btn primary" onClick={props.onBack}>
            Voltar
          </button>
        </div>
      </section>

      <div className="content-builder-layout">
        <aside className={`content-builder-sections-nav ${sidebarCollapsed ? "is-collapsed" : ""}`}>
          {sidebarCollapsed ? (
            <button
              type="button"
              className="content-builder-sidebar-toggle is-collapsed"
              onClick={() => dispatch({ type: "set_sidebar", payload: { collapsed: false } })}
              aria-label="Expandir navegacao"
            >
              <svg width="14" height="14" viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="2">
                <path d="M7 5l6 5-6 5" strokeLinecap="round" strokeLinejoin="round" />
              </svg>
            </button>
          ) : (
            <>
              <div className="content-builder-sidebar-header">
                <button
                  type="button"
                  className="content-builder-sidebar-toggle"
                  onClick={() => dispatch({ type: "set_sidebar", payload: { collapsed: true } })}
                  aria-label="Recolher navegacao"
                >
                  <svg width="14" height="14" viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="2">
                    <path d="M13 5l-6 5 6 5" strokeLinecap="round" strokeLinejoin="round" />
                  </svg>
                </button>
              </div>
              <ProgressSidebar
                title="Secoes"
                items={sections.map((section, index) => {
                  const isActive = currentSectionKey === section.key;
                  const isComplete = sectionCompletion[section.key] ?? false;
                  const stepStatus: StepStatus = isActive ? "active" : isComplete ? "done" : "pending";
                  return {
                    key: section.key,
                    label: section.title ?? section.key,
                    description: section.description ?? `Secao ${index + 1}`,
                    status: stepStatus,
                    isCurrent: isActive,
                    onSelect: () => handleSectionNav(section.key),
                  };
                })}
              />
            </>
          )}
        </aside>

        {previewCollapsed ? (
          <>
            {editorPane}
            {previewPane}
          </>
        ) : (
          <ResizableSplitPane left={editorPane} right={previewPane} />
        )}
      </div>

      <footer className="content-builder-footer">
        <div className="content-builder-footer-left">
          <span className="content-builder-version-pill">v{version ?? "-"}</span>
          <div className="content-builder-autosave">
            <span className={`content-builder-autosave-dot ${status === "dirty" ? "is-warn" : autoSave.isSaving || status === "saving" || status === "rendering" ? "is-info" : "is-ok"}`} />
            <span>{autosaveLabel}</span>
          </div>
        </div>
        <div className="content-builder-footer-actions">
          <button
            type="button"
            className="content-builder-btn ghost"
            onClick={handleExportDocx}
            disabled={isExporting || status === "saving" || status === "loading" || status === "rendering" || autoSave.isSaving}
          >
            Exportar .docx
          </button>
          <button
            type="button"
            className="content-builder-btn ghost"
            onClick={handleSave}
            disabled={isExporting || status === "saving" || status === "loading" || status === "rendering" || autoSave.isSaving}
          >
            Salvar rascunho
          </button>
          <button
            type="button"
            className="content-builder-btn primary"
            onClick={handleRenderPdf}
            disabled={isExporting || status === "saving" || status === "loading" || status === "rendering" || autoSave.isSaving}
          >
            Gerar PDF
          </button>
        </div>
      </footer>
    </section>
  );
}

function statusOf(error: unknown): number | undefined {
  if (error && typeof error === "object" && "status" in error && typeof (error as { status?: unknown }).status === "number") {
    return (error as { status: number }).status;
  }
  return undefined;
}
