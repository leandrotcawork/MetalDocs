import type { ReactNode } from "react";
import type { DocumentProfileSchemaItem } from "../../../lib.types";
import { PreviewDocumentPage } from "../../../components/content-builder/preview/PreviewDocumentPage";
import styles from "./DynamicEditor.module.css";
import {
  toRuntimeDocumentSchema,
  type RuntimeField,
  type RuntimeRepeatField,
  type RuntimeRichField,
  type RuntimeScalarField,
  type RuntimeTableField,
} from "./schemaRuntimeTypes";
import { ScalarField } from "./fields/ScalarField";
import { TableField } from "./fields/TableField";
import { RepeatField } from "./fields/RepeatField";
import { RichField } from "./fields/RichField";

type DynamicPreviewProps = {
  schema: DocumentProfileSchemaItem | null;
  content: Record<string, unknown>;
  profileCode: string;
  documentCode: string;
  title: string;
  documentStatus: string;
  version: number | null;
  activeSectionKey?: string | null;
};

export function DynamicPreview({
  schema,
  content,
  profileCode,
  documentCode,
  title,
  documentStatus,
  version,
  activeSectionKey,
}: DynamicPreviewProps) {
  const runtimeSchema = toRuntimeDocumentSchema(schema?.contentSchema);

  return (
    <PreviewDocumentPage profileCode={profileCode} documentCode={documentCode} title={title} documentStatus={documentStatus} version={version}>
      <div className={styles.editorRoot}>
        {runtimeSchema.sections.map((section, index) => {
          const sectionValue = getSectionValue(content, section.key);
          const isActive = activeSectionKey === section.key;

          return (
            <section
              key={section.key}
              data-preview-section={section.key}
              className={`${styles.section} ${styles.previewSectionMode} ${isActive ? styles.sectionActive : ""}`}
            >
              <div className={styles.sectionHeader}>
                <div className={styles.sectionHeading}>
                  <h2 className={styles.sectionTitle}>
                    {index + 1}. {section.title ?? section.key}
                  </h2>
                  {section.description && <div className={styles.sectionDescription}>{section.description}</div>}
                </div>
                <span className={styles.sectionBadge}>Preview</span>
              </div>
              <div className={styles.sectionBody}>
                {section.fields.map((field) => renderRuntimeField(field, sectionValue[field.key]))}
              </div>
            </section>
          );
        })}
      </div>
    </PreviewDocumentPage>
  );

  function renderRuntimeField(field: RuntimeField, fieldValue: unknown): ReactNode {
    switch (field.kind) {
      case "table":
        return <TableField field={field as RuntimeTableField} value={fieldValue} mode="preview" />;
      case "repeat":
        return (
          <RepeatField
            field={field as RuntimeRepeatField}
            value={fieldValue}
            mode="preview"
            renderField={renderRuntimeField}
          />
        );
      case "rich":
        return <RichField field={field as RuntimeRichField} value={fieldValue} mode="preview" />;
      case "scalar":
      default:
        return <ScalarField field={field as RuntimeScalarField} value={fieldValue} mode="preview" />;
    }
  }
}

function getSectionValue(value: Record<string, unknown>, sectionKey: string) {
  const sectionValue = value[sectionKey];
  if (sectionValue && typeof sectionValue === "object" && !Array.isArray(sectionValue)) {
    return sectionValue as Record<string, unknown>;
  }
  return {};
}
