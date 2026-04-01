import type { ReactNode } from "react";

type PreviewDocumentPageProps = {
  profileCode: string;
  documentCode: string;
  title: string;
  documentStatus: string;
  version: number | null;
  children: ReactNode;
};

export function PreviewDocumentPage({ profileCode, documentCode, title, documentStatus, version, children }: PreviewDocumentPageProps) {
  return (
    <div className="preview-document">
      <div className="preview-document-page">
        <header className="preview-document-header">
          <div className="preview-document-header-left">
            <div className="preview-document-logo-area">
              <svg width="28" height="28" viewBox="0 0 28 28" fill="none" stroke="var(--vinho)" strokeWidth="1.5">
                <rect x="3" y="3" width="22" height="22" rx="4" />
                <path d="M8 10h12M8 14h12M8 18h8" strokeLinecap="round" />
              </svg>
            </div>
            <div className="preview-document-header-info">
              <span className="preview-document-header-brand">MetalDocs</span>
              <span className="preview-document-header-profile">{profileCode.toUpperCase()}</span>
            </div>
          </div>
          <div className="preview-document-header-right">
            <span className="preview-document-header-status">{documentStatus}</span>
            <span className="preview-document-header-code">{documentCode}</span>
            <span className="preview-document-header-version">v{version ?? "-"}</span>
          </div>
        </header>

        <div className="preview-document-title-block">
          <h1 className="preview-document-title">{title || "Sem titulo"}</h1>
        </div>

        <div className="preview-document-content">{children}</div>

        <footer className="preview-document-footer">
          <span>{profileCode.toUpperCase()} — {documentCode}</span>
          <span>Versao {version ?? "-"}</span>
        </footer>
      </div>
    </div>
  );
}
