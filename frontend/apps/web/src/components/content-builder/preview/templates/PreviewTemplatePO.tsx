import type { PreviewTemplateProps } from "./PreviewTemplateGeneric";
import { PreviewSectionBlock } from "../PreviewSectionBlock";
import { PreviewFieldRenderer } from "../PreviewFieldRenderer";

const SECTION_ORDER = ["identificacao", "entradas_saidas", "processo", "indicadores"];
const SECTION_LABELS: Record<string, string> = {
  identificacao: "Identificacao",
  entradas_saidas: "Entradas e Saidas",
  processo: "Processo",
  indicadores: "Indicadores de Desempenho",
};

export function PreviewTemplatePO({ sections, content, activeSectionKey }: PreviewTemplateProps) {
  const orderedSections = SECTION_ORDER
    .map((key) => sections.find((s) => s.key === key))
    .filter(Boolean)
    .concat(sections.filter((s) => !SECTION_ORDER.includes(s.key)));

  return (
    <>
      {orderedSections.map((section, index) => {
        if (!section) return null;
        const sectionValue = (content[section.key] as Record<string, unknown>) ?? {};
        const isActive = activeSectionKey === section.key;
        const title = SECTION_LABELS[section.key] ?? section.title ?? section.key;

        return (
          <PreviewSectionBlock
            key={section.key}
            index={index}
            title={title}
            description={section.description}
            sectionKey={section.key}
          >
            <div className={`preview-section-fields ${isActive ? "is-active" : ""}`}>
              {(section.fields ?? []).map((field) => (
                <PreviewFieldRenderer
                  key={field.key}
                  field={field}
                  value={sectionValue[field.key]}
                />
              ))}
            </div>
          </PreviewSectionBlock>
        );
      })}
    </>
  );
}
