import { useEffect, useMemo } from "react";
import { EditorContent, useEditor } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import type { RuntimeDocumentSchema } from "../../runtime/schemaRuntimeTypes";
import styles from "../DocumentCanvas.module.css";
import { fromEnvelope, toEnvelope } from "../rich/metaldocsRich";
import { resolveCanvasSlotBinding } from "../slotBindings";
import { readCanvasSlotValue, writeCanvasSlotValue } from "../slotValues";

type RichSlotProps = {
  path: string;
  schema: RuntimeDocumentSchema | null;
  values: Record<string, unknown>;
  onChange: (next: Record<string, unknown>) => void;
  readOnly?: boolean;
};

export function RichSlot({ path, schema, values, onChange, readOnly }: RichSlotProps) {
  const binding = resolveCanvasSlotBinding(schema, path);
  const value = readCanvasSlotValue(values, path);
  const content = useMemo(() => fromEnvelope(value), [value]);
  const label = binding?.label ?? prettifyPath(path);
  const description = binding?.description ?? "";
  const required = Boolean(binding?.required);

  const editor = useEditor({
    extensions: [StarterKit],
    content,
    editable: !readOnly,
    editorProps: {
      attributes: {
        class: styles.richEditorContent,
      },
    },
    onUpdate({ editor: tiptapEditor }) {
      if (readOnly) {
        return;
      }
      onChange(writeCanvasSlotValue(values, path, toEnvelope(tiptapEditor.getJSON() as Record<string, unknown>)));
    },
  });

  useEffect(() => {
    if (!editor) {
      return;
    }
    const nextContent = content;
    if (JSON.stringify(editor.getJSON()) !== JSON.stringify(nextContent)) {
      editor.commands.setContent(nextContent, false);
    }
  }, [content, editor]);

  return (
    <div className={styles.slot}>
      <div className={styles.slotHeader}>
        <div className={styles.slotLabelRow}>
          <span className={styles.slotLabel}>{label}</span>
          {required ? <span className={styles.slotRequired}>*</span> : null}
        </div>
        <span className={styles.slotPath}>{binding?.sectionKey ?? path}</span>
      </div>
      {description ? <div className={styles.slotDescription}>{description}</div> : null}
      <div className={styles.richEditorShell}>
        <div className={styles.richToolbar}>
          <ToolbarButton editor={editor} action="bold">
            B
          </ToolbarButton>
          <ToolbarButton editor={editor} action="italic">
            I
          </ToolbarButton>
          <ToolbarButton editor={editor} action="bulletList">
            Lista
          </ToolbarButton>
          <ToolbarButton editor={editor} action="orderedList">
            Numerada
          </ToolbarButton>
          <ToolbarButton editor={editor} action="blockquote">
            Citar
          </ToolbarButton>
          <ToolbarButton editor={editor} action="undo">
            Undo
          </ToolbarButton>
          <ToolbarButton editor={editor} action="redo">
            Redo
          </ToolbarButton>
        </div>
        <div className={styles.richEditorBody}>
          <EditorContent editor={editor} />
        </div>
      </div>
    </div>
  );
}

type ToolbarAction = "bold" | "italic" | "bulletList" | "orderedList" | "blockquote" | "undo" | "redo";

type ToolbarButtonProps = {
  editor: ReturnType<typeof useEditor>;
  action: ToolbarAction;
  children: string;
};

function ToolbarButton({ editor, action, children }: ToolbarButtonProps) {
  if (!editor) {
    return (
      <button type="button" className={styles.richToolbarButton} disabled>
        {children}
      </button>
    );
  }

  const active =
    (action === "bold" && editor.isActive("bold")) ||
    (action === "italic" && editor.isActive("italic")) ||
    (action === "bulletList" && editor.isActive("bulletList")) ||
    (action === "orderedList" && editor.isActive("orderedList")) ||
    (action === "blockquote" && editor.isActive("blockquote"));

  return (
    <button
      type="button"
      className={`${styles.richToolbarButton} ${active ? styles.richToolbarButtonActive : ""}`}
      onMouseDown={(event) => event.preventDefault()}
      onClick={() => {
        if (!editor) {
          return;
        }
        switch (action) {
          case "bold":
            editor.chain().focus().toggleBold().run();
            break;
          case "italic":
            editor.chain().focus().toggleItalic().run();
            break;
          case "bulletList":
            editor.chain().focus().toggleBulletList().run();
            break;
          case "orderedList":
            editor.chain().focus().toggleOrderedList().run();
            break;
          case "blockquote":
            editor.chain().focus().toggleBlockquote().run();
            break;
          case "undo":
            editor.chain().focus().undo().run();
            break;
          case "redo":
            editor.chain().focus().redo().run();
            break;
        }
      }}
      disabled={!editor.isEditable}
    >
      {children}
    </button>
  );
}

function prettifyPath(path: string): string {
  const lastSegment = path.split(".").pop() ?? path;
  return lastSegment
    .split(/[_-]+/)
    .join(" ")
    .replace(/([a-z])([A-Z])/g, "$1 $2")
    .trim()
    .replace(/^./, (char) => char.toUpperCase());
}
