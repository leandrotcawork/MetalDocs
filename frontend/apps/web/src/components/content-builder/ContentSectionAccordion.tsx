import type { SchemaSection } from "./contentSchemaTypes";
import { SchemaFieldRenderer } from "./SchemaFieldRenderer";
import type { SchemaField } from "./contentSchemaTypes";

type ContentSectionAccordionProps = {
  section: SchemaSection;
  value: Record<string, unknown>;
  onChange: (next: Record<string, unknown>) => void;
  expanded: boolean;
  onToggle: () => void;
  anchorId: string;
  isActive?: boolean;
};

export function ContentSectionAccordion(props: ContentSectionAccordionProps) {
  const { section } = props;
  const sectionKey = section.key;
  const sectionValue = (props.value[sectionKey] as Record<string, unknown>) ?? {};
  const fields = section.fields ?? [];
  const sectionProgress = sectionCompletion(fields, sectionValue);

  function updateSectionField(fieldKey: string, nextValue: unknown) {
    const nextSection = { ...sectionValue, [fieldKey]: nextValue };
    props.onChange({ ...props.value, [sectionKey]: nextSection });
  }

  return (
    <div
      id={props.anchorId}
      data-section-key={sectionKey}
      className={`content-builder-section ${props.expanded ? "is-expanded" : "is-collapsed"} ${props.isActive ? "is-focused" : ""}`}
    >
      <div className="content-builder-section-head">
        <button type="button" className="content-builder-section-toggle" onClick={props.onToggle}>
          <div className="content-builder-section-title">
            <span className="content-builder-section-icon" aria-hidden="true">
              <svg width="14" height="14" viewBox="0 0 14 14" fill="none" stroke="currentColor" strokeWidth="1.4" strokeLinecap="round">
                <path d="M2 2h10v10H2z" strokeLinejoin="round" />
                <path d="M5 5.5h4M5 8h4M5 10.5h2" />
              </svg>
            </span>
            <div className="content-builder-section-copy">
              <strong>{section.title ?? section.key}</strong>
              {section.description && <small>{section.description}</small>}
            </div>
          </div>
          <div className="content-builder-section-progress">
            <span className="content-builder-progress-dots">
              {Array.from({ length: 3 }).map((_, index) => (
                <span
                  key={`${sectionKey}-dot-${index}`}
                  className={`content-builder-progress-dot ${sectionProgress.progressDots > index ? "is-filled" : ""}`}
                />
              ))}
            </span>
            <span>{sectionProgress.progressLabel}</span>
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

function sectionCompletion(fields: SchemaField[], sectionValue: Record<string, unknown>) {
  const total = fields.length;
  if (total === 0) {
    return { progressDots: 0, progressLabel: "0%" };
  }
  const completed = fields.reduce((acc, field) => {
    const value = sectionValue[field.key];
    return acc + (isFieldComplete(field, value) ? 1 : 0);
  }, 0);
  const ratio = Math.round((completed / total) * 100);
  return {
    progressDots: ratio === 0 ? 0 : ratio < 50 ? 1 : ratio < 90 ? 2 : 3,
    progressLabel: `${ratio}%`,
  };
}

function isFieldComplete(field: SchemaField, value: unknown) {
  if (field.type === "table") {
    return Array.isArray(value) && value.length > 0;
  }
  if (field.type === "array") {
    return Array.isArray(value) && value.some((item) => String(item ?? "").trim() !== "");
  }
  if (field.type === "checklist") {
    return Array.isArray(value) && value.some((item) => typeof item === "string" ? item.trim() !== "" : Boolean((item as { label?: string }).label));
  }
  if (field.type === "number") {
    return value !== null && value !== undefined && value !== "";
  }
  return String(value ?? "").trim() !== "";
}
