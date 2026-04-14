import type { TemplateListItemDTO } from "../../api/templates";

type TemplateRowActionsProps = {
  template: TemplateListItemDTO;
  onAction: (action: string) => void;
};

export function TemplateRowActions({ template, onAction }: TemplateRowActionsProps) {
  const { status } = template;

  return (
    <span style={{ display: "inline-flex", gap: "0.4rem", flexWrap: "wrap" }}>
      {status === "draft" && (
        <>
          <button data-testid={`template-action-edit-${template.templateKey}`} type="button" className="ghost-button" onClick={() => onAction("edit")}>Editar</button>
          <button data-testid={`template-action-clone-${template.templateKey}`} type="button" className="ghost-button" onClick={() => onAction("clone")}>Clonar</button>
          <button data-testid={`template-action-delete-${template.templateKey}`} type="button" className="ghost-button" onClick={() => onAction("delete")}>Excluir</button>
          <button data-testid={`template-action-discard-${template.templateKey}`} type="button" className="ghost-button" onClick={() => onAction("discard")}>Descartar</button>
        </>
      )}
      {status === "published" && (
        <>
          <button data-testid={`template-action-edit-${template.templateKey}`} type="button" className="ghost-button" onClick={() => onAction("edit")}>Editar</button>
          <button data-testid={`template-action-clone-${template.templateKey}`} type="button" className="ghost-button" onClick={() => onAction("clone")}>Clonar</button>
          <button data-testid={`template-action-deprecate-${template.templateKey}`} type="button" className="ghost-button" onClick={() => onAction("deprecate")}>Deprecar</button>
          <button data-testid={`template-action-export-${template.templateKey}`} type="button" className="ghost-button" onClick={() => onAction("export")}>Exportar</button>
        </>
      )}
      {status === "deprecated" && (
        <>
          <button data-testid={`template-action-clone-${template.templateKey}`} type="button" className="ghost-button" onClick={() => onAction("clone")}>Clonar</button>
          <button data-testid={`template-action-export-${template.templateKey}`} type="button" className="ghost-button" onClick={() => onAction("export")}>Exportar</button>
        </>
      )}
    </span>
  );
}
