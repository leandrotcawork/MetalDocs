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
  if (!props.url) {
    return null;
  }

  const pageWidth = props.width ?? 520;

  return (
    <div className={props.className ?? "create-doc-pdf-preview"}>
      <Document
        file={{ url: props.url }}
        loading={<div className="create-doc-pdf-loading">Carregando PDF...</div>}
        error={<div className="create-doc-pdf-error">Nao foi possivel carregar o PDF.</div>}
      >
        <Page pageNumber={1} width={pageWidth} />
      </Document>
    </div>
  );
}
