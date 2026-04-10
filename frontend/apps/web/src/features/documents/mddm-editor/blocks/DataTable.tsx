import { createReactBlockSpec } from "@blocknote/react";
import styles from "./DataTable.module.css";

type Column = { key: string; label: string };

export function parseDataTableColumns(columnsJson: string): Column[] {
  try {
    const parsed = JSON.parse(columnsJson);
    if (!Array.isArray(parsed)) {
      return [];
    }

    const columns: Column[] = [];
    const seenKeys = new Set<string>();

    for (const column of parsed) {
      if (!column || typeof column !== "object") {
        continue;
      }

      const key = typeof (column as { key?: unknown }).key === "string"
        ? (column as { key: string }).key.trim()
        : "";
      const label = typeof (column as { label?: unknown }).label === "string"
        ? (column as { label: string }).label.trim()
        : "";

      if (!key || !label || seenKeys.has(key)) {
        continue;
      }

      seenKeys.add(key);
      columns.push({ key, label });
    }

    return columns;
  } catch {
    return [];
  }
}

export const DataTable = createReactBlockSpec(
  {
    type: "dataTable",
    propSchema: {
      label: { default: "" },
      columnsJson: { default: "[]" },
      locked: { default: true },
      minRows: { default: 0 },
      maxRows: { default: 500 },
      density: { default: "normal" },
      __template_block_id: { default: "" },
    },
    content: "none",
  },
  {
    render: (props) => {
      const columns = parseDataTableColumns(props.block.props.columnsJson);

      return (
        <div
          className={styles.dataTable}
          data-mddm-block="dataTable"
          data-density={props.block.props.density || "normal"}
        >
          <div className={styles.dataTableHeader}>
            <strong className={styles.tableLabel}>
              {props.block.props.label || "Data Table"}
            </strong>
            <span className={styles.tableMeta}>
              {columns.length} colunas
            </span>
          </div>
          {columns.length > 0 ? (
            <div
              className={styles.tableGrid}
              style={{
                gridTemplateColumns: `repeat(${columns.length}, minmax(0, 1fr))`,
              }}
            >
              {columns.map((column) => (
                <div key={column.key} className={styles.tableHeaderCell}>
                  {column.label}
                </div>
              ))}
            </div>
          ) : null}
          <button
            type="button"
            className={styles.addRowButton}
            disabled
            aria-label="Adicionar linha, indisponível no momento"
          >
            + Adicionar linha
          </button>
        </div>
      );
    },
  },
);
