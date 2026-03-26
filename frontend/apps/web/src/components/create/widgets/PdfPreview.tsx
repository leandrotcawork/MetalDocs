import { useState } from "react";
import { Document, Page, pdfjs } from "react-pdf";

pdfjs.GlobalWorkerOptions.workerSrc = new URL(
  "pdfjs-dist/build/pdf.worker.min.mjs",
  import.meta.url,
).toString();

type PdfPreviewProps = {
  url: string;
  className?: string;
  width?: number;
};

export function PdfPreview(props: PdfPreviewProps) {
  const [numPages, setNumPages] = useState<number>(0);
  const [isLoading, setIsLoading] = useState(true);

  if (!props.url) {
    return null;
  }

  const pageWidth = props.width ?? 520;

  return (
    <div className={props.className ?? "create-doc-pdf-preview"} style={{ position: "relative" }}>
      {isLoading && (
        <div className="pdf-preview-loading-overlay">
          <span>Atualizando...</span>
        </div>
      )}
      <Document
        key={props.url}
        file={{ url: props.url }}
        loading={<div className="create-doc-pdf-loading">Carregando PDF...</div>}
        error={<div className="create-doc-pdf-error">Nao foi possivel carregar o PDF.</div>}
        onLoadSuccess={(pdf) => {
          setNumPages(pdf.numPages);
          setIsLoading(false);
        }}
        onLoadError={() => setIsLoading(false)}
      >
        {Array.from({ length: numPages }, (_, i) => (
          <Page key={i + 1} pageNumber={i + 1} width={pageWidth} />
        ))}
      </Document>
    </div>
  );
}
