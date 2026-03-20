import type { SchemaSection } from "./contentSchemaTypes";
import { SchemaFieldRenderer } from "./SchemaFieldRenderer";
import { sectionProgress } from "./contentBuilderUtils";

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
  const sectionProgressState = sectionProgress(fields, sectionValue);

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
                  className={`content-builder-progress-dot ${sectionProgressState.progressDots > index ? "is-filled" : ""}`}
                />
              ))}
            </span>
            <span>{sectionProgressState.progressLabel}</span>
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
