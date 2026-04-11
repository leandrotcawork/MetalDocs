import { createReactBlockSpec } from "@blocknote/react";
import styles from "./DataTableCell.module.css";
import { DataTableCellExternalHTML } from "../engine/external-html";
import { defaultLayoutTokens } from "../engine/layout-ir";

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
    toExternalHTML: ({ contentRef }) => (
      <DataTableCellExternalHTML tokens={defaultLayoutTokens}>
        <span ref={(el: HTMLSpanElement | null) => contentRef(el)} />
      </DataTableCellExternalHTML>
    ),
  },
);
