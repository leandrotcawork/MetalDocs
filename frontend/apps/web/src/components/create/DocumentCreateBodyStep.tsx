import { memo, useEffect, useState } from "react";
import type { ContentMode } from "./documentCreateTypes";
import { PdfPreview } from "./widgets/PdfPreview";

type DocumentCreateBodyStepProps = {
  contentMode: ContentMode;
  contentFile: File | null;
  contentPdfUrl: string;
  contentDocxUrl: string;
  contentStatus: "idle" | "saving" | "ready" | "error";
  contentError: string;
  profileCode: string;
  onContentModeChange: (mode: ContentMode) => void;
  onContentFileChange: (file: File | null) => void;
  onDownloadTemplate: (profileCode: string) => void | Promise<void>;
};

const DocumentCreateBodyStep = memo(function DocumentCreateBodyStep(props: DocumentCreateBodyStepProps) {
  const isNative = props.contentMode === "native";
  const isDocx = props.contentMode === "docx_upload";
  const fileName = props.contentFile?.name ?? "";
  const canDownloadTemplate = props.profileCode.trim().length > 0;
  const [templateDownloaded, setTemplateDownloaded] = useState(false);
  const hasFile = Boolean(props.contentFile);
  const hasPdf = Boolean(props.contentPdfUrl);

  useEffect(() => {
    setTemplateDownloaded(false);
  }, [props.profileCode, props.contentMode]);

  async function handleDownloadTemplate() {
    if (!canDownloadTemplate) {
      return;
    }
    try {
      await Promise.resolve(props.onDownloadTemplate(props.profileCode));
      setTemplateDownloaded(true);
    } catch {
      // Errors are surfaced by the caller; keep step state unchanged.
    }
  }

  const step1Status = templateDownloaded || hasFile ? "done" : "active";
  const step2Status = hasFile ? "done" : (templateDownloaded ? "active" : "pending");
  const step3Status = hasFile ? (hasPdf ? "done" : "active") : "pending";

  return (
    <div className="create-doc-content-module">
      <div className="create-doc-content-mode">
        <button
          type="button"
          className={`create-doc-content-card ${isNative ? "active" : ""}`}
          onClick={() => props.onContentModeChange("native")}
        >
          <span className="create-doc-content-card-icon" aria-hidden="true">
            <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" strokeLinejoin="round">
              <path d="M3 3h10v10H3z" />
              <path d="M5 6h6M5 9h6M5 12h3" />
            </svg>
          </span>
          <div className="create-doc-content-card-body">
            <strong>Editor nativo</strong>
            <small>Preencha o conteudo direto na plataforma em uma tela dedicada.</small>
            <span className="create-doc-content-badge">Recomendado</span>
          </div>
        </button>
        <button
          type="button"
          className={`create-doc-content-card ${isDocx ? "active" : ""}`}
          onClick={() => props.onContentModeChange("docx_upload")}
        >
          <span className="create-doc-content-card-icon" aria-hidden="true">
            <svg width="16" height="16" viewBox="0 0 16 16" fill="none" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round" strokeLinejoin="round">
              <path d="M4 2h5l3 3v9H4V2z" />
              <path d="M9 2v3h3" />
              <path d="M6 8h4M6 11h4" />
            </svg>
          </span>
          <div className="create-doc-content-card-body">
            <strong>Usar template Word</strong>
            <small>Baixe o .docx, preencha offline no Word e envie de volta.</small>
          </div>
        </button>
      </div>

      {isNative && (
        <div className="create-doc-content-block">
          <p className="create-doc-content-hint">
            O conteudo nativo e preenchido em uma tela dedicada apos a criacao do documento.
          </p>
        </div>
      )}

      {isDocx && (
        <div className="create-doc-content-block">
          <div className="create-doc-docx-steps">
            <div className={`create-doc-docx-step ${step1Status}`}>
              <span className="create-doc-docx-step-num">{step1Status === "done" ? "✓" : "1"}</span>
              <div className="create-doc-docx-step-body">
                <div className="create-doc-docx-step-title">Passo 1 de 3 — Baixe o template</div>
                <p className="create-doc-docx-step-desc">Use o template oficial do profile para manter o padrao documental.</p>
                <button
                  type="button"
                  className="ghost-button"
                  disabled={!canDownloadTemplate}
                  onClick={handleDownloadTemplate}
                >
                  Baixar template .docx
                </button>
              </div>
            </div>
            <div className={`create-doc-docx-step ${step2Status}`}>
              <span className="create-doc-docx-step-num">{step2Status === "done" ? "✓" : "2"}</span>
              <div className="create-doc-docx-step-body">
                <div className="create-doc-docx-step-title">Passo 2 de 3 — Preencha no Word</div>
                <p className="create-doc-docx-step-desc">Complete o conteudo offline no Word ou LibreOffice.</p>
              </div>
            </div>
            <div className={`create-doc-docx-step ${step3Status}`}>
              <span className="create-doc-docx-step-num">{step3Status === "done" ? "✓" : "3"}</span>
              <div className="create-doc-docx-step-body">
                <div className="create-doc-docx-step-title">Passo 3 de 3 — Envie o .docx preenchido</div>
                <p className="create-doc-docx-step-desc">O PDF sera gerado automaticamente apos o envio.</p>
                <label className="create-doc-docx-dropzone">
                  <input
                    type="file"
                    accept=".docx"
                    onChange={(event) => props.onContentFileChange(event.target.files?.[0] ?? null)}
                  />
                  <div className="create-doc-docx-dropzone-title">Arraste o .docx aqui ou clique para selecionar</div>
                  <div className="create-doc-docx-dropzone-hint">Apenas arquivos .docx · tamanho maximo 10MB</div>
                </label>
                {fileName && (
                  <div className="create-doc-docx-file">
                    <div>
                      <strong>{fileName}</strong>
                      <span>Arquivo pronto para envio.</span>
                    </div>
                    <button type="button" className="ghost-button" onClick={() => props.onContentFileChange(null)}>
                      Remover
                    </button>
                  </div>
                )}
              </div>
            </div>
          </div>
        </div>
      )}

      {props.contentStatus === "saving" && (
        <div className="create-doc-content-status">Processando arquivo e gerando PDF...</div>
      )}
      {props.contentStatus === "error" && props.contentError && (
        <div className="create-doc-content-error">{props.contentError}</div>
      )}

      {props.contentPdfUrl && isDocx && (
        <div className="create-doc-docx-preview">
          <div className="create-doc-docx-preview-title">Preview do PDF</div>
          <PdfPreview url={props.contentPdfUrl} />
        </div>
      )}
    </div>
  );
}, (prev, next) => (
  prev.contentMode === next.contentMode
  && prev.contentFile === next.contentFile
  && prev.contentPdfUrl === next.contentPdfUrl
  && prev.contentDocxUrl === next.contentDocxUrl
  && prev.contentStatus === next.contentStatus
  && prev.contentError === next.contentError
  && prev.profileCode === next.profileCode
));

export { DocumentCreateBodyStep };
