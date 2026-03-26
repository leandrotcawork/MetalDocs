import type { SchemaSection } from "../../contentSchemaTypes";
import { PreviewSectionBlock } from "../PreviewSectionBlock";
import { PreviewFieldRenderer } from "../PreviewFieldRenderer";

export type PreviewTemplateProps = {
  sections: SchemaSection[];
  content: Record<string, unknown>;
  activeSectionKey?: string | null;
};

export function PreviewTemplateGeneric({ sections, content, activeSectionKey }: PreviewTemplateProps) {
  return (
    <>
      {sections.map((section, index) => {
        const sectionValue = (content[section.key] as Record<string, unknown>) ?? {};
        const isActive = activeSectionKey === section.key;

        return (
          <PreviewSectionBlock
            key={section.key}
            index={index}
            title={section.title ?? section.key}
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
