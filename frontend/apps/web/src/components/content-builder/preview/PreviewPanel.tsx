import { useEffect, useRef, useState } from "react";
import type { DocumentProfileSchemaItem } from "../../../lib.types";
import type { SchemaSection } from "../contentSchemaTypes";
import { PdfPreview } from "../../create/widgets/PdfPreview";
import { DocumentPreviewRenderer } from "./DocumentPreviewRenderer";
import { normalizeDocumentTypeSchema } from "../../../features/documents/runtime/schemaRuntimeAdapters";

type PreviewPanelProps = {
  schema: DocumentProfileSchemaItem | null;
  contentDraft: Record<string, unknown>;
  pdfUrl: string;
  profileCode: string;
  documentCode: string;
  documentTitle: string;
  version: number | null;
  activeSectionKey?: string | null;
  isDirty: boolean;
  collapsed: boolean;
  onToggleCollapse: (collapsed: boolean) => void;
};

type PreviewMode = "live" | "pdf";

export function PreviewPanel(props: PreviewPanelProps) {
  const {
    schema,
    contentDraft,
    pdfUrl,
    profileCode,
    documentCode,
    documentTitle,
    version,
    activeSectionKey,
    isDirty,
    collapsed,
    onToggleCollapse,
  } = props;

  const [mode, setMode] = useState<PreviewMode>("live");
  const previewRef = useRef<HTMLDivElement | null>(null);

  const sections: SchemaSection[] = normalizeDocumentTypeSchema(schema?.contentSchema).sections;

  useEffect(() => {
    if (!activeSectionKey || !previewRef.current || mode !== "live") return;
    const target = previewRef.current.querySelector<HTMLElement>(
      `[data-preview-section="${activeSectionKey}"]`,
    );
    if (target) {
      target.scrollIntoView({ behavior: "smooth", block: "start" });
    }
  }, [activeSectionKey, mode]);

  if (collapsed) {
    return (
      <aside className="content-builder-preview is-collapsed">
        <button
          type="button"
          className="content-builder-preview-toggle is-collapsed"
          onClick={() => onToggleCollapse(false)}
          aria-label="Expandir preview"
        >
          <svg width="14" height="14" viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="2">
            <path d="M13 5l-6 5 6 5" strokeLinecap="round" strokeLinejoin="round" />
          </svg>
        </button>
      </aside>
    );
  }

  return (
    <aside className="content-builder-preview">
      <div className="content-builder-preview-inner">
        <div className="preview-panel-header">
          <button
            type="button"
            className="content-builder-preview-toggle"
            onClick={() => onToggleCollapse(true)}
            aria-label="Recolher preview"
          >
            <svg width="14" height="14" viewBox="0 0 20 20" fill="none" stroke="currentColor" strokeWidth="2">
              <path d="M7 5l6 5-6 5" strokeLinecap="round" strokeLinejoin="round" />
            </svg>
          </button>
          <nav className="preview-panel-tabs">
            <button
              type="button"
              className={`preview-panel-tab ${mode === "live" ? "is-active" : ""}`}
              onClick={() => setMode("live")}
            >
              Ao vivo
            </button>
            <button
              type="button"
              className={`preview-panel-tab ${mode === "pdf" ? "is-active" : ""}`}
              onClick={() => setMode("pdf")}
            >
              PDF
              {isDirty && pdfUrl && <span className="preview-panel-tab-badge" />}
            </button>
          </nav>
        </div>

        <div className="preview-panel-body" ref={previewRef}>
          {mode === "live" && (
            <DocumentPreviewRenderer
              sections={sections}
              content={contentDraft}
              profileCode={profileCode}
              documentCode={documentCode}
              title={documentTitle}
              version={version}
              activeSectionKey={activeSectionKey}
            />
          )}
          {mode === "pdf" && (
            <div className="preview-panel-pdf">
              {pdfUrl ? (
                <PdfPreview url={pdfUrl} className="content-builder-preview-frame" width={380} />
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
                  <span>O PDF sera gerado automaticamente ao salvar.</span>
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </aside>
  );
}
