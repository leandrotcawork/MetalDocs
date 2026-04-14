import { useRef, useState } from "react";

// ---------------------------------------------------------------------------
// Status badge — same color scheme as TemplateListPanel
// ---------------------------------------------------------------------------

const STATUS_BADGE: Record<string, { label: string; color: string; bg: string }> = {
  draft: { label: "Rascunho", color: "#92400e", bg: "#fef3c7" },
  published: { label: "Publicado", color: "#065f46", bg: "#d1fae5" },
  deprecated: { label: "Depreciado", color: "#6b7280", bg: "#f3f4f6" },
};

function StatusBadge({ status }: { status: string }) {
  const style = STATUS_BADGE[status] ?? { label: status, color: "#6b7280", bg: "#f3f4f6" };
  return (
    <span
      style={{
        display: "inline-block",
        padding: "2px 10px",
        borderRadius: "9999px",
        fontSize: "11px",
        fontWeight: 600,
        color: style.color,
        background: style.bg,
        flexShrink: 0,
      }}
    >
      {style.label}
    </span>
  );
}

// ---------------------------------------------------------------------------
// Props
// ---------------------------------------------------------------------------

export interface MetadataBarProps {
  templateName: string;
  profileCode: string;
  status: string;
  lockVersion: number;
  hasStrippedFields: boolean;
  isDirty: boolean;
  onSave: () => void;
  onPublish: () => void;
  onDiscard: () => void;
  onNameChange: (name: string) => void;
}

// ---------------------------------------------------------------------------
// MetadataBar
// ---------------------------------------------------------------------------

export function MetadataBar({
  templateName,
  profileCode,
  status,
  lockVersion,
  hasStrippedFields,
  isDirty,
  onSave,
  onPublish,
  onDiscard,
  onNameChange,
}: MetadataBarProps) {
  const [editingName, setEditingName] = useState(false);
  const [nameValue, setNameValue] = useState(templateName);
  const inputRef = useRef<HTMLInputElement>(null);

  // Keep local name in sync when templateName prop changes (e.g. after save)
  if (!editingName && nameValue !== templateName) {
    setNameValue(templateName);
  }

  function handleNameClick() {
    setEditingName(true);
    setTimeout(() => inputRef.current?.select(), 0);
  }

  function handleNameBlur() {
    setEditingName(false);
    if (nameValue !== templateName) {
      onNameChange(nameValue);
    }
  }

  function handleNameKeyDown(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === "Enter") {
      inputRef.current?.blur();
    }
    if (e.key === "Escape") {
      setNameValue(templateName);
      setEditingName(false);
    }
  }

  function handleDiscardClick() {
    const confirmed = window.confirm(
      "Tem certeza que deseja descartar este rascunho? Esta acao e irreversivel."
    );
    if (confirmed) {
      onDiscard();
    }
  }

  return (
    <div
      data-testid="metadata-bar"
      style={{
        display: "flex",
        alignItems: "center",
        gap: "0.75rem",
        padding: "0.5rem 1rem",
        borderBottom: "1px solid rgba(255,255,255,0.1)",
        background: "var(--color-surface, #1a1a2e)",
        flexShrink: 0,
        flexWrap: "wrap",
        minHeight: "48px",
      }}
    >
      {/* Editable template name */}
      <div style={{ display: "flex", alignItems: "center", gap: "0.375rem", flex: "1 1 200px", minWidth: 0 }}>
        {editingName ? (
          <input
            ref={inputRef}
            value={nameValue}
            onChange={(e) => setNameValue(e.target.value)}
            onBlur={handleNameBlur}
            onKeyDown={handleNameKeyDown}
            style={{
              fontSize: "14px",
              fontWeight: 600,
              background: "rgba(255,255,255,0.07)",
              border: "1px solid rgba(255,255,255,0.25)",
              borderRadius: "4px",
              padding: "2px 6px",
              color: "inherit",
              minWidth: 0,
              flex: 1,
            }}
            autoFocus
          />
        ) : (
          <button
            type="button"
            onClick={handleNameClick}
            title="Clique para renomear"
            style={{
              background: "none",
              border: "none",
              padding: "2px 4px",
              cursor: "text",
              fontSize: "14px",
              fontWeight: 600,
              color: "inherit",
              borderRadius: "4px",
              textAlign: "left",
              maxWidth: "300px",
              overflow: "hidden",
              textOverflow: "ellipsis",
              whiteSpace: "nowrap",
            }}
          >
            {nameValue}
            {isDirty && (
              <span
                title="Alteracoes nao salvas"
                style={{ marginLeft: "4px", fontSize: "10px", color: "#f59e0b", verticalAlign: "super" }}
              >
                *
              </span>
            )}
          </button>
        )}
      </div>

      {/* Profile badge */}
      <span
        style={{
          fontSize: "11px",
          color: "rgba(255,255,255,0.5)",
          fontFamily: "monospace",
          flexShrink: 0,
        }}
      >
        {profileCode}
      </span>

      {/* Status badge */}
      <StatusBadge status={status} />

      {/* Lock version indicator */}
      <span
        style={{
          fontSize: "11px",
          color: "rgba(255,255,255,0.4)",
          flexShrink: 0,
        }}
      >
        Edicao #{lockVersion}
      </span>

      {/* Stripped fields warning */}
      {hasStrippedFields && (
        <span
          title="Alguns campos foram removidos durante a importacao por nao corresponderem ao schema atual."
          style={{
            fontSize: "11px",
            color: "#f59e0b",
            fontWeight: 600,
            flexShrink: 0,
            cursor: "help",
          }}
        >
          ⚠ Campos removidos
        </span>
      )}

      {/* Spacer */}
      <div style={{ flex: 1 }} />

      {/* Action buttons */}
      <div style={{ display: "inline-flex", gap: "0.5rem", flexShrink: 0 }}>
        <button
          type="button"
          className="ghost-button"
          onClick={handleDiscardClick}
          title="Descartar rascunho e voltar para a versao publicada"
          style={{ fontSize: "13px" }}
        >
          Descartar rascunho
        </button>
        <button
          type="button"
          className="ghost-button"
          onClick={onSave}
          title={isDirty ? "Salvar alteracoes" : "Nenhuma alteracao pendente"}
          style={{
            fontSize: "13px",
            opacity: isDirty ? 1 : 0.6,
          }}
        >
          {isDirty ? "Salvar rascunho *" : "Salvar rascunho"}
        </button>
        <button
          type="button"
          onClick={onPublish}
          title="Publicar template (valida antes de enviar)"
          style={{ fontSize: "13px" }}
        >
          Publicar
        </button>
      </div>
    </div>
  );
}
