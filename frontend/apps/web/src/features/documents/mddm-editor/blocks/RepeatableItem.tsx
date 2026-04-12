import { createReactBlockSpec } from "@blocknote/react";
import { useMemo } from "react";
import { RepeatableItemExternalHTML } from "../engine/external-html";
import { getEditorTokens } from "../engine/editor-tokens";
import styles from "./RepeatableItem.module.css";

export function findItemIndex(document: any[], itemId: string): number {
  for (const block of document) {
    if (block.type === "repeatable" && block.children) {
      const repeatableItems = block.children.filter((child: any) => child.type === "repeatableItem");
      const idx = repeatableItems.findIndex((child: any) => child.id === itemId);
      if (idx >= 0) return idx + 1;
    }

    if (block.children) {
      const nested = findItemIndex(block.children, itemId);
      if (nested > 0) return nested;
    }
  }
  return 0;
}

function resolveItemIndex(document: any[], itemId: string): number {
  const itemIndex = findItemIndex(document, itemId);
  return itemIndex > 0 ? itemIndex : 1;
}

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
    render: (props) => {
      const itemNumber = useMemo(
        () => resolveItemIndex(props.editor.document, props.block.id ?? ""),
        // eslint-disable-next-line react-hooks/exhaustive-deps
        [props.editor.document, props.block.id],
      );

      const displayTitle = props.block.props.title
        ? `${itemNumber}. ${props.block.props.title}`
        : `Item ${itemNumber}`;

      return (
        <div
          className={styles.repeatableItem}
          data-mddm-block="repeatableItem"
          data-style={props.block.props.style || "bordered"}
        >
          <div className={styles.repeatableItemHeader}>
            <strong>{displayTitle}</strong>
          </div>
        </div>
      );
    },
    toExternalHTML: (props) => (
      <RepeatableItemExternalHTML
        tokens={getEditorTokens(props.editor)}
        title={props.block.props.title as string}
        itemNumber={resolveItemIndex(props.editor.document, props.block.id ?? "")}
      />
    ),
  },
);
