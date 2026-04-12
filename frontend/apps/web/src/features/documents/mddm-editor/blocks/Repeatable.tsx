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
    render: (props) => {
      const prefix = props.block.props.itemPrefix || "Item";
      return (
        <div className={styles.repeatable} data-mddm-block="repeatable">
          <div className={styles.repeatableHeader}>
            <strong className={styles.repeatableTitle}>
              {props.block.props.label || "Repeatable"}
            </strong>
            <span className={styles.repeatableMeta}>{prefix}</span>
          </div>
          {!props.block.props.locked && (
            <button
              type="button"
              className={styles.addItemButton}
              aria-label={`Adicionar ${prefix}`}
              onClick={() => {
                const newItem = {
                  type: "repeatableItem" as const,
                  props: { title: prefix, style: "bordered" },
                  children: [] as [],
                };
                props.editor.updateBlock(props.block, {
                  children: [...(props.block.children ?? []), newItem],
                });
              }}
            >
              + Adicionar {prefix}
            </button>
          )}
        </div>
      );
    },
  },
);
