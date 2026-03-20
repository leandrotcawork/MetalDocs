import { useEffect, useMemo, useReducer } from "react";
import { api } from "../../lib.api";
import type { DocumentListItem, DocumentProfileSchemaItem } from "../../lib.types";
import { PdfPreview } from "../create/widgets/PdfPreview";
import { ContentSchemaForm } from "./ContentSchemaForm";

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

  const documentCode = useMemo(() => {
    if (!props.document?.documentId) return "--";
    return props.document.documentId.slice(0, 8).toUpperCase();
  }, [props.document?.documentId]);

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
            pdfUrl: pdfResponse?.url ?? "",
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

  async function handleSave() {
    if (!documentId) return;
    dispatch({ type: "set_error", payload: { message: "" } });
    const parsedContent: Record<string, unknown> = contentDraft ?? {};
    dispatch({ type: "set_status", payload: { status: "saving" } });
    try {
      const response = await api.saveDocumentContentNative(documentId, { content: parsedContent });
      dispatch({ type: "set_pdf", payload: { pdfUrl: response.pdfUrl } });
      dispatch({ type: "load_success", payload: { contentDraft: parsedContent, schema, version: response.version ?? null, pdfUrl: response.pdfUrl } });
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
          <button type="button" className="ghost-button" onClick={props.onBack}>
            Voltar
          </button>
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
          <button type="button" className="ghost-button" onClick={props.onBack}>
            Voltar
          </button>
        </div>
      </section>

      <div className="content-builder-layout">
        <aside className="content-builder-sections-nav">
          <div className="content-builder-sections-title">Secoes</div>
          <div className="content-builder-sections-list">
            <div className="content-builder-section-link is-active">
              <span className="content-builder-section-num">1</span>
              <span>Identificacao</span>
            </div>
            <div className="content-builder-section-link">
              <span className="content-builder-section-num">2</span>
              <span>Entradas e saidas</span>
            </div>
            <div className="content-builder-section-link">
              <span className="content-builder-section-num">3</span>
              <span>Processo</span>
            </div>
            <div className="content-builder-section-link">
              <span className="content-builder-section-num">4</span>
              <span>Indicadores</span>
            </div>
          </div>
        </aside>

        <main className="content-builder-editor">
          <div className="content-builder-editor-inner">
            <ContentSchemaForm
              schema={schema}
              value={contentDraft}
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
                <div>
                  <strong>Preview do PDF</strong>
                  <small>Atualize para refletir as ultimas edicoes.</small>
                </div>
                <button type="button" className="ghost-button" onClick={() => dispatch({ type: "set_preview", payload: { collapsed: true } })}>
                  Recolher
                </button>
              </div>
              {pdfUrl ? (
                <PdfPreview url={pdfUrl} className="content-builder-preview-frame" width={320} />
              ) : (
                <div className="content-builder-preview-empty">
                  <strong>Nenhum PDF gerado</strong>
                  <span>Salve o conteudo e clique em "Gerar PDF" para visualizar.</span>
                </div>
              )}
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
        <span className="content-builder-footer-info">Versao ativa: {version ?? "-"}</span>
        <div className="content-builder-footer-actions">
          <button type="button" className="ghost-button" onClick={handleSave} disabled={status === "saving" || status === "loading" || status === "rendering"}>
            Salvar rascunho
          </button>
          <button type="button" onClick={handleRenderPdf} disabled={status === "saving" || status === "loading" || status === "rendering"}>
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
