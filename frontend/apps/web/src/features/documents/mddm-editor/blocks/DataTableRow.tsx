import { createReactBlockSpec } from "@blocknote/react";
import styles from "./DataTableRow.module.css";

export const DataTableRow = createReactBlockSpec(
  {
    type: "dataTableRow",
    propSchema: {},
    content: "none",
  },
  {
    render: () => (
      <div className={styles.row} data-mddm-block="dataTableRow" role="row" />
    ),
  },
);
