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
          <button type="button" className="ghost-button" onClick={() => onAction("edit")}>Editar</button>
          <button type="button" className="ghost-button" onClick={() => onAction("clone")}>Clonar</button>
          <button type="button" className="ghost-button" onClick={() => onAction("delete")}>Excluir</button>
          <button type="button" className="ghost-button" onClick={() => onAction("discard")}>Descartar</button>
        </>
      )}
      {status === "published" && (
        <>
          <button type="button" className="ghost-button" onClick={() => onAction("edit")}>Editar</button>
          <button type="button" className="ghost-button" onClick={() => onAction("clone")}>Clonar</button>
          <button type="button" className="ghost-button" onClick={() => onAction("deprecate")}>Deprecar</button>
          <button type="button" className="ghost-button" onClick={() => onAction("export")}>Exportar</button>
        </>
      )}
      {status === "deprecated" && (
        <>
          <button type="button" className="ghost-button" onClick={() => onAction("clone")}>Clonar</button>
          <button type="button" className="ghost-button" onClick={() => onAction("export")}>Exportar</button>
        </>
      )}
    </span>
  );
}
