import type { ReactNode } from "react";
import { useEffect, useMemo, useState } from "react";
import type { DocumentProfileSchemaItem } from "../../../lib.types";
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

type DynamicEditorProps = {
  schema: DocumentProfileSchemaItem | null;
  value: Record<string, unknown>;
  activeSectionKey?: string | null;
  onChange: (next: Record<string, unknown>) => void;
};

export function DynamicEditor({ schema, value, activeSectionKey, onChange }: DynamicEditorProps) {
  const runtimeSchema = useMemo(() => toRuntimeDocumentSchema(schema?.contentSchema), [schema?.contentSchema]);
  const [expandedSections, setExpandedSections] = useState<Record<string, boolean>>({});

  useEffect(() => {
    setExpandedSections((current) => {
      const next = { ...current };
      let changed = false;

      for (const section of runtimeSchema.sections) {
        if (!(section.key in next)) {
          next[section.key] = true;
          changed = true;
        }
      }

      return changed ? next : current;
    });
  }, [runtimeSchema.sections]);

  if (!schema) {
    return (
      <div className={styles.emptyState}>
        <strong>Sem schema ativo.</strong>
        <span>Selecione um profile com estrutura de conteudo para editar campos dinamicos.</span>
      </div>
    );
  }

  if (runtimeSchema.sections.length === 0) {
    return (
      <div className={styles.emptyState}>
        <strong>Schema sem secoes.</strong>
        <span>O profile atual nao possui campos runtime publicados.</span>
      </div>
    );
  }

  return (
    <div className={styles.editorRoot}>
      {runtimeSchema.sections.map((section, index) => {
        const sectionValue = getSectionValue(value, section.key);
        const isExpanded = expandedSections[section.key] ?? true;
        const isActive = activeSectionKey === section.key;

        return (
          <section
            key={section.key}
            id={`content-section-${section.key}`}
            data-section-key={section.key}
            className={`${styles.section} ${isActive ? styles.sectionActive : ""}`}
          >
            <button
              type="button"
              className={styles.sectionHeader}
              onClick={() => {
                setExpandedSections((current) => ({
                  ...current,
                  [section.key]: !isExpanded,
                }));
              }}
            >
              <div className={styles.sectionHeading}>
                <h2 className={styles.sectionTitle}>
                  {index + 1}. {section.title ?? section.key}
                </h2>
                {section.description && <div className={styles.sectionDescription}>{section.description}</div>}
              </div>
              <span className={styles.sectionBadge}>{isExpanded ? "Aberta" : "Fechada"}</span>
            </button>

            {isExpanded && (
              <div className={styles.sectionBody}>
                {section.fields.map((field) =>
                  renderRuntimeField(field, sectionValue[field.key], (nextValue) => {
                    onChange({
                      ...value,
                      [section.key]: {
                        ...sectionValue,
                        [field.key]: nextValue,
                      },
                    });
                  }),
                )}
              </div>
            )}
          </section>
        );
      })}
    </div>
  );

  function renderRuntimeField(field: RuntimeField, fieldValue: unknown, onFieldChange?: (next: unknown) => void): ReactNode {
    switch (field.kind) {
      case "table":
        return <TableField field={field as RuntimeTableField} value={fieldValue} mode="edit" onChange={onFieldChange} />;
      case "repeat":
        return (
          <RepeatField
            field={field as RuntimeRepeatField}
            value={fieldValue}
            mode="edit"
            onChange={onFieldChange}
            renderField={renderRuntimeField}
          />
        );
      case "rich":
        return <RichField field={field as RuntimeRichField} value={fieldValue} mode="edit" onChange={onFieldChange} />;
      case "scalar":
      default:
        return <ScalarField field={field as RuntimeScalarField} value={fieldValue} mode="edit" onChange={onFieldChange} />;
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
