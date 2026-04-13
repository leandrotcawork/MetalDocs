import { blockHasType, defaultProps } from "@blocknote/core";
import { useCallback } from "react";
import { useBlockNoteEditor, useComponentsContext, useEditorState } from "@blocknote/react";
import { RiAlignLeft, RiAlignCenter, RiAlignRight } from "react-icons/ri";
import type { IconType } from "react-icons";

type TextAlignment = "left" | "center" | "right";

const icons: Record<TextAlignment, IconType> = {
  left: RiAlignLeft,
  center: RiAlignCenter,
  right: RiAlignRight,
};

const labels: Record<TextAlignment, string> = {
  left: "Align text left",
  center: "Align text center",
  right: "Align text right",
};

function getTiptap(editor: any) {
  return editor._tiptapEditor;
}

/** Resolve the current alignment and whether we're in a table cell. */
function resolveAlignState(editor: any) {
  const tiptap = getTiptap(editor);
  const state = tiptap.state;
  const $from = state.selection.$from;

  // Walk up the resolved position to find a tableCell/tableHeader ancestor.
  for (let depth = $from.depth; depth >= 0; depth--) {
    const node = $from.node(depth);
    if (node.type.name === "tableCell" || node.type.name === "tableHeader") {
      return {
        mode: "cell" as const,
        cellDepth: depth,
        textAlignment: (node.attrs.textAlignment as TextAlignment) || "left",
      };
    }
  }

  // Not inside a table cell — fall back to block-level alignment.
  const block = editor.getTextCursorPosition().block;
  if (
    blockHasType(block, editor, block.type, {
      textAlignment: defaultProps.textAlignment,
    })
  ) {
    return {
      mode: "block" as const,
      block,
      textAlignment: ((block.props as any).textAlignment as TextAlignment) || "left",
    };
  }

  return undefined;
}

/**
 * Custom TextAlignButton that fixes cursor-jumping in native table cells.
 *
 * BlockNote's built-in TextAlignButton uses `editor.updateBlock()` +
 * `editor.setTextCursorPosition(block)` for tables, which resets the cursor
 * to the first cell. This version detects when the cursor is inside a table
 * cell and applies alignment directly on the ProseMirror `tableCell` node
 * via `setNodeMarkup`, which preserves the cursor position.
 */
export function MddmTextAlignButton({ textAlignment }: { textAlignment: TextAlignment }) {
  const Components = useComponentsContext()!;
  const editor = useBlockNoteEditor();

  const state = useEditorState({
    editor,
    selector: ({ editor }) => {
      if (!editor.isEditable) return undefined;
      return resolveAlignState(editor);
    },
  });

  const setAlign = useCallback(
    (alignment: TextAlignment) => {
      if (!state) return;

      editor.focus();

      if (state.mode === "cell") {
        // Apply alignment directly on the tableCell PM node.
        // This preserves cursor position — no setTextCursorPosition needed.
        const tiptap = getTiptap(editor);
        const pmState = tiptap.state;
        const $from = pmState.selection.$from;

        for (let depth = $from.depth; depth >= 0; depth--) {
          const node = $from.node(depth);
          if (node.type.name === "tableCell" || node.type.name === "tableHeader") {
            const pos = $from.before(depth);
            const tr = pmState.tr.setNodeMarkup(pos, undefined, {
              ...node.attrs,
              textAlignment: alignment,
            });
            tiptap.view.dispatch(tr);
            break;
          }
        }
      } else if (state.mode === "block") {
        // Standard block-level alignment through BlockNote API.
        editor.updateBlock(state.block, {
          props: { textAlignment: alignment },
        } as any);
      }
    },
    [editor, state],
  );

  if (!state) return null;

  const Icon = icons[textAlignment];
  return (
    <Components.FormattingToolbar.Button
      className="bn-button"
      data-test={`alignText${textAlignment.charAt(0).toUpperCase() + textAlignment.slice(1)}`}
      onClick={() => setAlign(textAlignment)}
      isSelected={state.textAlignment === textAlignment}
      label={labels[textAlignment]}
      mainTooltip={labels[textAlignment]}
      icon={<Icon />}
    />
  );
}
