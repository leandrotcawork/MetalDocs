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
        <div className="preview-doc-header-wrapper">
          <table className="preview-doc-header-table">
            <tbody>
              <tr>
                <td className="preview-doc-cell preview-doc-cell--purple preview-doc-cell--title">
                  <span className="preview-doc-type-label">{profileCode.toUpperCase()}</span>
                  <span className="preview-doc-title">{title || "Sem titulo"}</span>
                </td>
                <td className="preview-doc-cell preview-doc-cell--meta">
                  <span className="preview-doc-meta-label">Código</span>
                  <span className="preview-doc-meta-value">{documentCode}</span>
                </td>
                <td className="preview-doc-cell preview-doc-cell--meta">
                  <span className="preview-doc-meta-label">Versão</span>
                  <span className="preview-doc-meta-value">{version ?? "-"}</span>
                </td>
              </tr>
              <tr>
                <td className="preview-doc-cell preview-doc-cell--teal">
                  <span className="preview-doc-profile-label">{profileCode.toUpperCase()}</span>
                </td>
                <td className="preview-doc-cell preview-doc-cell--status" colSpan={2}>
                  <span className="preview-doc-meta-label">Status</span>
                  <span className="preview-doc-meta-value">{documentStatus}</span>
                </td>
              </tr>
            </tbody>
          </table>
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
