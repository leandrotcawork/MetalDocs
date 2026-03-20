import type { SchemaSection } from "./contentSchemaTypes";
import { SchemaFieldRenderer } from "./SchemaFieldRenderer";

type ContentSectionAccordionProps = {
  section: SchemaSection;
  value: Record<string, unknown>;
  onChange: (next: Record<string, unknown>) => void;
  expanded: boolean;
  onToggle: () => void;
};

export function ContentSectionAccordion(props: ContentSectionAccordionProps) {
  const { section } = props;
  const sectionKey = section.key;
  const sectionValue = (props.value[sectionKey] as Record<string, unknown>) ?? {};

  function updateSectionField(fieldKey: string, nextValue: unknown) {
    const nextSection = { ...sectionValue, [fieldKey]: nextValue };
    props.onChange({ ...props.value, [sectionKey]: nextSection });
  }

  return (
    <div className={`content-builder-section ${props.expanded ? "is-expanded" : "is-collapsed"}`}>
      <div className="content-builder-section-head">
        <button type="button" className="content-builder-section-toggle" onClick={props.onToggle}>
          <div className="content-builder-section-title">
            <strong>{section.title ?? section.key}</strong>
            {section.description && <small>{section.description}</small>}
          </div>
          <span className="content-builder-section-chevron" aria-hidden="true">
            {props.expanded ? "-" : "+"}
          </span>
        </button>
      </div>
      <div className="content-builder-section-body" aria-hidden={!props.expanded}>
        {(section.fields ?? []).map((field) => (
          <SchemaFieldRenderer
            key={`${sectionKey}-${field.key}`}
            field={field}
            value={sectionValue[field.key]}
            onChange={(next) => updateSectionField(field.key, next)}
          />
        ))}
      </div>
    </div>
  );
}
