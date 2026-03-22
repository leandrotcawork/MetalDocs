import { useEffect, useMemo, useReducer, useRef, useState } from "react";
import { api } from "../../lib.api";
import type { DocumentListItem, DocumentProfileSchemaItem } from "../../lib.types";
import { ProgressSidebar } from "../create/widgets/ProgressSidebar";
import type { StepStatus } from "../create/documentCreateTypes";
import { PdfPreview } from "../create/widgets/PdfPreview";
import { ContentSchemaForm } from "./ContentSchemaForm";
import type { SchemaSection } from "./contentSchemaTypes";
import { hasAnyValue, isFieldComplete, sectionAnchorId } from "./contentBuilderUtils";

type ContentBuilderViewProps = {
  document: DocumentListItem | null;
  onBack: () => void;
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
};

type BuilderAction =
  | { type: "load_start" }
  | { type: "load_success"; payload: { contentDraft: Record<string, unknown>; schema: DocumentProfileSchemaItem | null; version: number | null; pdfUrl: string } }
  | { type: "load_error"; payload: { message: string } }
  | { type: "set_draft"; payload: { contentDraft: Record<string, unknown> } }
  | { type: "set_status"; payload: { status: BuilderStatus } }
  | { type: "set_error"; payload: { message: string } }
  | { type: "set_pdf"; payload: { pdfUrl: string } }
  | { type: "set_preview"; payload: { collapsed: boolean } };

const initialState: BuilderState = {
  status: "loading",
  error: "",
  pdfUrl: "",
  version: null,
  contentDraft: {},
  schema: null,
  previewCollapsed: false,
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
    case "set_preview":
      return { ...state, previewCollapsed: action.payload.collapsed };
    default:
      return state;
  }
}

