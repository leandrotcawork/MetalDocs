import type { SchemaSection } from "../contentSchemaTypes";
import { PreviewDocumentPage } from "./PreviewDocumentPage";
import { PreviewSectionBlock } from "./PreviewSectionBlock";
import { PreviewFieldRenderer } from "./PreviewFieldRenderer";
import { getPreviewTemplate } from "./templates/templateRegistry";

type DocumentPreviewRendererProps = {
  sections: SchemaSection[];
  content: Record<string, unknown>;
  profileCode: string;
  documentCode: string;
  title: string;
  version: number | null;
  activeSectionKey?: string | null;
};

export function DocumentPreviewRenderer({
  sections,
  content,
  profileCode,
  documentCode,
  title,
  version,
  activeSectionKey,
}: DocumentPreviewRendererProps) {
  const Template = getPreviewTemplate(profileCode);

  if (Template) {
    return (
      <PreviewDocumentPage
        profileCode={profileCode}
        documentCode={documentCode}
        title={title}
        version={version}
      >
        <Template
          sections={sections}
          content={content}
          activeSectionKey={activeSectionKey}
        />
      </PreviewDocumentPage>
    );
  }

  return (
    <PreviewDocumentPage
      profileCode={profileCode}
      documentCode={documentCode}
      title={title}
      version={version}
    >
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
    </PreviewDocumentPage>
  );
}
