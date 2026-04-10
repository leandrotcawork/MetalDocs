import { createReactBlockSpec } from "@blocknote/react";
import styles from "./Field.module.css";

export const Field = createReactBlockSpec(
  {
    type: "field",
    propSchema: {
      label: { default: "" },
      valueMode: { default: "inline", values: ["inline"] as const },
      locked: { default: true },
      hint: { default: "" },
      layout: { default: "grid" },
      __template_block_id: { default: "" },
    },
    content: "inline",
  },
  {
    render: (props) => (
      <div
        className={styles.field}
        data-mddm-block="field"
        data-layout={props.block.props.layout || "grid"}
        data-locked={props.block.props.locked}
      >
        <div className={styles.fieldLabel}>
          <span>{props.block.props.label || "Field"}</span>
          {props.block.props.hint ? (
            <small className={styles.hint}>{props.block.props.hint}</small>
          ) : null}
        </div>
        <div className={styles.fieldValue}>
          <div ref={props.contentRef} className={styles.fieldContent} />
        </div>
      </div>
    ),
  },
);
