import { createReactBlockSpec } from "@blocknote/react";
import { useMemo } from "react";
import { RepeatableItemExternalHTML } from "../engine/external-html";
import { defaultLayoutTokens } from "../engine/layout-ir";
import styles from "./RepeatableItem.module.css";

function findItemIndex(document: any[], itemId: string): number {
  for (const block of document) {
    if (block.children) {
      const idx = block.children.findIndex((c: any) => c.id === itemId);
      if (idx >= 0) return idx + 1;
      // Recurse into children
      const nested = findItemIndex(block.children, itemId);
      if (nested > 0) return nested;
    }
  }
  return 1;
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
        () => findItemIndex(props.editor.document, props.block.id ?? ""),
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
        tokens={defaultLayoutTokens}
        title={props.block.props.title as string}
        itemNumber={1}
      />
    ),
  },
);
