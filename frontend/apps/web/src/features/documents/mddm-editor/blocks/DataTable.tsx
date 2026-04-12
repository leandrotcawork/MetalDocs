import { createReactBlockSpec } from "@blocknote/react";
import styles from "./DataTable.module.css";
import { DataTableExternalHTML } from "../engine/external-html";
import { getEditorTokens } from "../engine/editor-tokens";

// createReactBlockSpec's types restrict content to "inline" | "none",
// but the BlockNote runtime fully supports "table" content at the lower level.
// We cast through unknown to bypass the type-level restriction.
const _dataTableSpec = createReactBlockSpec(
  {
    type: "dataTable" as const,
    propSchema: {
      label: { default: "" },
      locked: { default: false },
      density: { default: "normal" },
      __template_block_id: { default: "" },
    },
    content: "none" as "none",
  },
  {
    render: (props) => (
      <div
        className={styles.dataTable}
        data-mddm-block="dataTable"
        data-density={props.block.props.density || "normal"}
        data-locked={String(props.block.props.locked)}
      >
        <div className={styles.dataTableHeader}>
          <strong className={styles.tableLabel}>
            {props.block.props.label || "Data Table"}
          </strong>
        </div>
        <div className={styles.tableContainer} ref={(props as any).contentRef} />
      </div>
    ),
    toExternalHTML: (props) => (
      <DataTableExternalHTML
        tokens={getEditorTokens(props.editor)}
        label={props.block.props.label as string}
        tableContent={props.block.content}
      />
    ),
  },
);

// Re-export with the config patched to content:"table" so BlockNote renders
// the native table grid and enables Tab navigation / cell editing.
export const DataTable: () => (typeof _dataTableSpec extends () => infer S ? S : never) =
  () => {
    const spec = (_dataTableSpec as () => any)();
    spec.config = { ...spec.config, content: "table" };
    return spec;
  };
