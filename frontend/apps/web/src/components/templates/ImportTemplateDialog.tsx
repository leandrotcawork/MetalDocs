import { useRef, useState } from "react";
import { importTemplate } from "../../api/templates";
import type { ImportResultDTO } from "../../api/templates";

type ImportTemplateDialogProps = {
  profileCode: string;
  onClose: () => void;
  onSuccess: (result: ImportResultDTO) => void;
};

export function ImportTemplateDialog({ profileCode, onClose, onSuccess }: ImportTemplateDialogProps) {
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const fileRef = useRef<HTMLInputElement>(null);

  async function handleImport() {
    const file = fileRef.current?.files?.[0];
    if (!file) {
      setError("Selecione um arquivo .json para importar.");
      return;
    }
    setLoading(true);
    setError(null);
    try {
      const result = await importTemplate(profileCode, file);
      onSuccess(result);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Erro ao importar o template.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <div
      role="dialog"
      aria-modal="true"
      aria-label="Importar template"
      style={{
        position: "fixed",
        inset: 0,
        background: "rgba(0,0,0,0.55)",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
        zIndex: 1000,
      }}
    >
      <div
        style={{
          background: "var(--surface-2, #1c1c24)",
          border: "1px solid rgba(255,255,255,0.12)",
          borderRadius: "10px",
          padding: "1.5rem",
          minWidth: "340px",
          maxWidth: "480px",
          width: "100%",
          display: "flex",
          flexDirection: "column",
          gap: "1rem",
        }}
      >
        <h3 style={{ margin: 0, fontSize: "15px" }}>Importar template</h3>
        <p style={{ margin: 0, fontSize: "13px", opacity: 0.7 }}>
          Selecione um arquivo <code>.json</code> exportado anteriormente.
        </p>
        <input
          data-testid="template-import-file-input"
          ref={fileRef}
          type="file"
          accept=".json"
          disabled={loading}
          style={{ fontSize: "13px" }}
        />
        {error && (
          <p style={{ margin: 0, fontSize: "13px", color: "var(--color-error, #f87171)" }}>
            {error}
          </p>
        )}
        <div style={{ display: "flex", gap: "0.5rem", justifyContent: "flex-end" }}>
          <button data-testid="template-import-cancel-btn" type="button" className="ghost-button" onClick={onClose} disabled={loading}>
            Cancelar
          </button>
          <button data-testid="template-import-submit-btn" type="button" onClick={() => void handleImport()} disabled={loading}>
            {loading ? "Importando..." : "Importar"}
          </button>
        </div>
      </div>
    </div>
  );
}
