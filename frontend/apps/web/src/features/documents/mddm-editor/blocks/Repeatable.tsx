import { createReactBlockSpec } from "@blocknote/react";
import { RepeatableExternalHTML } from "../engine/external-html";
import { getEditorTokens } from "../engine/editor-tokens";
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
      const maxItems = props.block.props.maxItems ?? 100;
      const currentChildren = props.block.children ?? [];
      const canAddItem = !props.block.props.locked && currentChildren.length < maxItems;

      return (
        <div className={styles.repeatable} data-mddm-block="repeatable" data-locked={props.block.props.locked}>
          <div className={styles.repeatableHeader}>
            <strong className={styles.repeatableTitle}>
              {props.block.props.label || "Repeatable"}
            </strong>
            <span className={styles.repeatableMeta}>{prefix}</span>
          </div>
          {canAddItem && (
            <button
              type="button"
              className={styles.addItemButton}
              aria-label={`Adicionar ${prefix}`}
              onClick={() => {
                // Read current block state at click time to avoid stale closure
                const currentBlock = props.editor.getBlock(props.block.id);
                const freshChildren = currentBlock?.children ?? [];
                const newItem = {
                  type: "repeatableItem" as const,
                  props: { title: `${prefix} ${freshChildren.length + 1}`, style: "bordered" },
                  children: [] as [],
                };
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                (props.editor as any).updateBlock(props.block, {
                  children: [...freshChildren, newItem],
                });
              }}
            >
              + Adicionar {prefix}
            </button>
          )}
        </div>
      );
    },
    toExternalHTML: (props) => (
      <RepeatableExternalHTML
        tokens={getEditorTokens(props.editor)}
        label={props.block.props.label as string}
      />
    ),
  },
);
