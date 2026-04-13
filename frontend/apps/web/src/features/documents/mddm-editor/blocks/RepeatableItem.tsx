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

/**
 * Returns { sectionNumber, itemNumber } for a repeatableItem so it can be
 * displayed as "5.1 Etapa 1" instead of "1. Etapa 1".
 *
 * Walks root-level sections, then their direct repeatable children, so the
 * section number always reflects which top-level section owns this item.
 */
function resolveItemContext(
  document: any[],
  itemId: string,
): { sectionNumber: number; itemNumber: number } {
  const rootSections = document.filter((b) => b.type === "section");

  for (let si = 0; si < rootSections.length; si++) {
    for (const child of rootSections[si].children ?? []) {
      if (child.type !== "repeatable" || !child.children) continue;
      const items = child.children.filter((c: any) => c.type === "repeatableItem");
      const idx = items.findIndex((c: any) => c.id === itemId);
      if (idx >= 0) return { sectionNumber: si + 1, itemNumber: idx + 1 };
    }
  }

  // Fallback: flat item index only (no section prefix)
  const itemNumber = findItemIndex(document, itemId);
  return { sectionNumber: 0, itemNumber: itemNumber > 0 ? itemNumber : 1 };
}

export const RepeatableItem = createReactBlockSpec(
  {
    type: "repeatableItem",
    propSchema: {
      title: { default: "" },
      style: { default: "bordered" },
      locked: { default: false },
      styleJson: { default: "{}" },
      capabilitiesJson: { default: "{}" },
    },
    content: "none",
  },
  {
    render: (props) => {
      const { sectionNumber, itemNumber } = useMemo(
        () => resolveItemContext(props.editor.document, props.block.id ?? ""),
        // eslint-disable-next-line react-hooks/exhaustive-deps
        [props.editor.document, props.block.id],
      );

      const prefix = sectionNumber > 0 ? `${sectionNumber}.${itemNumber}` : `${itemNumber}.`;
      const displayTitle = props.block.props.title
        ? `${prefix} ${props.block.props.title}`
        : `Item ${prefix}`;

      return (
        <div
          className={styles.repeatableItem}
          data-mddm-block="repeatableItem"
          data-style={props.block.props.style || "bordered"}
          data-locked={props.block.props.locked}
        >
          <div className={styles.repeatableItemHeader}>
            <strong>{displayTitle}</strong>
          </div>
        </div>
      );
    },
    toExternalHTML: (props) => {
      const { sectionNumber, itemNumber } = resolveItemContext(
        props.editor.document,
        props.block.id ?? "",
      );
      return (
        <RepeatableItemExternalHTML
          tokens={getEditorTokens(props.editor)}
          title={props.block.props.title as string}
          sectionNumber={sectionNumber}
          itemNumber={itemNumber}
        />
      );
    },
  },
);