export function ContentBuilderView(props: ContentBuilderViewProps) {
  const documentId = props.document?.documentId ?? "";
  const [state, dispatch] = useReducer(reducer, initialState);
  const { status, error, pdfUrl, version, contentDraft, schema, previewCollapsed } = state;
  const editorRef = useRef<HTMLDivElement | null>(null);
  const [activeSectionKey, setActiveSectionKey] = useState<string | null>(null);
  const [lastSavedAt, setLastSavedAt] = useState<Date | null>(null);
  const [autosaveLabel, setAutosaveLabel] = useState("Nao salvo");

  const documentCode = useMemo(() => {
    if (!props.document?.documentId) return "--";
    return props.document.documentId.slice(0, 8).toUpperCase();
  }, [props.document?.documentId]);

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

  useEffect(() => {
    if (!documentId) {
      dispatch({ type: "set_status", payload: { status: "idle" } });
      dispatch({ type: "set_draft", payload: { contentDraft: {} } });
      return;
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
            if (statusOf(err) === 404) {
              return null;
            }
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
    return () => {
      isActive = false;
    };
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
        if (visible.length === 0) {
          return;
        }
        const key = (visible[0].target as HTMLElement).dataset.sectionKey;
        if (key) {
          setActiveSectionKey(key);
        }
      },
      { root, threshold: [0.35, 0.6, 0.85] },
    );

    nodes.forEach((node) => observer.observe(node));
    return () => {
      observer.disconnect();
    };
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
    if (!documentId) return;
    dispatch({ type: "set_error", payload: { message: "" } });
    const parsedContent: Record<string, unknown> = contentDraft ?? {};
    dispatch({ type: "set_status", payload: { status: "saving" } });
    try {
      const response = await api.saveDocumentContentNative(documentId, { content: parsedContent });
      dispatch({ type: "set_pdf", payload: { pdfUrl: response.pdfUrl } });
      dispatch({ type: "load_success", payload: { contentDraft: parsedContent, schema, version: response.version ?? null, pdfUrl: response.pdfUrl } });
      setLastSavedAt(new Date());
    } catch {
      dispatch({ type: "load_error", payload: { message: "Falha ao salvar o conteudo." } });
    }
  }

  async function handleRenderPdf() {
    if (!documentId) return;
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

  useEffect(() => {
    if (status === "saving") {
      setAutosaveLabel("Salvando...");
      return;
    }
    if (status === "rendering") {
      setAutosaveLabel("Gerando PDF...");
      return;
    }
    if (status === "dirty") {
      setAutosaveLabel("Nao salvo");
      return;
    }
    if (status === "idle" && lastSavedAt) {
      setAutosaveLabel("Salvo agora");
      const timer = window.setTimeout(() => setAutosaveLabel("Salvo ha pouco"), 3000);
      return () => window.clearTimeout(timer);
    }
    setAutosaveLabel("Salvo");
  }, [status, lastSavedAt]);

  const statusLabel = status === "dirty"
    ? "Nao salvo"
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
          <span className="content-builder-breadcrumb-sep">›</span>
          <span className="content-builder-breadcrumb-link">Acervo</span>
          <span className="content-builder-breadcrumb-sep">›</span>
          <span className="content-builder-breadcrumb-link">{documentCode}</span>
          <span className="content-builder-breadcrumb-sep">›</span>
          <strong className="content-builder-breadcrumb-current">Editor de conteudo</strong>
        </div>
        <div className="content-builder-topbar-actions">
          <span className={`content-builder-status ${status === "dirty" ? "is-warning" : ""}`}>{statusLabel}</span>
          <button type="button" className="content-builder-btn ghost" onClick={props.onBack}>Voltar</button>
        </div>
      </header>

      <section className="content-builder-docbar">
        <div className="content-builder-docbar-left">
          <div className="content-builder-title">{props.document.title}</div>
          <div className="content-builder-meta">
            <span className="content-builder-pill">Profile: {props.document.documentProfile.toUpperCase()}</span>
            <span className="content-builder-pill">Status: {props.document.status}</span>
            <span className="content-builder-pill">Versao: {version ?? "-"}</span>
          </div>
        </div>
        <div className="content-builder-docbar-right">
          <button type="button" className="content-builder-btn ghost" onClick={props.onBack}>Voltar</button>
        </div>
      </section>

      <div className="content-builder-layout">
        <aside className="content-builder-sections-nav">
          <ProgressSidebar
            title="Secoes"
            items={sections.map((section, index) => {
              const isActive = currentSectionKey === section.key;
              const isComplete = sectionCompletion[section.key] ?? false;
              const status: StepStatus = isActive ? "active" : isComplete ? "done" : "pending";
              return {
                key: section.key,
                label: section.title ?? section.key,
                description: section.description ?? `Secao ${index + 1}`,
                status,
                isCurrent: isActive,
                onSelect: () => handleSectionNav(section.key),
              };
            })}
          />
        </aside>

        <main className="content-builder-editor" ref={editorRef}>
          <div className="content-builder-editor-inner">
            <ContentSchemaForm
              schema={schema}
              value={contentDraft}
              activeSectionKey={activeSectionKey}
              onChange={(next) => {
                dispatch({ type: "set_draft", payload: { contentDraft: next } });
                dispatch({ type: "set_status", payload: { status: "dirty" } });
              }}
            />
            {error && <div className="content-builder-error">{error}</div>}
          </div>
        </main>

        <aside className={`content-builder-preview ${previewCollapsed ? "is-collapsed" : ""}`}>
          {!previewCollapsed && (
            <div className="content-builder-preview-inner">
              <div className="content-builder-preview-header">
                <div className="content-builder-preview-title">
                  <strong>Preview do PDF</strong>
                  <small>Atualize para refletir as ultimas edicoes.</small>
                </div>
                <button type="button" className="ghost-button" onClick={() => dispatch({ type: "set_preview", payload: { collapsed: true } })}>
                  Recolher
                </button>
              </div>
              <div className="content-builder-preview-body">
                {pdfUrl ? (
                  <PdfPreview url={pdfUrl} className="content-builder-preview-frame" width={320} />
                ) : (
                  <div className="content-builder-preview-empty">
                    <div className="content-builder-preview-empty-icon" aria-hidden="true">
                      <svg width="18" height="18" viewBox="0 0 18 18" fill="none" stroke="currentColor" strokeWidth="1.4">
                        <path d="M4 2h7l3 3v11H4V2z" strokeLinejoin="round" />
                        <path d="M11 2v3h3" strokeLinejoin="round" />
                        <path d="M6 9h6M6 12h4" strokeLinecap="round" />
                      </svg>
                    </div>
                    <strong>Nenhum PDF gerado</strong>
                    <span>Salve o conteudo e clique em "Gerar PDF" para visualizar.</span>
                  </div>
                )}
                {status === "dirty" && pdfUrl && (
                  <div className="content-builder-preview-warning">
                    Preview pode estar desatualizado. Gere novamente para ver as ultimas edicoes.
                  </div>
                )}
              </div>
            </div>
          )}
          {previewCollapsed && (
            <button type="button" className="content-builder-preview-collapsed" onClick={() => dispatch({ type: "set_preview", payload: { collapsed: false } })}>
              Preview PDF
            </button>
          )}
        </aside>
      </div>

      <footer className="content-builder-footer">
        <div className="content-builder-footer-left">
          <span className="content-builder-version-pill">v{version ?? "-"}</span>
          <div className="content-builder-autosave">
            <span className={`content-builder-autosave-dot ${status === "dirty" ? "is-warn" : status === "saving" || status === "rendering" ? "is-info" : "is-ok"}`} />
            <span>{autosaveLabel}</span>
          </div>
        </div>
        <div className="content-builder-footer-actions">
          <button
            type="button"
            className="content-builder-btn ghost"
            onClick={handleSave}
            disabled={status === "saving" || status === "loading" || status === "rendering"}
          >
            Salvar rascunho
          </button>
          <button
            type="button"
            className="content-builder-btn primary"
            onClick={handleRenderPdf}
            disabled={status === "saving" || status === "loading" || status === "rendering"}
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
