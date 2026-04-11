import { useEffect, useRef, type CSSProperties } from "react";

export type SaveBeforeExportDialogProps = {
  open: boolean;
  isReleased: boolean;
  onSaveAndExport: () => void;
  onExportSaved: () => void;
  onCancel: () => void;
};

const overlayStyle: CSSProperties = {
  position: "fixed",
  inset: 0,
  background: "rgba(15, 15, 15, 0.55)",
  display: "flex",
  alignItems: "center",
  justifyContent: "center",
  zIndex: 9999,
};

const dialogStyle: CSSProperties = {
  background: "#ffffff",
  borderRadius: "8px",
  padding: "24px",
  width: "min(440px, 92vw)",
  boxShadow: "0 20px 40px rgba(0, 0, 0, 0.2)",
};

const actionsStyle: CSSProperties = {
  display: "flex",
  gap: "8px",
  justifyContent: "flex-end",
  marginTop: "20px",
};

const buttonStyle: CSSProperties = {
  padding: "8px 16px",
  borderRadius: "6px",
  border: "1px solid #cccccc",
  background: "#ffffff",
  cursor: "pointer",
};

const primaryButtonStyle: CSSProperties = {
  ...buttonStyle,
  background: "#6b1f2a",
  color: "#ffffff",
  border: "1px solid #6b1f2a",
};

export function SaveBeforeExportDialog({
  open,
  isReleased,
  onSaveAndExport,
  onExportSaved,
  onCancel,
}: SaveBeforeExportDialogProps) {
  const dialogRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) return;
    const el = dialogRef.current;
    if (!el) return;

    const focusable = el.querySelectorAll<HTMLElement>("button");
    const first = focusable[0];
    const last = focusable[focusable.length - 1];

    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        e.preventDefault();
        onCancel();
        return;
      }
      if (e.key !== "Tab") return;
      if (e.shiftKey) {
        if (document.activeElement === first) {
          e.preventDefault();
          last.focus();
        }
      } else {
        if (document.activeElement === last) {
          e.preventDefault();
          first.focus();
        }
      }
    };

    el.addEventListener("keydown", handleKeyDown);
    return () => el.removeEventListener("keydown", handleKeyDown);
  }, [open, onCancel]);

  if (!open) {
    return null;
  }

  const defaultActionLabel = isReleased ? "Exportar versão salva" : "Salvar e exportar";
  const defaultAction = isReleased ? onExportSaved : onSaveAndExport;
  const secondaryActionLabel = isReleased ? "Salvar e exportar" : "Exportar versão salva";
  const secondaryAction = isReleased ? onSaveAndExport : onExportSaved;

  return (
    <div ref={dialogRef} role="dialog" aria-modal="true" aria-labelledby="mddm-save-before-export-title" style={overlayStyle}>
      <div style={dialogStyle}>
        <h3 id="mddm-save-before-export-title" style={{ margin: 0, fontSize: "1.15rem" }}>
          Você tem alterações não salvas
        </h3>
        <p style={{ marginTop: "12px", color: "#555" }}>
          {isReleased
            ? "Este documento está publicado. Por padrão, a exportação usa a versão salva. Para incluir suas edições locais, salve primeiro."
            : "Para garantir rastreabilidade, a exportação sempre usa a última versão salva. Deseja salvar agora e exportar, ou exportar a versão salva atual?"}
        </p>
        <div style={actionsStyle}>
          <button type="button" style={buttonStyle} onClick={onCancel}>
            Cancelar
          </button>
          <button type="button" style={buttonStyle} onClick={secondaryAction}>
            {secondaryActionLabel}
          </button>
          <button type="button" style={primaryButtonStyle} onClick={defaultAction} autoFocus>
            {defaultActionLabel}
          </button>
        </div>
      </div>
    </div>
  );
}
