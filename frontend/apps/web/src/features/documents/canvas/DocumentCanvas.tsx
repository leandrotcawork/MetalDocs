import type { CanvasTemplatePage } from "./templateTypes";
import type { RuntimeDocumentSchema } from "../runtime/schemaRuntimeTypes";
import { TemplateNodeRenderer } from "./TemplateNodeRenderer";
import styles from "./DocumentCanvas.module.css";

type DocumentCanvasProps = {
  template: CanvasTemplatePage;
  schema: RuntimeDocumentSchema | null;
  values: Record<string, unknown>;
  onChange: (next: Record<string, unknown>) => void;
  readOnly?: boolean;
};

export function DocumentCanvas({ template, schema, values, onChange, readOnly }: DocumentCanvasProps) {
  return (
    <div className={styles.canvasPage}>
      <TemplateNodeRenderer node={template} schema={schema} values={values} onChange={onChange} readOnly={readOnly} />
    </div>
  );
}
