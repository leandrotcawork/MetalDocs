import type { CSSProperties } from "react";
import styles from "../DynamicEditor.module.css";
import type { RuntimeTableField } from "../schemaRuntimeTypes";
import { ScalarField } from "./ScalarField";

type RuntimeMode = "edit" | "preview";

type TableFieldProps = {
  field: RuntimeTableField;
  value: unknown;
  mode: RuntimeMode;
  onChange?: (next: unknown) => void;
};

export function TableField({ field, value, mode, onChange }: TableFieldProps) {
  const rows = normalizeRows(value, field.columns);
  const columnCount = Math.max(field.columns.length, 1);

  return (
    <div className={styles.field}>
      <div className={styles.fieldLabel}>
        <span>{field.label ?? field.key}</span>
        {field.required && <span className={styles.requiredMark}>*</span>}
      </div>
      {field.description && <div className={styles.fieldDescription}>{field.description}</div>}
      <div
        className={styles.tableShell}
        style={{ "--runtime-table-columns": String(columnCount) } as CSSProperties}
      >
        <div
          className={styles.tableHead}
          style={{ gridTemplateColumns: `repeat(${columnCount}, minmax(0, 1fr))${mode === "edit" ? " auto" : ""}` }}
        >
          {field.columns.map((column) => (
            <div key={column.key} className={styles.tableHeadCell}>
              {column.label ?? column.key}
            </div>
          ))}
          {mode === "edit" && <div className={styles.tableHeadCell}>Acoes</div>}
        </div>
        <div className={styles.tableRows}>
          {rows.length === 0 ? (
            <div className={styles.tableEmpty}>{mode === "edit" ? "Nenhuma linha adicionada." : "—"}</div>
          ) : (
            rows.map((row, rowIndex) => (
              <div key={`${field.key}-${rowIndex}`} className={styles.tableRow}>
                {field.columns.map((column) => (
                  <div key={column.key} className={styles.tableCell}>
                    <ScalarField
                      field={column}
                      value={row[column.key]}
                      mode={mode}
                      onChange={
                        mode === "edit"
                          ? (nextValue) => {
                              const nextRows = rows.slice();
                              nextRows[rowIndex] = { ...row, [column.key]: nextValue };
                              onChange?.(nextRows);
                            }
                          : undefined
                      }
                    />
                  </div>
                ))}
                {mode === "edit" && (
                  <div className={styles.tableRowActions}>
                    <button
                      type="button"
                      className={`${styles.tableButton} ${styles.tableButtonDanger}`}
                      onClick={() => onChange?.(rows.filter((_, index) => index !== rowIndex))}
                    >
                      Remover
                    </button>
                  </div>
                )}
              </div>
            ))
          )}
        </div>
        {mode === "edit" && (
          <div className={styles.tableActions}>
            <button type="button" className={styles.tableButton} onClick={() => onChange?.([...rows, createBlankRow(field.columns)])}>
              Adicionar linha
            </button>
          </div>
        )}
      </div>
    </div>
  );
}

function normalizeRows(value: unknown, columns: RuntimeTableField["columns"]) {
  if (!Array.isArray(value)) {
    return [];
  }

  return value.map((row) => {
    if (row && typeof row === "object" && !Array.isArray(row)) {
      return row as Record<string, unknown>;
    }

    if (columns.length === 1) {
      return { [columns[0].key]: row };
    }

    return createBlankRow(columns);
  });
}

function createBlankRow(columns: RuntimeTableField["columns"]) {
  return columns.reduce<Record<string, unknown>>((acc, column) => {
    acc[column.key] = column.input === "checkbox" ? false : "";
    return acc;
  }, {});
}
