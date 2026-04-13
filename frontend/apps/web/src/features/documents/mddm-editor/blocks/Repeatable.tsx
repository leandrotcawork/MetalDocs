import { createReactBlockSpec } from "@blocknote/react";
import { RepeatableExternalHTML } from "../engine/external-html";
import { getEditorTokens } from "../engine/editor-tokens";
import { interpretRepeatable } from "../engine/layout-interpreter/repeatable-interpreter";
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
      styleJson: { default: "{}" },
      capabilitiesJson: { default: "{}" },
    },
    content: "none",
  },
  {
    render: (props) => {
      const tokens = getEditorTokens(props.editor);
      const vm = interpretRepeatable(
        { props: props.block.props as Record<string, unknown>, children: props.block.children },
        tokens,
      );

      return (
        <div className={styles.repeatable} data-mddm-block="repeatable" data-locked={vm.locked}>
          <div className={styles.repeatableHeader}>
            <strong className={styles.repeatableTitle}>{vm.label || "Repeatable"}</strong>
            <span className={styles.repeatableMeta}>{vm.itemPrefix}</span>
          </div>
          {vm.canAddItems && (
            <button
              type="button"
              className={styles.addItemButton}
              aria-label={`Adicionar ${vm.itemPrefix}`}
              onClick={() => {
                // Read current block state at click time to avoid stale closure
                const currentBlock = props.editor.getBlock(props.block.id);
                const freshChildren = currentBlock?.children ?? [];
                const newItem = {
                  type: "repeatableItem" as const,
                  props: { title: `${vm.itemPrefix} ${freshChildren.length + 1}`, style: "bordered" },
                  children: [] as [],
                };
                // eslint-disable-next-line @typescript-eslint/no-explicit-any
                (props.editor as any).updateBlock(props.block, {
                  children: [...freshChildren, newItem],
                });
              }}
            >
              + Adicionar {vm.itemPrefix}
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
