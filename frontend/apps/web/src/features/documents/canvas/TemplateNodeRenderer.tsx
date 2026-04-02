import type { CanvasTemplateNode } from "./templateTypes";
import styles from "./DocumentCanvas.module.css";

type TemplateNodeRendererProps = {
  node: CanvasTemplateNode;
  values: Record<string, unknown>;
  onChange: (next: Record<string, unknown>) => void;
  readOnly?: boolean;
};

export function TemplateNodeRenderer({ node, values, onChange, readOnly }: TemplateNodeRendererProps) {
  switch (node.type) {
    case "page":
      return (
        <div className={styles.page}>
          {node.children.map((child) => (
            <TemplateNodeRenderer key={child.id} node={child} values={values} onChange={onChange} readOnly={readOnly} />
          ))}
        </div>
      );
    case "section-frame":
      return (
        <section className={styles.sectionFrame}>
          {node.title ? <h2 className={styles.sectionTitle}>{node.title}</h2> : null}
          <div className={styles.sectionBody}>
            {node.children.map((child) => (
              <TemplateNodeRenderer key={child.id} node={child} values={values} onChange={onChange} readOnly={readOnly} />
            ))}
          </div>
        </section>
      );
    case "label":
      return <div className={styles.label}>{node.text}</div>;
    case "field-slot":
      return (
        <div className={styles.fieldSlot}>
          <input
            className={styles.input}
            type="text"
            value={normalizeTextValue(readValue(values, node.path))}
            readOnly={readOnly}
            onChange={(event) => {
              if (readOnly) return;
              onChange(writeValue(values, node.path, event.target.value));
            }}
          />
        </div>
      );
    case "rich-slot":
      return (
        <div className={styles.richSlot}>
          <textarea
            className={styles.textarea}
            value={normalizeTextValue(readValue(values, node.path))}
            readOnly={readOnly}
            onChange={(event) => {
              if (readOnly) return;
              onChange(writeValue(values, node.path, event.target.value));
            }}
            rows={8}
          />
        </div>
      );
    case "table-slot":
      return (
        <div className={styles.unsupportedSlot}>
          <strong>Slot de tabela nao suportado neste piloto.</strong>
          <span>{node.path}</span>
        </div>
      );
    case "repeat-slot":
      return (
        <div className={styles.unsupportedSlot}>
          <strong>Slot repetitivo nao suportado neste piloto.</strong>
          <span>{node.path}</span>
        </div>
      );
    default:
      return null;
  }
}

function readValue(values: Record<string, unknown>, path: string): unknown {
  return path.split(".").reduce<unknown>((current, segment) => {
    if (!current || typeof current !== "object" || Array.isArray(current)) {
      return undefined;
    }
    return (current as Record<string, unknown>)[segment];
  }, values);
}

function writeValue(values: Record<string, unknown>, path: string, nextValue: unknown): Record<string, unknown> {
  const segments = path.split(".").map((segment) => segment.trim()).filter(Boolean);
  if (segments.length === 0) {
    return values;
  }

  const next = { ...values };
  let cursor: Record<string, unknown> = next;

  for (let index = 0; index < segments.length - 1; index += 1) {
    const segment = segments[index];
    const current = cursor[segment];
    const nextCursor = current && typeof current === "object" && !Array.isArray(current) ? { ...(current as Record<string, unknown>) } : {};
    cursor[segment] = nextCursor;
    cursor = nextCursor;
  }

  cursor[segments[segments.length - 1]] = nextValue;
  return next;
}

function normalizeTextValue(value: unknown): string {
  if (typeof value === "string") {
    return value;
  }
  if (value === null || value === undefined) {
    return "";
  }
  return String(value);
}
