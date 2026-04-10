import { createReactBlockSpec } from "@blocknote/react";
import styles from "./DataTableCell.module.css";

export const DataTableCell = createReactBlockSpec(
  {
    type: "dataTableCell",
    propSchema: {
      columnKey: { default: "" },
    },
    content: "inline",
  },
  {
    render: (props) => (
      <div
        className={styles.cell}
        data-mddm-block="dataTableCell"
        data-column-key={props.block.props.columnKey}
        role="cell"
      >
        <div ref={props.contentRef} className={styles.cellContent} />
      </div>
    ),
  },
);
