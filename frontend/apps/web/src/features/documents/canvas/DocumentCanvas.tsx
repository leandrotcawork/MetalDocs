import type { CanvasTemplatePage } from "./templateTypes";
import { TemplateNodeRenderer } from "./TemplateNodeRenderer";
import styles from "./DocumentCanvas.module.css";

type DocumentCanvasProps = {
  template: CanvasTemplatePage;
  values: Record<string, unknown>;
  onChange: (next: Record<string, unknown>) => void;
  readOnly?: boolean;
};

export function DocumentCanvas({ template, values, onChange, readOnly }: DocumentCanvasProps) {
  return (
    <div className={styles.canvasPage}>
      <TemplateNodeRenderer node={template} values={values} onChange={onChange} readOnly={readOnly} />
    </div>
  );
}
