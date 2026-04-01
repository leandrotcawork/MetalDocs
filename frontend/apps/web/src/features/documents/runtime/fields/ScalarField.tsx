import styles from "../DynamicEditor.module.css";
import type { RuntimeScalarField } from "../schemaRuntimeTypes";

type ScalarFieldProps = {
  field: RuntimeScalarField;
  value: unknown;
  onChange?: (next: unknown) => void;
};

export function ScalarField({ field, value, onChange }: ScalarFieldProps) {
  const label = field.label ?? field.key;

  return (
    <div className={styles.field}>
      <div className={styles.fieldLabel}>
        <span>{label}</span>
        {field.required && <span className={styles.requiredMark}>*</span>}
      </div>
      {field.description && <div className={styles.fieldDescription}>{field.description}</div>}
      {field.input === "textarea" ? (
        <textarea
          className={`${styles.control} ${styles.textarea}`}
          value={String(value ?? "")}
          onChange={(event) => onChange?.(event.target.value)}
        />
      ) : field.input === "select" ? (
        <select
          className={`${styles.control} ${styles.select}`}
          value={String(value ?? "")}
          onChange={(event) => onChange?.(event.target.value)}
        >
          <option value="">Selecione</option>
          {field.options.map((option) => (
            <option key={option} value={option}>
              {option}
            </option>
          ))}
        </select>
      ) : field.input === "number" ? (
        <input
          className={styles.control}
          type="number"
          value={value === null || value === undefined || value === "" ? "" : String(value)}
          onChange={(event) => {
            const nextValue = event.target.value;
            onChange?.(nextValue === "" ? "" : Number(nextValue));
          }}
        />
      ) : field.input === "checkbox" ? (
        <label className={`${styles.control} ${styles.checkboxRow}`}>
          <input
            className={styles.checkboxControl}
            type="checkbox"
            checked={Boolean(value)}
            onChange={(event) => onChange?.(event.target.checked)}
          />
          <span>{Boolean(value) ? "Sim" : "Nao"}</span>
        </label>
      ) : (
        <input
          className={styles.control}
          type="text"
          value={String(value ?? "")}
          onChange={(event) => onChange?.(event.target.value)}
        />
      )}
    </div>
  );
}

