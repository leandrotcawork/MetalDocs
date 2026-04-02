import type { CanvasTemplateNode } from "./templateTypes";
import type { RuntimeDocumentSchema } from "../runtime/schemaRuntimeTypes";
import styles from "./DocumentCanvas.module.css";
import { FieldSlot } from "./slots/FieldSlot";
import { RichSlot } from "./slots/RichSlot";

type TemplateNodeRendererProps = {
  node: CanvasTemplateNode;
  schema: RuntimeDocumentSchema | null;
  values: Record<string, unknown>;
  onChange: (next: Record<string, unknown>) => void;
  readOnly?: boolean;
};

export function TemplateNodeRenderer({ node, schema, values, onChange, readOnly }: TemplateNodeRendererProps) {
  switch (node.type) {
    case "page":
      return (
        <div className={styles.page}>
          {node.children.map((child) => (
            <TemplateNodeRenderer key={child.id} node={child} schema={schema} values={values} onChange={onChange} readOnly={readOnly} />
          ))}
        </div>
      );
    case "section-frame":
      return (
        <section className={styles.sectionFrame}>
          {node.title ? <h2 className={styles.sectionTitle}>{node.title}</h2> : null}
          <div className={styles.sectionBody}>
            {node.children.map((child) => (
              <TemplateNodeRenderer key={child.id} node={child} schema={schema} values={values} onChange={onChange} readOnly={readOnly} />
            ))}
          </div>
        </section>
      );
    case "label":
      return <div className={styles.label}>{node.text}</div>;
    case "field-slot":
      return <FieldSlot path={node.path} schema={schema} values={values} onChange={onChange} readOnly={readOnly} />;
    case "rich-slot":
      return <RichSlot path={node.path} schema={schema} values={values} onChange={onChange} readOnly={readOnly} />;
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
