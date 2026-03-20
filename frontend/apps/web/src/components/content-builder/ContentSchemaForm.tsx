import { useEffect, useState } from "react";
import type { DocumentProfileSchemaItem } from "../../lib.types";
import type { SchemaSection } from "./contentSchemaTypes";
import { ContentSectionAccordion } from "./ContentSectionAccordion";

type ContentSchemaFormProps = {
  schema: DocumentProfileSchemaItem | null;
  value: Record<string, unknown>;
  onChange: (next: Record<string, unknown>) => void;
};

export function ContentSchemaForm(props: ContentSchemaFormProps) {
  const schema = props.schema?.contentSchema as { sections?: SchemaSection[] } | undefined;
  const sections = Array.isArray(schema?.sections) ? schema?.sections : [];
  const [expandedSections, setExpandedSections] = useState<Record<string, boolean>>({});

  useEffect(() => {
    if (sections.length === 0) return;
    setExpandedSections((prev) => {
      let changed = false;
      const next = { ...prev };
      sections.forEach((section) => {
        if (!(section.key in next)) {
          next[section.key] = true;
          changed = true;
        }
      });
      return changed ? next : prev;
    });
  }, [sections]);

  if (!props.schema) {
    return (
      <div className="content-builder-section">
        <div className="content-builder-section-head">
          <strong>Conteudo estruturado</strong>
          <small>Schema nao disponivel para este profile.</small>
        </div>
        <div className="content-builder-empty">Sem schema ativo.</div>
      </div>
    );
  }

  return (
    <>
      {sections.map((section) => (
        <ContentSectionAccordion
          key={section.key}
          section={section}
          value={props.value}
          onChange={props.onChange}
          expanded={expandedSections[section.key] ?? true}
          onToggle={() =>
            setExpandedSections((prev) => ({
              ...prev,
              [section.key]: !(prev[section.key] ?? true),
            }))
          }
        />
      ))}
    </>
  );
}
