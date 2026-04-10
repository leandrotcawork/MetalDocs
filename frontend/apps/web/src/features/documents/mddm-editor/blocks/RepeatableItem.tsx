import { createReactBlockSpec } from "@blocknote/react";
import styles from "./RepeatableItem.module.css";

export const RepeatableItem = createReactBlockSpec(
  {
    type: "repeatableItem",
    propSchema: {
      title: { default: "" },
      style: { default: "bordered" },
    },
    content: "none",
  },
  {
    render: (props) => (
      <div
        className={styles.repeatableItem}
        data-mddm-block="repeatableItem"
        data-style={props.block.props.style || "bordered"}
      >
        <div className={styles.repeatableItemHeader}>
          <strong>{props.block.props.title || "Item"}</strong>
        </div>
      </div>
    ),
  },
);

