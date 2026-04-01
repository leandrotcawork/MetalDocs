import { useEffect, useMemo } from "react";
import type { Editor } from "@tiptap/react";
import { Color } from "@tiptap/extension-color";
import Image from "@tiptap/extension-image";
import Table from "@tiptap/extension-table";
import TableCell from "@tiptap/extension-table-cell";
import TableHeader from "@tiptap/extension-table-header";
import TableRow from "@tiptap/extension-table-row";
import { EditorContent, useEditor } from "@tiptap/react";
import StarterKit from "@tiptap/starter-kit";
import { TextStyle } from "@tiptap/extension-text-style";
import type { RuntimeRichField } from "../schemaRuntimeTypes";
import editorStyles from "../DynamicEditor.module.css";
import styles from "../RichField.module.css";

type RuntimeMode = "edit" | "preview";

type RichFieldProps = {
  field: RuntimeRichField;
  value: unknown;
  mode: RuntimeMode;
  onChange?: (next: unknown) => void;
};

const EMPTY_HTML = "<p></p>";

export function RichField({ field, value, mode, onChange }: RichFieldProps) {
  const content = useMemo(() => normalizeRichValue(value), [value]);
  const label = field.label ?? field.key;

  if (mode === "preview") {
    return (
      <div className={styles.richRoot}>
        <div className={editorStyles.fieldLabel}>
          <span>{label}</span>
          {field.required && <span className={editorStyles.requiredMark}>*</span>}
        </div>
        {field.description && <div className={editorStyles.fieldDescription}>{field.description}</div>}
        <div className={styles.editorShell}>
          <div className={styles.previewBody} dangerouslySetInnerHTML={{ __html: content || EMPTY_HTML }} />
        </div>
      </div>
    );
  }

  return <RichEditor field={field} value={content} onChange={onChange} label={label} />;
}

function RichEditor({
  value,
  onChange,
  field,
  label,
}: Pick<RichFieldProps, "value" | "onChange" | "field"> & { label: string }) {
  const editor = useEditor({
    extensions: [
      StarterKit,
      TextStyle,
      Color,
      Image.configure({ inline: true }),
      Table.configure({ resizable: true }),
      TableRow,
      TableHeader,
      TableCell,
    ],
    content: normalizeRichValue(value),
    editorProps: {
      attributes: {
        class: "tiptap-editor",
      },
    },
    onUpdate({ editor: tiptapEditor }) {
      onChange?.(tiptapEditor.getHTML());
    },
  });

  useEffect(() => {
    if (!editor) return;
    const nextValue = normalizeRichValue(value);
    if (editor.getHTML() !== nextValue) {
      editor.commands.setContent(nextValue, false);
    }
  }, [editor, value]);

  return (
    <div className={styles.richRoot}>
      <div className={editorStyles.fieldLabel}>
        <span>{label}</span>
        {field.required && <span className={editorStyles.requiredMark}>*</span>}
      </div>
      {field.description && <div className={editorStyles.fieldDescription}>{field.description}</div>}
      <div className={styles.editorShell}>
        <div className={styles.toolbar}>
          <div className={styles.toolbarGroup}>
            <ToolbarButton editor={editor} command="bold">
              B
            </ToolbarButton>
            <ToolbarButton editor={editor} command="italic">
              I
            </ToolbarButton>
            <ToolbarButton editor={editor} command="strike">
              S
            </ToolbarButton>
          </div>
          <div className={styles.toolbarGroup}>
            <ToolbarButton editor={editor} command="paragraph">
              Paragrafo
            </ToolbarButton>
            <ToolbarButton editor={editor} command="heading" level={1}>
              H1
            </ToolbarButton>
            <ToolbarButton editor={editor} command="heading" level={2}>
              H2
            </ToolbarButton>
            <ToolbarButton editor={editor} command="heading" level={3}>
              H3
            </ToolbarButton>
          </div>
          <div className={styles.toolbarGroup}>
            <ToolbarButton editor={editor} command="bulletList">
              Lista
            </ToolbarButton>
            <ToolbarButton editor={editor} command="orderedList">
              Numerada
            </ToolbarButton>
            <ToolbarButton editor={editor} command="blockquote">
              Citar
            </ToolbarButton>
          </div>
          <div className={styles.toolbarGroup}>
            <ToolbarButton editor={editor} command="insertTable">
              Tabela
            </ToolbarButton>
            <ToolbarButton editor={editor} command="image">
              Imagem
            </ToolbarButton>
            <ToolbarButton editor={editor} command="undo">
              Undo
            </ToolbarButton>
            <ToolbarButton editor={editor} command="redo">
              Redo
            </ToolbarButton>
          </div>
          <div className={styles.toolbarGroup}>
            <input
              className={styles.colorInput}
              type="color"
              aria-label="Cor do texto"
              onChange={(event) => {
                if (!editor) return;
                editor.chain().focus().setColor(event.target.value).run();
              }}
            />
            <ToolbarButton editor={editor} command="clearColor">
              Cor padrao
            </ToolbarButton>
          </div>
        </div>
        <div className={styles.editorBody}>
          <EditorContent editor={editor} />
        </div>
      </div>
    </div>
  );
}

