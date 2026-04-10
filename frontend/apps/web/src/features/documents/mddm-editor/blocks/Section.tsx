import { createReactBlockSpec } from "@blocknote/react";
import styles from "./Section.module.css";

export const Section = createReactBlockSpec(
  {
    type: "section",
    propSchema: {
      title: { default: "" },
      color: { default: "#6b1f2a" },
      locked: { default: true },
      optional: { default: false },
      variant: { default: "bar" },
      __template_block_id: { default: "" },
    },
    content: "none",
  },
  {
    render: (props) => (
      <div
        className={styles.section}
        data-mddm-block="section"
        data-variant={props.block.props.variant || "bar"}
        data-locked={props.block.props.locked}
      >
        <div className={styles.sectionHeader}>
          <span className={styles.sectionTitle}>
            {props.block.props.title || "Section"}
          </span>
          {props.block.props.optional ? (
            <span className={styles.optionalBadge}>Opcional</span>
          ) : null}
        </div>
      </div>
    ),
  },
);
