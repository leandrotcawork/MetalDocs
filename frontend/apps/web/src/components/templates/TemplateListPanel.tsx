import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import {
  listTemplates,
  createTemplate,
  editPublished,
  cloneTemplate,
  deleteDraft,
  discardDraft,
  deprecateTemplate,
  exportTemplate,
} from "../../api/templates";
import type { TemplateListItemDTO, ImportResultDTO } from "../../api/templates";
import { buildTemplateEditorPath } from "../../routing/workspaceRoutes";
import { TemplateRowActions } from "./TemplateRowActions";
import { ImportTemplateDialog } from "./ImportTemplateDialog";

type LoadState = "idle" | "loading" | "ready" | "error";

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
        padding: "2px 8px",
        borderRadius: "9999px",
        fontSize: "11px",
        fontWeight: 600,
        color: style.color,
        background: style.bg,
      }}
    >
      {style.label}
    </span>
  );
}

type TemplateListPanelProps = {
  profileCode: string;
};

export function TemplateListPanel({ profileCode }: TemplateListPanelProps) {
  const navigate = useNavigate();
  const [loadState, setLoadState] = useState<LoadState>("idle");
  const [templates, setTemplates] = useState<TemplateListItemDTO[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [creating, setCreating] = useState(false);
  const [showImport, setShowImport] = useState(false);

  async function fetchTemplates() {
    setLoadState("loading");
    setError(null);
    try {
      const items = await listTemplates(profileCode);
      setTemplates(Array.isArray(items) ? items : []);
      setLoadState("ready");
    } catch (err) {
      setError(err instanceof Error ? err.message : "Erro ao carregar templates.");
      setLoadState("error");
    }
  }

  useEffect(() => {
    void fetchTemplates();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [profileCode]);

  async function handleCreate() {
    setCreating(true);
    try {
      const draft = await createTemplate(profileCode, "Novo template");
      navigate(buildTemplateEditorPath({ profileCode, templateKey: draft.templateKey }));
    } catch (err) {
      setError(err instanceof Error ? err.message : "Erro ao criar template.");
    } finally {
      setCreating(false);
    }
  }

  async function handleAction(template: TemplateListItemDTO, action: string) {
    try {
      if (action === "edit") {
        if (template.status === "published") {
          const draft = await editPublished(template.templateKey);
          navigate(buildTemplateEditorPath({ profileCode, templateKey: draft.templateKey }));
          return;
        }
        navigate(buildTemplateEditorPath({ profileCode, templateKey: template.templateKey }));
        return;
      }
      if (action === "clone") {
        const draft = await cloneTemplate(template.templateKey, `${template.name} (copia)`);
        navigate(buildTemplateEditorPath({ profileCode, templateKey: draft.templateKey }));
        return;
      }
      if (action === "delete") {
        await deleteDraft(template.templateKey);
        await fetchTemplates();
        return;
      }
      if (action === "discard") {
        await discardDraft(template.templateKey);
        await fetchTemplates();
        return;
      }
      if (action === "deprecate") {
        await deprecateTemplate(template.templateKey, template.version);
        await fetchTemplates();
        return;
      }
      if (action === "export") {
        const blob = await exportTemplate(template.templateKey, template.version);
        const url = URL.createObjectURL(blob);
        const a = document.createElement("a");
        a.href = url;
        a.download = `${template.templateKey}-v${template.version}.json`;
        a.click();
        URL.revokeObjectURL(url);
        return;
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : `Erro na acao "${action}".`);
    }
  }

  function handleImportSuccess(result: ImportResultDTO) {
    setShowImport(false);
    navigate(buildTemplateEditorPath({ profileCode, templateKey: result.templateKey }));
  }

  return (
    <div className="catalog-card" data-testid="template-list-panel">
      <div style={{ display: "flex", alignItems: "center", justifyContent: "space-between", gap: "0.75rem", flexWrap: "wrap", marginBottom: "0.75rem" }}>
        <h3 style={{ margin: 0 }}>Templates</h3>
        <span style={{ display: "inline-flex", gap: "0.5rem" }}>
          <button data-testid="template-import-open-btn" type="button" className="ghost-button" onClick={() => setShowImport(true)}>
            Importar
          </button>
          <button data-testid="template-create-btn" type="button" onClick={() => void handleCreate()} disabled={creating}>
            {creating ? "Criando..." : "Novo template"}
          </button>
        </span>
      </div>

      {loadState === "loading" && (
        <p className="catalog-muted">Carregando templates...</p>
      )}

      {loadState === "error" && (
        <p style={{ fontSize: "13px", color: "var(--color-error, #f87171)" }}>
          {error}
          <button type="button" className="ghost-button" style={{ marginLeft: "0.5rem" }} onClick={() => void fetchTemplates()}>
            Tentar novamente
          </button>
        </p>
      )}

      {loadState === "ready" && templates.length === 0 && (
        <p className="catalog-muted">Nenhum template cadastrado para este perfil.</p>
      )}

      {loadState === "ready" && templates.length > 0 && (
        <table style={{ width: "100%", borderCollapse: "collapse", fontSize: "13px" }}>
          <thead>
            <tr style={{ borderBottom: "1px solid rgba(255,255,255,0.1)" }}>
              <th style={{ textAlign: "left", padding: "6px 8px", fontWeight: 500, opacity: 0.7 }}>Nome</th>
              <th style={{ textAlign: "left", padding: "6px 8px", fontWeight: 500, opacity: 0.7 }}>Status</th>
              <th style={{ textAlign: "left", padding: "6px 8px", fontWeight: 500, opacity: 0.7 }}>Versao</th>
              <th style={{ textAlign: "right", padding: "6px 8px", fontWeight: 500, opacity: 0.7 }}>Acoes</th>
            </tr>
          </thead>
          <tbody>
            {templates.map((tpl) => (
              <tr data-testid={`template-row-${tpl.templateKey}`} key={tpl.templateKey} style={{ borderBottom: "1px solid rgba(255,255,255,0.06)" }}>
                <td style={{ padding: "6px 8px" }}>{tpl.name}</td>
                <td style={{ padding: "6px 8px" }}>
                  <StatusBadge status={tpl.status} />
                </td>
                <td style={{ padding: "6px 8px", opacity: 0.7 }}>
                  {tpl.status === "draft" ? "rascunho" : `v${tpl.version}`}
                </td>
                <td style={{ padding: "6px 8px", textAlign: "right" }}>
                  <TemplateRowActions template={tpl} onAction={(action) => void handleAction(tpl, action)} />
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      )}

      {error && loadState !== "error" && (
        <p style={{ marginTop: "0.5rem", fontSize: "13px", color: "var(--color-error, #f87171)" }}>
          {error}
        </p>
      )}

      {showImport && (
        <ImportTemplateDialog
          profileCode={profileCode}
          onClose={() => setShowImport(false)}
          onSuccess={handleImportSuccess}
        />
      )}
    </div>
  );
}
