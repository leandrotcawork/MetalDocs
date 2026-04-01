import { useEffect, useRef, useState } from "react";
import { PdfPreview } from "../../create/widgets/PdfPreview";

type PreviewPanelProps = {
  pdfUrl: string;
  isDirty: boolean;
  isBusy: boolean;
  collapsed: boolean;
  onToggleCollapse: (collapsed: boolean) => void;
};

export function PreviewPanel({
  pdfUrl,
  isDirty,
  isBusy,
  collapsed,
  onToggleCollapse,
}: PreviewPanelProps) {
  const bodyRef = useRef<HTMLDivElement | null>(null);
  const [bodyWidth, setBodyWidth] = useState(0);

  useEffect(() => {
    const el = bodyRef.current;
    if (!el) return;
    const observer = new ResizeObserver((entries) => {
      for (const entry of entries) {
        setBodyWidth(Math.floor(entry.contentRect.width));
      }
    });
    observer.observe(el);
    return () => observer.disconnect();
  }, []);

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
          <span className="preview-panel-title-text">Preview</span>
        </div>

        <div className="preview-panel-body" ref={bodyRef}>
          {!pdfUrl ? (
            <div className="content-builder-preview-empty">
              <div className="content-builder-preview-empty-icon" aria-hidden="true">
                <svg width="18" height="18" viewBox="0 0 18 18" fill="none" stroke="currentColor" strokeWidth="1.4">
                  <path d="M4 2h7l3 3v11H4V2z" strokeLinejoin="round" />
                  <path d="M11 2v3h3" strokeLinejoin="round" />
                  <path d="M6 9h6M6 12h4" strokeLinecap="round" />
                </svg>
              </div>
              <strong>Nenhum preview disponivel</strong>
              <span>Salve o documento para gerar o preview.</span>
            </div>
          ) : (
            <div className="preview-panel-pdf-wrapper">
              {(isDirty || isBusy) && (
                <div className={`preview-panel-overlay ${isBusy ? "is-saving" : "is-stale"}`}>
                  {isBusy ? (
                    <>
                      <span className="preview-panel-spinner" />
                      <span>Atualizando preview...</span>
                    </>
                  ) : (
                    <span className="preview-panel-stale-badge">Alteracoes nao salvas</span>
                  )}
                </div>
              )}
              {bodyWidth > 0 && (
                <PdfPreview url={pdfUrl} width={bodyWidth} />
              )}
            </div>
          )}
        </div>
      </div>
    </aside>
  );
}
