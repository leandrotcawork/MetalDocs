import { createReactBlockSpec } from "@blocknote/react";
import styles from "./FieldGroup.module.css";

export const FieldGroup = createReactBlockSpec(
  {
    type: "fieldGroup",
    propSchema: {
      columns: { default: 1, values: [1, 2] as const },
      locked: { default: true },
      __template_block_id: { default: "" },
    },
    content: "none",
  },
  {
    render: (props) => (
      <div
        className={styles.fieldGroup}
        data-mddm-block="fieldGroup"
        data-columns={props.block.props.columns}
        data-locked={props.block.props.locked}
      />
    ),
  },
);