type ToolbarCommand =
  | "bold"
  | "italic"
  | "strike"
  | "paragraph"
  | "heading"
  | "bulletList"
  | "orderedList"
  | "blockquote"
  | "insertTable"
  | "image"
  | "undo"
  | "redo"
  | "clearColor";

type ToolbarButtonProps = {
  editor: Editor | null;
  command: ToolbarCommand;
  level?: 1 | 2 | 3 | 4 | 5 | 6;
  children: string;
};

function ToolbarButton({ editor, command, level, children }: ToolbarButtonProps) {
  if (!editor) {
    return (
      <button type="button" className={styles.toolbarButton} disabled>
        {children}
      </button>
    );
  }

  const active =
    (command === "bold" && editor.isActive("bold")) ||
    (command === "italic" && editor.isActive("italic")) ||
    (command === "strike" && editor.isActive("strike")) ||
    (command === "paragraph" && editor.isActive("paragraph")) ||
    (command === "heading" && level ? editor.isActive("heading", { level }) : false) ||
    (command === "bulletList" && editor.isActive("bulletList")) ||
    (command === "orderedList" && editor.isActive("orderedList")) ||
    (command === "blockquote" && editor.isActive("blockquote"));

  return (
    <button
      type="button"
      className={`${styles.toolbarButton} ${active ? styles.toolbarButtonActive : ""}`}
      onMouseDown={(event) => event.preventDefault()}
      onClick={() => {
        if (!editor) return;
        switch (command) {
          case "bold":
            editor.chain().focus().toggleBold().run();
            break;
          case "italic":
            editor.chain().focus().toggleItalic().run();
            break;
          case "strike":
            editor.chain().focus().toggleStrike().run();
            break;
          case "paragraph":
            editor.chain().focus().setParagraph().run();
            break;
          case "heading":
            editor.chain().focus().toggleHeading({ level: level ?? 1 }).run();
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
          case "insertTable":
            editor.chain().focus().insertTable({ rows: 2, cols: 2, withHeaderRow: true }).run();
            break;
          case "image": {
            const src = window.prompt("URL da imagem");
            if (src) {
              editor.chain().focus().setImage({ src }).run();
            }
            break;
          }
          case "undo":
            editor.chain().focus().undo().run();
            break;
          case "redo":
            editor.chain().focus().redo().run();
            break;
          case "clearColor":
            editor.chain().focus().unsetColor().run();
            break;
        }
      }}
    >
      {children}
    </button>
  );
}

function normalizeRichValue(value: unknown) {
  if (typeof value === "string") {
    return value.trim() ? value : EMPTY_HTML;
  }

  if (value && typeof value === "object" && !Array.isArray(value)) {
    const richRecord = value as Record<string, unknown>;
    if (typeof richRecord.html === "string") {
      return richRecord.html;
    }
    if (typeof richRecord.content === "string") {
      return richRecord.content;
    }
  }

  return EMPTY_HTML;
}
