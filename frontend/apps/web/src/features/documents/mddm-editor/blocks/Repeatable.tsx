import { createReactBlockSpec } from "@blocknote/react";
import styles from "./Repeatable.module.css";

export const Repeatable = createReactBlockSpec(
  {
    type: "repeatable",
    propSchema: {
      label: { default: "" },
      itemPrefix: { default: "Item" },
      locked: { default: true },
      minItems: { default: 0 },
      maxItems: { default: 100 },
      __template_block_id: { default: "" },
    },
    content: "none",
  },
  {
    render: (props) => (
      <div className={styles.repeatable} data-mddm-block="repeatable">
        <div className={styles.repeatableHeader}>
          <strong className={styles.repeatableTitle}>
            {props.block.props.label || "Repeatable"}
          </strong>
          <span className={styles.repeatableMeta}>
            {props.block.props.itemPrefix || "Item"}
          </span>
        </div>
      </div>
    ),
  },
);
