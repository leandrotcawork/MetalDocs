import { useEffect, useMemo, useState } from "react";
import { api } from "../../lib.api";
import type { DocumentListItem } from "../../lib.types";
import { PdfPreview } from "../create/widgets/PdfPreview";

type ContentBuilderViewProps = {
  document: DocumentListItem | null;
  onBack: () => void;
};

type BuilderStatus = "loading" | "idle" | "dirty" | "saving" | "error";

export function ContentBuilderView(props: ContentBuilderViewProps) {
  const documentId = props.document?.documentId ?? "";
  const [contentDraft, setContentDraft] = useState("{\n\n}");
  const [status, setStatus] = useState<BuilderStatus>("loading");
  const [error, setError] = useState("");
  const [pdfUrl, setPdfUrl] = useState("");
  const [version, setVersion] = useState<number | null>(null);
  const [previewCollapsed, setPreviewCollapsed] = useState(false);

  const documentCode = useMemo(() => {
    if (!props.document?.documentId) return "--";
    return props.document.documentId.slice(0, 8).toUpperCase();
  }, [props.document?.documentId]);

  useEffect(() => {
    if (!documentId) {
      setStatus("idle");
      setContentDraft("{\n\n}");
      return;
    }
    let isActive = true;
    async function loadContent() {
      setStatus("loading");
      setError("");
      try {
        const response = await api.getDocumentContentNative(documentId);
        if (!isActive) return;
        setVersion(response.version);
        setContentDraft(JSON.stringify(response.content ?? {}, null, 2));
        setStatus("idle");
      } catch (err) {
        if (!isActive) return;
        if (statusOf(err) === 404) {
          setContentDraft("{\n\n}");
          setStatus("idle");
          return;
        }
        setError("Falha ao carregar o conteudo nativo.");
        setStatus("error");
      }
    }
    void loadContent();
    return () => {
      isActive = false;
    };
  }, [documentId]);

  async function handleSave() {
    if (!documentId) return;
    setError("");
    let parsedContent: Record<string, unknown> = {};
    if (contentDraft.trim()) {
      try {
        parsedContent = JSON.parse(contentDraft) as Record<string, unknown>;
      } catch {
        setError("JSON invalido. Corrija o conteudo antes de salvar.");
        setStatus("error");
        return;
      }
    }
    setStatus("saving");
    try {
      const response = await api.saveDocumentContentNative(documentId, { content: parsedContent });
      setVersion(response.version);
      setPdfUrl(response.pdfUrl);
      setStatus("idle");
    } catch {
      setError("Falha ao salvar o conteudo.");
      setStatus("error");
    }
  }

  async function handleRenderPdf() {
    if (!documentId) return;
    if (status === "dirty") {
      await handleSave();
      return;
    }
    setError("");
    setStatus("saving");
    try {
      const response = await api.renderDocumentContentPdf(documentId);
      setPdfUrl(response.pdfUrl);
      setStatus("idle");
    } catch {
      setError("Nao foi possivel gerar o PDF.");
      setStatus("error");
    }
  }

  const statusLabel = status === "dirty"
    ? "Nao salvo"
    : status === "saving"
      ? "Salvando..."
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
      <header className="content-builder-header">
        <div>
          <div className="content-builder-code">{documentCode}</div>
          <h2 className="content-builder-title">{props.document.title}</h2>
          <div className="content-builder-meta">
            <span>Profile: {props.document.documentProfile.toUpperCase()}</span>
            <span>Status: {props.document.status}</span>
          </div>
        </div>
        <div className="content-builder-header-actions">
          <span className={`content-builder-status ${status === "dirty" ? "is-warning" : ""}`}>{statusLabel}</span>
          <button type="button" className="ghost-button" onClick={props.onBack}>
            Voltar
          </button>
        </div>
      </header>

      <div className="content-builder-body">
        <div className="content-builder-editor">
          <div className="content-builder-section">
            <div className="content-builder-section-head">
              <strong>Conteudo estruturado (JSON)</strong>
              <small>Substituido por editor orientado a schema na Task 066.</small>
            </div>
            <textarea
              className="content-builder-textarea"
              value={contentDraft}
              rows={18}
              onChange={(event) => {
                setContentDraft(event.target.value);
                setStatus((current) => (current === "dirty" ? current : "dirty"));
              }}
              placeholder={`{\n  "section": "preencher"\n}`}
            />
          </div>
          {error && <div className="content-builder-error">{error}</div>}
        </div>

        <aside className={`content-builder-preview ${previewCollapsed ? "is-collapsed" : ""}`}>
          {!previewCollapsed && (
            <div className="content-builder-preview-inner">
              <div className="content-builder-preview-header">
                <strong>Preview do PDF</strong>
                <button type="button" className="ghost-button" onClick={() => setPreviewCollapsed(true)}>
                  Recolher
                </button>
              </div>
              {pdfUrl ? (
                <PdfPreview url={pdfUrl} className="content-builder-preview-frame" width={300} />
              ) : (
                <div className="content-builder-preview-empty">
                  <strong>Nenhum PDF gerado</strong>
                  <span>Salve o conteudo e clique em "Gerar PDF" para visualizar.</span>
                </div>
              )}
            </div>
          )}
          {previewCollapsed && (
            <button type="button" className="content-builder-preview-collapsed" onClick={() => setPreviewCollapsed(false)}>
              Preview PDF
            </button>
          )}
        </aside>
      </div>

      <footer className="content-builder-footer">
        <span>Versao: {version ?? "-"}</span>
        <div className="content-builder-footer-actions">
          <button type="button" className="ghost-button" onClick={handleSave} disabled={status === "saving" || status === "loading"}>
            Salvar
          </button>
          <button type="button" onClick={handleRenderPdf} disabled={status === "saving" || status === "loading"}>
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
